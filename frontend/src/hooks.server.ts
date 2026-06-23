import type { Handle } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { PB_COOKIE_NAME } from '$lib/pocketbase';
import { generateAccentCSSVars } from '$lib/colors';
import { generateFontCSSVars, getGoogleFontsUrl, DEFAULT_FONT_PACK } from '$lib/fonts';

export const handle: Handle = async ({ event, resolve }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	event.locals.pb = new PocketBase(pbUrl);

	const cookie = event.request.headers.get('cookie') || '';
	event.locals.pb.authStore.loadFromCookie(cookie, PB_COOKIE_NAME);

	// Snapshot the token so we can detect if auth changed during request handling
	const initialToken = event.locals.pb.authStore.token;

	// Fetch accent color and font pack for SSR injection (eliminates flash on first paint)
	let accentCSSVars = '';
	let fontCSSVars = '';
	let fontPack = '';
	let operatorFontPack = '';
	try {
		const homepageRes = await fetch(`${pbUrl}/api/homepage`, {
			headers: { 'X-Internal': 'true' }
		});
		if (homepageRes.ok) {
			const homepage = await homepageRes.json();
			accentCSSVars = generateAccentCSSVars(
				homepage.profile?.accent_color,
				homepage.profile?.custom_hex_color
			);
			operatorFontPack = homepage.profile?.font_pack || '';
		}
	} catch { /* silent - fallback to client-side application */ }

	// Fetch the opt-in visual design mode. "classic" (the default for every
	// existing instance) emits no attribute, so the rendered HTML is byte-identical
	// to before; only "soft-premium" adds data-design so the warm token layer applies.
	let design = 'classic';
	try {
		const settingsRes = await fetch(`${pbUrl}/api/site-settings`, {
			headers: { 'X-Internal': 'true' }
		});
		if (settingsRes.ok) {
			const settings = await settingsRes.json();
			if (settings.design === 'soft-premium') design = 'soft-premium';
		}
	} catch { /* silent - fallback to classic */ }

	// Soft Premium ships its own font pack (Hanken Grotesk + Newsreader) as the
	// default, but an operator who explicitly picked a pack keeps it — the classic
	// escape hatch. Emitted inline below, which (unlike a :root[data-design] rule)
	// correctly overrides the static defaults.
	const effectiveFontPack =
		design === 'soft-premium' && (!operatorFontPack || operatorFontPack === DEFAULT_FONT_PACK)
			? 'soft-premium'
			: operatorFontPack;
	fontCSSVars = generateFontCSSVars(effectiveFontPack);
	fontPack = effectiveFontPack;

	const response = await resolve(event, {
		transformPageChunk: ({ html }) => {
			// Combine accent and font CSS vars into a single inline style on <html>
			const varParts: string[] = [];
			if (accentCSSVars) {
				varParts.push(
					accentCSSVars
						.replace(/^:root\s*\{/, '')
						.replace(/\}\s*$/, '')
						.trim()
						.replace(/\s+/g, ' ')
						.replace(/;?\s*$/, ';')
				);
			}
			if (fontCSSVars) {
				varParts.push(
					fontCSSVars
						.replace(/^:root\s*\{/, '')
						.replace(/\}\s*$/, '')
						.trim()
						.replace(/\s+/g, ' ')
						.replace(/;?\s*$/, ';')
				);
			}

			let result = html;

			// Compose all <html> attributes into a single replace so the design
			// attribute and the inline style vars both land. classic => no attribute.
			const htmlAttrs: string[] = [];
			if (design === 'soft-premium') {
				htmlAttrs.push('data-design="soft-premium"');
			}
			if (varParts.length > 0) {
				htmlAttrs.push(`style="${varParts.join(' ')}"`);
			}
			if (htmlAttrs.length > 0) {
				result = result.replace('<html lang="en">', `<html lang="en" ${htmlAttrs.join(' ')}>`);
			}

			// Inject Google Fonts link for non-default font packs
			if (fontPack && fontPack !== DEFAULT_FONT_PACK) {
				const fontsUrl = getGoogleFontsUrl(fontPack);
				result = result.replace(
					'</head>',
					`<link rel="stylesheet" href="${fontsUrl}">\n</head>`
				);
			}

			return result;
		}
	});

	// Strip oversized Link headers from HTML responses to prevent proxy failures.
	// Only for HTML - API endpoints may use Link for pagination (rel="next").
	// See: https://github.com/sveltejs/kit/issues/6819
	const contentType = response.headers.get('content-type') || '';
	if (contentType.includes('text/html')) {
		response.headers.delete('link');
	}

	// Only set the auth cookie if auth state changed during this request.
	// Blindly re-exporting the cookie on every response causes redirect loops:
	// the client clears a stale cookie, but the server response re-sets it
	// because it loaded the stale token from the incoming request.
	const currentToken = event.locals.pb.authStore.token;
	if (currentToken !== initialToken) {
		// Determine cookie security from the actual request protocol, not NODE_ENV.
		// NODE_ENV is always 'production' in Docker, but the connection may be HTTP
		// (e.g., local access, behind a reverse proxy that terminates TLS upstream).
		//
		// Priority: X-Forwarded-Proto (set by Caddy) > event.url.protocol.
		// Note: event.url.protocol requires PROTOCOL_HEADER env to be set in adapter-node,
		// otherwise it defaults to 'https:'. We set PROTOCOL_HEADER in the Dockerfile,
		// but X-Forwarded-Proto is the most reliable signal.
		const forwardedProto = event.request.headers.get('x-forwarded-proto');
		const isSecure = forwardedProto
			? forwardedProto === 'https'
			: event.url.protocol === 'https:';
		const exportedCookie = event.locals.pb.authStore.exportToCookie({
			httpOnly: false,
			secure: isSecure,
			sameSite: 'Lax',
			path: '/'
		}, PB_COOKIE_NAME);

		response.headers.append('set-cookie', exportedCookie);
	}

	return response;
};

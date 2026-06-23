import type { Handle } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { PB_COOKIE_NAME } from '$lib/pocketbase';
import { generateAccentCSSVars, SOFT_PREMIUM_DEFAULT_ACCENT } from '$lib/colors';
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
	let operatorAccent = '';
	let operatorCustomHex = '';
	try {
		const homepageRes = await fetch(`${pbUrl}/api/homepage`, {
			headers: { 'X-Internal': 'true' }
		});
		if (homepageRes.ok) {
			const homepage = await homepageRes.json();
			operatorAccent = homepage.profile?.accent_color || '';
			operatorCustomHex = homepage.profile?.custom_hex_color || '';
			operatorFontPack = homepage.profile?.font_pack || '';
		}
	} catch { /* silent - fallback to client-side application */ }

	// Soft Premium is the only design — always on. The warm token layer keys off
	// the data-design attribute (injected below), so it is hardcoded here rather
	// than fetched/branched on a site setting.
	const design = 'soft-premium';

	// Fonts are fully governed by the operator's font-pack pick. Whatever they
	// selected is used; if they never chose one, fall back to the Soft Premium
	// default pack (DEFAULT_FONT_PACK). Emitted inline below, which (unlike a
	// :root rule) correctly overrides the static defaults in app.css.
	const effectiveFontPack = operatorFontPack || DEFAULT_FONT_PACK;
	fontCSSVars = generateFontCSSVars(effectiveFontPack);
	fontPack = effectiveFontPack;

	// Accent: Soft Premium defaults to a warm terracotta when the operator hasn't
	// set an accent, so the accent harmonizes with the stone surfaces. An operator
	// accent (named or custom hex) always wins.
	if (!operatorAccent && !operatorCustomHex) {
		accentCSSVars = generateAccentCSSVars(null, SOFT_PREMIUM_DEFAULT_ACCENT);
	} else {
		accentCSSVars = generateAccentCSSVars(operatorAccent, operatorCustomHex);
	}

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
			// attribute and the inline style vars both land. Soft Premium is always
			// on, so data-design is always emitted.
			const htmlAttrs: string[] = ['data-design="soft-premium"'];
			if (varParts.length > 0) {
				htmlAttrs.push(`style="${varParts.join(' ')}"`);
			}
			result = result.replace('<html lang="en">', `<html lang="en" ${htmlAttrs.join(' ')}>`);

			// Inject the Google Fonts link for the resolved pack. app.html statically
			// loads only the editorial pack, so the chosen pack's fonts (including the
			// Soft Premium default — Hanken/Newsreader) must always be loaded here or
			// they'd be missing.
			if (fontPack) {
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

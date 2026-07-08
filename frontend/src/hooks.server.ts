import type { Handle, HandleFetch } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { PB_COOKIE_NAME } from '$lib/pocketbase';
import { generateAccentCSSVars, generateTextInkCSSVars, SOFT_PREMIUM_DEFAULT_ACCENT } from '$lib/colors';
import { generateFontCSSVars, getGoogleFontsUrl, DEFAULT_FONT_PACK } from '$lib/fonts';
import { fetchWith503Retry } from '$lib/server/internal-api-retry';

export const handle: Handle = async ({ event, resolve }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	event.locals.pb = new PocketBase(pbUrl);

	const cookie = event.request.headers.get('cookie') || '';
	event.locals.pb.authStore.loadFromCookie(cookie, PB_COOKIE_NAME);
	event.locals.siteSettings = {
		faviconUrl: null,
		customCSS: null,
		defaultLocale: null,
		showAvatar: true,
		defaultThemeMode: 'system',
		design: 'classic',
	};

	// Snapshot the token so we can detect if auth changed during request handling
	const initialToken = event.locals.pb.authStore.token;

	// Fetch accent color and font pack for SSR injection (eliminates flash on first paint)
	let accentCSSVars = '';
	let fontCSSVars = '';
	let fontPack = '';
	let operatorFontPack = '';
	let operatorAccent = '';
	let operatorCustomHex = '';
	let operatorTextColor = '';
	let design: App.SiteSettings['design'] = 'classic';
	try {
		const settingsRes = await fetch(`${pbUrl}/api/site-settings`, {
			headers: { 'X-Internal': 'true' },
		});
		if (settingsRes.ok) {
			const settings = await settingsRes.json();
			design = settings.design === 'soft-premium' ? 'soft-premium' : 'classic';
			event.locals.siteSettings = {
				faviconUrl: settings.favicon || null,
				customCSS: settings.custom_css || null,
				defaultLocale: settings.default_locale || null,
				showAvatar: settings.show_avatar !== false,
				defaultThemeMode: settings.default_theme_mode || 'system',
				design,
			};
		}
	} catch {
		/* silent - default to classic */
	}

	try {
		const homepageRes = await fetch(`${pbUrl}/api/homepage`, {
			headers: { 'X-Internal': 'true' },
		});
		if (homepageRes.ok) {
			const homepage = await homepageRes.json();
			operatorAccent = homepage.profile?.accent_color || '';
			operatorCustomHex = homepage.profile?.custom_hex_color || '';
			operatorFontPack = homepage.profile?.font_pack || '';
			operatorTextColor = homepage.profile?.text_color || '';
		}
	} catch {
		/* silent - fallback to client-side application */
	}

	// Text (font) color: derive a per-mode AAA-clamped ink from the operator's
	// chosen hue and inject --text-ink (light) + --text-ink-dark (dark). Empty ⇒
	// '' ⇒ nothing injected ⇒ default render unchanged. The clamp lives in
	// colors.ts (deriveTextInk) and is exercised by the AAA proof test.
	const textInkCSSVars = generateTextInkCSSVars(operatorTextColor);

	// Fonts are fully governed by the operator's font-pack pick. Whatever they
	// selected is used. If they never chose one, Classic keeps the static
	// Lora/Jakarta defaults from app.css; Soft Premium gets the redesigned default.
	const effectiveFontPack = operatorFontPack || (design === 'soft-premium' ? DEFAULT_FONT_PACK : '');
	if (effectiveFontPack) {
		fontCSSVars = generateFontCSSVars(effectiveFontPack);
		fontPack = effectiveFontPack;
	}

	// Accent: Soft Premium defaults to warm terracotta only when the operator has
	// not set an accent. Classic keeps the stock sky palette from app.css.
	if (design === 'soft-premium' && !operatorAccent && !operatorCustomHex) {
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
						.replace(/;?\s*$/, ';'),
				);
			}
			if (fontCSSVars) {
				varParts.push(
					fontCSSVars
						.replace(/^:root\s*\{/, '')
						.replace(/\}\s*$/, '')
						.trim()
						.replace(/\s+/g, ' ')
						.replace(/;?\s*$/, ';'),
				);
			}
			if (textInkCSSVars) {
				varParts.push(
					textInkCSSVars
						.replace(/^:root\s*\{/, '')
						.replace(/\}\s*$/, '')
						.trim()
						.replace(/\s+/g, ' ')
						.replace(/;?\s*$/, ';'),
				);
			}

			let result = html;

			// Compose all <html> attributes into a single replace so the design
			// attribute and the inline style vars both land.
			const htmlAttrs: string[] = [];
			if (design === 'soft-premium') {
				htmlAttrs.push('data-design="soft-premium"');
			}
			// data-text-ink gates the scoped text-color rules in app.css. Present
			// ONLY when the operator set a color, so the default render carries no
			// text-ink rules at all (byte-identical to today).
			if (textInkCSSVars) {
				htmlAttrs.push('data-text-ink');
			}
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
				result = result.replace('</head>', `<link rel="stylesheet" href="${fontsUrl}">\n</head>`);
			}

			return result;
		},
	});
	const headers = new Headers(response.headers);

	// Strip oversized Link headers from HTML responses to prevent proxy failures.
	// Only for HTML - API endpoints may use Link for pagination (rel="next").
	// See: https://github.com/sveltejs/kit/issues/6819
	const contentType = headers.get('content-type') || '';
	if (contentType.includes('text/html')) {
		headers.delete('link');
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
		const isSecure = forwardedProto ? forwardedProto === 'https' : event.url.protocol === 'https:';
		const exportedCookie = event.locals.pb.authStore.exportToCookie(
			{
				httpOnly: false,
				secure: isSecure,
				sameSite: 'Lax',
				path: '/',
			},
			PB_COOKIE_NAME,
		);

		headers.append('set-cookie', exportedCookie);
	}

	return new Response(response.body, {
		status: response.status,
		statusText: response.statusText,
		headers,
	});
};

export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	return fetchWith503Retry(request, fetch, {
		apiOrigins: [event.url.origin, pbUrl],
	});
};

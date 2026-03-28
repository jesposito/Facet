import type { Handle } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { PB_COOKIE_NAME } from '$lib/pocketbase';
import { generateAccentCSSVars } from '$lib/colors';

export const handle: Handle = async ({ event, resolve }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	
	event.locals.pb = new PocketBase(pbUrl);

	const cookie = event.request.headers.get('cookie') || '';
	event.locals.pb.authStore.loadFromCookie(cookie, PB_COOKIE_NAME);

	// Snapshot the token so we can detect if auth changed during request handling
	const initialToken = event.locals.pb.authStore.token;

	// Fetch accent color for SSR injection (eliminates color flash on first paint)
	let accentCSSVars = '';
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
		}
	} catch { /* silent - fallback to client-side application */ }

	const response = await resolve(event, {
		transformPageChunk: ({ html }) => {
			if (accentCSSVars) {
				// Inject accent CSS vars into <html> style attribute
				const vars = accentCSSVars
					.replace(/^:root\s*\{/, '')
					.replace(/\}\s*$/, '')
					.trim()
					.replace(/\s+/g, ' ');
				return html.replace('<html lang="en"', `<html lang="en" style="${vars}"`);
			}
			return html;
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

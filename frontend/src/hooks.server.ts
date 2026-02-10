import type { Handle } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { PB_COOKIE_NAME } from '$lib/pocketbase';

export const handle: Handle = async ({ event, resolve }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	
	event.locals.pb = new PocketBase(pbUrl);
	
	const cookie = event.request.headers.get('cookie') || '';
	event.locals.pb.authStore.loadFromCookie(cookie, PB_COOKIE_NAME);

	const response = await resolve(event);

	// Strip oversized Link headers from HTML responses to prevent proxy failures.
	// Only for HTML - API endpoints may use Link for pagination (rel="next").
	// See: https://github.com/sveltejs/kit/issues/6819
	const contentType = response.headers.get('content-type') || '';
	if (contentType.includes('text/html')) {
		response.headers.delete('link');
	}

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

	return response;
};

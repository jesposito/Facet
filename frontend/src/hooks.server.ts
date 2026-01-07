import type { Handle } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { PB_COOKIE_NAME } from '$lib/pocketbase';

export const handle: Handle = async ({ event, resolve }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	
	event.locals.pb = new PocketBase(pbUrl);
	
	const cookie = event.request.headers.get('cookie') || '';
	event.locals.pb.authStore.loadFromCookie(cookie, PB_COOKIE_NAME);

	const response = await resolve(event);

	const isProd = process.env.NODE_ENV === 'production';
	const exportedCookie = event.locals.pb.authStore.exportToCookie({
		httpOnly: false,
		secure: isProd,
		sameSite: 'Lax',
		path: '/'
	}, PB_COOKIE_NAME);

	response.headers.append('set-cookie', exportedCookie);

	return response;
};

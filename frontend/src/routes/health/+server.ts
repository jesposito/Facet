import type { RequestHandler } from './$types';
import { checkBackendHealth, resolvePocketBaseUrl } from '$lib/server/health';

export const GET: RequestHandler = async ({ fetch }) => {
	const status = await checkBackendHealth(fetch, resolvePocketBaseUrl());

	return new Response(JSON.stringify(status), {
		status: status.ok ? 200 : 503,
		headers: {
			'Content-Type': 'application/json',
			'Cache-Control': 'no-store'
		}
	});
};

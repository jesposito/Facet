import type { RequestHandler } from './$types';
import { buildRobotsTxt, publicOrigin } from '$lib/server/robots';

export const GET: RequestHandler = async ({ url }) => {
	const origin = publicOrigin(process.env, url.origin);

	return new Response(buildRobotsTxt(origin), {
		headers: {
			'Content-Type': 'text/plain; charset=utf-8',
			'Cache-Control': 'public, max-age=86400' // Cache for 24 hours
		}
	});
};

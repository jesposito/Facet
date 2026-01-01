/**
 * Talk detail route: /talks/[slug]
 *
 * Displays a single talk with full details and video embed.
 * Only public, non-draft talks are accessible.
 * Private/unlisted/draft talks return 404 (non-discoverable).
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;
	const fromView = url.searchParams.get('from');

	try {
		const response = await fetch(`${pbUrl}/api/talk/${slug}`);

		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const talk = await response.json();

		return {
			talk: {
				id: talk.id,
				title: talk.title,
				slug: talk.slug,
				event: talk.event || null,
				event_url: talk.event_url || null,
				date: talk.date || null,
				location: talk.location || null,
				description: talk.description || '',
				slides_url: talk.slides_url || null,
				video_url: talk.video_url || null,
				created: talk.created,
				updated: talk.updated
			},
			profile: talk.profile || null,
			prev_talk: talk.prev_talk || null,
			next_talk: talk.next_talk || null,
			fromView: fromView || null
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load talk:', err);
		throw error(404, 'Not Found');
	}
};

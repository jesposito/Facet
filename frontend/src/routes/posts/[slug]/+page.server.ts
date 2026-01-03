/**
 * Post detail route: /posts/[slug]
 *
 * Displays a single blog post with full content.
 * Only public, non-draft posts are accessible.
 * Private/unlisted/draft posts return 404 (non-discoverable).
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;
	const fromView = url.searchParams.get('from');

	try {
		const response = await fetch(`${pbUrl}/api/post/${slug}`);
		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const post = await response.json();

		const mediaRefsIds: string[] = post.media_refs || [];
		let mediaRefs: any[] = [];
		if (mediaRefsIds.length > 0) {
			const filter = mediaRefsIds.map((id) => `id="${id}"`).join(' || ');
			const res = await fetch(
				`${pbUrl}/api/collections/external_media/records?filter=${encodeURIComponent(filter)}&perPage=${mediaRefsIds.length}`
			);
			if (res.ok) {
				const data = await res.json();
				mediaRefs = data.items || [];
			}
		}

		return {
			post: {
				id: post.id,
				title: post.title,
				slug: post.slug,
				excerpt: post.excerpt || null,
				content: post.content || '',
				tags: post.tags || [],
				published_at: post.published_at || null,
				created: post.created,
				updated: post.updated,
				cover_image_url: post.cover_image_url || null
			},
			media_refs: mediaRefs,
			profile: post.profile || null,
			prev_post: post.prev_post || null,
			next_post: post.next_post || null,
			fromView: fromView || null
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load post:', err);
		throw error(404, 'Not Found');
	}
};

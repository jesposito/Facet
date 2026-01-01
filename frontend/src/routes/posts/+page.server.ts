/**
 * Posts index route: /posts
 *
 * Lists all non-private, non-draft posts with tag filtering.
 * Uses the custom /api/posts endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	console.log('[POSTS PAGE] ========== LOAD START ==========');
	console.log('[POSTS PAGE] URL:', url.toString());

	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const tag = url.searchParams.get('tag');
	const fromView = url.searchParams.get('from');

	console.log('[POSTS PAGE] pbUrl:', pbUrl);
	console.log('[POSTS PAGE] tag:', tag);
	console.log('[POSTS PAGE] fromView:', fromView);

	try {
		// Use custom API endpoint that bypasses collection access rules
		const apiUrl = `${pbUrl}/api/posts`;
		console.log('[POSTS PAGE] Fetching:', apiUrl);

		const response = await fetch(apiUrl);
		console.log('[POSTS PAGE] Response status:', response.status);

		if (!response.ok) {
			const errorText = await response.text();
			console.error('[POSTS PAGE] API error:', response.status, errorText);
			return {
				posts: [],
				profile: null,
				selectedTag: tag,
				allTags: [],
				fromView: fromView || null
			};
		}

		const data = await response.json();
		console.log('[POSTS PAGE] API response data:', JSON.stringify(data, null, 2));

		let posts = data.posts || [];
		const profile = data.profile || null;

		console.log('[POSTS PAGE] Posts count:', posts.length);
		console.log('[POSTS PAGE] Profile:', profile);

		// If tag filter is specified, filter client-side
		if (tag) {
			const beforeFilter = posts.length;
			posts = posts.filter((post: { tags?: string[] }) =>
				post.tags?.some((t: string) => t.toLowerCase() === tag.toLowerCase())
			);
			console.log('[POSTS PAGE] Filtered by tag:', tag, 'from', beforeFilter, 'to', posts.length);
		}

		// Get unique tags from all posts for filter UI
		const allTags = new Set<string>();
		(data.posts || []).forEach((post: { tags?: string[] }) => {
			post.tags?.forEach((t: string) => allTags.add(t));
		});

		const result = {
			posts,
			profile,
			selectedTag: tag,
			allTags: Array.from(allTags).sort(),
			fromView: fromView || null
		};
		console.log('[POSTS PAGE] Returning:', JSON.stringify({ ...result, posts: `[${posts.length} items]` }));
		console.log('[POSTS PAGE] ========== LOAD END ==========');

		return result;
	} catch (err) {
		console.error('[POSTS PAGE] EXCEPTION:', err);
		return {
			posts: [],
			profile: null,
			selectedTag: tag,
			allTags: [],
			fromView: fromView || null
		};
	}
};

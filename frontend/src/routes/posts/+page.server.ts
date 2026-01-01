/**
 * Posts index route: /posts
 *
 * Lists all non-private, non-draft posts with tag filtering.
 * Uses the custom /api/posts endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const tag = url.searchParams.get('tag');

	try {
		// Use custom API endpoint that bypasses collection access rules
		const response = await fetch(`${pbUrl}/api/posts`);

		if (!response.ok) {
			console.error('Posts API error:', response.status);
			return {
				posts: [],
				profile: null,
				selectedTag: tag,
				allTags: []
			};
		}

		const data = await response.json();
		let posts = data.posts || [];
		const profile = data.profile || null;

		// If tag filter is specified, filter client-side
		if (tag) {
			posts = posts.filter((post: { tags?: string[] }) =>
				post.tags?.some((t: string) => t.toLowerCase() === tag.toLowerCase())
			);
		}

		// Get unique tags from all posts for filter UI
		const allTags = new Set<string>();
		(data.posts || []).forEach((post: { tags?: string[] }) => {
			post.tags?.forEach((t: string) => allTags.add(t));
		});

		return {
			posts,
			profile,
			selectedTag: tag,
			allTags: Array.from(allTags).sort()
		};
	} catch (err) {
		console.error('Failed to load posts:', err);
		return {
			posts: [],
			profile: null,
			selectedTag: tag,
			allTags: []
		};
	}
};

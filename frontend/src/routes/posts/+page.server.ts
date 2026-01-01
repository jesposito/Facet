/**
 * Posts index route: /posts
 *
 * Lists all public, non-draft posts with tag filtering.
 * Private/unlisted/draft posts are excluded.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const tag = url.searchParams.get('tag');

	try {
		// Build filter for public, non-draft posts
		let filter = "visibility = 'public' && is_draft = false";

		// Fetch posts from PocketBase
		const postsResponse = await fetch(
			`${pbUrl}/api/collections/posts/records?filter=${encodeURIComponent(filter)}&sort=-published_at,-created`
		);

		// Fetch profile for page context
		const profileResponse = await fetch(
			`${pbUrl}/api/collections/profile/records?perPage=1`
		);

		let posts = [];
		if (postsResponse.ok) {
			const postsData = await postsResponse.json();
			posts = postsData.items || [];

			// If tag filter is specified, filter client-side (PocketBase JSON field filtering is limited)
			if (tag) {
				posts = posts.filter((post: { tags?: string[] }) =>
					post.tags?.some((t: string) => t.toLowerCase() === tag.toLowerCase())
				);
			}

			// Add cover image URLs
			posts = posts.map((post: { id: string; collectionId: string; cover_image?: string }) => ({
				...post,
				cover_image_url: post.cover_image
					? `/api/files/${post.collectionId}/${post.id}/${post.cover_image}`
					: null
			}));
		}

		let profile = null;
		if (profileResponse.ok) {
			const profileData = await profileResponse.json();
			profile = profileData.items?.[0] || null;
		}

		// Get unique tags from all posts for filter UI
		const allTags = new Set<string>();
		posts.forEach((post: { tags?: string[] }) => {
			post.tags?.forEach(t => allTags.add(t));
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

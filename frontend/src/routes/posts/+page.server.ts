/**
 * Posts index route: /posts
 *
 * Lists all non-private, non-draft posts with tag filtering.
 * Uses the custom /api/posts endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';
import { logger } from '$lib/logger';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const tag = url.searchParams.get('tag');
	const fromView = url.searchParams.get('from');

	try {
		const response = await fetch(`${pbUrl}/api/posts`);

		if (!response.ok) {
			let landingPageMessage = '';
			try {
				const errorData = await response.json();
				if (errorData?.homepage_enabled === false) {
					return {
						posts: [],
						profile: null,
						selectedTag: tag,
						allTags: [],
						fromView: fromView || null,
						homepageDisabled: true,
						landingPageMessage: errorData.landing_page_message || ''
					};
				}
				landingPageMessage = errorData?.error || '';
			} catch {
				// Failed to parse error response
			}

			const errorText = landingPageMessage || (await response.text());
			logger.error('[POSTS PAGE] API error:', response.status, errorText);
			return {
				posts: [],
				profile: null,
				selectedTag: tag,
				allTags: [],
				fromView: fromView || null,
				homepageDisabled: false,
				landingPageMessage: landingPageMessage
			};
		}

		const data = await response.json();
		let posts = data.posts || [];
		const profile = data.profile || null;

		if (tag) {
			posts = posts.filter((post: { tags?: string[] }) =>
				post.tags?.some((t: string) => t.toLowerCase() === tag.toLowerCase())
			);
		}

		const allTags = new Set<string>();
		(data.posts || []).forEach((post: { tags?: string[] }) => {
			post.tags?.forEach((t: string) => allTags.add(t));
		});

		return {
			posts,
			profile,
			selectedTag: tag,
			allTags: Array.from(allTags).sort(),
			fromView: fromView || null,
			homepageDisabled: false,
			landingPageMessage: ''
		};
	} catch (err) {
		logger.error('[POSTS PAGE] Exception:', err);
		return {
			posts: [],
			profile: null,
			selectedTag: tag,
			allTags: [],
			fromView: fromView || null,
			homepageDisabled: false,
			landingPageMessage: ''
		};
	}
};

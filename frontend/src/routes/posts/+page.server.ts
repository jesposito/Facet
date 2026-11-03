/**
 * Posts index route: /posts
 *
 * Lists all non-private, non-draft posts with tag filtering.
 * Uses the custom /api/posts endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';
import type { Profile } from '$lib/pocketbase';
import { logger } from '$lib/logger';
import { navigationFacet } from '$lib/discovery';
import { internalApiHeaders, parsePublicPosts } from '$lib/server/public-api';

export const load: PageServerLoad = async ({ fetch, url, request, locals }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const tag = url.searchParams.get('tag');
	const fromHints = url.searchParams.getAll('from');
	const fromView = fromHints.length === 1 ? navigationFacet(fromHints[0]) : '';
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	const headers = {
		...internalApiHeaders(request),
		...(pbAuthToken ? { Authorization: `Bearer ${pbAuthToken}` } : {})
	};

	try {
		const response = await fetch(`${pbUrl}/api/posts`, { headers });

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

		const data = (await response.json()) as { posts?: unknown; profile?: Profile | null };
		const publicPosts = parsePublicPosts(data.posts).filter(
			(post) => post.visibility === 'public' && !post.is_draft
		);
		let posts = publicPosts;
		const profile = data.profile || null;

		if (tag) {
			posts = posts.filter((post: { tags?: string[] }) =>
				post.tags?.some((t: string) => t.toLowerCase() === tag.toLowerCase())
			);
		}

		const allTags = new Set<string>();
		publicPosts.forEach((post) => {
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

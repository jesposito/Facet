/**
 * Post detail route: /posts/[slug]
 *
 * Displays a single blog post with full content.
 * Public and unlisted posts open directly. A private post opens only from a
 * Facet that shows it, for a visitor who can read that Facet.
 * Anything else returns 404 (non-discoverable).
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';
import { navigationFacet } from '$lib/discovery';
import { buildTokenHeaders, getPasswordToken, getShareToken, getShareVisit } from '$lib/tokens';
import { internalApiHeaders, parseMediaRefs } from '$lib/server/public-api';

export const load: PageServerLoad = async ({ params, fetch, url, request, locals, cookies }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;
	const fromHints = url.searchParams.getAll('from');
	const fromView = fromHints.length === 1 ? navigationFacet(fromHints[0]) : '';

	const headers: Record<string, string> = {
		...internalApiHeaders(request),
		...buildTokenHeaders(getShareToken(cookies), getPasswordToken(cookies), getShareVisit(cookies))
	};
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	if (pbAuthToken) {
		headers['Authorization'] = `Bearer ${pbAuthToken}`;
	}
	const accessQuery = fromView ? `?from=${encodeURIComponent(fromView)}` : '';

	try {
		const response = await fetch(`${pbUrl}/api/post/${slug}${accessQuery}`, { headers });
		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const post = await response.json();

		const mediaRefs = parseMediaRefs(post.media_refs_expand);

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
				cover_image_url: post.cover_image_url || null,
				cover_image_thumb_url:
					typeof post.cover_image_thumb_url === 'string' ? post.cover_image_thumb_url : null,
				cover_image_large_url:
					typeof post.cover_image_large_url === 'string' ? post.cover_image_large_url : null,
				visibility: post.visibility || 'public',
				is_draft: post.is_draft || false,
				comments_enabled: post.comments_enabled ?? false,
				has_downloads: post.has_downloads === true,
				is_gated: post.is_gated === true,
				is_purchased: post.is_purchased === true,
				access_tier: post.access_tier || 'free',
				price: Number(post.price) || 0,
				paywall_preview_paragraphs: Number(post.paywall_preview_paragraphs) || 0,
				is_preview_content: post.is_preview_content === true
			},
			media_refs: post.is_gated === true ? [] : mediaRefs,
			profile: post.profile || null,
			prev_post: post.prev_post || null,
			next_post: post.next_post || null,
			fromView: fromView || null,
			isAuthenticated: post.is_authenticated || false
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load post:', err);
		throw error(404, 'Not Found');
	}
};

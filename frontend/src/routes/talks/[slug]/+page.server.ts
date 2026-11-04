/**
 * Talk detail route: /talks/[slug]
 *
 * Displays a single talk with full details and video embed.
 * Public and unlisted talks open directly. A private talk opens only from a
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
		const response = await fetch(`${pbUrl}/api/talk/${slug}${accessQuery}`, { headers });

		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const talk = await response.json();

		const mediaRefs = parseMediaRefs(talk.media_refs_expand);

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
				cover_image_library_url: talk.cover_image_library_url || null,
				created: talk.created,
				updated: talk.updated,
				visibility: talk.visibility || 'public',
				is_draft: talk.is_draft || false,
				is_gated: talk.is_gated === true,
				is_purchased: talk.is_purchased === true,
				access_tier: talk.access_tier || 'free',
				price: Number(talk.price) || 0
			},
			media_refs: talk.is_gated === true ? [] : mediaRefs,
			profile: talk.profile || null,
			prev_talk: talk.prev_talk || null,
			next_talk: talk.next_talk || null,
			fromView: fromView || null,
			isAuthenticated: talk.is_authenticated || false
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load talk:', err);
		throw error(404, 'Not Found');
	}
};

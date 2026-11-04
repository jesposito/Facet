/** Project detail route: /projects/[slug]. */

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
		const response = await fetch(`${pbUrl}/api/project/${slug}${accessQuery}`, { headers });

		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const project = await response.json();

		const mediaRefs = parseMediaRefs(project.media_refs_expand);

		return {
			project: {
				id: project.id,
				title: project.title,
				slug: project.slug,
				summary: project.summary,
				description: project.description,
				experience: project.experience || null,
				tech_stack: project.tech_stack || [],
				links: project.links || [],
				categories: project.categories || [],
				is_featured: project.is_featured,
				cover_image_url: project.cover_image_url || null,
				cover_image_thumb_url:
					typeof project.cover_image_thumb_url === 'string' ? project.cover_image_thumb_url : null,
				cover_image_large_url:
					typeof project.cover_image_large_url === 'string' ? project.cover_image_large_url : null,
				media_urls: project.is_gated === true ? [] : project.media_urls || [],
				visibility: project.visibility || 'public',
				is_draft: project.is_draft || false,
				has_downloads: project.has_downloads === true,
				related_experience: project.related_experience_expand || [],
				related_education: project.related_education_expand || [],
				is_gated: project.is_gated === true,
				is_purchased: project.is_purchased === true,
				access_tier: project.access_tier || 'free',
				price: Number(project.price) || 0
			},
			media_refs: project.is_gated === true ? [] : mediaRefs,
			profile: project.profile || null,
			fromView: fromView || null,
			isAuthenticated: project.is_authenticated || false
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load project:', err);
		throw error(404, 'Not Found');
	}
};

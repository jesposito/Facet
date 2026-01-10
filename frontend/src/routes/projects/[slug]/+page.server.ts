/**
 * Project detail route: /projects/[slug]
 *
 * Displays a single project with full details.
 * Only public, non-draft projects are accessible.
 * Private/unlisted/draft projects return 404 (non-discoverable).
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch, url, locals }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;
	const fromView = url.searchParams.get('from');

	const headers: Record<string, string> = {};
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	if (pbAuthToken) {
		headers['Authorization'] = `Bearer ${pbAuthToken}`;
	}

	try {
		const response = await fetch(`${pbUrl}/api/project/${slug}`, { headers });

		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const project = await response.json();

		const mediaRefs: any[] = project.media_refs_expand || [];

		return {
			project: {
				id: project.id,
				title: project.title,
				slug: project.slug,
				summary: project.summary,
				description: project.description,
				tech_stack: project.tech_stack || [],
				links: project.links || [],
				categories: project.categories || [],
				is_featured: project.is_featured,
				cover_image_url: project.cover_image_url || null,
				media_urls: project.media_urls || [],
				visibility: project.visibility || 'public',
				is_draft: project.is_draft || false
			},
			media_refs: mediaRefs,
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

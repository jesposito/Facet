/**
 * Project detail route: /projects/[slug]
 *
 * Displays a single project with full details.
 * Only public, non-draft projects are accessible.
 * Private/unlisted/draft projects return 404 (non-discoverable).
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;
	const fromView = url.searchParams.get('from');

	try {
		const response = await fetch(`${pbUrl}/api/project/${slug}`);

		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const project = await response.json();

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
				media_urls: project.media_urls || []
			},
			profile: project.profile || null,
			fromView: fromView || null
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load project:', err);
		throw error(404, 'Not Found');
	}
};

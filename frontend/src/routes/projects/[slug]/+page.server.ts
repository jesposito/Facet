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
		// Prefer expanded fetch to include media_refs for rendering
		let project: any = null;
		const expanded = await fetch(
			`${pbUrl}/api/collections/projects/records?filter=${encodeURIComponent(`slug="${slug}"`)}&expand=media_refs&perPage=1`
		);
		if (expanded.ok) {
			const data = await expanded.json();
			project = data.items?.[0] ?? null;
		}

		if (!project) {
			const response = await fetch(`${pbUrl}/api/project/${slug}`);

			if (!response.ok) {
				throw error(404, 'Not Found');
			}

			project = await response.json();
		}

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
			media_refs: project.expand?.media_refs || project.media_refs || [],
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

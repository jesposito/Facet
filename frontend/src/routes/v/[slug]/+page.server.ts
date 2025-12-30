import type { PageServerLoad } from './$types';
import PocketBase from 'pocketbase';
import { error } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params }) => {
	const pb = new PocketBase(process.env.POCKETBASE_URL || 'http://localhost:8090');
	const { slug } = params;

	try {
		// Get view access info
		const response = await fetch(`${pb.baseURL}/api/view/${slug}/access`);
		if (!response.ok) {
			throw error(404, 'View not found');
		}

		const accessInfo = await response.json();

		// Handle different visibility types
		if (accessInfo.visibility === 'private') {
			throw error(404, 'View not found');
		}

		if (accessInfo.visibility === 'unlisted') {
			// Unlisted views require a share token
			throw error(404, 'View not found');
		}

		// Fetch view data
		const dataResponse = await fetch(`${pb.baseURL}/api/view/${slug}/data`);
		if (!dataResponse.ok) {
			throw error(404, 'View not found');
		}

		const viewData = await dataResponse.json();

		// Fetch profile
		const profileRecords = await pb.collection('profile').getList(1, 1);
		const profile = profileRecords.items[0] || null;

		return {
			view: {
				id: viewData.id,
				slug: viewData.slug,
				name: viewData.name,
				hero_headline: viewData.hero_headline,
				hero_summary: viewData.hero_summary,
				cta_text: viewData.cta_text,
				cta_url: viewData.cta_url
			},
			profile: profile ? {
				...profile,
				hero_image: profile.hero_image ? pb.files.getURL(profile, profile.hero_image) : null,
				avatar: profile.avatar ? pb.files.getURL(profile, profile.avatar) : null
			} : null,
			sections: viewData.sections || {},
			requiresPassword: accessInfo.visibility === 'password'
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load view:', err);
		return {
			view: null,
			profile: null,
			sections: {},
			error: 'View not found'
		};
	}
};

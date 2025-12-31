import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	try {
		// Fetch all public content from the API
		const response = await fetch(`${pbUrl}/api/homepage`);

		if (!response.ok) {
			console.error('Homepage API error:', response.status);
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				skills: [],
				error: 'Failed to load profile'
			};
		}

		const data = await response.json();

		// Check if profile is private
		if (!data.profile) {
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				skills: [],
				error: 'Profile is private'
			};
		}

		// Map API response to expected format
		const profile = data.profile ? {
			...data.profile,
			hero_image: data.profile.hero_image_url || null,
			avatar: data.profile.avatar_url || null
		} : null;

		const projects = (data.projects || []).map((p: Record<string, unknown>) => ({
			...p,
			cover_image: p.cover_image_url || null
		}));

		return {
			profile,
			experience: data.experience || [],
			projects,
			education: data.education || [],
			skills: data.skills || []
		};
	} catch (error) {
		console.error('Failed to load profile data:', error);
		return {
			profile: null,
			experience: [],
			projects: [],
			education: [],
			skills: [],
			error: 'Failed to load profile'
		};
	}
};

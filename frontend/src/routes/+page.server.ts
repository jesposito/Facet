import type { PageServerLoad } from './$types';
import PocketBase from 'pocketbase';

export const load: PageServerLoad = async ({ fetch }) => {
	const pb = new PocketBase(process.env.POCKETBASE_URL || 'http://localhost:8090');

	try {
		// Fetch profile
		const profileRecords = await pb.collection('profile').getList(1, 1);
		const profile = profileRecords.items[0] || null;

		// Check visibility
		if (profile && profile.visibility === 'private') {
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				skills: [],
				error: 'Profile is private'
			};
		}

		// Fetch all public content in parallel
		const [experienceRes, projectsRes, educationRes, skillsRes] = await Promise.all([
			pb.collection('experience').getList(1, 100, {
				filter: "visibility != 'private' && is_draft = false",
				sort: '-sort_order,-start_date'
			}).catch(() => ({ items: [] })),
			pb.collection('projects').getList(1, 100, {
				filter: "visibility != 'private' && is_draft = false",
				sort: '-is_featured,-sort_order'
			}).catch(() => ({ items: [] })),
			pb.collection('education').getList(1, 100, {
				filter: "visibility != 'private' && is_draft = false",
				sort: '-sort_order,-end_date'
			}).catch(() => ({ items: [] })),
			pb.collection('skills').getList(1, 200, {
				filter: "visibility != 'private'",
				sort: 'category,sort_order'
			}).catch(() => ({ items: [] }))
		]);

		return {
			profile: profile ? {
				...profile,
				hero_image: profile.hero_image ? pb.files.getURL(profile, profile.hero_image) : null,
				avatar: profile.avatar ? pb.files.getURL(profile, profile.avatar) : null
			} : null,
			experience: experienceRes.items,
			projects: projectsRes.items.map((p: Record<string, unknown>) => ({
				...p,
				cover_image: p.cover_image ? pb.files.getURL(p, p.cover_image as string) : null
			})),
			education: educationRes.items,
			skills: skillsRes.items
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

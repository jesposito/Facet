/**
 * Talks index route: /talks
 *
 * Lists all non-private, non-draft talks with year filtering.
 * Only private and draft talks are excluded.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const year = url.searchParams.get('year');

	try {
		// Build filter for non-private, non-draft talks (matches profile behavior)
		const filter = "visibility != 'private' && is_draft = false";

		// Fetch talks from PocketBase
		const talksResponse = await fetch(
			`${pbUrl}/api/collections/talks/records?filter=${encodeURIComponent(filter)}&sort=-date,-sort_order`
		);

		// Fetch profile for page context
		const profileResponse = await fetch(
			`${pbUrl}/api/collections/profile/records?perPage=1`
		);

		let talks = [];
		if (talksResponse.ok) {
			const talksData = await talksResponse.json();
			talks = talksData.items || [];

			// If year filter is specified, filter client-side
			if (year) {
				talks = talks.filter((talk: { date?: string }) => {
					if (!talk.date) return false;
					return new Date(talk.date).getFullYear().toString() === year;
				});
			}
		}

		let profile = null;
		if (profileResponse.ok) {
			const profileData = await profileResponse.json();
			profile = profileData.items?.[0] || null;
		}

		// Get unique years from all talks for filter UI
		const allYears = new Set<string>();
		talks.forEach((talk: { date?: string }) => {
			if (talk.date) {
				allYears.add(new Date(talk.date).getFullYear().toString());
			}
		});

		// If we filtered, we need to get years from all talks (unfiltered)
		if (year) {
			// Refetch without year filter for year list
			const allTalksResponse = await fetch(
				`${pbUrl}/api/collections/talks/records?filter=${encodeURIComponent(filter)}&sort=-date`
			);
			if (allTalksResponse.ok) {
				const allTalksData = await allTalksResponse.json();
				(allTalksData.items || []).forEach((talk: { date?: string }) => {
					if (talk.date) {
						allYears.add(new Date(talk.date).getFullYear().toString());
					}
				});
			}
		}

		return {
			talks,
			profile,
			selectedYear: year,
			allYears: Array.from(allYears).sort((a, b) => parseInt(b) - parseInt(a)) // Descending
		};
	} catch (err) {
		console.error('Failed to load talks:', err);
		return {
			talks: [],
			profile: null,
			selectedYear: year,
			allYears: []
		};
	}
};

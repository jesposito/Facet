/**
 * Talks index route: /talks
 *
 * Lists all non-private, non-draft talks with year filtering.
 * Uses the custom /api/talks endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const year = url.searchParams.get('year');

	try {
		// Use custom API endpoint that bypasses collection access rules
		const response = await fetch(`${pbUrl}/api/talks`);

		if (!response.ok) {
			console.error('Talks API error:', response.status);
			return {
				talks: [],
				profile: null,
				selectedYear: year,
				allYears: []
			};
		}

		const data = await response.json();
		let talks = data.talks || [];
		const profile = data.profile || null;

		// Get unique years from all talks for filter UI (before filtering)
		const allYears = new Set<string>();
		talks.forEach((talk: { date?: string }) => {
			if (talk.date) {
				allYears.add(new Date(talk.date).getFullYear().toString());
			}
		});

		// If year filter is specified, filter client-side
		if (year) {
			talks = talks.filter((talk: { date?: string }) => {
				if (!talk.date) return false;
				return new Date(talk.date).getFullYear().toString() === year;
			});
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

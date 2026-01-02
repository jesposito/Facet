/**
 * Talks index route: /talks
 *
 * Lists all non-private, non-draft talks with year filtering.
 * Uses the custom /api/talks endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	console.log('[TALKS PAGE] ========== LOAD START ==========');
	console.log('[TALKS PAGE] URL:', url.toString());

	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const year = url.searchParams.get('year');
	const fromView = url.searchParams.get('from');

	console.log('[TALKS PAGE] pbUrl:', pbUrl);
	console.log('[TALKS PAGE] year:', year);
	console.log('[TALKS PAGE] fromView:', fromView);

	try {
		// Use custom API endpoint that bypasses collection access rules
		const apiUrl = `${pbUrl}/api/talks`;
		console.log('[TALKS PAGE] Fetching:', apiUrl);

		const response = await fetch(apiUrl);
		console.log('[TALKS PAGE] Response status:', response.status);

		if (!response.ok) {
			let landingPageMessage = '';
			try {
				const errorData = await response.json();
				if (errorData?.homepage_enabled === false) {
					return {
						talks: [],
						profile: null,
						selectedYear: year,
						allYears: [],
						fromView: fromView || null,
						homepageDisabled: true,
						landingPageMessage: errorData.landing_page_message || ''
					};
				}
				landingPageMessage = errorData?.error || '';
			} catch (parseErr) {
				console.error('[TALKS PAGE] Failed to parse error response:', parseErr);
			}

			const errorText = landingPageMessage || (await response.text());
			console.error('[TALKS PAGE] API error:', response.status, errorText);
			return {
				talks: [],
				profile: null,
				selectedYear: year,
				allYears: [],
				fromView: fromView || null,
				homepageDisabled: false,
				landingPageMessage: landingPageMessage
			};
		}

		const data = await response.json();
		console.log('[TALKS PAGE] API response data:', JSON.stringify(data, null, 2));

		let talks = data.talks || [];
		const profile = data.profile || null;

		console.log('[TALKS PAGE] Talks count:', talks.length);
		console.log('[TALKS PAGE] Profile:', profile);

		// Get unique years from all talks for filter UI (before filtering)
		const allYears = new Set<string>();
		talks.forEach((talk: { date?: string }) => {
			if (talk.date) {
				allYears.add(new Date(talk.date).getFullYear().toString());
			}
		});

		// If year filter is specified, filter client-side
		if (year) {
			const beforeFilter = talks.length;
			talks = talks.filter((talk: { date?: string }) => {
				if (!talk.date) return false;
				return new Date(talk.date).getFullYear().toString() === year;
			});
			console.log('[TALKS PAGE] Filtered by year:', year, 'from', beforeFilter, 'to', talks.length);
		}

		const result = {
			talks,
			profile,
			selectedYear: year,
			allYears: Array.from(allYears).sort((a, b) => parseInt(b) - parseInt(a)), // Descending
			fromView: fromView || null,
			homepageDisabled: false,
			landingPageMessage: ''
		};
		console.log('[TALKS PAGE] Returning:', JSON.stringify({ ...result, talks: `[${talks.length} items]` }));
		console.log('[TALKS PAGE] ========== LOAD END ==========');

		return result;
	} catch (err) {
		console.error('[TALKS PAGE] EXCEPTION:', err);
		return {
			talks: [],
			profile: null,
			selectedYear: year,
			allYears: [],
			fromView: fromView || null,
			homepageDisabled: false,
			landingPageMessage: ''
		};
	}
};

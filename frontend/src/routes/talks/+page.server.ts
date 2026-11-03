/**
 * Talks index route: /talks
 *
 * Lists all non-private, non-draft talks with year filtering.
 * Uses the custom /api/talks endpoint which bypasses collection access rules.
 */

import type { PageServerLoad } from './$types';
import type { Profile } from '$lib/pocketbase';
import { logger } from '$lib/logger';
import { navigationFacet } from '$lib/discovery';
import { internalApiHeaders, parsePublicTalks } from '$lib/server/public-api';

export const load: PageServerLoad = async ({ fetch, url, request, locals }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const year = url.searchParams.get('year');
	const fromHints = url.searchParams.getAll('from');
	const fromView = fromHints.length === 1 ? navigationFacet(fromHints[0]) : '';
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	const headers = {
		...internalApiHeaders(request),
		...(pbAuthToken ? { Authorization: `Bearer ${pbAuthToken}` } : {})
	};

	try {
		const response = await fetch(`${pbUrl}/api/talks`, { headers });

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
			} catch {
				// Failed to parse error response
			}

			const errorText = landingPageMessage || (await response.text());
			logger.error('[TALKS PAGE] API error:', response.status, errorText);
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

		const data = (await response.json()) as { talks?: unknown; profile?: Profile | null };
		const publicTalks = parsePublicTalks(data.talks).filter(
			(talk) => talk.visibility === 'public' && !talk.is_draft
		);
		let talks = publicTalks;
		const profile = data.profile || null;

		const allYears = new Set<string>();
		talks.forEach((talk: { date?: string }) => {
			if (talk.date) {
				allYears.add(new Date(talk.date).getFullYear().toString());
			}
		});

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
			allYears: Array.from(allYears).sort((a, b) => parseInt(b) - parseInt(a)),
			fromView: fromView || null,
			homepageDisabled: false,
			landingPageMessage: ''
		};
	} catch (err) {
		logger.error('[TALKS PAGE] Exception:', err);
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

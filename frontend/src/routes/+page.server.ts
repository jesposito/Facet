/**
 * Homepage route: /
 *
 * Renders the default public profile view. Behavior:
 * 1. Check if a default view is configured (is_default = true)
 * 2. If yes: render that view's curated content
 * 3. If no: fallback to legacy homepage (all public content aggregated)
 *
 * This route does NOT handle password or unlisted views at the root.
 * Those must be accessed via /<slug> with proper tokens.
 */

import type { PageServerLoad } from './$types';
import { logger } from '$lib/logger';

export const load: PageServerLoad = async ({ fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	try {
		const defaultViewResponse = await fetch(`${pbUrl}/api/default-view`);

		if (defaultViewResponse.ok) {
			const defaultViewInfo = await defaultViewResponse.json();

			if (defaultViewInfo.homepage_enabled === false) {
				logger.debug('[ROOT PAGE] Homepage disabled via settings');
				return {
					homepageDisabled: true,
					landingPageMessage: defaultViewInfo.landing_page_message || '',
					hideLoginButton: defaultViewInfo.hide_login_button || false
				};
			}

			if (defaultViewInfo.has_default && defaultViewInfo.slug) {
				const viewDataResponse = await fetch(`${pbUrl}/api/view/${defaultViewInfo.slug}/data`);

				if (viewDataResponse.ok) {
					const viewData = await viewDataResponse.json();
					const profile = viewData.profile || null;

					let posts = viewData.sections?.posts || [];
					let talks = viewData.sections?.talks || [];
					let awards = viewData.sections?.awards || [];

					if (posts.length === 0 || talks.length === 0 || awards.length === 0) {
						try {
							const homepageResponse = await fetch(`${pbUrl}/api/homepage`);
							if (homepageResponse.ok) {
								const homepageData = await homepageResponse.json();
								if (posts.length === 0 && homepageData.posts) {
									posts = homepageData.posts.map((p: Record<string, unknown>) => ({
										...p,
										cover_image: p.cover_image_url || null,
										cover_image_thumb_url: p.cover_image_thumb_url || p.cover_image_url || null,
										cover_image_large_url: p.cover_image_large_url || p.cover_image_url || null
									}));
								}
								if (talks.length === 0 && homepageData.talks) {
									talks = homepageData.talks;
								}
								if (awards.length === 0 && homepageData.awards) {
									awards = homepageData.awards;
								}
							}
						} catch {
							// Ignore homepage fallback errors
						}
					}

					return {
						view: {
							id: viewData.id,
							slug: viewData.slug,
							name: viewData.name,
							hero_headline: viewData.hero_headline,
							hero_summary: viewData.hero_summary,
							hero_location: viewData.hero_location,
							cta_text: viewData.cta_text,
							cta_url: viewData.cta_url,
							cta_button_text: viewData.cta_button_text,
							accent_color: viewData.accent_color || null
						},
						profile: profile
							? {
									...profile,
									hero_image: profile.hero_image_url || null,
									avatar: profile.avatar_url || null
								}
							: null,
						sections: viewData.sections || {},
						sectionOrder: viewData.section_order || [],
						sectionLayouts: viewData.section_layouts || {},
						sectionWidths: viewData.section_widths || {},
						experience: viewData.sections?.experience || [],
						projects: viewData.sections?.projects || [],
						education: viewData.sections?.education || [],
						certifications: viewData.sections?.certifications || [],
						awards,
						skills: viewData.sections?.skills || [],
						posts,
						talks,
						testimonials: viewData.sections?.testimonials || [],
						contacts: viewData.sections?.contacts || [],
						isDefaultView: true,
						homepageDisabled: false,
						landingPageMessage: defaultViewInfo.landing_page_message || '',
						hideLoginButton: defaultViewInfo.hide_login_button || false
					};
				}
			}
		}

		const response = await fetch(`${pbUrl}/api/homepage`);

		if (!response.ok) {
			const errorText = await response.text();
			logger.error('[ROOT PAGE] Homepage API error:', response.status, errorText);
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				certifications: [],
				awards: [],
				skills: [],
				posts: [],
				talks: [],
				testimonials: [],
				contacts: [],
				view: null,
				sections: {},
				sectionOrder: [],
				sectionLayouts: {},
				sectionWidths: {},
				error: 'Failed to load profile',
				isDefaultView: false,
				homepageDisabled: false,
				landingPageMessage: '',
				hideLoginButton: false
			};
		}

		const data = await response.json();
		if (data.homepage_enabled === false) {
			return {
				homepageDisabled: true,
				landingPageMessage: data.landing_page_message || '',
				hideLoginButton: data.hide_login_button || false,
				profile: null,
				experience: [],
				projects: [],
				education: [],
				certifications: [],
				awards: [],
				skills: [],
				posts: [],
				talks: [],
				view: null,
				isDefaultView: false
			};
		}

		if (!data.profile) {
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				certifications: [],
				awards: [],
				skills: [],
				posts: [],
				talks: [],
				view: null,
				error: 'Profile is private',
				isDefaultView: false,
				hideLoginButton: data.hide_login_button || false
			};
		}

		const profile = data.profile
			? {
					...data.profile,
					hero_image: data.profile.hero_image_url || null,
					avatar: data.profile.avatar_url || null
				}
			: null;

		const projects = (data.projects || []).map((p: Record<string, unknown>) => ({
			...p,
			cover_image: p.cover_image_url || null
		}));

		const posts = (data.posts || []).map((p: Record<string, unknown>) => ({
			...p,
			cover_image: p.cover_image_url || null
		}));

		return {
			profile,
			experience: data.experience || [],
			projects,
			education: data.education || [],
			certifications: data.certifications || [],
			awards: data.awards || [],
			skills: data.skills || [],
			posts,
			talks: data.talks || [],
			view: null,
			isDefaultView: false,
			hideLoginButton: data.hide_login_button || false
		};
	} catch (error) {
		logger.error('[ROOT PAGE] Exception:', error);
		return {
			profile: null,
			experience: [],
			projects: [],
			education: [],
			certifications: [],
			awards: [],
			skills: [],
			posts: [],
			talks: [],
			view: null,
			error: 'Failed to load profile',
			isDefaultView: false,
			hideLoginButton: false
		};
	}
};

/**
 * Homepage route: /
 *
 * Always renders the homepage configuration from /api/homepage.
 * Facets are accessed via their own slug routes (/<slug>).
 *
 * This route does NOT handle password or unlisted views at the root.
 * Those must be accessed via /<slug> with proper tokens.
 */

import type { PageServerLoad } from './$types';
import { logger } from '$lib/logger';

export const load: PageServerLoad = async ({ fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	try {
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
				sectionFeaturedIds: {},
				postsTotalCount: 0,
				talksTotalCount: 0,
				error: 'Failed to load profile',
				isDefaultView: false,
				homepageDisabled: false,
				landingPageMessage: '',
				hideLoginButton: false,
				siteCtaEnabled: true
			};
		}

		const data = await response.json();
		if (data.homepage_enabled === false) {
			return {
				homepageDisabled: true,
				landingPageMessage: data.landing_page_message || '',
				hideLoginButton: data.hide_login_button || false,
				siteCtaEnabled: data.site_cta_enabled !== false,
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
				isDefaultView: false,
				sectionFeaturedIds: {},
				postsTotalCount: 0,
				talksTotalCount: 0
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
				hideLoginButton: data.hide_login_button || false,
				siteCtaEnabled: data.site_cta_enabled !== false,
				sectionFeaturedIds: {},
				postsTotalCount: 0,
				talksTotalCount: 0
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
			postsTotalCount: data.posts_total_count ?? 0,
			talksTotalCount: data.talks_total_count ?? 0,
			testimonials: data.testimonials || [],
			contacts: data.contacts || [],
			customContent: data.custom_content || [],
			homepageCustomContentConfig: data.homepage_custom_content || [],
			homepageSectionOrder: data.homepage_section_order || [],
			homepageSections: data.homepage_sections || {},
			skillsCategoryOrder: data.skills_category_order || [],
			sectionFeaturedIds: {},
			view: null,
			isDefaultView: false,
			hideLoginButton: data.hide_login_button || false,
			siteCtaEnabled: data.site_cta_enabled !== false
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
			testimonials: [],
			contacts: [],
			view: null,
			sectionFeaturedIds: {},
			postsTotalCount: 0,
			talksTotalCount: 0,
			error: 'Failed to load profile',
			isDefaultView: false,
			hideLoginButton: false,
			siteCtaEnabled: true
		};
	}
};

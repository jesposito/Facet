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

export const load: PageServerLoad = async ({ fetch, parent }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const parentData = await parent();

	// Common empty shape so TypeScript knows view can have properties (not just null)
	const emptyView = null as {
		slug: string;
		hero_headline: string;
		hero_summary: string;
		hero_location: string;
		accent_color: string | null;
		cta_text: string;
		cta_url: string;
		cta_button_text: string;
		cta_enabled: boolean;
		font_pack: string | null;
		hero_layout: string | null;
	} | null;

	try {
		const response = await fetch(`${pbUrl}/api/homepage`, {
			headers: { 'X-Internal': 'true' }
		});

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
				courses: [],
				testimonials: [],
				contacts: [],
				view: emptyView,
				sections: {} as Record<string, any[]>,
				sectionOrder: [],
				sectionLayouts: {} as Record<string, string>,
				sectionWidths: {} as Record<string, string>,
				sectionCategoryOrders: {} as Record<string, string[]>,
				sectionDisabledCategories: {} as Record<string, string[]>,
				sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
				sectionFeaturedIds: {} as Record<string, string>,
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
				courses: [],
				view: emptyView,
				isDefaultView: false,
				sectionFeaturedIds: {} as Record<string, string>,
				sectionCategoryOrders: {} as Record<string, string[]>,
				sectionDisabledCategories: {} as Record<string, string[]>,
				sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
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
				courses: [],
				view: emptyView,
				error: 'Profile is private',
				isDefaultView: false,
				hideLoginButton: data.hide_login_button || false,
				siteCtaEnabled: data.site_cta_enabled !== false,
				sectionFeaturedIds: {} as Record<string, string>,
				sectionCategoryOrders: {} as Record<string, string[]>,
				sectionDisabledCategories: {} as Record<string, string[]>,
				sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
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

		const courses = data.courses || [];

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
			courses,
			postsTotalCount: data.posts_total_count ?? 0,
			talksTotalCount: data.talks_total_count ?? 0,
			testimonials: data.testimonials || [],
			contacts: data.contacts || [],
			customContent: data.custom_content || [],
			homepageCustomContentConfig: data.homepage_custom_content || [],
			homepageSectionOrder: data.homepage_section_order || [],
			homepageSections: data.homepage_sections || {},
			skillsCategoryOrder: data.skills_category_order || [],
			sectionFeaturedIds: {} as Record<string, string>,
			sectionCategoryOrders: {} as Record<string, string[]>,
			sectionDisabledCategories: {} as Record<string, string[]>,
			sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
			view: emptyView,
			isDefaultView: false,
			hideLoginButton: data.hide_login_button || false,
			siteCtaEnabled: data.site_cta_enabled !== false,
			siteNav: parentData.siteNav
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
			courses: [],
			testimonials: [],
			contacts: [],
			view: emptyView,
			sectionFeaturedIds: {} as Record<string, string>,
			sectionCategoryOrders: {} as Record<string, string[]>,
			sectionDisabledCategories: {} as Record<string, string[]>,
			sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
			postsTotalCount: 0,
			talksTotalCount: 0,
			error: 'Failed to load profile',
			isDefaultView: false,
			hideLoginButton: false,
			siteCtaEnabled: true
		};
	}
};

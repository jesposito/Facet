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

type ViewData = {
	hero_headline?: string;
	hero_summary?: string;
	hero_location?: string;
	slug?: string;
	accent_color?: string;
	cta_url?: string;
	cta_button_text?: string;
	cta_text?: string;
	cta_enabled?: boolean;
	font_pack?: string | null;
	hero_layout?: string | null;
} | null;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type SectionsData = Record<string, any[]>;

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
				view: null as ViewData,
				sections: {} as SectionsData,
				sectionOrder: [],
				sectionLayouts: {} as Record<string, string>,
				sectionWidths: {} as Record<string, string>,
				sectionFeaturedIds: {} as Record<string, string>,
				sectionCategoryOrders: {} as Record<string, string[]>,
				sectionDisabledCategories: {} as Record<string, string[]>,
				sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
				homepageSections: {} as Record<string, Record<string, unknown>>,
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
				view: null as ViewData,
				isDefaultView: false,
				sectionFeaturedIds: {} as Record<string, string>,
				sectionCategoryOrders: {} as Record<string, string[]>,
				sectionDisabledCategories: {} as Record<string, string[]>,
				sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
				homepageSections: {} as Record<string, Record<string, unknown>>,
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
				view: null as ViewData,
				error: 'Profile is private',
				isDefaultView: false,
				hideLoginButton: data.hide_login_button || false,
				siteCtaEnabled: data.site_cta_enabled !== false,
				sectionFeaturedIds: {} as Record<string, string>,
				sectionCategoryOrders: {} as Record<string, string[]>,
				sectionDisabledCategories: {} as Record<string, string[]>,
				sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
				homepageSections: {} as Record<string, Record<string, unknown>>,
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
			homepageSections: data.homepage_sections || {} as Record<string, Record<string, unknown>>,
			skillsCategoryOrder: data.skills_category_order || [],
			sectionFeaturedIds: {} as Record<string, string>,
			sectionLayouts: {} as Record<string, string>,
			sectionWidths: {} as Record<string, string>,
			sectionCategoryOrders: {} as Record<string, string[]>,
			sectionDisabledCategories: {} as Record<string, string[]>,
			sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
			view: null as ViewData,
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
			view: null as ViewData,
			sectionFeaturedIds: {} as Record<string, string>,
			sectionCategoryOrders: {} as Record<string, string[]>,
			sectionDisabledCategories: {} as Record<string, string[]>,
			sectionCategoryDisplayModes: {} as Record<string, Record<string, string>>,
			homepageSections: {} as Record<string, Record<string, unknown>>,
			postsTotalCount: 0,
			talksTotalCount: 0,
			error: 'Failed to load profile',
			isDefaultView: false,
			hideLoginButton: false,
			siteCtaEnabled: true
		};
	}
};

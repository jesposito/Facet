/**
 * Public view route: /<slug>
 *
 * This is the canonical URL for named views. Features:
 * - Renders public views directly
 * - Handles unlisted views with share token (from cookie)
 * - Handles password-protected views with password prompt and JWT flow
 * - Returns 404 for private views (non-discoverable)
 *
 * Token flow:
 * - Share tokens: Set by /s/[token], stored in me_share_token cookie
 * - Password JWTs: Set via form action, stored in me_password_token cookie
 * - Legacy ?t= links: Redirected to /s/[token] before any other work
 */

import type { PageServerLoad, Actions } from './$types';
import { error, redirect } from '@sveltejs/kit';
import {
	getShareToken,
	getPasswordToken,
	setPasswordToken,
	getShareVisit,
	storeShareVisit,
	SHARE_VISIT_HEADER,
	isSecureRequest
} from '$lib/tokens';
import { internalApiHeaders } from '$lib/server/public-api';

export const load: PageServerLoad = async ({
	params,
	cookies,
	url,
	fetch,
	locals,
	request,
	setHeaders,
	parent
}) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;
	if (url.searchParams.has('t')) {
		setHeaders({ 'Cache-Control': 'private, no-store', 'Referrer-Policy': 'no-referrer' });

		// /s/ owns the whole token check, so a legacy link recovers the same way.
		// An empty value is no token, and `.` or `..` would vanish from the path.
		const urlToken = url.searchParams.get('t');
		if (urlToken && urlToken !== '.' && urlToken !== '..') {
			throw redirect(302, `/s/${encodeURIComponent(urlToken)}`);
		}
	}

	const shareToken = getShareToken(cookies);
	const passwordToken = getPasswordToken(cookies);

	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;

	const effectiveShareToken = shareToken;

	try {
		const accessHeaders: Record<string, string> = internalApiHeaders(request);
		if (pbAuthToken) {
			accessHeaders['Authorization'] = `Bearer ${pbAuthToken}`;
		}
		if (effectiveShareToken) {
			accessHeaders['X-Share-Token'] = effectiveShareToken;
			const shareVisit = getShareVisit(cookies);
			if (shareVisit) accessHeaders[SHARE_VISIT_HEADER] = shareVisit;
		}

		const response = await fetch(`${pbUrl}/api/view/${slug}/access`, {
			headers: accessHeaders
		});

		if (!response.ok) {
			throw error(404, 'Not Found');
		}

		const accessInfo = await response.json();

		const isAuthenticated = accessInfo.is_authenticated === true;

		if (accessInfo.visibility === 'private' && !isAuthenticated) {
			throw error(404, 'Not Found');
		}

		if (accessInfo.visibility === 'unlisted' && !isAuthenticated && !effectiveShareToken) {
			throw error(404, 'Not Found');
		}

		if (accessInfo.visibility === 'password' && !isAuthenticated && !passwordToken) {
			return {
				view: {
					id: accessInfo.view_id,
					slug,
					name: accessInfo.view_name || 'Protected View',
					hero_headline: undefined,
					hero_summary: undefined,
					hero_location: undefined,
					cta_text: undefined,
					cta_url: undefined,
					cta_button_text: undefined,
					accent_color: undefined,
					hero_image_url: undefined,
					font_pack: undefined,
					hero_layout: undefined,
					hero_spacing: undefined,
					hero_bg_color: undefined
				},
				profile: null,
				sections: {},
				sectionFeaturedIds: {},
				postsTotalCount: 0,
				talksTotalCount: 0,
				requiresPassword: true
			};
		}

		const dataHeaders: Record<string, string> = internalApiHeaders(request);
		if (effectiveShareToken) {
			dataHeaders['X-Share-Token'] = effectiveShareToken;
			const shareVisit = getShareVisit(cookies);
			if (shareVisit) {
				dataHeaders[SHARE_VISIT_HEADER] = shareVisit;
			}
		}
		if (pbAuthToken) {
			dataHeaders['Authorization'] = `Bearer ${pbAuthToken}`;
		} else if (passwordToken) {
			dataHeaders['Authorization'] = `Bearer ${passwordToken}`;
		}

		const dataResponse = await fetch(`${pbUrl}/api/view/${slug}/data`, { headers: dataHeaders });

		if (!dataResponse.ok) {
			if (dataResponse.status === 401 || dataResponse.status === 403) {
				if (accessInfo.visibility === 'password') {
					return {
						view: {
							id: accessInfo.view_id,
							slug,
							name: accessInfo.view_name || 'Protected View',
							hero_headline: undefined,
							hero_summary: undefined,
							hero_location: undefined,
							cta_text: undefined,
							cta_url: undefined,
							cta_button_text: undefined,
							accent_color: undefined,
							hero_image_url: undefined,
							font_pack: undefined,
							hero_layout: undefined
						},
						profile: null,
						sections: {},
						sectionFeaturedIds: {},
						postsTotalCount: 0,
						talksTotalCount: 0,
						requiresPassword: true
					};
				}
				// Invalid share token for unlisted = 404 (not discoverable)
				throw error(404, 'Not Found');
			}
			throw error(404, 'Not Found');
		}

		// A share token use was counted for this request; keep the visit open.
		storeShareVisit(cookies, dataResponse, isSecureRequest(url, request));

		const viewData = await dataResponse.json();

		// Profile data comes from the API response (no direct PocketBase access)
		const profile = viewData.profile || null;

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
				accent_color: viewData.accent_color || null,
				hero_image_url: viewData.hero_image_url || null,
				cta_enabled: viewData.cta_enabled !== false,
				font_pack: viewData.font_pack || null,
				hero_layout: viewData.hero_layout || null,
				hero_spacing: viewData.hero_spacing || null,
				hero_bg_color: viewData.hero_bg_color || null
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
			sectionCategoryOrders: viewData.section_category_orders || {},
			sectionDisabledCategories: viewData.section_disabled_categories || {},
			sectionCategoryDisplayModes: viewData.section_category_display_modes || {},
			sectionFeaturedIds: viewData.section_featured_ids || {},
			postsTotalCount: viewData.posts_total_count ?? 0,
			talksTotalCount: viewData.talks_total_count ?? 0,
			requiresPassword: false,
			shareToken: effectiveShareToken || null,
			isPublicView: accessInfo.visibility === 'public',
			siteCtaEnabled: viewData.site_cta_enabled !== false,
			footerCtaEnabled: viewData.footer_cta_enabled !== false,
			siteNavEnabled: viewData.site_nav_enabled === true,
			siteNav: (await parent()).siteNav
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load view:', err);
		throw error(404, 'Not Found');
	}
};

// Form action to set password token cookie
export const actions: Actions = {
	setPasswordToken: async ({ cookies, request, url }) => {
		const data = await request.formData();
		const token = data.get('token') as string;
		const maxAge = parseInt(data.get('maxAge') as string) || 3600;

		if (token) {
			setPasswordToken(cookies, token, maxAge, isSecureRequest(url, request));
		}

		return { success: true };
	}
};

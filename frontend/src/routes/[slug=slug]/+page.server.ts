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
 */

import type { PageServerLoad, Actions } from './$types';
import { error, redirect } from '@sveltejs/kit';
import { getShareToken, getPasswordToken, setPasswordToken, setShareToken } from '$lib/tokens';

export const load: PageServerLoad = async ({ params, cookies, url, fetch, locals }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;

	const shareToken = getShareToken(cookies);
	const passwordToken = getPasswordToken(cookies);
	
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;

	// Check for URL token (legacy ?t= parameter)
	// If present, validate and redirect to clean URL
	const urlToken = url.searchParams.get('t');
	if (urlToken) {
		// Validate the token via API
		const validateResponse = await fetch(`${pbUrl}/api/share/validate`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ token: urlToken })
		});

		if (validateResponse.ok) {
			const result = await validateResponse.json();
			if (result.valid) {
				// Store token in cookie and redirect to clean URL
				setShareToken(cookies, urlToken, 7 * 24 * 60 * 60);
				throw redirect(302, `/${slug}`);
			}
		}
		// If token invalid, continue without it (will fail below if needed)
	}

	const effectiveShareToken = shareToken;

	try {
		const accessHeaders: Record<string, string> = {};
		if (pbAuthToken) {
			accessHeaders['Authorization'] = `Bearer ${pbAuthToken}`;
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
					name: accessInfo.view_name || 'Protected View'
				},
				profile: null,
				sections: {},
				requiresPassword: true
			};
		}

		const dataHeaders: Record<string, string> = {};
		if (effectiveShareToken) {
			dataHeaders['X-Share-Token'] = effectiveShareToken;
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
						view: { id: accessInfo.view_id, slug, name: accessInfo.view_name || 'Protected View' },
						profile: null,
						sections: {},
						requiresPassword: true
					};
				}
				// Invalid share token for unlisted = 404 (not discoverable)
				throw error(404, 'Not Found');
			}
			throw error(404, 'Not Found');
		}

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
				cta_text: viewData.cta_text,
				cta_url: viewData.cta_url,
				accent_color: viewData.accent_color || null
			},
			profile: profile ? {
				...profile,
				// File URLs come from API already resolved
				hero_image: profile.hero_image_url || null,
				avatar: profile.avatar_url || null
			} : null,
			sections: viewData.sections || {},
			sectionOrder: viewData.section_order || [],
			sectionLayouts: viewData.section_layouts || {},
			sectionWidths: viewData.section_widths || {},
			requiresPassword: false
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
	setPasswordToken: async ({ cookies, request }) => {
		const data = await request.formData();
		const token = data.get('token') as string;
		const maxAge = parseInt(data.get('maxAge') as string) || 3600;

		if (token) {
			setPasswordToken(cookies, token, maxAge);
		}

		return { success: true };
	}
};

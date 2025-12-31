import type { PageServerLoad, Actions } from './$types';
import { error } from '@sveltejs/kit';
import { getShareToken, getPasswordToken, setPasswordToken, buildTokenHeaders } from '$lib/tokens';

export const load: PageServerLoad = async ({ params, cookies, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;

	// Get tokens from cookies (set by /s/[token] or password flow)
	const shareToken = getShareToken(cookies);
	const passwordToken = getPasswordToken(cookies);

	// Also check for legacy URL token (for backwards compatibility, will be cleaned client-side)
	const urlToken = url.searchParams.get('t');
	const effectiveShareToken = shareToken || urlToken;

	try {
		// Get view access info
		const response = await fetch(`${pbUrl}/api/view/${slug}/access`);
		if (!response.ok) {
			throw error(404, 'View not found');
		}

		const accessInfo = await response.json();

		// Handle different visibility types
		if (accessInfo.visibility === 'private') {
			throw error(404, 'View not found');
		}

		// For unlisted views, we need a share token
		if (accessInfo.visibility === 'unlisted' && !effectiveShareToken) {
			throw error(404, 'View not found');
		}

		// For password-protected views without a token, return minimal data for password prompt
		if (accessInfo.visibility === 'password' && !passwordToken) {
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

		// Build headers with available tokens
		const headers = buildTokenHeaders(effectiveShareToken, passwordToken);

		// Fetch view data with tokens
		const dataResponse = await fetch(`${pbUrl}/api/view/${slug}/data`, { headers });

		if (!dataResponse.ok) {
			// If we have a token but it's invalid, show appropriate error
			if (dataResponse.status === 401 || dataResponse.status === 403) {
				if (accessInfo.visibility === 'password') {
					return {
						view: { id: accessInfo.view_id, slug, name: accessInfo.view_name || 'Protected View' },
						profile: null,
						sections: {},
						requiresPassword: true
					};
				}
				throw error(404, 'View not found');
			}
			throw error(404, 'View not found');
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
				cta_url: viewData.cta_url
			},
			profile: profile ? {
				...profile,
				// File URLs come from API already resolved
				hero_image: profile.hero_image_url || null,
				avatar: profile.avatar_url || null
			} : null,
			sections: viewData.sections || {},
			requiresPassword: false
		};
	} catch (err) {
		if ((err as { status?: number }).status === 404) {
			throw err;
		}
		console.error('Failed to load view:', err);
		return {
			view: null,
			profile: null,
			sections: {},
			error: 'View not found'
		};
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

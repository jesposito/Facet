import type { PageServerLoad, Actions } from './$types';
import { error, redirect } from '@sveltejs/kit';
import { getShareToken, getPasswordToken, setPasswordToken, setShareToken, buildTokenHeaders } from '$lib/tokens';

export const load: PageServerLoad = async ({ params, cookies, url, fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const { slug } = params;

	// Get tokens from cookies (set by /s/[token] or password flow)
	const shareToken = getShareToken(cookies);
	const passwordToken = getPasswordToken(cookies);

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
				throw redirect(302, `/v/${slug}`);
			}
		}
		// If token invalid, continue without it (will fail below if needed)
	}

	const effectiveShareToken = shareToken;

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

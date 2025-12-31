/**
 * Share link entry point: /s/<token>
 *
 * This route handles share links for unlisted views:
 * 1. Validates the share token server-side
 * 2. Sets an httpOnly cookie (me_share_token) for subsequent requests
 * 3. Redirects to the canonical URL /<slug> (token NOT in URL)
 *
 * The token is never exposed in the final URL, which:
 * - Prevents token leakage via browser history
 * - Prevents token leakage via Referer headers
 * - Keeps URLs clean and shareable
 */

import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';
import { setShareToken } from '$lib/tokens';

export const load: PageServerLoad = async ({ params, fetch, cookies }) => {
	const { token } = params;
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	try {
		// Validate the share token
		const response = await fetch(`${pbUrl}/api/share/validate`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ token })
		});

		if (!response.ok) {
			return { error: 'Invalid or expired share link' };
		}

		const result = await response.json();

		if (!result.valid) {
			return { error: result.error || 'Invalid or expired share link' };
		}

		// Store the validated token in a cookie for SSR access
		// Token is valid for 7 days (same as backend expiry)
		setShareToken(cookies, token, 7 * 24 * 60 * 60);

		// Redirect to the canonical URL WITHOUT token in URL (clean URLs)
		throw redirect(302, `/${result.view_slug}`);
	} catch (err) {
		if ((err as { status?: number }).status === 302) {
			throw err; // Re-throw redirect
		}
		console.error('Share token validation failed:', err);
		return { error: 'Invalid or expired share link' };
	}
};

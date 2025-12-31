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

		// Redirect to the view WITHOUT token in URL (clean URLs)
		throw redirect(302, `/v/${result.view_slug}`);
	} catch (err) {
		if ((err as { status?: number }).status === 302) {
			throw err; // Re-throw redirect
		}
		console.error('Share token validation failed:', err);
		return { error: 'Invalid or expired share link' };
	}
};

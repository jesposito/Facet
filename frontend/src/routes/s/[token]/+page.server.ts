import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params, fetch }) => {
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

		// Redirect to the view with a session marker
		// The view will be accessible because the token was validated
		throw redirect(302, `/v/${result.view_slug}?t=${token}`);
	} catch (err) {
		if ((err as { status?: number }).status === 302) {
			throw err; // Re-throw redirect
		}
		console.error('Share token validation failed:', err);
		return { error: 'Invalid or expired share link' };
	}
};

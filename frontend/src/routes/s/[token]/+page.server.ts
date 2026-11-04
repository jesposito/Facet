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
 *
 * A failed check answers with a code, so a busy or broken backend is never
 * reported as a bad link. A check can spend a use of the token, so this route
 * never repeats one: another attempt is always the recipient's own request.
 */

import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';
import { navigationFacet } from '$lib/discovery';
import { setShareToken, storeShareVisit, isSecureRequest } from '$lib/tokens';

// Longer waits are cut short so nobody watches an hour-long countdown. The
// backend asks for seconds, and an early retry is only denied again.
const MAX_RETRY_AFTER_SECONDS = 300;

type EntryFailure = {
	code: 'invalid' | 'rate_limited' | 'unavailable';
	retryAfter: number | null;
};

function failure(code: EntryFailure['code'], retryAfter: number | null = null): EntryFailure {
	return { code, retryAfter };
}

function retryAfterSeconds(value: string | null): number | null {
	if (!value || !/^\d+$/.test(value)) return null;
	const seconds = Number(value);
	return seconds > 0 ? Math.min(seconds, MAX_RETRY_AFTER_SECONDS) : null;
}

export const load: PageServerLoad = async ({
	params,
	fetch,
	cookies,
	url,
	request,
	setHeaders,
	getClientAddress
}) => {
	setHeaders({ 'Cache-Control': 'private, no-store', 'Referrer-Policy': 'no-referrer' });
	const { token } = params;
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	let response: Response;
	let result: { valid?: unknown; view_slug?: unknown } | null;

	try {
		// Validate the share token
		response = await fetch(`${pbUrl}/api/share/validate`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				// The backend keys this check's rate limit on this address.
				'X-Forwarded-For': getClientAddress()
			},
			body: JSON.stringify({ token }),
			signal: AbortSignal.timeout(15_000)
		});

		// A 429 is refused before the token is looked at, so it spent no use.
		if (response.status === 429) {
			const retryAfter = retryAfterSeconds(response.headers.get('Retry-After'));
			if (retryAfter) setHeaders({ 'Retry-After': String(retryAfter) });
			return failure('rate_limited', retryAfter);
		}

		// The backend may have counted a use before it failed, so this is neither
		// a bad link nor a good one.
		if (response.status >= 500) {
			console.error('Share entry check failed: backend status', response.status);
			return failure('unavailable');
		}

		if (!response.ok) {
			return failure('invalid');
		}

		result = await response.json();
	} catch (err) {
		// Only the error name: the message can carry the request URL.
		console.error('Share entry check failed:', err instanceof Error ? err.name : 'unknown');
		return failure('unavailable');
	}

	if (result?.valid === false) {
		return failure('invalid');
	}

	const slug = result?.view_slug;
	if (
		result?.valid !== true ||
		typeof slug !== 'string' ||
		!slug ||
		navigationFacet(slug) !== slug
	) {
		console.error('Share entry check failed: unexpected backend answer');
		return failure('unavailable');
	}

	// Store the validated token in a cookie for SSR access
	// Token is valid for 7 days (same as backend expiry)
	const secure = isSecureRequest(url, request);
	setShareToken(cookies, token, 7 * 24 * 60 * 60, secure);
	// Validation spent a use of the token; the visit proof lets this visit
	// continue on the view without spending another.
	storeShareVisit(cookies, response, secure);

	// Redirect to the canonical URL WITHOUT token in URL (clean URLs)
	throw redirect(302, `/${slug}`);
};

/**
 * Checkout return: /checkout/return?type=<type>&slug=<slug>[&session_id=cs_…][&cancelled=1]
 *
 * Stripe sends buyers here after checkout. The query string only says which
 * item and session to look at; it is never treated as proof of payment.
 * Access is re-checked with the backend using the visitor's purchase cookie,
 * which the backend sets only from a confirmed purchase's access link.
 *
 * The session itself is confirmed by the browser, not here. The confirm
 * endpoint is rate limited per client address, and a server-side call would
 * count every buyer against this server's address. The browser call also
 * means a link scanner fetching this page never triggers an email.
 */

import type { PageServerLoad } from './$types';
import { contentPath, isCheckoutSessionId, isPurchasableType } from '$lib/commerce';
import { internalApiHeaders } from '$lib/server/public-api';

const detailEndpoint = {
	posts: 'post',
	projects: 'project',
	talks: 'talk',
	courses: 'courses'
} as const;

export const load: PageServerLoad = async ({ url, fetch, request, locals, setHeaders }) => {
	// The address can hold a checkout session ID, so keep the page out of
	// caches, search results and Referer headers.
	setHeaders({
		'Cache-Control': 'private, no-store',
		'X-Robots-Tag': 'noindex, nofollow',
		'Referrer-Policy': 'no-referrer'
	});

	const type = url.searchParams.get('type');
	const slug = url.searchParams.get('slug') ?? '';
	if (!isPurchasableType(type) || !/^[a-z0-9][a-z0-9-]{0,199}$/i.test(slug)) {
		return {
			status: 'invalid' as const,
			type: null,
			slug: null,
			title: null,
			href: null,
			hasSession: false
		};
	}

	const href = contentPath(type, slug);
	// Anyone can type this address, so it only means the buyer left
	// checkout. It says nothing about whether a payment went through.
	if (url.searchParams.get('cancelled') === '1') {
		return { status: 'cancelled' as const, type, slug, title: null, href, hasSession: false };
	}

	// Only whether a well-formed ID is present. The browser reads the ID from
	// its own address bar, so it isn't copied into the page data.
	const hasSession = isCheckoutSessionId(url.searchParams.get('session_id'));

	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	const headers = {
		...internalApiHeaders(request),
		...(pbAuthToken ? { Authorization: `Bearer ${pbAuthToken}` } : {})
	};

	try {
		const res = await fetch(`${pbUrl}/api/${detailEndpoint[type]}/${encodeURIComponent(slug)}`, {
			headers
		});
		if (!res.ok) return { status: 'unknown' as const, type, slug, title: null, href, hasSession };
		const item = await res.json();
		const unlocked = type === 'courses' ? item.is_purchased === true : item.is_gated === false;
		return {
			status: unlocked ? ('unlocked' as const) : ('locked' as const),
			type,
			slug,
			title: typeof item.title === 'string' ? item.title : null,
			href,
			hasSession
		};
	} catch (err) {
		console.error(
			'Failed to check purchase after checkout:',
			err instanceof Error ? err.message : 'unknown error'
		);
		return { status: 'unknown' as const, type, slug, title: null, href, hasSession };
	}
};

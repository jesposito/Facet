// Server side of the /unsubscribe page. Backend contract (backend/hooks/newsletter.go):
//   GET  /api/public/unsubscribe/{token}  200 { email: masked, status } | 404 { error }
//   POST /api/public/unsubscribe          { token } -> 200 { status: 'unsubscribed' | 'already_unsubscribed' } | 400/404 { error }
//   POST /api/public/unsubscribe/by-email { email } -> 200 { status: 'received' } whether or not the address exists

// Newsletter tokens are 32 random bytes in unpadded base64url (43 characters).
// The range leaves room for a longer token later without accepting slashes,
// dots or percent signs that could change the backend path.
const TOKEN_PATTERN = /^[A-Za-z0-9_-]{16,128}$/;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export type TokenCheck =
	| { state: 'ready'; maskedEmail: string }
	| { state: 'already'; maskedEmail: string }
	| { state: 'invalid' }
	| { state: 'unavailable' };

export type UnsubscribeResult = 'unsubscribed' | 'already_unsubscribed' | 'invalid' | 'unavailable';

export function isValidUnsubscribeToken(token: string | undefined): token is string {
	return typeof token === 'string' && TOKEN_PATTERN.test(token);
}

export function normaliseEmail(value: unknown): string | null {
	if (typeof value !== 'string') return null;
	const email = value.trim().toLowerCase();
	return email.length <= 254 && EMAIL_PATTERN.test(email) ? email : null;
}

function backendUrl(path: string): string {
	const pbUrl = (process.env.POCKETBASE_URL || 'http://localhost:8090').replace(/\/$/, '');
	return `${pbUrl}${path}`;
}

async function readJson(response: Response): Promise<Record<string, unknown>> {
	try {
		const data = await response.json();
		return data && typeof data === 'object' ? (data as Record<string, unknown>) : {};
	} catch {
		return {};
	}
}

// Looks the token up without changing anything. Mail scanners open links, so
// the GET has to be safe; the unsubscribe itself happens on POST.
export async function checkUnsubscribeToken(
	fetcher: typeof fetch,
	token: string
): Promise<TokenCheck> {
	if (!isValidUnsubscribeToken(token)) return { state: 'invalid' };
	let response: Response;
	try {
		response = await fetcher(backendUrl(`/api/public/unsubscribe/${encodeURIComponent(token)}`));
	} catch {
		return { state: 'unavailable' };
	}
	if (response.status === 400 || response.status === 404) return { state: 'invalid' };
	if (!response.ok) return { state: 'unavailable' };

	const data = await readJson(response);
	const maskedEmail = typeof data.email === 'string' ? data.email : '';
	return data.status === 'unsubscribed'
		? { state: 'already', maskedEmail }
		: { state: 'ready', maskedEmail };
}

export async function confirmUnsubscribe(
	fetcher: typeof fetch,
	token: string
): Promise<UnsubscribeResult> {
	if (!isValidUnsubscribeToken(token)) return 'invalid';
	let response: Response;
	try {
		response = await fetcher(backendUrl('/api/public/unsubscribe'), {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ token })
		});
	} catch {
		return 'unavailable';
	}
	if (response.status === 400 || response.status === 404) return 'invalid';
	if (!response.ok) return 'unavailable';

	const data = await readJson(response);
	if (data.status === 'already_unsubscribed') return 'already_unsubscribed';
	if (data.status === 'unsubscribed') return 'unsubscribed';
	return 'unavailable';
}

// The backend answers "received" for every address so nobody can use this
// form to learn who subscribes. The page has to say the same thing.
export async function requestUnsubscribeByEmail(
	fetcher: typeof fetch,
	email: string
): Promise<boolean> {
	try {
		const response = await fetcher(backendUrl('/api/public/unsubscribe/by-email'), {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email })
		});
		return response.ok;
	} catch {
		return false;
	}
}

/**
 * Headers for every unsubscribe response. The token in the URL works like a
 * password for one subscription, so the page must not be cached, indexed or
 * leaked through a Referer header to anything it links to.
 *
 * The root layout already sets Cache-Control for signed-in visitors, and
 * SvelteKit refuses to set a header twice, so pass whether it did.
 */
export function unsubscribeHeaders(layoutSetCacheControl: boolean): Record<string, string> {
	return {
		...(layoutSetCacheControl ? {} : { 'Cache-Control': 'private, no-store' }),
		'X-Robots-Tag': 'noindex, nofollow',
		'Referrer-Policy': 'no-referrer'
	};
}

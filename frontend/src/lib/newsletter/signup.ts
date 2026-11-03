/**
 * Public newsletter signup. A form either names the list it signs people up
 * to or leaves it out and gets the site's default list. The server refuses a
 * list that isn't open to the public instead of quietly using another, and
 * this module passes that refusal on instead of showing success.
 */

export type SignupInput = {
	email: string;
	name?: string;
	source?: string;
	/** List id or slug. Empty or missing means the default list. */
	list?: string;
	turnstileToken?: string;
};

export function buildSignupBody(input: SignupInput): Record<string, unknown> {
	const body: Record<string, unknown> = {
		email: input.email.trim().toLowerCase(),
		name: (input.name ?? '').trim()
	};
	if (input.source) body.source = input.source;
	const list = (input.list ?? '').trim();
	if (list) body.list = list;
	if (input.turnstileToken) body.turnstile_token = input.turnstileToken;
	return body;
}

export type SignupOutcome = 'confirmation_required' | 'list_unavailable' | 'rate_limited' | 'error';

/** Reads the signup reply. Only a 200 counts as success. */
export async function readSignupOutcome(response: Response): Promise<SignupOutcome> {
	if (response.ok) return 'confirmation_required';
	if (response.status === 429) return 'rate_limited';
	const data = await response.json().catch(() => ({}));
	if (data && data.code === 'list_unavailable') return 'list_unavailable';
	return 'error';
}

export type PublicNewsletterList = {
	id: string;
	slug: string;
	name: string;
	description: string;
	is_default: boolean;
};

/** The lists a visitor may join, for forms that let them choose. */
export async function loadPublicNewsletterLists(
	fetchImpl: typeof fetch = fetch
): Promise<PublicNewsletterList[]> {
	try {
		const response = await fetchImpl('/api/public/newsletter/lists');
		if (!response.ok) return [];
		const data = await response.json();
		return Array.isArray(data?.data) ? data.data : [];
	} catch {
		return [];
	}
}

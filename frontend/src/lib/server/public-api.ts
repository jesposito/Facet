const INTERNAL_AUTH_COOKIES = ['facet_purchases', 'facet_totp_session'] as const;
const PRIVATE_RESPONSE_COOKIES = [
	...INTERNAL_AUTH_COOKIES,
	'pb_auth',
	'me_share_token',
	'me_password_token'
] as const;
const COOKIE_VALUE = /^[\x21\x23-\x2B\x2D-\x3A\x3C-\x5B\x5D-\x7E]+$/;

type JsonObject = { [key: string]: unknown };

export interface PublicMediaRef {
	id: string;
	url: string;
	title?: string;
	alt_text?: string;
	description?: string;
	provider?: string;
	mime?: string;
	thumbnail_url?: string;
}

export interface PublicPost {
	id: string;
	title: string;
	slug: string;
	excerpt?: string;
	published_at?: string;
	cover_image_url?: string;
	cover_image_large_url?: string;
	cover_image_thumb_url?: string;
	visibility?: string;
	is_draft?: boolean;
	tags?: string[];
}

export interface PublicTalk {
	id: string;
	title: string;
	slug: string;
	event?: string;
	event_url?: string;
	date?: string;
	location?: string;
	slides_url?: string;
	video_url?: string;
	cover_image_library_url?: string;
	visibility?: string;
	is_draft?: boolean;
}

function selectedCookies(header: string | null, names: readonly string[]): Map<string, string> {
	const selected = new Map<string, string>();
	if (!header) return selected;

	for (const part of header.split(';')) {
		const separator = part.indexOf('=');
		if (separator < 1) continue;
		const name = part.slice(0, separator).trim();
		const value = part.slice(separator + 1).trim();
		if (names.includes(name) && COOKIE_VALUE.test(value) && !selected.has(name)) {
			selected.set(name, value);
		}
	}
	return selected;
}

export function internalApiHeaders(request: Request): Record<string, string> {
	const cookies = selectedCookies(request.headers.get('cookie'), INTERNAL_AUTH_COOKIES);
	const cookie = Array.from(cookies, ([name, value]) => `${name}=${value}`).join('; ');
	return {
		'X-Internal': 'true',
		...(cookie ? { Cookie: cookie } : {})
	};
}

export function publicCacheHeaders(
	request: Request,
	hasAdditionalAuth = false
): Record<string, string> {
	const hasAuthCookie =
		selectedCookies(request.headers.get('cookie'), PRIVATE_RESPONSE_COOKIES).size > 0;
	return {
		Vary: 'Cookie',
		...(hasAdditionalAuth || hasAuthCookie ? { 'Cache-Control': 'private, no-store' } : {})
	};
}

export function applyPublicCacheHeaders(
	request: Request,
	headers: Headers,
	hasAdditionalAuth = false
): void {
	const cacheHeaders = publicCacheHeaders(request, hasAdditionalAuth);
	const vary = (headers.get('Vary') || '')
		.split(',')
		.map((token) => token.trim())
		.filter(Boolean);
	if (!vary.some((token) => token === '*' || token.toLowerCase() === 'cookie')) {
		vary.push('Cookie');
	}
	headers.set('Vary', vary.join(', '));
	if (cacheHeaders['Cache-Control']) {
		headers.set('Cache-Control', cacheHeaders['Cache-Control']);
	}
}

function asObject(value: unknown): JsonObject | null {
	return typeof value === 'object' && value !== null && !Array.isArray(value)
		? (value as JsonObject)
		: null;
}

function requiredString(record: JsonObject, key: string): string | null {
	const value = record[key];
	return typeof value === 'string' && value.length > 0 ? value : null;
}

function optionalStrings<const Key extends string>(
	record: JsonObject,
	keys: readonly Key[]
): Partial<Record<Key, string>> {
	const result: Partial<Record<Key, string>> = {};
	for (const key of keys) {
		if (typeof record[key] === 'string') result[key] = record[key];
	}
	return result;
}

function optionalDraft(record: JsonObject): { is_draft?: boolean } {
	return typeof record.is_draft === 'boolean' ? { is_draft: record.is_draft } : {};
}

function stringArray(value: unknown): string[] | undefined {
	return Array.isArray(value) && value.every((item) => typeof item === 'string')
		? value
		: undefined;
}

function parseMediaRef(value: unknown): PublicMediaRef | null {
	const record = asObject(value);
	if (!record) return null;
	const id = requiredString(record, 'id');
	const url = requiredString(record, 'url');
	if (!id || !url) return null;
	return {
		id,
		url,
		...optionalStrings(record, [
			'title',
			'alt_text',
			'description',
			'provider',
			'mime',
			'thumbnail_url'
		])
	};
}

export function parseMediaRefs(value: unknown): PublicMediaRef[] {
	return Array.isArray(value)
		? value.map(parseMediaRef).filter((media): media is PublicMediaRef => media !== null)
		: [];
}

function parsePublicPost(value: unknown): PublicPost | null {
	const record = asObject(value);
	if (!record) return null;
	const id = requiredString(record, 'id');
	const title = requiredString(record, 'title');
	const slug = requiredString(record, 'slug');
	if (!id || !title || !slug) return null;
	const tags = stringArray(record.tags);
	return {
		id,
		title,
		slug,
		...optionalStrings(record, [
			'excerpt',
			'published_at',
			'cover_image_url',
			'cover_image_large_url',
			'cover_image_thumb_url',
			'visibility'
		]),
		...optionalDraft(record),
		...(tags ? { tags } : {})
	};
}

export function parsePublicPosts(value: unknown): PublicPost[] {
	return Array.isArray(value)
		? value.map(parsePublicPost).filter((post): post is PublicPost => post !== null)
		: [];
}

function parsePublicTalk(value: unknown): PublicTalk | null {
	const record = asObject(value);
	if (!record) return null;
	const id = requiredString(record, 'id');
	const title = requiredString(record, 'title');
	const slug = requiredString(record, 'slug');
	if (!id || !title || !slug) return null;
	return {
		id,
		title,
		slug,
		...optionalStrings(record, [
			'event',
			'event_url',
			'date',
			'location',
			'slides_url',
			'video_url',
			'cover_image_library_url',
			'visibility'
		]),
		...optionalDraft(record)
	};
}

export function parsePublicTalks(value: unknown): PublicTalk[] {
	return Array.isArray(value)
		? value.map(parsePublicTalk).filter((talk): talk is PublicTalk => talk !== null)
		: [];
}

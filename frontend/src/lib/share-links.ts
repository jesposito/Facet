/** Highest use limit a share link can have. Matches maxShareTokenUses in the backend. */
export const MAX_SHARE_LINK_USES = 1_000_000;

let createdLink: { viewId: string; url: string } | null = null;

/** Carry a newly revealed link across client navigation without storing it. */
export function handoffCreatedShareLink(viewId: string, url: string): void {
	createdLink = { viewId, url };
}

export function takeCreatedShareLink(viewId: string): string | null {
	if (createdLink?.viewId !== viewId) return null;
	const url = createdLink.url;
	createdLink = null;
	return url;
}

/**
 * A datetime-local value is wall-clock time with no zone, so it has to become
 * an absolute instant here, where the browser knows the owner's timezone.
 * Returns '' for no expiry and null when the value isn't a real date.
 */
export function localExpiryToISO(value: string): string | null {
	if (!value) return '';
	const parts = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/.exec(value);
	if (!parts) return null;
	const [year, month, day, hour, minute, second] = parts.slice(1).map(Number);
	const date = new Date(value);
	// Date normalizes impossible calendar dates and daylight-saving gaps.
	// Refuse those values instead of silently choosing a different time.
	if (
		date.getFullYear() !== year ||
		date.getMonth() + 1 !== month ||
		date.getDate() !== day ||
		date.getHours() !== hour ||
		date.getMinutes() !== minute ||
		date.getSeconds() !== (second || 0) ||
		date.getTime() <= Date.now()
	)
		return null;
	return date.toISOString();
}

/** An empty number input binds as null, which means no limit. */
export function isValidMaxUses(value: number | null | undefined): boolean {
	const uses = value ?? 0;
	return Number.isInteger(uses) && uses >= 0 && uses <= MAX_SHARE_LINK_USES;
}

/**
 * Replaces a share link in one server-side step. The raw token in the result
 * is the only copy, so callers show it once and keep it out of storage.
 */
export async function regenerateShareLink(
	tokenId: string,
	authToken: string,
	fallbackError: string
): Promise<{ id: string; token: string; name: string }> {
	const response = await fetch(`/api/share/regenerate/${tokenId}`, {
		method: 'POST',
		headers: { Authorization: `Bearer ${authToken}` }
	});
	const data = await response.json().catch(() => null);
	if (
		!response.ok ||
		typeof data?.token !== 'string' ||
		!data.token ||
		typeof data.id !== 'string'
	) {
		throw new Error(data?.error || fallbackError);
	}
	return { id: data.id, token: data.token, name: typeof data.name === 'string' ? data.name : '' };
}

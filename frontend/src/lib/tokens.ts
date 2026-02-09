/**
 * Token management utilities for share tokens and password JWTs
 *
 * Tokens are stored in httpOnly cookies for SSR access and cleaned from URLs.
 */

import type { Cookies } from '@sveltejs/kit';

export const TOKEN_COOKIES = {
	SHARE: 'me_share_token',
	PASSWORD: 'me_password_token'
} as const;

/**
 * Determine if cookies should use the Secure flag based on request context.
 * Priority: X-Forwarded-Proto (set by Caddy/reverse proxies) > URL protocol.
 * NODE_ENV is NOT used because it's always 'production' in Docker regardless of protocol.
 * Note: url.protocol requires PROTOCOL_HEADER env in adapter-node; without it, defaults to 'https:'.
 */
export function isSecureRequest(url: URL, request?: Request): boolean {
	const forwardedProto = request?.headers.get('x-forwarded-proto');
	return forwardedProto
		? forwardedProto === 'https'
		: url.protocol === 'https:';
}

// Base cookie options (secure flag added per-request)
function getCookieOptions(secure: boolean) {
	return {
		path: '/',
		httpOnly: true,
		sameSite: 'lax' as const,
		secure
	};
}

/**
 * Set a share token cookie
 */
export function setShareToken(cookies: Cookies, token: string, maxAge = 7 * 24 * 60 * 60, secure = false) {
	cookies.set(TOKEN_COOKIES.SHARE, token, {
		...getCookieOptions(secure),
		maxAge
	});
}

/**
 * Get share token from cookies
 */
export function getShareToken(cookies: Cookies): string | null {
	return cookies.get(TOKEN_COOKIES.SHARE) || null;
}

/**
 * Clear share token cookie
 */
export function clearShareToken(cookies: Cookies) {
	cookies.delete(TOKEN_COOKIES.SHARE, { path: '/' });
}

/**
 * Set a password JWT cookie
 */
export function setPasswordToken(cookies: Cookies, token: string, maxAge = 60 * 60, secure = false) {
	cookies.set(TOKEN_COOKIES.PASSWORD, token, {
		...getCookieOptions(secure),
		maxAge
	});
}

/**
 * Get password token from cookies
 */
export function getPasswordToken(cookies: Cookies): string | null {
	return cookies.get(TOKEN_COOKIES.PASSWORD) || null;
}

/**
 * Clear password token cookie
 */
export function clearPasswordToken(cookies: Cookies) {
	cookies.delete(TOKEN_COOKIES.PASSWORD, { path: '/' });
}

/**
 * Build headers for API requests with available tokens
 *
 * Token transport:
 * - Password JWT: Authorization: Bearer <jwt> (standards compliant)
 * - Share token: X-Share-Token: <token> (custom header, avoids Authorization conflict)
 */
export function buildTokenHeaders(shareToken: string | null, passwordToken: string | null): Record<string, string> {
	const headers: Record<string, string> = {};

	// Share tokens use a custom header to avoid conflict with password JWT
	if (shareToken) {
		headers['X-Share-Token'] = shareToken;
	}

	// Password JWT uses standard Bearer authentication
	if (passwordToken) {
		headers['Authorization'] = `Bearer ${passwordToken}`;
	}

	return headers;
}

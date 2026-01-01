/**
 * Client-side hooks for debugging navigation issues
 */

import { beforeNavigate, afterNavigate } from '$app/navigation';

// Log navigation events in development
if (typeof window !== 'undefined') {
	console.log('[HOOKS CLIENT] Initializing client hooks');

	// Note: These hooks need to be set up in a component, not in hooks.client.ts
	// This file is for handle/handleError hooks only
}

/** @type {import('@sveltejs/kit').HandleClientError} */
export function handleError({ error, event, status, message }) {
	console.error('[HOOKS CLIENT] Client error:', {
		status,
		message,
		url: event?.url?.href,
		error
	});

	return {
		message: message || 'An error occurred',
		status
	};
}

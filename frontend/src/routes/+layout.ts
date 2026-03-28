import { waitLocale } from '$lib/i18n';
import type { LoadEvent } from '@sveltejs/kit';

/**
 * Universal load function to ensure i18n is ready before rendering.
 * SSR data (favicon, accent color, plan config) is handled by +layout.server.ts.
 */
export const load = async ({ fetch }: LoadEvent) => {
	await waitLocale();
	return {};
};

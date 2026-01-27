// Import from our i18n module to ensure init() is called at module load time
import { waitLocale } from '$lib/i18n';

/**
 * Universal load function to ensure i18n is ready before rendering
 * This prevents hydration errors by waiting for the locale to load
 * before the page renders on both server and client
 */
export const load = async () => {
	await waitLocale();
	return {};
};

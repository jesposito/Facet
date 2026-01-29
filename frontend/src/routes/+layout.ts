// Import from our i18n module to ensure init() is called at module load time
import { waitLocale } from '$lib/i18n';
import type { LoadEvent } from '@sveltejs/kit';

/**
 * Universal load function to ensure i18n is ready before rendering
 * and to load site settings (including favicon) for SSR
 */
export const load = async ({ fetch }: LoadEvent) => {
	await waitLocale();

	// Load site settings server-side so favicon is correct on initial render
	let faviconUrl: string | null = null;
	try {
		const response = await fetch(`/api/site-settings?_=${Date.now()}`);
		if (response.ok) {
			const data = await response.json();
			if (data.favicon) {
				faviconUrl = `${data.favicon}?v=${Date.now()}`;
			}
		}
	} catch {
		// Silently fail - will use default favicon
	}

	return { faviconUrl };
};

import { initI18n, waitLocale } from '$lib/i18n';
import type { LayoutLoad } from './$types';

/**
 * Universal load function to ensure i18n is ready before rendering.
 *
 * IMPORTANT: must forward `data` from +layout.server.ts. SvelteKit replaces
 * the server data with whatever this universal load returns, so returning
 * `{}` silently discards faviconUrl / planConfig / siteNav / accentColor /
 * customCSS / etc. — which caused the favicon SSR regression where the
 * page rendered the fallback `<link rel="icon" href="/favicon.png">`
 * even though the server fetched the correct `/api/favicon` URL.
 */
export const load: LayoutLoad = async ({ data }) => {
	initI18n(data.defaultLocale || undefined);
	await waitLocale();
	return { ...data };
};

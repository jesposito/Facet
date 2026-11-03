import type { SiteFeatures } from '$lib/stores/features';

const GA_SCRIPT_ID = 'ga-script';
const GA_INLINE_ID = 'ga-inline';

/**
 * Whether the public site may load Google Analytics. The owner's Analytics
 * switch wins over a saved measurement ID: with the switch off, nothing from
 * Google is loaded at all.
 */
export function shouldLoadGA(
	features: Pick<SiteFeatures, 'analytics'>,
	id: string | null | undefined
): boolean {
	return features.analytics === true && typeof id === 'string' && sanitizeGAId(id) !== '';
}

export function sanitizeGAId(id: string): string {
	return id.trim().replace(/[^A-Za-z0-9-]/g, '');
}

/** Adds the GA4 snippet once. */
export function injectGA(doc: Document, id: string) {
	const clean = sanitizeGAId(id);
	if (!clean || doc.getElementById(GA_SCRIPT_ID)) return;

	const script = doc.createElement('script');
	script.id = GA_SCRIPT_ID;
	script.async = true;
	script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(clean)}`;
	doc.head.appendChild(script);

	const inline = doc.createElement('script');
	inline.id = GA_INLINE_ID;
	inline.textContent = `
	  window.dataLayer = window.dataLayer || [];
	  function gtag(){dataLayer.push(arguments);}
	  gtag('js', new Date());
	  gtag('config', '${clean}');
	`;
	doc.head.appendChild(inline);
}

/**
 * Takes the snippet back out when Analytics is switched off during a visit,
 * and sets Google's documented opt-out flag so an already loaded gtag stops
 * sending hits for this ID.
 */
export function removeGA(doc: Document, id?: string) {
	doc.getElementById(GA_SCRIPT_ID)?.remove();
	doc.getElementById(GA_INLINE_ID)?.remove();
	const clean = id ? sanitizeGAId(id) : '';
	const win = doc.defaultView as (Window & Record<string, unknown>) | null;
	if (win && clean) {
		win[`ga-disable-${clean}`] = true;
	}
}

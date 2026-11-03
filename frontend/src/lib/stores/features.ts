import { writable } from 'svelte/store';

/**
 * The owner's feature switches from Settings → Features
 * (site_settings.enabled_features). Every feature is on unless it was saved
 * as false, matching the backend, so a missing key never hides anything.
 */
export type SiteFeatures = {
	analytics: boolean;
	courses: boolean;
	newsletter: boolean;
	discussions: boolean;
};

export const ALL_FEATURES_ON: SiteFeatures = {
	analytics: true,
	courses: true,
	newsletter: true,
	discussions: true
};

export function resolveSiteFeatures(raw: unknown): SiteFeatures {
	const saved = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
	return {
		analytics: saved.analytics !== false,
		courses: saved.courses !== false,
		newsletter: saved.newsletter !== false,
		discussions: saved.discussions !== false
	};
}

export const siteFeatures = writable<SiteFeatures>({ ...ALL_FEATURES_ON });

export function setSiteFeatures(raw: unknown) {
	siteFeatures.set(resolveSiteFeatures(raw));
}

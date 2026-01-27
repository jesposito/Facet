/**
 * i18n configuration for Facet
 *
 * Uses svelte-i18n with simple JSON translation files.
 * Community members can contribute translations by editing files in src/locales/
 *
 * @see CONTRIBUTING_TRANSLATIONS.md for contribution guidelines
 */

import { register, init, getLocaleFromNavigator, locale, waitLocale } from 'svelte-i18n';
import { browser } from '$app/environment';

// Register available locales - add new languages here
register('en', () => import('../../locales/en.json'));
// register('es', () => import('../../locales/es.json'));
// register('fr', () => import('../../locales/fr.json'));
// register('de', () => import('../../locales/de.json'));

// Supported locales for validation and UI
export const SUPPORTED_LOCALES = ['en'] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

// Human-readable locale names for language switcher
export const LOCALE_NAMES: Record<string, string> = {
	en: 'English',
	es: 'Espanol',
	fr: 'Francais',
	de: 'Deutsch',
	ja: 'Japanese',
	'pt-BR': 'Portugues (Brasil)',
	'zh-CN': 'Chinese (Simplified)'
};

const STORAGE_KEY = 'facet-locale';

/**
 * Get stored locale from localStorage
 */
function getStoredLocale(): string | null {
	if (!browser) return null;
	return localStorage.getItem(STORAGE_KEY);
}

/**
 * Initialize i18n - call this in root layout onMount
 */
export function initI18n() {
	const storedLocale = getStoredLocale();
	const browserLocale = browser ? getLocaleFromNavigator() : null;

	// Determine initial locale: stored > browser > default
	let initialLocale = 'en';
	if (storedLocale && isSupported(storedLocale)) {
		initialLocale = storedLocale;
	} else if (browserLocale) {
		// Try exact match first, then language code only
		const langCode = browserLocale.split('-')[0];
		if (isSupported(browserLocale)) {
			initialLocale = browserLocale;
		} else if (isSupported(langCode)) {
			initialLocale = langCode;
		}
	}

	init({
		fallbackLocale: 'en',
		initialLocale
	});
}

/**
 * Set and persist locale preference
 */
export function setLocale(newLocale: string) {
	if (!browser) return;
	if (!isSupported(newLocale)) {
		console.warn(`Locale "${newLocale}" is not supported, falling back to "en"`);
		newLocale = 'en';
	}
	localStorage.setItem(STORAGE_KEY, newLocale);
	locale.set(newLocale);
}

/**
 * Check if a locale is supported
 */
export function isSupported(loc: string): loc is SupportedLocale {
	return SUPPORTED_LOCALES.includes(loc as SupportedLocale);
}

/**
 * Wait for locale to be loaded - use in layout before rendering
 */
export { waitLocale };

/**
 * Re-export locale store for reactive access
 */
export { locale };

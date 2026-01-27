/**
 * i18n configuration for Facet
 *
 * Uses svelte-i18n with simple JSON translation files.
 * Community members can contribute translations by editing files in src/locales/
 *
 * @see CONTRIBUTING_TRANSLATIONS.md for contribution guidelines
 */

import { register, init, locale, waitLocale } from 'svelte-i18n';

// Register available locales - add new languages here
register('en', () => import('../../locales/en.json'));
register('de', () => import('../../locales/de.json'));
register('elvish', () => import('../../locales/elvish.json'));
register('klingon', () => import('../../locales/klingon.json'));
register('lolcat', () => import('../../locales/lolcat.json'));
// register('es', () => import('../../locales/es.json'));
// register('fr', () => import('../../locales/fr.json'));

// Supported locales for validation and UI
export const SUPPORTED_LOCALES = ['en', 'de', 'elvish', 'klingon', 'lolcat'] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

// Human-readable locale names for language switcher
export const LOCALE_NAMES: Record<string, string> = {
	en: 'English',
	de: 'Deutsch',
	elvish: 'Sindarin (Elvish)',
	klingon: 'tlhIngan Hol (Klingon)',
	lolcat: 'LOLcat',
	es: 'Espanol',
	fr: 'Francais',
	ja: 'Japanese',
	'pt-BR': 'Portugues (Brasil)',
	'zh-CN': 'Chinese (Simplified)'
};

// Initialize i18n immediately for SSR support
// This ensures $t() works during server-side rendering
init({
	fallbackLocale: 'en',
	initialLocale: 'en'
});

/**
 * Initialize i18n with user preferences - call this in root layout onMount
 * This re-initializes with stored/browser locale on the client
 * @param serverDefaultLocale - Optional default locale from server settings
 */
export function initI18n(serverDefaultLocale?: string) {
	// Determine initial locale: server setting > default
	let initialLocale = 'en';
	if (serverDefaultLocale && isSupported(serverDefaultLocale)) {
		initialLocale = serverDefaultLocale;
	}

	// Set the locale (init was already called at module load for SSR)
	locale.set(initialLocale);
}

/**
 * Set the current locale (does not persist - use server settings for persistence)
 */
export function setLocale(newLocale: string) {
	if (!isSupported(newLocale)) {
		console.warn(`Locale "${newLocale}" is not supported, falling back to "en"`);
		newLocale = 'en';
	}
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

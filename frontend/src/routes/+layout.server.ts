import type { LayoutServerLoad } from './$types';
import type { PlanConfig } from '$lib/stores/plan';
import { logger } from '$lib/logger';

export const load: LayoutServerLoad = async ({ fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	// Fetch site settings server-side (favicon, custom CSS, locale)
	let faviconUrl: string | null = null;
	let customCSS: string | null = null;
	let defaultLocale: string | null = null;
	try {
		const siteSettingsResponse = await fetch(`${pbUrl}/api/site-settings`, {
			headers: { 'X-Internal': 'true' }
		});
		if (siteSettingsResponse.ok) {
			const siteSettings = await siteSettingsResponse.json();
			faviconUrl = siteSettings.favicon || null;
			customCSS = siteSettings.custom_css || null;
			defaultLocale = siteSettings.default_locale || null;
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load site settings:', error);
	}

	// Fetch plan config server-side (FOUC fix)
	let planConfig: PlanConfig | null = null;
	try {
		const planResponse = await fetch(`${pbUrl}/api/plan`, {
			headers: { 'X-Internal': 'true' }
		});
		if (planResponse.ok) {
			planConfig = await planResponse.json();
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load plan config:', error);
	}

	// Fetch site-nav server-side to eliminate nav pop-in on page load
	let siteNav = { enabled: false, items: [] as Array<{ viewId: string; slug: string; label: string; name: string }> };
	try {
		const navResponse = await fetch(`${pbUrl}/api/site-nav`, {
			headers: { 'X-Internal': 'true' }
		});
		if (navResponse.ok) {
			const navData = await navResponse.json();
			siteNav = { enabled: navData.enabled === true, items: navData.items || [] };
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load site nav:', error);
	}

	// Fetch profile accent color server-side to eliminate color flash
	let accentColor: string | null = null;
	let customHexColor: string | null = null;
	try {
		const homepageResponse = await fetch(`${pbUrl}/api/homepage`, {
			headers: { 'X-Internal': 'true' }
		});
		if (homepageResponse.ok) {
			const homepage = await homepageResponse.json();
			accentColor = homepage.profile?.accent_color || null;
			customHexColor = homepage.profile?.custom_hex_color || null;
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load accent color:', error);
	}

	return {
		faviconUrl,
		planConfig,
		siteNav,
		accentColor,
		customHexColor,
		customCSS,
		defaultLocale
	};
};

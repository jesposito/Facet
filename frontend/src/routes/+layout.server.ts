/**
 * Root layout server load: Preload plan config server-side to eliminate FOUC
 *
 * Fetches plan configuration from PocketBase before first render, ensuring
 * the "Powered by Facet" badge visibility state is correct on initial paint.
 *
 * Pro/Creator users (managed, !badge_forced) will never see the badge flash.
 */

import type { LayoutServerLoad } from './$types';
import type { PlanConfig } from '$lib/stores/plan';
import { logger } from '$lib/logger';
import { SOFT_PREMIUM_DEFAULT_ACCENT } from '$lib/colors';
import { DEFAULT_FONT_PACK } from '$lib/fonts';

export const load: LayoutServerLoad = async ({ fetch, locals }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	const {
		faviconUrl,
		customCSS,
		defaultLocale,
		showAvatar,
		defaultThemeMode,
		design,
	} = locals.siteSettings ?? {
		faviconUrl: null,
		customCSS: null,
		defaultLocale: null,
		showAvatar: true,
		defaultThemeMode: 'system',
		design: 'classic' as const,
	};

	// Fetch plan config server-side (new functionality - FOUC fix)
	let planConfig: PlanConfig | null = null;
	try {
		const planResponse = await fetch(`${pbUrl}/api/plan`, {
			headers: { 'X-Internal': 'true' },
		});
		if (planResponse.ok) {
			planConfig = await planResponse.json();
			logger.debug('[LAYOUT SSR] Loaded plan config:', planConfig?.plan);
		} else {
			logger.debug('[LAYOUT SSR] Plan API returned non-OK status:', planResponse.status);
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load plan config:', error);
	}

	// Pass APP_URL to the client for canonical/OG URLs.
	// SSR $page.url.origin reflects the internal HTTP origin (behind Caddy proxy),
	// not the public HTTPS URL. APP_URL is authoritative.
	const appUrl = process.env.APP_URL || '';

	// Fetch site-nav server-side to eliminate nav pop-in on page load
	let siteNav = {
		enabled: false,
		mode: 'bar',
		position: 'below',
		items: [] as Array<{
			viewId: string;
			slug: string;
			label: string;
			name: string;
		}>,
	};
	try {
		const navResponse = await fetch(`${pbUrl}/api/site-nav`, {
			headers: { 'X-Internal': 'true' },
		});
		if (navResponse.ok) {
			const navData = await navResponse.json();
			siteNav = {
				enabled: navData.enabled === true,
				mode: navData.mode || 'bar',
				position: navData.position || 'below',
				items: navData.items || [],
			};
			logger.debug('[LAYOUT SSR] Loaded site nav:', siteNav.enabled, siteNav.items.length, 'items');
		} else {
			logger.debug('[LAYOUT SSR] Site nav API returned:', navResponse.status);
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load site nav:', error);
	}

	// Fetch profile accent color and font pack server-side to eliminate flash on page load
	let accentColor: string | null = null;
	let customHexColor: string | null = null;
	let fontPack: string | null = null;
	let accentExplicit = false;
	let fontPackExplicit = false;
	try {
		const homepageResponse = await fetch(`${pbUrl}/api/homepage`, {
			headers: { 'X-Internal': 'true' },
		});
		if (homepageResponse.ok) {
			const homepage = await homepageResponse.json();
			accentColor = homepage.profile?.accent_color || null;
			customHexColor = homepage.profile?.custom_hex_color || null;
			fontPack = homepage.profile?.font_pack || null;
			accentExplicit = !!(accentColor || customHexColor);
			fontPackExplicit = !!fontPack;
		}
	} catch (error) {
		logger.debug('[LAYOUT SSR] Failed to load accent color:', error);
	}

	// Soft Premium defaults to warm terracotta and Hanken/Newsreader only when the
	// operator has not chosen an accent/font. Classic keeps the static defaults.
	if (design === 'soft-premium' && !accentColor && !customHexColor) {
		customHexColor = SOFT_PREMIUM_DEFAULT_ACCENT;
	}
	if (design === 'soft-premium' && !fontPack) {
		fontPack = DEFAULT_FONT_PACK;
	}

	return {
		faviconUrl,
		planConfig,
		appUrl,
		siteNav,
		accentColor,
		customHexColor,
		accentExplicit,
		fontPack,
		fontPackExplicit,
		customCSS,
		defaultLocale,
		showAvatar,
		defaultThemeMode,
		design,
	};
};

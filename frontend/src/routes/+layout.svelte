<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { beforeNavigate, afterNavigate } from '$app/navigation';
	import { theme, toasts } from '$lib/stores';
	import Toast from '$components/shared/Toast.svelte';
	import ConfirmDialog from '$components/shared/ConfirmDialog.svelte';
	import { ACCENT_COLORS, DEFAULT_ACCENT_COLOR, type AccentColor, generatePaletteFromHex, hexToRgbChannels, buildPaletteRootBlock } from '$lib/colors';
	import { getFontPack, DEFAULT_FONT_PACK, type FontPack } from '$lib/fonts';
	import { initI18n, setLocale, waitLocale } from '$lib/i18n';
	import { isLoading as i18nLoading, t } from 'svelte-i18n';
	import { initPlan, initPlanFromSSR, type PlanConfig } from '$lib/stores/plan';
	import { siteNavStore } from '$lib/stores/siteNav';
	interface Props {
		children?: import('svelte').Snippet;
		data: {
			faviconUrl: string | null;
			planConfig: PlanConfig | null;
			appUrl: string;
			siteNav: { enabled: boolean; mode?: string; position?: string; items: Array<{ viewId: string; slug: string; label: string; name: string }> };
			accentColor: string | null;
			customHexColor: string | null;
			fontPack: string | null;
			customCSS: string | null;
			defaultLocale: string | null;
			showAvatar: boolean;
			defaultThemeMode: string;
		};
	}

	let { children, data }: Props = $props();

	// Initialize plan config from SSR data immediately (before onMount)
	// This eliminates FOUC on Pro/Creator plans where badge should be hidden
	$effect(() => {
		if (data.planConfig) {
			initPlanFromSSR(data.planConfig);
		}
	});

	// Initialize site nav store from SSR data — updates on every navigation
	// so the store always reflects the latest server data.
	$effect(() => {
		if (data.siteNav) {
			siteNavStore.initFromSSR(data.siteNav);
		}
	});

	// Apply accent color from SSR data immediately via $effect (before onMount)
	// This runs during hydration, before paint, eliminating the color flash
	$effect(() => {
		if (!browser) return;
		if (data.customHexColor) {
			applyFlatAccent(data.customHexColor);
		} else if (data.accentColor) {
			applyAccentColor(data.accentColor as AccentColor);
		}
	});

	// Apply font pack from SSR data (eliminates font flash on first paint)
	$effect(() => {
		if (!browser) return;
		if (data.fontPack) {
			applyFontPack(data.fontPack);
		}
	});

	// Debug navigation (only log in dev mode)
	beforeNavigate((navigation) => {
		if (import.meta.env.DEV) {
			console.log('[NAVIGATION] Before navigate:', {
				from: navigation.from?.url?.pathname,
				to: navigation.to?.url?.pathname,
				type: navigation.type,
				willUnload: navigation.willUnload
			});
		}
	});

	afterNavigate((navigation) => {
		if (import.meta.env.DEV) {
			console.log('[NAVIGATION] After navigate:', {
				from: navigation.from?.url?.pathname,
				to: navigation.to?.url?.pathname,
				type: navigation.type
			});
		}
	});

	let themeColor = $state('#0ea5e9'); // Default sky-500
	// Seed from SSR data so the $effect at the bottom of <script> applies the
	// admin-configured CSS on mount. Without this, the effect would call
	// applyCustomCSS('') and wipe the <style id="custom-css"> element
	// immediately after onMount injected it (regression in v2.21.15).
	let customCSS = $state(data.customCSS ?? '');
	let lastCustomCSS = $state('');
	let mounted = $state(false);
let gaMeasurementId = $state('');
let gaInitialized = $state(false);
let accentStyleEl: HTMLStyleElement | null = $state(null);
let fontStyleEl: HTMLStyleElement | null = $state(null);
let fontLinkEl: HTMLLinkElement | null = $state(null);
let customPaletteLocked = false;
// Initialize from server-loaded data so SSR renders the correct favicon.
// A $state(null) + $effect pattern is broken here because $effect never
// runs during SSR, so the server-rendered HTML would always fall back to
// /favicon.png even when a custom favicon is configured. $state-with-initial
// is writable, so onMount/event-handlers below can still reassign it.
let faviconUrl = $state<string | null>(data.faviconUrl);

function applyPaletteFromCSS(css: string) {
	if (!browser || !css) return;

	// Collect explicit primary tokens from user CSS
	const matches = [...css.matchAll(/--color-primary-(50|100|200|300|400|500|600|700|800|900|950)\s*:\s*([^;]+);?/gi)];
	const palette: Record<string, string> = {};
	for (const [, token, value] of matches) {
		palette[token] = value.trim();
	}

	if (Object.keys(palette).length === 0) return;

	// If only 500 is provided, generate a full palette from it
	if (Object.keys(palette).length === 1 && palette['500']) {
		const scale = generatePaletteFromHex(palette['500']);
		for (const step of ['50', '100', '200', '300', '400', '500', '600', '700', '800', '900', '950'] as const) {
			palette[step] = scale[step];
		}
		customPaletteLocked = true;
		applyFlatAccent(palette['500']);
	} else {
		customPaletteLocked = true;
	}

	// Apply directly to :root so Tailwind classes pick up overrides immediately
	for (const [token, value] of Object.entries(palette)) {
		document.documentElement.style.setProperty(`--color-primary-${token}`, value);
		document.documentElement.style.setProperty(`--color-primary-${token}-rgb`, hexToRgbChannels(value));
	}
}

function applyFlatAccent(color: string) {
	if (!browser || !color) return;

	const scale = generatePaletteFromHex(color);

	if (!accentStyleEl) {
		accentStyleEl = document.createElement('style');
		accentStyleEl.id = 'accent-colors';
		document.head.appendChild(accentStyleEl);
	}

	accentStyleEl.textContent = buildPaletteRootBlock(scale);
	themeColor = color;
}

	function applyAccentColor(colorName: AccentColor) {
		if (!browser) return;
		if (customPaletteLocked) return;

		const color = ACCENT_COLORS[colorName];
		if (!color) return;

		if (!accentStyleEl) {
			accentStyleEl = document.createElement('style');
			accentStyleEl.id = 'accent-colors';
			document.head.appendChild(accentStyleEl);
		}

		accentStyleEl.textContent = buildPaletteRootBlock(color.scale);

		// Update theme-color meta tag for browser chrome
		themeColor = color.scale[500];
	}

	function applyFontPack(packName: string) {
		if (!browser) return;

		const pack = getFontPack(packName);

		if (!fontStyleEl) {
			fontStyleEl = document.createElement('style');
			fontStyleEl.id = 'font-pack';
			document.head.appendChild(fontStyleEl);
		}

		fontStyleEl.textContent = `:root {
  --font-heading: '${pack.heading}', ${pack.headingFallback};
  --font-body: '${pack.body}', ${pack.bodyFallback};
  --font-code: '${pack.code}', ${pack.codeFallback};
}`;

		// Load Google Fonts for non-default packs
		if (packName !== DEFAULT_FONT_PACK) {
			if (!fontLinkEl) {
				fontLinkEl = document.createElement('link');
				fontLinkEl.id = 'dynamic-google-fonts';
				fontLinkEl.rel = 'stylesheet';
				document.head.appendChild(fontLinkEl);
			}
			fontLinkEl.href = pack.googleFontsUrl;
		} else if (fontLinkEl) {
			fontLinkEl.remove();
			fontLinkEl = null;
		}
	}

	async function loadAccentColor() {
		try {
			// Fetch profile via public API endpoint
			const response = await fetch('/api/homepage');
			if (response.ok) {
				const data = await response.json();
				// Custom hex color takes precedence over named palette
				if (data.profile?.custom_hex_color) {
					applyFlatAccent(data.profile.custom_hex_color);
				} else if (data.profile?.accent_color) {
					applyAccentColor(data.profile.accent_color as AccentColor);
				}
			}
		} catch (err) {
			// Silently fail - use default colors
			console.debug('Using default accent color');
		}
	}

	function maybeDeriveAccentFromCustomCSS(css: string) {
		// If user only set --color-primary-500, mirror it across the palette for convenience
		const primary500 = css.match(/--color-primary-500\s*:\s*([^;]+);?/i);
		if (!primary500) return;

		// If they already set other palette tokens, don't override
		const hasOtherTokens = /--color-primary-(50|100|200|300|400|600|700|800|900|950)\s*:/i.test(css);
		if (hasOtherTokens) return;

		const color = primary500[1].trim();
		if (color) {
			customPaletteLocked = true;
			applyFlatAccent(color);
		}
	}

async function loadSiteSettings(): Promise<string | undefined> {
	try {
		// Add cache buster to prevent browser from caching stale settings
		const response = await fetch(`/api/site-settings?_=${Date.now()}`);
		if (response.ok) {
			const data = await response.json();
			customCSS = data.custom_css || '';
			gaMeasurementId = data.ga_measurement_id || '';
			// Add cache buster to favicon URL to prevent browser caching
			faviconUrl = data.favicon ? `${data.favicon}?v=${Date.now()}` : null;
			applyPaletteFromCSS(customCSS);
			applyCustomCSS(customCSS);
			return data.default_locale || undefined;
		}
	} catch (err) {
		console.debug('No custom CSS loaded');
	}
	return undefined;
}

function applyCustomCSS(css: string) {
	if (!browser) return;
	lastCustomCSS = css;
	applyPaletteFromCSS(css);
	const existing = document.getElementById('custom-css');
	if (existing) {
		existing.remove();
	}
	if (!css) return;
	const style = document.createElement('style');
	style.id = 'custom-css';
	style.textContent = css;
	document.head.appendChild(style);
}

onMount(() => {
	mounted = true;
	// Pass admin-configured default ('system' | 'light' | 'dark') so theme.initialize()
	// can fall back to it when the user has no localStorage preference yet.
	theme.initialize(data.defaultThemeMode);
	(async () => {
		// Load plan config only if SSR didn't provide it (fallback for client-side navigation)
		if (!data.planConfig) {
			try {
				await initPlan();
			} catch {
				// Continue - plan fetch failure shouldn't block rendering
			}
		}
		// SSR provides locale and custom CSS from +layout.server.ts.
		// Fall back to client-side loadSiteSettings() if SSR didn't provide them.
		// customCSS is already seeded from data.customCSS via the $state initializer;
		// the $effect at the bottom of <script> applies it on mount.
		if (data.defaultLocale || data.customCSS) {
			initI18n(data.defaultLocale || undefined);
		} else {
			const serverLocale = await loadSiteSettings();
			initI18n(serverLocale);
			if (lastCustomCSS) {
				applyCustomCSS(lastCustomCSS);
			}
		}
		await waitLocale();
		// Skip client-side accent fetch if SSR already provided it
		if (!data.accentColor && !data.customHexColor) {
			await loadAccentColor();
		}
	})();

	// Listen for accent color changes from settings page
	const handleColorChange = (event: CustomEvent<AccentColor>) => {
		customPaletteLocked = false;
		applyAccentColor(event.detail);
		if (lastCustomCSS) {
			applyCustomCSS(lastCustomCSS);
		}
	};
	window.addEventListener('accent-color-changed', handleColorChange as EventListener);

	// Listen for custom hex color changes from settings page
	const handleHexColorChange = (event: CustomEvent<string>) => {
		applyFlatAccent(event.detail);
		if (lastCustomCSS) {
			applyCustomCSS(lastCustomCSS);
		}
	};
	window.addEventListener('custom-hex-color-changed', handleHexColorChange as EventListener);

	// Listen for favicon changes from settings page
	// The {#key faviconUrl} block in <svelte:head> handles DOM updates via Svelte reactivity
	const handleFaviconChange = (event: CustomEvent<string | null>) => {
		faviconUrl = event.detail;
	};
	window.addEventListener('favicon-changed', handleFaviconChange as EventListener);

	// Listen for font pack changes from settings page
	const handleFontPackChange = (event: CustomEvent<string>) => {
		applyFontPack(event.detail);
	};
	window.addEventListener('font-pack-changed', handleFontPackChange as EventListener);

	return () => {
		window.removeEventListener('accent-color-changed', handleColorChange as EventListener);
		window.removeEventListener('custom-hex-color-changed', handleHexColorChange as EventListener);
		window.removeEventListener('favicon-changed', handleFaviconChange as EventListener);
		window.removeEventListener('font-pack-changed', handleFontPackChange as EventListener);
	};
});



	function injectGA(id: string) {
		if (!browser || !id) return;
		if (document.getElementById('ga-script')) return;

		// GA4 snippet (minimal)
		const script = document.createElement('script');
		script.id = 'ga-script';
		script.async = true;
		script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(id)}`;
		document.head.appendChild(script);

		const inline = document.createElement('script');
		inline.textContent = `
		  window.dataLayer = window.dataLayer || [];
		  function gtag(){dataLayer.push(arguments);}
		  gtag('js', new Date());
		  gtag('config', '${id.replace(/[^A-Za-z0-9-]/g, '')}');
		`;
		document.head.appendChild(inline);
	}
$effect(() => {
		if (mounted) {
		applyCustomCSS(customCSS);
		if (!gaInitialized && gaMeasurementId) {
			injectGA(gaMeasurementId);
			gaInitialized = true;
		}
	}
	});
// Ensure custom CSS stays last after accent updates
$effect(() => {
		if (mounted && lastCustomCSS && accentStyleEl) {
		// Re-append custom CSS to the end of head so it wins cascade against accent variables
		applyCustomCSS(lastCustomCSS);
	}
	});
</script>

<svelte:head>
	<meta name="theme-color" content={themeColor} />
	<link rel="alternate" type="application/rss+xml" title="RSS Feed" href="/rss.xml" />
	<!-- Use {#key} to force browser to fetch new favicon when URL changes -->
	{#key faviconUrl}
		{#if faviconUrl}
			<link rel="icon" type="image/x-icon" href={faviconUrl} />
		{:else}
			<link rel="icon" type="image/png" href="/favicon.png" />
		{/if}
	{/key}
</svelte:head>

<!-- Skip link for keyboard navigation -->
<a href="#main-content" class="skip-link">
	{$t('shared.aria.skip_to_content')}
</a>

{@render children?.()}

<!-- Toast notifications - live region for screen readers -->
<div
	class="fixed bottom-4 right-4 left-4 sm:left-auto z-50 flex flex-col gap-2"
	role="region"
	aria-label={$t('shared.aria.notifications')}
>
	{#each $toasts as toast (toast.id)}
		<Toast {toast} on:dismiss={() => toasts.remove(toast.id)} />
	{/each}
</div>

<!-- Confirm dialog -->
<ConfirmDialog />

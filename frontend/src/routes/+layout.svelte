<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { beforeNavigate, afterNavigate } from '$app/navigation';
	import { theme, toasts } from '$lib/stores';
	import Toast from '$components/shared/Toast.svelte';
	import ConfirmDialog from '$components/shared/ConfirmDialog.svelte';
	import { ACCENT_COLORS, DEFAULT_ACCENT_COLOR, type AccentColor } from '$lib/colors';

	// Debug navigation in development
	beforeNavigate((navigation) => {
		console.log('[NAVIGATION] Before navigate:', {
			from: navigation.from?.url?.pathname,
			to: navigation.to?.url?.pathname,
			type: navigation.type,
			willUnload: navigation.willUnload
		});
	});

	afterNavigate((navigation) => {
		console.log('[NAVIGATION] After navigate:', {
			from: navigation.from?.url?.pathname,
			to: navigation.to?.url?.pathname,
			type: navigation.type
		});
	});

	let themeColor = '#0ea5e9'; // Default sky-500
	let customCSS = '';
	let lastCustomCSS = '';
	let mounted = false;
let gaMeasurementId = '';
let gaInitialized = false;
let accentStyleEl: HTMLStyleElement | null = null;
let customPaletteLocked = false;

function applyPaletteFromCSS(css: string) {
	if (!browser || !css) return;

	// Collect explicit primary tokens from user CSS
	const matches = [...css.matchAll(/--color-primary-(50|100|200|300|400|500|600|700|800|900|950)\s*:\s*([^;]+);?/gi)];
	const palette: Record<string, string> = {};
	for (const [, token, value] of matches) {
		palette[token] = value.trim();
	}

	if (Object.keys(palette).length === 0) return;

	// If only 500 is provided, fan it out to all tokens
	if (Object.keys(palette).length === 1 && palette['500']) {
		const color = palette['500'];
		for (const step of ['50', '100', '200', '300', '400', '500', '600', '700', '800', '900', '950']) {
			palette[step] = color;
		}
		customPaletteLocked = true;
		applyFlatAccent(color);
	} else {
		customPaletteLocked = true;
	}

	// Apply directly to :root so Tailwind classes pick up overrides immediately
	for (const [token, value] of Object.entries(palette)) {
		document.documentElement.style.setProperty(`--color-primary-${token}`, value);
	}
}

function applyFlatAccent(color: string) {
	if (!browser || !color) return;

	if (!accentStyleEl) {
		accentStyleEl = document.createElement('style');
			accentStyleEl.id = 'accent-colors';
			document.head.appendChild(accentStyleEl);
		}

		accentStyleEl.textContent = `
:root {
  --color-primary-50: ${color};
  --color-primary-100: ${color};
  --color-primary-200: ${color};
  --color-primary-300: ${color};
  --color-primary-400: ${color};
  --color-primary-500: ${color};
  --color-primary-600: ${color};
  --color-primary-700: ${color};
  --color-primary-800: ${color};
  --color-primary-900: ${color};
  --color-primary-950: ${color};
}
		`.trim();
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

		accentStyleEl.textContent = `
:root {
  --color-primary-50: ${color.scale[50]};
  --color-primary-100: ${color.scale[100]};
  --color-primary-200: ${color.scale[200]};
  --color-primary-300: ${color.scale[300]};
  --color-primary-400: ${color.scale[400]};
  --color-primary-500: ${color.scale[500]};
  --color-primary-600: ${color.scale[600]};
  --color-primary-700: ${color.scale[700]};
  --color-primary-800: ${color.scale[800]};
  --color-primary-900: ${color.scale[900]};
  --color-primary-950: ${color.scale[950]};
}
		`.trim();

		// Update theme-color meta tag for browser chrome
		themeColor = color.scale[500];
	}

	async function loadAccentColor() {
		try {
			// Fetch profile via public API endpoint
			const response = await fetch('/api/homepage');
			if (response.ok) {
				const data = await response.json();
				if (data.profile?.accent_color) {
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

async function loadSiteSettings() {
	try {
		const response = await fetch('/api/site-settings');
		if (response.ok) {
			const data = await response.json();
			customCSS = data.custom_css || '';
			gaMeasurementId = data.ga_measurement_id || '';
			applyPaletteFromCSS(customCSS);
			applyCustomCSS(customCSS);
		}
	} catch (err) {
		console.debug('No custom CSS loaded');
	}
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
	theme.initialize();
	(async () => {
		await loadSiteSettings();
		await loadAccentColor();
		if (lastCustomCSS) {
			applyCustomCSS(lastCustomCSS);
		}
	})();

	// Listen for accent color changes from settings page
	const handleColorChange = (event: CustomEvent<AccentColor>) => {
		applyAccentColor(event.detail);
		if (lastCustomCSS) {
			applyCustomCSS(lastCustomCSS);
		}
	};
	window.addEventListener('accent-color-changed', handleColorChange as EventListener);

	return () => {
		window.removeEventListener('accent-color-changed', handleColorChange as EventListener);
	};
});

$: if (mounted) {
	applyCustomCSS(customCSS);
	if (!gaInitialized && gaMeasurementId) {
		injectGA(gaMeasurementId);
		gaInitialized = true;
	}
}

// Ensure custom CSS stays last after accent updates
$: if (mounted && lastCustomCSS && accentStyleEl) {
	// Re-append custom CSS to the end of head so it wins cascade against accent variables
	applyCustomCSS(lastCustomCSS);
}

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
		  gtag('config', '${id}');
		`;
		document.head.appendChild(inline);
	}
</script>

<svelte:head>
	<meta name="theme-color" content={themeColor} />
</svelte:head>

<!-- Skip link for keyboard navigation -->
<a href="#main-content" class="skip-link">
	Skip to main content
</a>

<slot />

<!-- Toast notifications - live region for screen readers -->
<div
	class="fixed bottom-4 right-4 z-50 flex flex-col gap-2"
	role="region"
	aria-label="Notifications"
	aria-live="polite"
>
	{#each $toasts as toast (toast.id)}
		<Toast {toast} on:dismiss={() => toasts.remove(toast.id)} />
	{/each}
</div>

<!-- Confirm dialog -->
<ConfirmDialog />

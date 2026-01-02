<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { beforeNavigate, afterNavigate } from '$app/navigation';
	import { theme, toasts } from '$lib/stores';
	import Toast from '$components/shared/Toast.svelte';
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
	let mounted = false;
	let gaMeasurementId = '';
	let gaInitialized = false;

	function applyAccentColor(colorName: AccentColor) {
		if (!browser) return;

		const color = ACCENT_COLORS[colorName];
		if (!color) return;

		const root = document.documentElement;
		root.style.setProperty('--color-primary-50', color.scale[50]);
		root.style.setProperty('--color-primary-100', color.scale[100]);
		root.style.setProperty('--color-primary-200', color.scale[200]);
		root.style.setProperty('--color-primary-300', color.scale[300]);
		root.style.setProperty('--color-primary-400', color.scale[400]);
		root.style.setProperty('--color-primary-500', color.scale[500]);
		root.style.setProperty('--color-primary-600', color.scale[600]);
		root.style.setProperty('--color-primary-700', color.scale[700]);
		root.style.setProperty('--color-primary-800', color.scale[800]);
		root.style.setProperty('--color-primary-900', color.scale[900]);
		root.style.setProperty('--color-primary-950', color.scale[950]);

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

	async function loadSiteSettings() {
		try {
			const response = await fetch('/api/site-settings');
			if (response.ok) {
				const data = await response.json();
				customCSS = data.custom_css || '';
				gaMeasurementId = data.ga_measurement_id || '';
			}
		} catch (err) {
			console.debug('No custom CSS loaded');
		}
	}

	function applyCustomCSS(css: string) {
		if (!browser) return;
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
		loadAccentColor();
		loadSiteSettings();

		// Listen for accent color changes from settings page
		const handleColorChange = (event: CustomEvent<AccentColor>) => {
			applyAccentColor(event.detail);
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

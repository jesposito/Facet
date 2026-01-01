<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { theme, toasts } from '$lib/stores';
	import Toast from '$components/shared/Toast.svelte';
	import { ACCENT_COLORS, DEFAULT_ACCENT_COLOR, type AccentColor } from '$lib/colors';

	let themeColor = '#0ea5e9'; // Default sky-500

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

	onMount(() => {
		theme.initialize();
		loadAccentColor();

		// Listen for accent color changes from settings page
		const handleColorChange = (event: CustomEvent<AccentColor>) => {
			applyAccentColor(event.detail);
		};
		window.addEventListener('accent-color-changed', handleColorChange as EventListener);

		return () => {
			window.removeEventListener('accent-color-changed', handleColorChange as EventListener);
		};
	});
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

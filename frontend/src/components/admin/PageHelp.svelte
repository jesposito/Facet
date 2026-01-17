<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import type { Snippet } from 'svelte';

	interface Props {
		/** Unique key for localStorage persistence */
		pageKey: string;
		/** Whether help should be expanded by default for first-time users */
		defaultExpanded?: boolean;
		/** Content to render inside the help section */
		children: Snippet;
	}

	let { pageKey, defaultExpanded = true, children }: Props = $props();

	const STORAGE_KEY = 'facet_page_help_collapsed';

	// Use a function to compute initial value to avoid the warning
	function getInitialExpanded(): boolean {
		return defaultExpanded;
	}

	let isExpanded = $state(getInitialExpanded());
	let initialized = $state(false);

	onMount(() => {
		if (browser) {
			try {
				const stored = localStorage.getItem(STORAGE_KEY);
				if (stored) {
					const collapsed: Record<string, boolean> = JSON.parse(stored);
					// If user has explicitly collapsed this page's help, respect that
					if (collapsed[pageKey] === true) {
						isExpanded = false;
					}
				}
			} catch {
				// Ignore localStorage errors
			}
			initialized = true;
		}
	});

	function toggle() {
		isExpanded = !isExpanded;
		if (browser) {
			try {
				const stored = localStorage.getItem(STORAGE_KEY);
				const collapsed: Record<string, boolean> = stored ? JSON.parse(stored) : {};
				collapsed[pageKey] = !isExpanded;
				localStorage.setItem(STORAGE_KEY, JSON.stringify(collapsed));
			} catch {
				// Ignore localStorage errors
			}
		}
	}
</script>

{#if initialized}
	<div class="mb-6 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg overflow-hidden">
		<button
			type="button"
			class="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-blue-100/50 dark:hover:bg-blue-900/30 transition-colors"
			onclick={toggle}
			aria-expanded={isExpanded}
		>
			<div class="flex items-center gap-2 text-blue-700 dark:text-blue-300">
				<svg class="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				<span class="font-medium text-sm">Page Help</span>
			</div>
			<svg
				class="w-5 h-5 text-blue-500 dark:text-blue-400 transition-transform duration-200 {isExpanded ? 'rotate-180' : ''}"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</button>
		
		{#if isExpanded}
			<div class="px-4 pb-4 pt-1 text-sm text-blue-800 dark:text-blue-200 space-y-2">
				{@render children()}
			</div>
		{/if}
	</div>
{/if}

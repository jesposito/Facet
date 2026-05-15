<script lang="ts">
	/**
	 * AccentPicker — accessible color swatch selector with keyboard navigation.
	 *
	 * WAI-ARIA APG radio pattern:
	 * - role="radiogroup" with roving tabindex
	 * - Arrow keys move focus + selection, wrapping at boundaries
	 * - Home → first, End → last
	 * - Tab escapes the group naturally
	 *
	 * Backported from Facet Cloud (v3.24.0) with adaptation for self-hosted's
	 * 6-curated-color system and existing color utilities.
	 */
	import { tick } from 'svelte';
	import { t } from 'svelte-i18n';
	import {
		ACCENT_COLORS,
		ACCENT_COLOR_LIST,
		DEFAULT_ACCENT_COLOR,
		type AccentColor
	} from '$lib/colors';

	let {
		value = $bindable(DEFAULT_ACCENT_COLOR),
		onchange
	}: {
		value?: AccentColor;
		onchange?: (color: AccentColor) => void;
	} = $props();

	// Refs for roving-tabindex focus management
	let swatchRefs: HTMLButtonElement[] = $state([]);

	const colorInfo = $derived(ACCENT_COLORS[value] ?? ACCENT_COLORS[DEFAULT_ACCENT_COLOR]);

	// Index of the swatch that owns the radiogroup's tabstop.
	const focusedIndex = $derived(ACCENT_COLOR_LIST.indexOf(value));

	function pick(color: AccentColor) {
		value = color;
		onchange?.(color);
	}

	// WAI-ARIA APG radio pattern keyboard handler
	async function handleSwatchKeyDown(e: KeyboardEvent) {
		const last = ACCENT_COLOR_LIST.length - 1;
		let next = focusedIndex;
		switch (e.key) {
			case 'ArrowRight':
			case 'ArrowDown':
				next = focusedIndex >= last ? 0 : focusedIndex + 1;
				break;
			case 'ArrowLeft':
			case 'ArrowUp':
				next = focusedIndex <= 0 ? last : focusedIndex - 1;
				break;
			case 'Home':
				next = 0;
				break;
			case 'End':
				next = last;
				break;
			default:
				return;
		}
		e.preventDefault();
		pick(ACCENT_COLOR_LIST[next]);
		await tick();
		swatchRefs[next]?.focus();
	}
</script>

<fieldset class="space-y-4">
	<legend class="text-sm font-semibold text-gray-900 dark:text-white mb-2">
		{$t('admin.settings_page.appearance.accent_color_label')}
	</legend>

	<!-- Curated swatches — WAI-ARIA APG radio pattern with roving tabindex -->
	<div
		role="radiogroup"
		aria-label={$t('admin.accent_picker.curated_colors')}
		tabindex={-1}
		class="flex flex-wrap gap-3"
		onkeydown={handleSwatchKeyDown}
	>
		{#each ACCENT_COLOR_LIST as color, i (color)}
			{@const info = ACCENT_COLORS[color]}
			<button
				type="button"
				role="radio"
				aria-checked={value === color}
				aria-label={info.label}
				tabindex={i === focusedIndex ? 0 : -1}
				bind:this={swatchRefs[i]}
				onclick={() => pick(color)}
				class="relative group aspect-square w-12 h-12 rounded-xl border-2 transition-all duration-200 ring-offset-2 ring-offset-white dark:ring-offset-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900 dark:focus-visible:ring-white {value ===
				color
					? 'border-gray-900 dark:border-white scale-110'
					: 'border-transparent hover:scale-105'}"
				style="background-color: {info.scale[500]}"
			>
				{#if value === color}
					<div class="absolute inset-0 flex items-center justify-center">
						<svg class="w-5 h-5 text-white drop-shadow" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
							<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
						</svg>
					</div>
				{/if}
			</button>
		{/each}
	</div>

	<!-- Color name labels -->
	<div class="flex flex-wrap gap-3">
		{#each ACCENT_COLOR_LIST as color}
			<span
				class="w-12 text-center text-xs text-gray-600 dark:text-gray-400 {value === color ? 'font-semibold' : ''}"
				aria-hidden="true"
			>
				{ACCENT_COLORS[color].label}
			</span>
		{/each}
	</div>

	<!-- Live preview -->
	<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mt-4">
		<span class="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide font-medium mb-3 block">
			{$t('admin.settings_page.appearance.preview_label')}
		</span>
		<div class="flex flex-wrap items-center gap-4">
			<button
				type="button"
				class="px-4 py-2 rounded-lg font-medium text-white transition-colors"
				style="background-color: {colorInfo.scale[600]}"
			>
				{$t('admin.settings_page.appearance.primary_button')}
			</button>
			<button
				type="button"
				class="px-4 py-2 rounded-lg font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200"
			>
				{$t('admin.settings_page.appearance.secondary_button')}
			</button>
			<a
				href="#appearance"
				class="font-medium underline underline-offset-2"
				style="color: {colorInfo.scale[600]}"
			>
				{$t('admin.settings_page.appearance.link_example')}
			</a>
			<span
				class="px-2 py-1 rounded text-sm font-medium"
				style="background-color: {colorInfo.scale[100]}; color: {colorInfo.scale[700]}"
			>
				{$t('admin.settings_page.appearance.badge')}
			</span>
		</div>
		<p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
			{colorInfo.description}
		</p>
	</div>
</fieldset>

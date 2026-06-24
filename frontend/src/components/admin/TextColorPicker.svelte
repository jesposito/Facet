<script lang="ts">
	/**
	 * TextColorPicker — operator-settable body/heading text color for the public
	 * profile, AAA-readable BY CONSTRUCTION.
	 *
	 * The operator picks a hue; deriveTextInk() clamps it per-mode (darken for
	 * light, lighten for dark) so it always clears 7:1 against the lightest
	 * surface text sits on. The picker shows the CLAMPED result live — what you
	 * pick is what renders, and it is always readable.
	 *
	 * Accessibility:
	 * - Curated suggestions follow the WAI-ARIA APG radio pattern (roving
	 *   tabindex, arrow keys move focus + selection, Home/End jump).
	 * - Hex input validates on blur (less noisy for screen readers); a persistent
	 *   aria-live region announces commit + error.
	 * - The live preview swatches carry text alternatives (not color-only).
	 */
	import { tick } from 'svelte';
	import { t } from 'svelte-i18n';
	import { isValidHexColor, deriveTextInk } from '$lib/colors';

	let {
		value = $bindable(''),
		onchange
	}: {
		value?: string;
		onchange?: (hex: string) => void;
	} = $props();

	// Curated, harmonious ink hues. Operators can still type any hex — these are
	// just sensible starting points (warm + cool, all distinct from the accent).
	const SUGGESTIONS: Array<{ hex: string; labelKey: string }> = [
		{ hex: '#1f2937', labelKey: 'admin.text_color_picker.suggestion_slate' },
		{ hex: '#3f3f46', labelKey: 'admin.text_color_picker.suggestion_graphite' },
		{ hex: '#7c2d12', labelKey: 'admin.text_color_picker.suggestion_umber' },
		{ hex: '#1e3a8a', labelKey: 'admin.text_color_picker.suggestion_navy' },
		{ hex: '#14532d', labelKey: 'admin.text_color_picker.suggestion_forest' },
		{ hex: '#581c87', labelKey: 'admin.text_color_picker.suggestion_plum' }
	];

	let swatchRefs: HTMLButtonElement[] = $state([]);
	let hexInput = $state(value || '');
	let hexError = $state('');
	let announce = $state('');

	const normalized = $derived(value && isValidHexColor(value) ? value : '');
	// Live clamped inks — exactly what hooks.server.ts injects and the page renders.
	const ink = $derived(normalized ? deriveTextInk(normalized) : null);

	const selectedIndex = $derived(SUGGESTIONS.findIndex((s) => s.hex.toLowerCase() === normalized.toLowerCase()));
	// Roving tabstop owner: the selected suggestion, else the first.
	const focusedIndex = $derived(selectedIndex >= 0 ? selectedIndex : 0);

	function normalizeHex(raw: string): string {
		const trimmed = raw.trim();
		if (!trimmed) return '';
		return trimmed.startsWith('#') ? trimmed : `#${trimmed}`;
	}

	function pick(hex: string) {
		const norm = normalizeHex(hex);
		value = norm;
		hexInput = norm;
		hexError = '';
		announce = $t('admin.text_color_picker.announced', { values: { hex: norm } });
		onchange?.(norm);
	}

	function commitHex() {
		const norm = normalizeHex(hexInput);
		if (!norm) {
			value = '';
			hexError = '';
			announce = $t('admin.text_color_picker.announced_reset');
			onchange?.('');
			return;
		}
		if (!isValidHexColor(norm)) {
			hexError = $t('admin.text_color_picker.invalid');
			return;
		}
		hexError = '';
		value = norm;
		hexInput = norm;
		announce = $t('admin.text_color_picker.announced', { values: { hex: norm } });
		onchange?.(norm);
	}

	function handleHexInput(e: Event) {
		hexInput = (e.currentTarget as HTMLInputElement).value;
		if (hexError) hexError = '';
	}

	function reset() {
		value = '';
		hexInput = '';
		hexError = '';
		announce = $t('admin.text_color_picker.announced_reset');
		onchange?.('');
	}

	async function handleSwatchKeyDown(e: KeyboardEvent) {
		const last = SUGGESTIONS.length - 1;
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
		pick(SUGGESTIONS[next].hex);
		await tick();
		swatchRefs[next]?.focus();
	}
</script>

<fieldset class="space-y-4">
	<legend class="text-sm font-semibold text-gray-900 dark:text-white mb-1">
		{$t('admin.text_color_picker.label')}
	</legend>
	<p class="text-sm text-gray-600 dark:text-gray-400 mb-2">
		{$t('admin.text_color_picker.description')}
	</p>

	<!-- Curated suggestions — WAI-ARIA APG radio pattern with roving tabindex -->
	<div
		role="radiogroup"
		aria-label={$t('admin.text_color_picker.suggestions_label')}
		tabindex={-1}
		class="flex flex-wrap gap-3"
		onkeydown={handleSwatchKeyDown}
	>
		{#each SUGGESTIONS as s, i (s.hex)}
			{@const isChecked = normalized.toLowerCase() === s.hex.toLowerCase()}
			<button
				type="button"
				role="radio"
				aria-checked={isChecked}
				aria-label={$t(s.labelKey)}
				tabindex={i === focusedIndex ? 0 : -1}
				bind:this={swatchRefs[i]}
				onclick={() => pick(s.hex)}
				class="relative aspect-square w-12 h-12 rounded-xl border-2 transition-all duration-200 ring-offset-2 ring-offset-white dark:ring-offset-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900 dark:focus-visible:ring-white {isChecked
					? 'border-gray-900 dark:border-white scale-110'
					: 'border-transparent hover:scale-105'}"
				style="background-color: {s.hex}"
			>
				{#if isChecked}
					<div class="absolute inset-0 flex items-center justify-center">
						<svg class="w-5 h-5 text-white drop-shadow" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
							<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
						</svg>
					</div>
				{/if}
			</button>
		{/each}
	</div>

	<!-- Custom hex input -->
	<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
		<label for="text-color-hex-input" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
			{$t('admin.text_color_picker.custom_label')}
		</label>
		<div class="flex items-center gap-2">
			<span
				class="inline-block w-11 h-11 rounded-lg border border-gray-300 dark:border-gray-600 shrink-0"
				style="background-color: {normalized || '#ffffff'}"
				aria-hidden="true"
			></span>
			<input
				id="text-color-hex-input"
				type="text"
				bind:value={hexInput}
				oninput={handleHexInput}
				onblur={commitHex}
				onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); commitHex(); } }}
				placeholder={$t('admin.text_color_picker.custom_placeholder')}
				maxlength="7"
				spellcheck="false"
				autocomplete="off"
				autocapitalize="off"
				autocorrect="off"
				aria-invalid={hexError !== ''}
				aria-describedby="text-color-hex-help text-color-hex-error"
				class="input flex-1 font-mono"
			/>
			{#if normalized}
				<button
					type="button"
					onclick={reset}
					class="px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors min-h-11"
				>
					{$t('admin.text_color_picker.reset')}
				</button>
			{/if}
		</div>
		<p id="text-color-hex-help" class="text-xs text-gray-500 dark:text-gray-400 mt-1">
			{$t('admin.text_color_picker.custom_help')}
		</p>
		<p
			id="text-color-hex-error"
			role="status"
			aria-live="polite"
			aria-atomic="true"
			class="text-sm text-red-600 dark:text-red-400 mt-1 min-h-[1.25rem]"
		>
			{hexError}
		</p>
	</div>

	<!-- Live region for confirmation announcements (visually hidden) -->
	<span class="sr-only" aria-live="polite" aria-atomic="true">{announce}</span>

	<!-- Live CLAMPED preview — shows the actual rendered ink for both modes, so
	     "what you pick is what you get, and it is always readable". -->
	{#if ink}
		<div class="rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700 mt-4">
			<span class="block px-4 pt-3 text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide font-medium">
				{$t('admin.text_color_picker.preview_label')}
			</span>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-px bg-gray-200 dark:bg-gray-700 mt-3">
				<!-- Light mode preview (clamped vs white surface) -->
				<div class="p-4" style="background-color: #ffffff;">
					<span class="block text-[10px] uppercase tracking-wide font-semibold mb-2" style="color: #6b7280;">
						{$t('admin.text_color_picker.preview_light')} · {ink.light}
					</span>
					<span class="block text-lg font-bold" style="color: {ink.light};">
						{$t('admin.text_color_picker.preview_heading')}
					</span>
					<span class="block text-sm" style="color: {ink.light};">
						{$t('admin.text_color_picker.preview_body')}
					</span>
				</div>
				<!-- Dark mode preview (clamped vs dark surface) -->
				<div class="p-4" style="background-color: #221e1a;">
					<span class="block text-[10px] uppercase tracking-wide font-semibold mb-2" style="color: #ada199;">
						{$t('admin.text_color_picker.preview_dark')} · {ink.dark}
					</span>
					<span class="block text-lg font-bold" style="color: {ink.dark};">
						{$t('admin.text_color_picker.preview_heading')}
					</span>
					<span class="block text-sm" style="color: {ink.dark};">
						{$t('admin.text_color_picker.preview_body')}
					</span>
				</div>
			</div>
			<p class="px-4 py-3 text-xs text-gray-500 dark:text-gray-400">
				{$t('admin.text_color_picker.aaa_note')}
			</p>
		</div>
	{/if}
</fieldset>

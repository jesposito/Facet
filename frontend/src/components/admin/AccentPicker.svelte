<script lang="ts">
	/**
	 * AccentPicker — accessible color swatch selector with keyboard navigation
	 * AND a custom hex input.
	 *
	 * WAI-ARIA APG radio pattern:
	 * - role="radiogroup" with roving tabindex
	 * - Arrow keys move focus + selection, wrapping at boundaries
	 * - Home → first, End → last
	 * - Tab escapes the group naturally
	 *
	 * Custom hex input:
	 * - Below the swatches
	 * - Validates on blur (not on every keystroke — less noisy for SR)
	 * - When valid: deselects all radio swatches (aria-checked=false on all)
	 * - Persistent live region announces commit + error state (aria-live="polite")
	 * - Reset button returns to curated default
	 */
	import { tick } from 'svelte';
	import { t } from 'svelte-i18n';
	import {
		ACCENT_COLORS,
		ACCENT_COLOR_LIST,
		DEFAULT_ACCENT_COLOR,
		isValidHexColor,
		type AccentColor
	} from '$lib/colors';

	let {
		value = $bindable(DEFAULT_ACCENT_COLOR),
		customHex = $bindable(''),
		onchange,
		onhexchange
	}: {
		value?: AccentColor;
		customHex?: string;
		onchange?: (color: AccentColor) => void;
		onhexchange?: (hex: string) => void;
	} = $props();

	// Refs for roving-tabindex focus management
	let swatchRefs: HTMLButtonElement[] = $state([]);

	// Hex input internal state
	let hexInput = $state(customHex || '');
	let hexError = $state('');
	let hexCommitted = $state(customHex !== '');

	// Live announce text — toggled when user commits a color, read by aria-live region
	let announce = $state('');

	const colorInfo = $derived(ACCENT_COLORS[value] ?? ACCENT_COLORS[DEFAULT_ACCENT_COLOR]);

	// The currently active hex (either the committed custom hex, or the named color's 500 step)
	const activeHex = $derived(hexCommitted && customHex ? customHex : colorInfo.scale[500]);

	// Index of the swatch that owns the radiogroup's tabstop.
	// When a custom hex is active, no swatch is checked but roving tabindex still
	// needs exactly one tabstop — fall back to the position of the named value or 0.
	const focusedIndex = $derived(ACCENT_COLOR_LIST.indexOf(value) >= 0 ? ACCENT_COLOR_LIST.indexOf(value) : 0);

	function normalizeHex(raw: string): string {
		const trimmed = raw.trim();
		if (!trimmed) return '';
		return trimmed.startsWith('#') ? trimmed : `#${trimmed}`;
	}

	function pickColor(color: AccentColor) {
		value = color;
		// Clear custom hex when picking a curated swatch
		if (hexCommitted) {
			customHex = '';
			hexInput = '';
			hexCommitted = false;
			hexError = '';
			onhexchange?.('');
		}
		announce = $t('admin.accent_picker.announced_curated', { values: { name: ACCENT_COLORS[color].label } });
		onchange?.(color);
	}

	function commitHex() {
		const normalized = normalizeHex(hexInput);
		if (!normalized) {
			// Empty — treat as reset to curated default
			customHex = '';
			hexCommitted = false;
			hexError = '';
			onhexchange?.('');
			return;
		}
		if (!isValidHexColor(normalized)) {
			hexError = $t('admin.accent_picker.custom_invalid');
			return;
		}
		hexError = '';
		hexCommitted = true;
		customHex = normalized;
		hexInput = normalized;
		announce = $t('admin.accent_picker.announced_custom', { values: { hex: normalized } });
		onhexchange?.(normalized);
	}

	function handleHexInput(e: Event) {
		const target = e.currentTarget as HTMLInputElement;
		hexInput = target.value;
		// Clear error as soon as the user resumes typing — gentler UX than per-keystroke validation
		if (hexError) hexError = '';
	}

	function handleHexBlur() {
		commitHex();
	}

	function handleHexKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			commitHex();
		}
	}

	function resetToDefault() {
		hexInput = '';
		customHex = '';
		hexCommitted = false;
		hexError = '';
		value = DEFAULT_ACCENT_COLOR;
		announce = $t('admin.accent_picker.announced_curated', { values: { name: ACCENT_COLORS[DEFAULT_ACCENT_COLOR].label } });
		onhexchange?.('');
		onchange?.(DEFAULT_ACCENT_COLOR);
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
		pickColor(ACCENT_COLOR_LIST[next]);
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
			{@const isCheckedSwatch = !hexCommitted && value === color}
			<button
				type="button"
				role="radio"
				aria-checked={isCheckedSwatch}
				aria-label={info.label}
				tabindex={i === focusedIndex ? 0 : -1}
				bind:this={swatchRefs[i]}
				onclick={() => pickColor(color)}
				class="relative group aspect-square w-12 h-12 rounded-xl border-2 transition-all duration-200 ring-offset-2 ring-offset-white dark:ring-offset-gray-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900 dark:focus-visible:ring-white {isCheckedSwatch
					? 'border-gray-900 dark:border-white scale-110'
					: 'border-transparent hover:scale-105'}"
				style="background-color: {info.scale[500]}"
			>
				{#if isCheckedSwatch}
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
				class="w-12 text-center text-xs text-gray-600 dark:text-gray-400 {!hexCommitted && value === color ? 'font-semibold' : ''}"
				aria-hidden="true"
			>
				{ACCENT_COLORS[color].label}
			</span>
		{/each}
	</div>

	<!-- Custom hex input -->
	<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
		<label for="accent-hex-input" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
			{$t('admin.accent_picker.custom_label')}
		</label>
		<div class="flex items-center gap-2">
			<!-- Preview swatch (decorative — value is announced via the input itself) -->
			<span
				class="inline-block w-11 h-11 rounded-lg border border-gray-300 dark:border-gray-600 shrink-0"
				style="background-color: {activeHex}"
				aria-hidden="true"
			></span>
			<input
				id="accent-hex-input"
				type="text"
				bind:value={hexInput}
				oninput={handleHexInput}
				onblur={handleHexBlur}
				onkeydown={handleHexKeyDown}
				placeholder={$t('admin.accent_picker.custom_placeholder')}
				maxlength="7"
				spellcheck="false"
				autocomplete="off"
				autocapitalize="off"
				autocorrect="off"
				aria-invalid={hexError !== ''}
				aria-describedby="accent-hex-help accent-hex-error"
				class="input flex-1 font-mono"
			/>
			<button
				type="button"
				onclick={resetToDefault}
				class="px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors min-h-11"
			>
				{$t('admin.accent_picker.reset')}
			</button>
		</div>
		<p id="accent-hex-help" class="text-xs text-gray-500 dark:text-gray-400 mt-1">
			{$t('admin.accent_picker.custom_help')}
		</p>
		<!-- Persistent error region — toggle text, not the element, so screen readers consistently announce -->
		<p
			id="accent-hex-error"
			role="status"
			aria-live="polite"
			aria-atomic="true"
			class="text-sm text-red-600 dark:text-red-400 mt-1 min-h-[1.25rem]"
		>
			{hexError}
		</p>
		<!-- Custom-active confirmation (non-color text signal — WCAG 1.4.1) -->
		{#if hexCommitted && customHex}
			<p class="text-xs text-gray-700 dark:text-gray-200 font-medium mt-1">
				{$t('admin.accent_picker.custom_active', { values: { hex: customHex } })}
			</p>
		{/if}
	</div>

	<!-- Live region for confirmation announcements (visually hidden) -->
	<span class="sr-only" aria-live="polite" aria-atomic="true">{announce}</span>

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

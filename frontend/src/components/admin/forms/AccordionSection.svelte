<script lang="ts">
	/**
	 * AccordionSection — collapsible card with an APG-compliant disclosure pattern.
	 *
	 * - Button triggers `aria-expanded` + `aria-controls`
	 * - Panel has `role="region"` + `aria-labelledby`
	 * - Body only rendered when open (collapses tab order naturally)
	 * - Optional leading icon, description, trailing badge
	 *
	 * Backported from Facet Cloud, adapted to self-hosted (no lucide-svelte
	 * dependency — inline chevron SVG).
	 */
	import type { Snippet } from 'svelte';

	interface Props {
		/** Section heading shown in the trigger button */
		title: string;
		/** Optional short description shown under the title */
		description?: string;
		/** Optional leading icon SVG path (single `d` attribute) */
		iconPath?: string;
		/** Whether the section is open initially. Can be two-way bound. */
		open?: boolean;
		/** Unique id used for aria-controls / aria-labelledby. Generated if omitted. */
		id?: string;
		/** Hide the chevron (for decorative-only use) */
		hideChevron?: boolean;
		/** Optional trailing badge content (e.g. item count or status pill) */
		badge?: Snippet;
		/** Body content rendered inside the collapsible panel */
		children: Snippet;
	}

	let {
		title,
		description,
		iconPath,
		open = $bindable(false),
		id = `accordion-${crypto.randomUUID().slice(0, 8)}`,
		hideChevron = false,
		badge,
		children
	}: Props = $props();

	// svelte-ignore state_referenced_locally
	const buttonId = `${id}-trigger`;
	// svelte-ignore state_referenced_locally
	const panelId = `${id}-panel`;

	function toggle() {
		open = !open;
	}
</script>

<div class="card overflow-hidden">
	<h3 class="m-0">
		<button
			type="button"
			id={buttonId}
			aria-expanded={open}
			aria-controls={panelId}
			onclick={toggle}
			class="w-full flex items-center gap-3 px-5 py-4 text-start hover:bg-stone-50 dark:hover:bg-stone-700/30 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-inset"
		>
			{#if iconPath}
				<span class="w-9 h-9 rounded-xl bg-primary-50 dark:bg-primary-900/20 flex items-center justify-center flex-shrink-0">
					<svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" d={iconPath} />
					</svg>
				</span>
			{/if}
			<span class="flex-1 min-w-0">
				<span class="block text-base font-semibold text-stone-900 dark:text-white">{title}</span>
				{#if description}
					<span class="block text-sm text-stone-500 dark:text-stone-400 mt-0.5">{description}</span>
				{/if}
			</span>
			{#if badge}
				<span class="flex-shrink-0">{@render badge()}</span>
			{/if}
			{#if !hideChevron}
				<svg
					class="w-[18px] h-[18px] text-stone-400 flex-shrink-0 transition-transform duration-200 {open ? 'rotate-180' : ''}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
				</svg>
			{/if}
		</button>
	</h3>
	{#if open}
		<div
			id={panelId}
			role="region"
			aria-labelledby={buttonId}
			class="border-t border-stone-100 dark:border-stone-700/60 px-5 py-5"
		>
			{@render children()}
		</div>
	{/if}
</div>

<script lang="ts" generics="T extends { id: string }">
	import { tick } from 'svelte';
	import { t } from 'svelte-i18n';

	interface Props {
		/** Currently selected record ids */
		selectedIds: string[];
		/** Records available for selection */
		options: T[];
		/** Primary display label for a record */
		getLabel: (record: T) => string;
		/** Optional secondary label (e.g. dates) shown beneath the primary label */
		getSubLabel?: (record: T) => string;
		/** Whether option records are still loading */
		loading?: boolean;
		/** Text for the open/edit trigger button */
		addLabel: string;
		editLabel: string;
		/** Accessible label for the listbox of options */
		listLabel: string;
		/** Message shown when there are no options to choose from */
		emptyLabel: string;
		/** ID of the label element for a11y association */
		labelledBy?: string;
		onchange?: (ids: string[]) => void;
	}

	let {
		selectedIds = $bindable([]),
		options,
		getLabel,
		getSubLabel,
		loading = false,
		addLabel,
		editLabel,
		listLabel,
		emptyLabel,
		labelledBy,
		onchange
	}: Props = $props();

	let showDropdown = $state(false);
	let triggerEl = $state<HTMLButtonElement>();
	let dropdownEl = $state<HTMLElement>();

	async function openDropdown() {
		showDropdown = true;
		await tick();
		dropdownEl?.querySelector<HTMLButtonElement>('button')?.focus();
	}

	function closeDropdown() {
		showDropdown = false;
		triggerEl?.focus();
	}

	function toggle(id: string) {
		const newIds = selectedIds.includes(id)
			? selectedIds.filter((existing) => existing !== id)
			: [...selectedIds, id];
		selectedIds = newIds;
		onchange?.(newIds);
	}

	function remove(id: string) {
		const newIds = selectedIds.filter((existing) => existing !== id);
		selectedIds = newIds;
		onchange?.(newIds);
	}

	let selectedRecords = $derived(options.filter((record) => selectedIds.includes(record.id)));
</script>

<svelte:window onkeydown={(e) => { if (showDropdown && e.key === 'Escape') closeDropdown(); }} />

<div class="relative">
	{#if selectedRecords.length > 0}
		<ul class="flex flex-wrap gap-1.5 mb-2 list-none p-0 m-0">
			{#each selectedRecords as record (record.id)}
				<li
					class="inline-flex items-center gap-1.5 pl-2.5 pr-1 py-1 text-sm rounded-full bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-200 border border-primary-200 dark:border-primary-800"
				>
					<span>{getLabel(record)}</span>
					<button
						type="button"
						class="inline-flex items-center justify-center w-5 h-5 rounded-full text-primary-500 hover:text-primary-700 hover:bg-primary-100 dark:hover:bg-primary-800 dark:hover:text-primary-100 transition-colors"
						onclick={() => remove(record.id)}
						aria-label={$t('admin.related.remove_item', { values: { name: getLabel(record) } })}
					>
						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</li>
			{/each}
		</ul>
	{/if}

	<button
		bind:this={triggerEl}
		type="button"
		class="btn btn-secondary btn-sm"
		onclick={() => (showDropdown ? closeDropdown() : openDropdown())}
		aria-labelledby={labelledBy}
		aria-expanded={showDropdown}
		aria-haspopup="true"
	>
		{selectedRecords.length > 0 ? editLabel : addLabel}
	</button>

	{#if showDropdown}
		<div
			bind:this={dropdownEl}
			class="absolute z-50 mt-1 p-2 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 min-w-[240px]"
		>
			{#if loading}
				<p class="text-sm text-gray-500 dark:text-gray-400 p-2">{$t('admin.related.loading')}</p>
			{:else if options.length === 0}
				<p class="text-sm text-gray-500 dark:text-gray-400 p-2 text-center">{emptyLabel}</p>
			{:else}
				<ul class="space-y-1 max-h-56 overflow-y-auto list-none p-0 m-0" role="group" aria-label={listLabel}>
					{#each options as record (record.id)}
						{@const isSelected = selectedIds.includes(record.id)}
						<li>
							<button
								type="button"
								aria-pressed={isSelected}
								class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-left text-sm transition-colors
									{isSelected ? 'bg-gray-100 dark:bg-gray-700' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}"
								onclick={() => toggle(record.id)}
							>
								<span class="flex-1 min-w-0">
									<span class="block truncate text-gray-700 dark:text-gray-200">{getLabel(record)}</span>
									{#if getSubLabel}
										{@const sub = getSubLabel(record)}
										{#if sub}
											<span class="block truncate text-xs text-gray-500 dark:text-gray-400">{sub}</span>
										{/if}
									{/if}
								</span>
								{#if isSelected}
									<svg class="w-4 h-4 text-primary-600 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			{/if}
			<button
				type="button"
				class="absolute top-1 right-1 p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
				onclick={closeDropdown}
				aria-label={$t('shared.aria.close')}
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}
</div>

{#if showDropdown}
	<button
		type="button"
		class="fixed inset-0 z-40"
		onclick={() => (showDropdown = false)}
		aria-label={$t('shared.aria.close_menu')}
	></button>
{/if}

<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { t } from 'svelte-i18n';

	interface Props {
		selectedCount: number;
		totalCount: number;
		showVisibilityActions?: boolean;
		showDeleteAction?: boolean;
	}

	let {
		selectedCount,
		totalCount,
		showVisibilityActions = true,
		showDeleteAction = true
	}: Props = $props();

	const dispatch = createEventDispatcher<{
		selectAll: void;
		clearSelection: void;
		setVisibility: 'public' | 'unlisted' | 'private';
		delete: void;
		cancel: void;
	}>();

	let showVisibilityMenu = $state(false);
	let visibilityButton: HTMLButtonElement | undefined = $state();

	function handleVisibility(visibility: 'public' | 'unlisted' | 'private') {
		dispatch('setVisibility', visibility);
		showVisibilityMenu = false;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key !== 'Escape' || !showVisibilityMenu) return;
		event.preventDefault();
		event.stopPropagation();
		showVisibilityMenu = false;
		visibilityButton?.focus();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div
	role="region"
	aria-label={$t('admin.bulk_actions.region_label')}
	class="sticky top-16 z-10 w-full min-w-0 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 px-3 sm:px-4 py-3 flex flex-wrap items-center justify-between gap-x-4 gap-y-2 shadow-sm"
>
	<div class="flex min-w-0 max-w-full flex-wrap items-center gap-x-4 gap-y-2">
		<span
			role="status"
			aria-atomic="true"
			class="text-sm font-medium text-gray-700 dark:text-gray-300"
		>
			{$t('admin.bulk_actions.selected_count', {
				values: { selected: selectedCount, total: totalCount }
			})}
		</span>
		<button
			type="button"
			class="min-w-0 max-w-full whitespace-normal break-words text-left text-sm text-primary-700 hover:text-primary-800 dark:text-primary-400 hover:underline"
			onclick={() => dispatch('selectAll')}
		>
			{$t('admin.bulk_actions.select_all')}
		</button>
		<button
			type="button"
			class="min-w-0 max-w-full whitespace-normal break-words text-left text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 hover:underline"
			onclick={() => dispatch('clearSelection')}
		>
			{$t('admin.bulk_actions.clear')}
		</button>
	</div>

	<div class="flex min-w-0 max-w-full flex-wrap items-center gap-2">
		{#if showVisibilityActions}
			<div class="relative min-w-0 max-w-full">
				<button
					bind:this={visibilityButton}
					type="button"
					disabled={selectedCount === 0}
					class="btn btn-secondary min-w-0 max-w-full whitespace-normal break-words text-left text-sm flex items-center gap-1 disabled:cursor-not-allowed disabled:opacity-50"
					onclick={() => (showVisibilityMenu = !showVisibilityMenu)}
					aria-expanded={showVisibilityMenu}
				>
					{$t('admin.bulk_actions.change_visibility')}
					<svg
						class="w-4 h-4 shrink-0 transition-transform duration-200"
						class:rotate-180={showVisibilityMenu}
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M19 9l-7 7-7-7"
						/>
					</svg>
				</button>

				{#if showVisibilityMenu}
					<button
						type="button"
						tabindex="-1"
						class="fixed inset-0 z-20"
						onclick={() => (showVisibilityMenu = false)}
						aria-label={$t('shared.aria.close_menu')}
					></button>
					<div
						class="absolute right-0 mt-1 w-40 max-w-full bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-30 animate-slide-in-down"
					>
						<button
							type="button"
							class="w-full min-w-0 whitespace-normal break-words px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
							onclick={() => handleVisibility('public')}
						>
							<span class="w-2 h-2 shrink-0 rounded-full bg-green-500"></span>
							{$t('admin.content.common.visibility_public')}
						</button>
						<button
							type="button"
							class="w-full min-w-0 whitespace-normal break-words px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
							onclick={() => handleVisibility('unlisted')}
						>
							<span class="w-2 h-2 shrink-0 rounded-full bg-yellow-500"></span>
							{$t('admin.content.common.visibility_unlisted')}
						</button>
						<button
							type="button"
							class="w-full min-w-0 whitespace-normal break-words px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
							onclick={() => handleVisibility('private')}
						>
							<span class="w-2 h-2 shrink-0 rounded-full bg-red-500"></span>
							{$t('admin.content.common.visibility_private')}
						</button>
					</div>
				{/if}
			</div>
		{/if}

		{#if showDeleteAction}
			<button
				type="button"
				disabled={selectedCount === 0}
				class="btn min-w-0 max-w-full whitespace-normal break-words text-left text-sm bg-red-600 hover:bg-red-700 text-white disabled:cursor-not-allowed disabled:opacity-50"
				onclick={() => dispatch('delete')}
			>
				{$t('shared.actions.delete')}
			</button>
		{/if}

		<button
			type="button"
			class="btn btn-ghost min-w-0 max-w-full whitespace-normal break-words text-left text-sm"
			onclick={() => dispatch('cancel')}
		>
			{$t('shared.actions.cancel')}
		</button>
	</div>
</div>

<style>
	button {
		overflow-wrap: anywhere;
	}

	button:hover,
	button:active {
		transform: none;
	}
</style>

<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { getTagColor, TAG_COLOR_LIST, type TagColor } from '$lib/colors';
	import AdminTagBadge from './AdminTagBadge.svelte';
	import { t } from 'svelte-i18n';

	interface AdminTag {
		id: string;
		name: string;
		color: string;
		sort_order: number;
	}

	interface Props {
		selectedIds: string[];
		onchange?: (ids: string[]) => void;
		compact?: boolean;
		/** ID of the label element for a11y association */
		labelledBy?: string;
	}

	let { selectedIds = $bindable([]), onchange, compact = false, labelledBy }: Props = $props();

	let availableTags: AdminTag[] = $state([]);
	let loading = $state(true);
	let showDropdown = $state(false);

	onMount(async () => {
		await loadTags();
	});

	async function loadTags() {
		loading = true;
		try {
			const result = await pb.collection('admin_tags').getList<AdminTag>(1, 100, {
				sort: 'sort_order,name'
			});
			availableTags = result.items;
		} catch (err) {
			console.error('Failed to load admin tags:', err);
			availableTags = [];
		} finally {
			loading = false;
		}
	}

	function toggleTag(tagId: string) {
		const newIds = selectedIds.includes(tagId)
			? selectedIds.filter(id => id !== tagId)
			: [...selectedIds, tagId];
		selectedIds = newIds;
		onchange?.(newIds);
	}

	function removeTag(tagId: string) {
		const newIds = selectedIds.filter(id => id !== tagId);
		selectedIds = newIds;
		onchange?.(newIds);
	}

	let selectedTags = $derived(
		availableTags.filter(tag => selectedIds.includes(tag.id))
	);

	let unselectedTags = $derived(
		availableTags.filter(tag => !selectedIds.includes(tag.id))
	);
</script>

<div class="relative">
	{#if compact}
		<div class="flex flex-wrap items-center gap-1">
			{#each selectedTags as tag (tag.id)}
				<AdminTagBadge
					name={tag.name}
					color={tag.color}
					size="sm"
					removable
					onRemove={() => removeTag(tag.id)}
				/>
			{/each}
			<button
				type="button"
				class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs rounded border border-dashed border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:border-gray-400 dark:hover:border-gray-500 transition-colors"
				onclick={() => showDropdown = !showDropdown}
				aria-label={labelledBy ? undefined : $t('admin.tags.add_tag')}
				aria-labelledby={labelledBy || undefined}
				aria-expanded={showDropdown}
			>
				<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				Tag
			</button>
		</div>
	{:else}
		<div class="flex flex-wrap gap-1 mb-2">
			{#each selectedTags as tag (tag.id)}
				<AdminTagBadge
					name={tag.name}
					color={tag.color}
					size="md"
					removable
					onRemove={() => removeTag(tag.id)}
				/>
			{/each}
		</div>
		<button
			type="button"
			class="btn btn-secondary btn-sm"
			onclick={() => showDropdown = !showDropdown}
			aria-labelledby={labelledBy}
			aria-expanded={showDropdown}
		>
			{selectedTags.length > 0 ? 'Edit Tags' : 'Add Tags'}
		</button>
	{/if}

	{#if showDropdown}
		<div
			class="absolute z-50 mt-1 p-2 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 min-w-[200px]"
		>
			{#if loading}
				<p class="text-sm text-gray-500 dark:text-gray-400 p-2">Loading tags...</p>
			{:else if availableTags.length === 0}
				<div class="p-2 text-center">
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-2">No tags yet</p>
					<a href="/admin/settings/tags" class="text-xs text-primary-600 hover:underline">
						Create tags in Settings
					</a>
				</div>
			{:else}
				<div class="space-y-1 max-h-48 overflow-y-auto">
					{#each availableTags as tag (tag.id)}
						{@const isSelected = selectedIds.includes(tag.id)}
						{@const colorInfo = getTagColor(tag.color)}
						<button
							type="button"
							class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-left text-sm transition-colors
								{isSelected ? 'bg-gray-100 dark:bg-gray-700' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'}"
							onclick={() => toggleTag(tag.id)}
						>
							<span
								class="w-3 h-3 rounded-full {colorInfo.bg} {colorInfo.border} border"
							></span>
							<span class="flex-1 text-gray-700 dark:text-gray-200">{tag.name}</span>
							{#if isSelected}
								<svg class="w-4 h-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							{/if}
						</button>
					{/each}
				</div>
				<div class="border-t border-gray-200 dark:border-gray-700 mt-2 pt-2">
					<a
						href="/admin/settings/tags"
						class="block text-xs text-center text-gray-500 dark:text-gray-400 hover:text-primary-600"
					>
						Manage tags
					</a>
				</div>
			{/if}
			<button
				type="button"
				class="absolute top-1 right-1 p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
				onclick={() => showDropdown = false}
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
		onclick={() => showDropdown = false}
		aria-label={$t('shared.aria.close_menu')}
	></button>
{/if}

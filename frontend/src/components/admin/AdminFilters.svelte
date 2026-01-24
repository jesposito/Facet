<script lang="ts">
	import { icon } from '$lib/icons';
	import AdminTagBadge from './AdminTagBadge.svelte';
	import type { FilterStateStore } from '$lib/admin/filterState.svelte.ts';

	interface AdminTag {
		id: string;
		name: string;
		color: string;
	}

	let {
		filterStore,
		availableTags = [],
		showAdvanced = $bindable(false)
	}: {
		filterStore: FilterStateStore;
		availableTags?: AdminTag[];
		showAdvanced?: boolean;
	} = $props();

	let showTagSelector = $state(false);

	function handleQuickFilter(type: 'drafts' | 'published' | 'private' | 'public') {
		switch (type) {
			case 'drafts':
				filterStore.setDraftFilter('drafts');
				filterStore.setVisibilityFilter('all');
				break;
			case 'published':
				filterStore.setDraftFilter('published');
				filterStore.setVisibilityFilter('all');
				break;
			case 'private':
				filterStore.setVisibilityFilter('private');
				filterStore.setDraftFilter('all');
				break;
			case 'public':
				filterStore.setVisibilityFilter('public');
				filterStore.setDraftFilter('all');
				break;
		}
	}

	$effect(() => {
		if (filterStore.selectedTagIds.length === 0) {
			showTagSelector = false;
		}
	});
</script>

<div class="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
	<div class="p-4 space-y-3">

		{#if filterStore.config.enableSearch}
			<div class="relative">
				<div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
					<svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
				</div>
				<input
					type="text"
					placeholder={filterStore.config.searchPlaceholder}
					value={filterStore.searchQuery}
					oninput={(e) => filterStore.setSearchQuery(e.currentTarget.value)}
					class="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
					       bg-white dark:bg-gray-800 text-gray-900 dark:text-white 
					       placeholder-gray-500 dark:placeholder-gray-400
					       focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-colors"
				/>
				{#if filterStore.searchQuery}
					<button
						type="button"
						onclick={() => filterStore.setSearchQuery('')}
						class="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
						title="Clear search"
					>
						{@html icon('x')}
					</button>
				{/if}
			</div>
		{/if}


		<div class="flex flex-wrap items-center gap-2">
			{#if filterStore.config.enableDraftFilter}
				<button
					type="button"
					onclick={() => handleQuickFilter('drafts')}
					class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full transition-colors
					       {filterStore.draftFilter === 'drafts' 
					         ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' 
					         : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'}"
				>
					<div class="w-2 h-2 rounded-full bg-current opacity-60"></div>
					Drafts
				</button>
				
				<button
					type="button"
					onclick={() => handleQuickFilter('published')}
					class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full transition-colors
					       {filterStore.draftFilter === 'published' 
					         ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' 
					         : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'}"
				>
					<div class="w-2 h-2 rounded-full bg-current opacity-60"></div>
					Published
				</button>
			{/if}

			{#if filterStore.config.enableVisibilityFilter}
				<button
					type="button"
					onclick={() => handleQuickFilter('private')}
					class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full transition-colors
					       {filterStore.visibilityFilter === 'private' 
					         ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' 
					         : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'}"
				>
					<div class="w-2 h-2 rounded-full bg-current opacity-60"></div>
					Private
				</button>
			{/if}

			{#if filterStore.config.enableTagFilter && filterStore.selectedTagIds.length > 0}
				{#each filterStore.selectedTagIds as tagId (tagId)}
					{@const tag = availableTags.find(t => t.id === tagId)}
					{#if tag}
						<button
							type="button"
							onclick={() => filterStore.toggleTag(tagId)}
							class="inline-flex items-center gap-1 pl-2 pr-1 py-1 rounded-full bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-200 hover:bg-primary-200 dark:hover:bg-primary-800 transition-colors group"
							title="Remove {tag.name} filter"
						>
							<AdminTagBadge name={tag.name} color={tag.color} />
							<span class="w-4 h-4 rounded-full flex items-center justify-center group-hover:bg-primary-300 dark:group-hover:bg-primary-700">
								<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</span>
						</button>
					{/if}
				{/each}
			{/if}


			<button
				type="button"
				onclick={() => showAdvanced = !showAdvanced}
				class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4" />
				</svg>
				More filters
			</button>

			{#if filterStore.hasActiveFilters()}
				<button
					type="button"
					onclick={() => filterStore.clearAllFilters()}
					class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 transition-colors"
				>
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
					Clear all
				</button>
			{/if}
		</div>

		{#if showAdvanced}
			<div class="pt-3 border-t border-gray-200 dark:border-gray-700 space-y-3">
				<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
					{#if filterStore.config.enableVisibilityFilter}
						<div>
							<label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
								Visibility
							</label>
							<select
								value={filterStore.visibilityFilter}
								onchange={(e) => filterStore.setVisibilityFilter(e.currentTarget.value as any)}
								class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
								       bg-white dark:bg-gray-800 text-gray-900 dark:text-white 
								       focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
							>
								<option value="all">All visibility levels</option>
								<option value="public">Public</option>
								<option value="unlisted">Unlisted</option>
								<option value="private">Private</option>
							</select>
						</div>
					{/if}

					{#if filterStore.config.enableDraftFilter}
						<div>
							<label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
								Status
							</label>
							<select
								value={filterStore.draftFilter}
								onchange={(e) => filterStore.setDraftFilter(e.currentTarget.value as any)}
								class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
								       bg-white dark:bg-gray-800 text-gray-900 dark:text-white 
								       focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
							>
								<option value="all">All statuses</option>
								<option value="published">Published</option>
								<option value="drafts">Drafts</option>
							</select>
						</div>
					{/if}

					{#if filterStore.config.enableTagFilter && availableTags.length > 0}
						<div>
							<label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
								Tags
							</label>
							<div class="relative">
								<button
									type="button"
									onclick={() => showTagSelector = !showTagSelector}
									class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg 
									       bg-white dark:bg-gray-800 text-gray-900 dark:text-white 
									       focus:ring-2 focus:ring-primary-500 focus:border-primary-500
									       text-left flex items-center justify-between"
								>
									<span class="text-sm">
										{filterStore.selectedTagIds.length === 0 
											? 'Select tags...' 
											: `${filterStore.selectedTagIds.length} tag${filterStore.selectedTagIds.length > 1 ? 's' : ''} selected`}
									</span>
									<svg class="w-4 h-4 transition-transform {showTagSelector ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
									</svg>
								</button>

								{#if showTagSelector}
									<div class="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg shadow-lg z-10 max-h-48 overflow-y-auto">
										{#each availableTags as tag (tag.id)}
											<button
												type="button"
												onclick={() => filterStore.toggleTag(tag.id)}
												class="w-full px-3 py-2 text-left hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2 transition-colors"
											>
												<input
													type="checkbox"
													checked={filterStore.selectedTagIds.includes(tag.id)}
													readonly
													class="w-4 h-4 text-primary-600 rounded"
												/>
												<AdminTagBadge name={tag.name} color={tag.color} />
											</button>
										{/each}
									</div>
								{/if}
							</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>

	{#if filterStore.hasActiveFilters()}
		<div class="px-4 py-2 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700">
			<div class="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
				<span>
					{#if filterStore.searchQuery}Search: "{filterStore.searchQuery}"{/if}
					{#if filterStore.visibilityFilter !== 'all'} · Visibility: {filterStore.visibilityFilter}{/if}
					{#if filterStore.draftFilter !== 'all'} · Status: {filterStore.draftFilter}{/if}
					{#if filterStore.selectedTagIds.length > 0} · {filterStore.selectedTagIds.length} tag{filterStore.selectedTagIds.length > 1 ? 's' : ''}{/if}
				</span>
			</div>
		</div>
	{/if}
</div>
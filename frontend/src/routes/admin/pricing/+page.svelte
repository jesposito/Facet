<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { toasts } from '$lib/stores';
	import { brandName } from '$lib/stores/plan';

	interface ContentItem {
		id: string;
		collection: string;
		title: string;
		access_tier: string;
		price: number;
		visibility: string;
		is_draft: boolean;
	}

	const CONTENT_TYPES = [
		{ collection: 'posts', labelKey: 'admin.pricing.type_posts', titleField: 'title' },
		{ collection: 'talks', labelKey: 'admin.pricing.type_talks', titleField: 'title' },
		{ collection: 'courses', labelKey: 'admin.pricing.type_courses', titleField: 'title' },
		{ collection: 'projects', labelKey: 'admin.pricing.type_projects', titleField: 'title' },
		{ collection: 'custom_content', labelKey: 'admin.pricing.type_custom', titleField: 'title' },
	];

	let items: ContentItem[] = $state([]);
	let loading = $state(true);
	let saving = $state(false);
	let selectedIds: Set<string> = $state(new Set());
	let bulkTier = $state('paid');
	let bulkPrice = $state('');
	let filterType = $state('all');
	let filterTier = $state('all');

	let filteredItems = $derived(
		items.filter(item => {
			if (filterType !== 'all' && item.collection !== filterType) return false;
			if (filterTier === 'free' && item.access_tier !== 'free') return false;
			if (filterTier === 'paid' && item.access_tier !== 'paid') return false;
			return true;
		})
	);

	let allSelected = $derived(
		filteredItems.length > 0 && filteredItems.every(i => selectedIds.has(i.id))
	);

	onMount(loadAll);

	async function loadAll() {
		loading = true;
		try {
			const allItems: ContentItem[] = [];
			for (const ct of CONTENT_TYPES) {
				try {
					const records = await pb.collection(ct.collection).getList(1, 200, {
						sort: '-created',
						fields: `id,${ct.titleField},access_tier,price,visibility,is_draft`,
					});
					for (const r of records.items) {
						allItems.push({
							id: r.id,
							collection: ct.collection,
							title: r[ct.titleField] || $t('admin.pricing.untitled'),
							access_tier: r.access_tier || 'free',
							price: r.price || 0,
							visibility: r.visibility || 'public',
							is_draft: r.is_draft || false,
						});
					}
				} catch {
					// Collection might not have access_tier field - skip silently
				}
			}
			items = allItems;
		} catch (err) {
			console.error('Failed to load content:', err);
			toasts.add('error', $t('admin.pricing.error_load'));
		} finally {
			loading = false;
		}
	}

	function toggleSelect(id: string) {
		const next = new Set(selectedIds);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selectedIds = next;
	}

	function toggleAll() {
		if (allSelected) {
			selectedIds = new Set();
		} else {
			selectedIds = new Set(filteredItems.map(i => i.id));
		}
	}

	function collectionLabel(collection: string): string {
		const ct = CONTENT_TYPES.find(c => c.collection === collection);
		return ct ? $t(ct.labelKey) : collection;
	}

	function tierBadgeClass(tier: string): string {
		return tier === 'paid'
			? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
			: 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400';
	}

	async function applyBulkPricing() {
		const selected = items.filter(i => selectedIds.has(i.id));
		if (selected.length === 0) {
			toasts.add('error', $t('admin.pricing.error_no_selection'));
			return;
		}

		const priceInCents = bulkTier === 'paid' && bulkPrice
			? Math.round((parseFloat(bulkPrice) || 0) * 100)
			: 0;

		if (bulkTier === 'paid' && priceInCents < 50) {
			toasts.add('error', $t('admin.pricing.error_min_price'));
			return;
		}

		saving = true;
		let successCount = 0;
		let failCount = 0;

		for (const item of selected) {
			try {
				await pb.collection(item.collection).update(item.id, {
					access_tier: bulkTier,
					price: bulkTier === 'paid' ? priceInCents : 0,
				});
				// Update local state
				item.access_tier = bulkTier;
				item.price = bulkTier === 'paid' ? priceInCents : 0;
				successCount++;
			} catch (err) {
				console.error(`Failed to update ${item.collection}/${item.id}:`, err);
				failCount++;
				toasts.add('error', $t('admin.pricing.error_update_item', { values: { title: item.title } }));
			}
		}

		if (successCount > 0) {
			toasts.add('success', $t('admin.pricing.updated_count', { values: { count: successCount } }));
		}
		if (failCount > 0) {
			toasts.add('error', $t('admin.pricing.failed_count', { values: { count: failCount } }));
		}

		selectedIds = new Set();
		saving = false;
	}
</script>

<svelte:head>
	<title>{$t('admin.pricing.page_title')} | {$brandName}</title>
</svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.pricing.page_title')}</h1>
		<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
			{$t('admin.pricing.description')}
		</p>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
		</div>
	{:else}
		<!-- Filters -->
		<div class="flex flex-wrap items-center gap-3">
			<select bind:value={filterType} class="input w-auto text-sm">
				<option value="all">{$t('admin.pricing.filter_all_types')}</option>
				{#each CONTENT_TYPES as ct}
					<option value={ct.collection}>{$t(ct.labelKey)}</option>
				{/each}
			</select>
			<select bind:value={filterTier} class="input w-auto text-sm">
				<option value="all">{$t('admin.pricing.filter_all_tiers')}</option>
				<option value="free">{$t('admin.pricing.filter_free')}</option>
				<option value="paid">{$t('admin.pricing.filter_paid')}</option>
			</select>
			<span class="text-sm text-gray-500">{$t('admin.pricing.item_count', { values: { count: filteredItems.length } })}</span>
		</div>

		<!-- Bulk action bar -->
		{#if selectedIds.size > 0}
			<div class="card p-4 flex flex-wrap items-center gap-3 border-primary-500/50 bg-primary-50/10">
				<span class="text-sm font-medium text-gray-900 dark:text-white">
					{$t('admin.pricing.selected_count', { values: { count: selectedIds.size } })}
				</span>
				<select bind:value={bulkTier} class="input w-auto text-sm">
					<option value="paid">{$t('admin.pricing.set_paid')}</option>
					<option value="free">{$t('admin.pricing.set_free')}</option>
				</select>
				{#if bulkTier === 'paid'}
					<div class="relative">
						<span class="absolute left-2 top-1/2 -translate-y-1/2 text-gray-500 text-sm">$</span>
						<input
							type="number"
							bind:value={bulkPrice}
							class="input w-24 pl-6 text-sm"
							placeholder="9.99"
							min="0.50"
							step="0.01"
							aria-label={$t('admin.pricing.price_label')}
						/>
					</div>
				{/if}
				<button
					class="btn btn-primary btn-sm"
					onclick={applyBulkPricing}
					disabled={saving}
				>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-1 h-3 w-3" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{$t('admin.pricing.apply')}
				</button>
				<button class="btn btn-secondary btn-sm" onclick={() => selectedIds = new Set()}>
					{$t('admin.pricing.clear')}
				</button>
			</div>
		{/if}

		<!-- Content table -->
		<div class="card overflow-hidden">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
						<th class="w-10 px-4 py-3">
							<input
								type="checkbox"
								checked={allSelected}
								onchange={toggleAll}
								class="rounded border-gray-300 dark:border-gray-600"
								aria-label={$t('admin.pricing.select_all')}
							/>
						</th>
						<th scope="col" class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">{$t('admin.pricing.col_title')}</th>
						<th scope="col" class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400 hidden sm:table-cell">{$t('admin.pricing.col_type')}</th>
						<th scope="col" class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">{$t('admin.pricing.col_tier')}</th>
						<th scope="col" class="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-400">{$t('admin.pricing.col_price')}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
					{#each filteredItems as item (item.id)}
						<tr
							class="hover:bg-gray-50 dark:hover:bg-gray-800/30 transition-colors {selectedIds.has(item.id) ? 'bg-primary-50 dark:bg-primary-900/10' : ''}"
						>
							<td class="w-10 px-4 py-3">
								<input
									type="checkbox"
									checked={selectedIds.has(item.id)}
									onchange={() => toggleSelect(item.id)}
									class="rounded border-gray-300 dark:border-gray-600"
									aria-label={$t('admin.pricing.select_item', { values: { title: item.title } })}
								/>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<span class="font-medium text-gray-900 dark:text-white truncate max-w-xs">{item.title}</span>
									{#if item.is_draft}
										<span class="px-1.5 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">{$t('admin.pricing.draft')}</span>
									{/if}
								</div>
							</td>
							<td class="px-4 py-3 hidden sm:table-cell">
								<span class="text-gray-500 dark:text-gray-400">{collectionLabel(item.collection)}</span>
							</td>
							<td class="px-4 py-3">
								<span class="px-2 py-0.5 text-xs rounded {tierBadgeClass(item.access_tier)}">
									{item.access_tier === 'paid' ? $t('admin.pricing.tier_paid') : $t('admin.pricing.tier_free')}
								</span>
							</td>
							<td class="px-4 py-3 text-right text-gray-900 dark:text-white">
								{#if item.access_tier === 'paid' && item.price > 0}
									${(item.price / 100).toFixed(2)}
								{:else}
									<span class="text-gray-400">-</span>
								{/if}
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="5" class="px-4 py-8 text-center text-gray-500">
								{$t('admin.pricing.no_items')}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if items.length === 0}
			<div class="text-center py-12">
				<p class="text-gray-500 dark:text-gray-400">{$t('admin.pricing.empty_title')}</p>
				<p class="text-sm text-gray-400 mt-1">{$t('admin.pricing.empty_description')}</p>
			</div>
		{/if}
	{/if}
</div>

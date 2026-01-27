<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { pb, type Award } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import { formatDate, toDateInputValue, truncate } from '$lib/utils';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';

	let awards: Award[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingAward: Award | null = $state(null);
	let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});

	// Form fields
	let title = $state('');
	let issuer = $state('');
	let awardedAt = $state('');
	let description = $state('');
	let url = $state('');
	let visibility = $state('public');
	let isDraft = $state(false);
	let sortOrder = $state(0);
	let saving = $state(false);

let selectMode = $state(false);
let selectedIds: Set<string> = $state(new Set());

	const autosave = createAutosave('admin-awards', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: true,
		enableDraftFilter: true,
		enableTagFilter: false,
		searchPlaceholder: 'Search awards...'
	});
	let showAdvancedFilters = $state(false);

	let filteredAwards = $derived(
		filterStore.filterItems(
			awards,
			(award) => `${award.title} ${award.issuer || ''} ${award.description || ''}`,
			(award) => award.visibility,
			(award) => award.is_draft
		)
	);

	function getFormData() {
	return { title, issuer, awardedAt, description, url, visibility, isDraft, sortOrder };
}

function restoreFromDraft(data: Record<string, any>) {
	title = data.title || '';
	issuer = data.issuer || '';
	awardedAt = data.awardedAt || '';
	description = data.description || '';
	url = data.url || '';
	visibility = data.visibility || 'public';
	isDraft = data.isDraft || false;
	sortOrder = data.sortOrder || 0;
}

function handleFormChange() {
	if (showForm) {
		autosave.save(getFormData(), !!editingAward, editingAward?.id);
	}
}

function handleRestoreDraft() {
	const draft = autosave.loadDraft();
	if (draft?.data) {
		restoreFromDraft(draft.data);
		if (draft.isEditing && draft.editingId) {
			const award = awards.find(a => a.id === draft.editingId);
			if (award) editingAward = award;
		}
		showForm = true;
	}
	showRecoveryBanner = false;
}

function handleDismissDraft() {
	autosave.clearDraft();
	showRecoveryBanner = false;
	recoveryData = null;
}

afterNavigate(() => {
	showForm = false;
	editingAward = null;
	selectMode = false;
	selectedIds = new Set();
	
	const draft = autosave.loadDraft();
	if (draft?.data && Object.values(draft.data).some(v => v !== '' && v !== false && v !== 0)) {
		recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
		showRecoveryBanner = true;
	} else {
		showRecoveryBanner = false;
		recoveryData = null;
	}
});

onMount(loadAwards);

	async function loadAwards() {
		loading = true;
		try {
			const [records, membershipResp] = await Promise.all([
				await collection('awards').getList(1, 100, {
					sort: '-sort_order,-awarded_at'
				}),
				fetch('/api/admin/view-memberships?collection=awards', {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				}).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed memberships'))))
			]);
			awards = records.items as unknown as Award[];
			memberships = (membershipResp.memberships as typeof memberships) || {};
		} catch (err) {
			console.error('Failed to load awards:', err);
			toasts.add('error', 'Failed to load awards');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		title = '';
		issuer = '';
		awardedAt = '';
		description = '';
		url = '';
		visibility = 'public';
		isDraft = false;
		sortOrder = 0;
		editingAward = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function openEditForm(award: Award) {
		editingAward = award;
		title = award.title;
		issuer = award.issuer || '';
		awardedAt = toDateInputValue(award.awarded_at);
		description = award.description || '';
		url = award.url || '';
		visibility = award.visibility;
		isDraft = award.is_draft;
		sortOrder = award.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
	}

	async function handleSubmit() {
		if (!title.trim()) {
			toasts.add('error', 'Title is required');
			return;
		}

		const parsedSort = Number(sortOrder);
		const finalSort = Number.isFinite(parsedSort) ? parsedSort : 0;

		saving = true;
		try {
			const data = {
				title: title.trim(),
				issuer: issuer.trim(),
				awarded_at: awardedAt ? new Date(awardedAt).toISOString() : null,
				description: description.trim(),
				url: url.trim(),
				visibility,
				is_draft: isDraft,
				sort_order: finalSort
			};

			if (editingAward) {
				await await collection('awards').update(editingAward.id, data);
				toasts.add('success', 'Award updated successfully');
			} else {
				await await collection('awards').create(data);
				toasts.add('success', 'Award created successfully');
			}

			closeForm();
			await loadAwards();
		} catch (err) {
			console.error('Failed to save award:', err);
			const message =
				(err as any)?.data?.data &&
				Object.entries((err as any).data.data)
					.map(([field, info]) => `${field}: ${(info as any).message}`)
					.join(', ');
			toasts.add('error', message || 'Failed to save award');
		} finally {
			saving = false;
		}
	}

	async function deleteAward(award: Award) {
		const confirmed = await confirm({
			title: 'Delete Award',
			message: `Are you sure you want to delete "${award.title}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await await collection('awards').delete(award.id);
			toasts.add('success', 'Award deleted');
			await loadAwards();
		} catch (err) {
			console.error('Failed to delete award:', err);
			toasts.add('error', 'Failed to delete award');
		}
	}

	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		if (selectedIds.has(id)) selectedIds.delete(id);
		else selectedIds.add(id);
		selectedIds = selectedIds;
	}

	function selectAll() { selectedIds = new Set(awards.map(e => e.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('awards').update(id, { visibility });
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadAwards();
		} catch (err) {
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: 'Delete Awards',
			message: `Are you sure you want to delete ${ids.length} award(s)? This action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;
		try {
			for (const id of ids) await collection('awards').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadAwards();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.content.awards.title')} {$t('admin.content.common.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="awards">
		{@html $t('admin.content.awards.help_text')}
		<p><strong>Tip:</strong> {$t('admin.content.awards.help_tip_1')}</p>
	</PageHelp>

	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={awards.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.content.awards.title')}</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Showcase notable awards, honors, fellowships, and accolades.
			</p>
		</div>
		<div class="flex items-center gap-2">
			{#if awards.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={toggleSelectMode}
				>
					{selectMode ? $t('admin.content.common.cancel') : $t('admin.content.common.select')}
				</button>
			{/if}
			<button class="btn btn-primary" onclick={openNewForm}>
				+ {$t('admin.content.common.new_button', { values: { type: $t('admin.content.awards.title').slice(0, -1) } })}
			</button>
		</div>
	</div>

	{#if showRecoveryBanner && recoveryData}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

	<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} />

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.content.common.loading', { values: { type: $t('admin.content.awards.title').toLowerCase() } })}</div>
		</div>
	{:else if showForm}
		<!-- Award Form (Inline) -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingAward ? 'Edit Award' : 'New Award'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" onclick={closeForm} aria-label="Close form">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="title" class="label">Title *</label>
					<input
						type="text"
						id="title"
						bind:value={title}
						class="input"
						placeholder="e.g., ACM Distinguished Engineer"
						required
					/>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="issuer" class="label">Issuer</label>
						<input
							type="text"
							id="issuer"
							bind:value={issuer}
							class="input"
							placeholder="Organization or event"
						/>
					</div>
					<div>
						<label for="awarded_at" class="label">Awarded on</label>
						<input
							type="text"
							id="awarded_at"
							bind:value={awardedAt}
							placeholder="2023 or 2023-11"
							class="input"
						/>
					</div>
				</div>

				<div>
					<label for="description" class="label">Description</label>
					<textarea
						id="description"
						bind:value={description}
						class="input h-28"
						placeholder="Optional summary or citation"
					></textarea>
				</div>

				<div>
					<label for="url" class="label">Link</label>
					<input
						type="url"
						id="url"
						bind:value={url}
						class="input"
						placeholder="https://example.com/announcement"
					/>
					<p class="text-xs text-gray-500 mt-1">Link to announcement or verification</p>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Settings</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="visibility" class="label">Visibility</label>
						<select id="visibility" bind:value={visibility} class="input">
							<option value="public">Public</option>
							<option value="unlisted">Unlisted</option>
							<option value="private">Private</option>
						</select>
					</div>

					<div>
						<label for="sort_order" class="label">Sort Order</label>
						<input
							type="number"
							id="sort_order"
							bind:value={sortOrder}
							class="input"
							min="0"
						/>
						<p class="text-xs text-gray-500 mt-1">Higher numbers appear first</p>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<input
						type="checkbox"
						id="is_draft"
						bind:checked={isDraft}
						class="w-4 h-4 text-primary-600 rounded border-gray-300"
					/>
					<label for="is_draft" class="text-sm text-gray-700 dark:text-gray-300">
						Save as draft (won't be visible publicly)
					</label>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<button type="button" class="btn btn-secondary" onclick={closeForm}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingAward ? 'Update Award' : 'Create Award'}
				</button>
			</div>
		</form>
	{:else if awards.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No awards yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your awards, honors, fellowships, and recognitions.
			</p>
			<button class="btn btn-primary" onclick={openNewForm}>
				+ Add Your First Award
			</button>
		</div>
	{:else if filteredAwards.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No awards match your filters</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Try adjusting your search or filter criteria.
			</p>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				Clear All Filters
			</button>
		</div>
	{:else}
		<!-- Awards List -->
		<div class="space-y-3">
			{#each filteredAwards as award (award.id)}
				<div class="card p-4 {selectMode && selectedIds.has(award.id) ? 'ring-2 ring-primary-500' : ''}">
					<div class="flex items-start justify-between gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(award.id)}
								onchange={() => toggleSelect(award.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-gray-900 dark:text-white">
									{award.title}
								</h3>
								{#if award.is_draft}
									<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
										Draft
									</span>
								{/if}
								{#if award.visibility !== 'public'}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{award.visibility}
									</span>
								{/if}
							</div>

							{#if memberships[award.id]?.length}
								<div class="flex flex-wrap gap-1 mt-2">
									{#each memberships[award.id].slice(0, 3) as viewRef}
										<a
											href={`/admin/views/${viewRef.id}`}
											target="_blank"
											class="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
											title={`Open view: ${viewRef.name}`}
										>
											{viewRef.slug || viewRef.name}
										</a>
									{/each}
									{#if memberships[award.id].length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-400">
											+{memberships[award.id].length - 3}
										</span>
									{/if}
								</div>
							{:else}
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">Not in any view</p>
							{/if}

							<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
								{#if award.issuer}
									<span>{award.issuer}</span>
								{/if}
								{#if award.awarded_at}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
										</svg>
										{formatDate(award.awarded_at, { month: 'long', year: 'numeric' })}
									</span>
								{/if}
							</div>

							{#if award.description}
								<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
									{truncate(award.description, 200)}
								</p>
							{/if}

							{#if award.url}
								<div class="mt-2">
									<a href={award.url} target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 dark:text-primary-400 hover:underline flex items-center gap-1">
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
										</svg>
										View announcement
									</a>
								</div>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<button
								class="p-3 text-gray-500 hover:text-blue-600"
								onclick={() => openEditForm(award)}
								title="Edit"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="p-3 text-gray-500 hover:text-red-600"
								onclick={() => deleteAward(award)}
								title="Delete"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
								</svg>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

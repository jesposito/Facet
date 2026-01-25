<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { pb, type Custom } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import { truncate } from '$lib/utils';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';

	let items: Custom[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingItem: Custom | null = $state(null);
	let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});

	// Form fields
	let title = $state('');
	let content = $state('');
	let visibility = $state('public');
	let isDraft = $state(false);
	let sortOrder = $state(0);
	let saving = $state(false);

	let selectMode = $state(false);
	let selectedIds: Set<string> = $state(new Set());

	const autosave = createAutosave('admin-custom', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: true,
		enableDraftFilter: true,
		enableTagFilter: false,
		searchPlaceholder: 'Search custom content...'
	});
	let showAdvancedFilters = $state(false);

	let filteredItems = $derived(
		filterStore.filterItems(
			items,
			(item) => `${item.title} ${item.content || ''}`,
			(item) => item.visibility,
			(item) => item.is_draft
		)
	);

	function getFormData() {
		return { title, content, visibility, isDraft, sortOrder };
	}

	function restoreFromDraft(data: Record<string, any>) {
		title = data.title || '';
		content = data.content || '';
		visibility = data.visibility || 'public';
		isDraft = data.isDraft || false;
		sortOrder = data.sortOrder || 0;
	}

	function handleFormChange() {
		if (showForm) {
			autosave.save(getFormData(), !!editingItem, editingItem?.id);
		}
	}

	function handleRestoreDraft() {
		const draft = autosave.loadDraft();
		if (draft?.data) {
			restoreFromDraft(draft.data);
			if (draft.isEditing && draft.editingId) {
				const item = items.find((a) => a.id === draft.editingId);
				if (item) editingItem = item;
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
		editingItem = null;
		selectMode = false;
		selectedIds = new Set();

		const draft = autosave.loadDraft();
		if (draft?.data && Object.values(draft.data).some((v) => v !== '' && v !== false && v !== 0)) {
			recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
			showRecoveryBanner = true;
		} else {
			showRecoveryBanner = false;
			recoveryData = null;
		}
	});

	onMount(loadItems);

	async function loadItems() {
		loading = true;
		try {
			const [records, membershipResp] = await Promise.all([
				await collection('custom').getList(1, 100, {
					sort: '-sort_order,-created',
					requestKey: null
				}),
				fetch('/api/admin/view-memberships?collection=custom', {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				}).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed memberships'))))
			]);
			items = records.items as unknown as Custom[];
			memberships = (membershipResp.memberships as typeof memberships) || {};
		} catch (err) {
			console.error('Failed to load custom content:', err);
			toasts.add('error', 'Failed to load custom content');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		title = '';
		content = '';
		visibility = 'public';
		isDraft = false;
		sortOrder = 0;
		editingItem = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function openEditForm(item: Custom) {
		editingItem = item;
		title = item.title;
		content = item.content || '';
		visibility = item.visibility;
		isDraft = item.is_draft;
		sortOrder = item.sort_order;
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
				content: content.trim(),
				visibility,
				is_draft: isDraft,
				sort_order: finalSort
			};

			if (editingItem) {
				await await collection('custom').update(editingItem.id, data);
				toasts.add('success', 'Content updated successfully');
			} else {
				await await collection('custom').create(data);
				toasts.add('success', 'Content created successfully');
			}

			closeForm();
			await loadItems();
		} catch (err) {
			console.error('Failed to save custom content:', err);
			const message =
				(err as any)?.data?.data &&
				Object.entries((err as any).data.data)
					.map(([field, info]) => `${field}: ${(info as any).message}`)
					.join(', ');
			toasts.add('error', message || 'Failed to save custom content');
		} finally {
			saving = false;
		}
	}

	async function deleteItem(item: Custom) {
		const confirmed = await confirm({
			title: 'Delete Custom Content',
			message: `Are you sure you want to delete "${item.title}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await await collection('custom').delete(item.id);
			toasts.add('success', 'Content deleted');
			await loadItems();
		} catch (err) {
			console.error('Failed to delete custom content:', err);
			toasts.add('error', 'Failed to delete custom content');
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

	function selectAll() {
		selectedIds = new Set(items.map((e) => e.id));
	}
	function clearSelection() {
		selectedIds = new Set();
	}

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('custom').update(id, { visibility });
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadItems();
		} catch (err) {
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: 'Delete Custom Content',
			message: `Are you sure you want to delete ${ids.length} item(s)? This action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;
		try {
			for (const id of ids) await collection('custom').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadItems();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}

	// Strip HTML for preview
	function stripHtml(html: string): string {
		const tmp = document.createElement('div');
		tmp.innerHTML = html;
		return tmp.textContent || tmp.innerText || '';
	}
</script>

<svelte:head>
	<title>Custom Content | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="custom">
		<p><strong>Custom Content</strong> lets you create freeform sections for your profile.</p>
		<p>
			Use it for anything that doesn't fit other categories: personal statements, mission
			statements, custom announcements, or specialized content sections.
		</p>
		<p>
			<strong>Tip:</strong> Each custom content item can be added to your views as its own section,
			giving you complete control over your profile layout.
		</p>
	</PageHelp>

	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={items.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Custom Content</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Create freeform content sections for your profile.
			</p>
		</div>
		<div class="flex items-center gap-2">
			{#if items.length > 0}
				<button class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}" onclick={toggleSelectMode}>
					{selectMode ? 'Cancel' : 'Select'}
				</button>
			{/if}
			<button class="btn btn-primary" onclick={openNewForm}> + New Content </button>
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
			<div class="animate-pulse">Loading custom content...</div>
		</div>
	{:else if showForm}
		<!-- Custom Content Form (Inline) -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingItem ? 'Edit Content' : 'New Content'}
					</h2>
					<button
						type="button"
						class="text-gray-500 hover:text-gray-700"
						onclick={closeForm}
						aria-label="Close form"
					>
						<svg
							class="w-5 h-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>

				<div>
					<label for="title" class="label">Section Header *</label>
					<input
						type="text"
						id="title"
						bind:value={title}
						class="input"
						placeholder="e.g., About Me, Mission Statement, Current Focus"
						required
					/>
					<p class="text-xs text-gray-500 mt-1">This will be displayed as the section heading on your profile</p>
				</div>

				<div>
					<label for="content" class="label">Content</label>
					<textarea
						id="content"
						bind:value={content}
						class="input h-48"
						placeholder="Write your content here. Supports Markdown formatting."
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Supports Markdown for formatting</p>
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
						<input type="number" id="sort_order" bind:value={sortOrder} class="input" min="0" />
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
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					{/if}
					{editingItem ? 'Update Content' : 'Create Content'}
				</button>
			</div>
		</form>
	{:else if items.length === 0}
		<div class="card p-8 text-center">
			<svg
				class="w-12 h-12 mx-auto text-gray-400 mb-4"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
				/>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No custom content yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Create freeform content sections for your profile.
			</p>
			<button class="btn btn-primary" onclick={openNewForm}> + Add Your First Content </button>
		</div>
	{:else if filteredItems.length === 0}
		<div class="card p-8 text-center">
			<svg
				class="w-12 h-12 mx-auto text-gray-400 mb-4"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				/>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
				No content matches your filters
			</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Try adjusting your search or filter criteria.
			</p>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				Clear All Filters
			</button>
		</div>
	{:else}
		<!-- Custom Content List -->
		<div class="space-y-3">
			{#each filteredItems as item (item.id)}
				<div
					class="card p-4 {selectMode && selectedIds.has(item.id)
						? 'ring-2 ring-primary-500'
						: ''}"
				>
					<div class="flex items-start justify-between gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(item.id)}
								onchange={() => toggleSelect(item.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-gray-900 dark:text-white">
									{item.title}
								</h3>
								{#if item.is_draft}
									<span
										class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded"
									>
										Draft
									</span>
								{/if}
								{#if item.visibility !== 'public'}
									<span
										class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded"
									>
										{item.visibility}
									</span>
								{/if}
							</div>

							{#if memberships[item.id]?.length}
								<div class="flex flex-wrap gap-1 mt-2">
									{#each memberships[item.id].slice(0, 3) as viewRef}
										<a
											href={`/admin/views/${viewRef.id}`}
											target="_blank"
											class="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
											title={`Open view: ${viewRef.name}`}
										>
											{viewRef.slug || viewRef.name}
										</a>
									{/each}
									{#if memberships[item.id].length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-400">
											+{memberships[item.id].length - 3}
										</span>
									{/if}
								</div>
							{:else}
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">Not in any view</p>
							{/if}

							{#if item.content}
								<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
									{truncate(stripHtml(item.content), 200)}
								</p>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<button
								class="p-3 text-gray-500 hover:text-blue-600"
								onclick={() => openEditForm(item)}
								title="Edit"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/>
								</svg>
							</button>
							<button
								class="p-3 text-gray-500 hover:text-red-600"
								onclick={() => deleteItem(item)}
								title="Delete"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									/>
								</svg>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Award } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { formatDate, truncate } from '$lib/utils';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';

	let awards: Award[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingAward: Award | null = $state(null);

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

	onMount(loadAwards);

	async function loadAwards() {
		loading = true;
		try {
			const records = await await collection('awards').getList(1, 100, {
				sort: '-sort_order,-awarded_at'
			});
			awards = records.items as unknown as Award[];
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
	}

	function openEditForm(award: Award) {
		editingAward = award;
		title = award.title;
		issuer = award.issuer || '';
		awardedAt = award.awarded_at ? award.awarded_at.split('T')[0] : '';
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
	<title>Awards | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
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
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Awards & Honors</h1>
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
					{selectMode ? 'Cancel' : 'Select'}
				</button>
			{/if}
			<button class="btn btn-primary" onclick={openNewForm}>
				Add Award
			</button>
		</div>
	</div>

	{#if loading}
		<div class="text-gray-500 dark:text-gray-400">Loading awards...</div>
	{:else if awards.length === 0}
		<div class="text-gray-500 dark:text-gray-400">No awards added yet.</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each awards as award (award.id)}
				<article class="card p-5 flex flex-col gap-2 {selectMode && selectedIds.has(award.id) ? 'ring-2 ring-primary-500' : ''}">
					<div class="flex items-start justify-between gap-2">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(award.id)}
								onchange={() => toggleSelect(award.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div>
							<p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
								{award.visibility}{award.is_draft ? ' • Draft' : ''}
							</p>
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{award.title}</h2>
							{#if award.issuer}
								<p class="text-sm text-gray-600 dark:text-gray-400">{award.issuer}</p>
							{/if}
						</div>
						<div class="flex items-center gap-2">
							<button class="btn btn-ghost btn-sm" onclick={() => openEditForm(award)}>
								Edit
							</button>
							<button class="btn btn-ghost btn-sm text-red-600" onclick={() => deleteAward(award)}>
								Delete
							</button>
						</div>
					</div>

					{#if award.awarded_at}
						<p class="text-sm text-gray-600 dark:text-gray-400">
							Awarded {formatDate(award.awarded_at, { month: 'long', year: 'numeric' })}
						</p>
					{/if}

					{#if award.description}
						<p class="text-sm text-gray-700 dark:text-gray-300">{truncate(award.description, 200)}</p>
					{/if}

					{#if award.url}
						<a
							href={award.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
							</svg>
							View link
						</a>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</div>

<!-- Award Form Modal -->
{#if showForm}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
		<div class="bg-white dark:bg-gray-900 rounded-xl shadow-xl w-full max-w-2xl border border-gray-200 dark:border-gray-700">
			<div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
				<div>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingAward ? 'Edit Award' : 'Add Award'}
					</h2>
					<p class="text-sm text-gray-600 dark:text-gray-400">Add details about an award or honor.</p>
				</div>
				<button class="btn btn-ghost btn-sm" onclick={closeForm}>Close</button>
			</div>

			<div class="p-4 space-y-4">
				<div>
					<label class="label" for="title">Title</label>
					<input id="title" class="input" bind:value={title} placeholder="e.g., ACM Distinguished Engineer" />
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="label" for="issuer">Issuer</label>
						<input id="issuer" class="input" bind:value={issuer} placeholder="Organization or event" />
					</div>
					<div>
						<label class="label" for="awarded_at">Awarded on</label>
						<input id="awarded_at" type="date" class="input" bind:value={awardedAt} />
					</div>
				</div>

				<div>
					<label class="label" for="description">Description</label>
					<textarea
						id="description"
						class="input h-28"
						placeholder="Optional summary or citation"
						bind:value={description}
					></textarea>
				</div>

				<div>
					<label class="label" for="url">Link</label>
					<input id="url" class="input" bind:value={url} placeholder="Optional link to announcement" />
				</div>

				<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
					<div>
						<label class="label" for="visibility">Visibility</label>
						<select id="visibility" class="input" bind:value={visibility}>
							<option value="public">Public</option>
							<option value="unlisted">Unlisted</option>
							<option value="private">Private</option>
						</select>
					</div>
					<div class="flex items-center gap-2">
						<input id="draft" type="checkbox" bind:checked={isDraft} class="h-4 w-4 text-primary-600" />
						<label for="draft" class="text-sm text-gray-700 dark:text-gray-300">Mark as draft</label>
					</div>
					<div>
						<label class="label" for="sort">Sort order</label>
						<input id="sort" type="number" class="input" bind:value={sortOrder} />
					</div>
				</div>
			</div>

			<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
				<button class="btn btn-ghost" onclick={closeForm}>Cancel</button>
				<button class="btn btn-primary" onclick={handleSubmit} disabled={saving}>
					{saving ? 'Saving...' : 'Save'}
				</button>
			</div>
		</div>
	</div>
{/if}

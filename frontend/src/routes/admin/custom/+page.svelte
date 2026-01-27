<script lang="ts">
	import { preventDefault } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { browser } from '$app/environment';
	import { flip } from 'svelte/animate';
	import { pb, type CustomContent, getFileUrl } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import AdminTagSelector from '$components/admin/AdminTagSelector.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';

	let items: CustomContent[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingItem: CustomContent | null = $state(null);
	let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});

	// Form fields
	let title = $state('');
	let content = $state('');
	let visibility = $state('public');
	let isDraft = $state(false);
	let sortOrder = $state(0);
	let coverImageFile: FileList | null = $state(null);
	let pendingMediaFiles: File[] = $state([]); // Accumulated new files to upload
	let mediaToKeep: string[] = $state([]); // Track existing media filenames to keep
	let saving = $state(false);
	let adminTagIds: string[] = $state([]);

	// Media library refs
	let mediaRefs: string[] = $state([]);
	let mediaOptions: { id: string; title: string; provider?: string; url?: string }[] = $state([]);
	let mediaSearch = $state('');
	let loadingMedia = $state(false);

	let selectMode = $state(false);
	let selectedIds: Set<string> = $state(new Set());

	let reorderMode = $state(false);
	let reorderItems: Array<{ id: string; title: string; is_draft: boolean }> = $state([]);
	let savingOrder = $state(false);
	const flipDurationMs = 200;

	let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
	let dndLoaded = $state(false);

	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: true,
		enableDraftFilter: true,
		enableTagFilter: true,
		searchPlaceholder: 'Search custom content...'
	});
	let availableTags: Array<{ id: string; name: string; color: string }> = $state([]);
	let showAdvancedFilters = $state(false);

	const autosave = createAutosave('admin-custom-content', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	function getFormData() {
		return { title, content, visibility, isDraft, sortOrder, mediaRefs };
	}

	function restoreFromDraft(data: Record<string, any>) {
		title = data.title || '';
		content = data.content || '';
		visibility = data.visibility || 'public';
		isDraft = data.isDraft || false;
		sortOrder = data.sortOrder || 0;
		mediaRefs = data.mediaRefs || [];
	}

	function handleFormChange() {
		if (showForm) {
			autosave.save(getFormData(), !!editingItem, editingItem?.id);
		}
	}

	afterNavigate(() => {
		showForm = false;
		editingItem = null;
		selectMode = false;
		selectedIds = new Set();

		const draft = autosave.loadDraft();
		if (
			draft?.data &&
			Object.values(draft.data).some(
				(v) => v !== '' && v !== false && v !== 0 && !(Array.isArray(v) && v.length === 0)
			)
		) {
			recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
			showRecoveryBanner = true;
		} else {
			showRecoveryBanner = false;
			recoveryData = null;
		}
	});

	onMount(loadItems);
	onMount(loadAvailableTags);
	onMount(loadMediaOptions);
	onMount(async () => {
		if (browser) {
			const { dndzone: dnd } = await import('svelte-dnd-action');
			dndzone = dnd;
			dndLoaded = true;
		}
	});

	async function loadAvailableTags() {
		try {
			const records = await collection('admin_tags').getList(1, 100, {
				sort: 'sort_order,name'
			});
			availableTags = records.items.map((tag) => ({
				id: tag.id,
				name: tag.name,
				color: tag.color
			}));
		} catch (err) {
			console.error('Failed to load available tags:', err);
		}
	}

	async function loadMediaOptions(search = '') {
		loadingMedia = true;
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};

			const mediaParams = new URLSearchParams({ perPage: '50' });
			if (search) mediaParams.set('q', search);

			const externalFilter = search ? `&filter=${encodeURIComponent(`title~"${search}" || url~"${search}"`)}` : '';

			const [mediaRes, externalRes] = await Promise.all([
				fetch(`/api/media?${mediaParams.toString()}`, { headers }),
				fetch(`/api/collections/external_media/records?perPage=50${externalFilter}`, { headers })
			]);

			const mediaData = mediaRes.ok ? await mediaRes.json() : { items: [] };
			const externalData = externalRes.ok ? await externalRes.json() : { items: [] };

			const options: { id: string; title: string; provider?: string; url?: string }[] = [];

			for (const item of mediaData.items || []) {
				options.push({
					id: item.record_id || item.relative_path || item.url,
					title: item.display_name || item.filename || item.url,
					provider: item.provider || (item.external ? 'external' : 'upload'),
					url: item.url
				});
			}

			for (const item of externalData.items || []) {
				if (!options.find((opt) => opt.id === item.id || opt.url === item.url)) {
					options.push({
						id: item.id,
						title: item.title || item.url,
						provider: 'external',
						url: item.url
					});
				}
			}

			mediaOptions = options;
		} catch (err) {
			console.error('Failed to load media options', err);
		} finally {
			loadingMedia = false;
		}
	}

	async function resolveMediaRefs(selected: string[]) {
		const headers: Record<string, string> = pb.authStore.isValid
			? { Authorization: `Bearer ${pb.authStore.token}` }
			: {};
		const optionMap = new Map(mediaOptions.map((opt) => [opt.id, opt]));
		const resolved: string[] = [];

		const toAbsolute = (url?: string) => {
			if (!url) return '';
			if (/^https?:\/\//i.test(url)) return url;
			const base = pb.baseUrl.replace(/\/$/, '');
			return `${base}${url.startsWith('/') ? url : `/${url}`}`;
		};

		for (const id of selected) {
			const opt = optionMap.get(id);
			if (!opt) continue;
			if (opt.provider === 'upload' && opt.url) {
				try {
					const filter = encodeURIComponent(`url="${opt.url}"`);
					const existingRes = await fetch(
						`/api/collections/external_media/records?perPage=1&filter=${filter}`,
						{ headers }
					);
					if (existingRes.ok) {
						const existing = await existingRes.json();
						if (existing.items?.[0]?.id) {
							resolved.push(existing.items[0].id);
							continue;
						}
					}
					const created = await fetch('/api/media/external', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json', ...headers },
						body: JSON.stringify({ url: toAbsolute(opt.url), title: opt.title })
					});
					if (created.ok) {
						const body = await created.json();
						if (body.id) {
							resolved.push(body.id);
							continue;
						}
					}
				} catch (err) {
					console.error('Failed to mirror upload to external_media', err);
				}
			} else {
				resolved.push(id);
			}
		}

		return resolved;
	}

	let filteredItems = $derived(
		filterStore.filterItems(
			items,
			(item) => `${item.title} ${item.content || ''}`,
			(item) => item.visibility,
			(item) => item.is_draft,
			(item) => (item as any).admin_tags || []
		)
	);

	async function loadItems() {
		loading = true;
		try {
			const [itemsResp, membershipResp] = await Promise.all([
				await collection('custom_content').getList(1, 100, {
					sort: 'sort_order',
					expand: 'admin_tags'
				}),
				fetch('/api/admin/view-memberships?collection=custom_content', {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				}).then((r) => (r.ok ? r.json() : { memberships: {} }))
			]);
			items = itemsResp.items as unknown as CustomContent[];
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
		coverImageFile = null;
		pendingMediaFiles = [];
		mediaToKeep = [];
		editingItem = null;
		adminTagIds = [];
		mediaRefs = [];
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function handleRestoreDraft() {
		const draft = autosave.loadDraft();
		if (draft?.data) {
			restoreFromDraft(draft.data);
			if (draft.isEditing && draft.editingId) {
				const item = items.find((i) => i.id === draft.editingId);
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

	function openEditForm(item: CustomContent) {
		editingItem = item;
		title = item.title;
		content = item.content || '';
		visibility = item.visibility;
		isDraft = item.is_draft;
		sortOrder = item.sort_order;
		adminTagIds = item.admin_tags || [];
		coverImageFile = null;
		pendingMediaFiles = [];
		mediaToKeep = item.media ? [...item.media] : []; // Keep all existing media by default
		mediaRefs = (item as any).media_refs || [];
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

		saving = true;
		try {
			const formData = new FormData();
			formData.append('title', title.trim());
			formData.append('content', content);
			formData.append('visibility', visibility);
			formData.append('is_draft', String(isDraft));
			formData.append('sort_order', String(sortOrder));

			// Handle admin tags
			if (adminTagIds.length > 0) {
				adminTagIds.forEach((tagId) => formData.append('admin_tags', tagId));
			}

			// Handle cover image upload
			if (coverImageFile && coverImageFile.length > 0) {
				formData.append('cover_image', coverImageFile[0]);
			}

			// Handle media files - include existing files to keep AND new files to add
			if (editingItem) {
				// When editing, first add existing media filenames to keep
				for (const filename of mediaToKeep) {
					formData.append('media', filename);
				}
			}
			// Then add any new pending files
			for (const file of pendingMediaFiles) {
				formData.append('media', file);
			}

			// Handle media library refs
			const resolvedRefs = await resolveMediaRefs(mediaRefs);
			formData.append('media_refs', JSON.stringify(resolvedRefs));

			if (editingItem) {
				await collection('custom_content').update(editingItem.id, formData);
				toasts.add('success', 'Custom content updated successfully');
			} else {
				await collection('custom_content').create(formData);
				toasts.add('success', 'Custom content created successfully');
			}

			closeForm();
			await loadItems();
		} catch (err) {
			console.error('Failed to save custom content:', err);
			toasts.add('error', 'Failed to save custom content');
		} finally {
			saving = false;
		}
	}

	async function deleteItem(item: CustomContent) {
		const confirmed = await confirm({
			title: 'Delete Custom Content',
			message: `Are you sure you want to delete "${item.title}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;

		try {
			await collection('custom_content').delete(item.id);
			toasts.add('success', 'Custom content deleted');
			await loadItems();
		} catch (err) {
			console.error('Failed to delete custom content:', err);
			toasts.add('error', 'Failed to delete custom content');
		}
	}

	async function togglePublish(item: CustomContent) {
		try {
			const newDraftState = !item.is_draft;
			await collection('custom_content').update(item.id, {
				is_draft: newDraftState
			});
			toasts.add('success', newDraftState ? 'Content set to draft' : 'Content published');
			await loadItems();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update custom content');
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

	async function bulkSetVisibility(vis: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('custom_content').update(id, { visibility: vis });
			toasts.add('success', `Updated ${ids.length} items to ${vis}`);
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
			for (const id of ids) await collection('custom_content').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadItems();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}

	// Reorder mode functions
	function enterReorderMode() {
		reorderItems = items.map((i) => ({ id: i.id, title: i.title, is_draft: i.is_draft }));
		reorderMode = true;
	}

	function cancelReorder() {
		reorderMode = false;
		reorderItems = [];
	}

	function handleDndConsider(e: CustomEvent<{ items: typeof reorderItems }>) {
		reorderItems = e.detail.items;
	}

	function handleDndFinalize(e: CustomEvent<{ items: typeof reorderItems }>) {
		reorderItems = e.detail.items;
	}

	async function saveOrder() {
		savingOrder = true;
		try {
			for (let i = 0; i < reorderItems.length; i++) {
				await collection('custom_content').update(reorderItems[i].id, { sort_order: i });
			}
			toasts.add('success', 'Order saved');
			reorderMode = false;
			await loadItems();
		} catch (err) {
			toasts.add('error', 'Failed to save order');
		} finally {
			savingOrder = false;
		}
	}

	async function removeCoverImage() {
		if (!editingItem) return;
		try {
			await collection('custom_content').update(editingItem.id, { cover_image: null });
			toasts.add('success', 'Cover image removed');
			await loadItems();
			// Re-open form with updated item
			const updated = items.find((i) => i.id === editingItem!.id);
			if (updated) openEditForm(updated);
		} catch (err) {
			toasts.add('error', 'Failed to remove cover image');
		}
	}

	function removeMediaFromKeep(filename: string) {
		mediaToKeep = mediaToKeep.filter((f) => f !== filename);
	}

	function handleMediaFilesSelected(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files && input.files.length > 0) {
			// Accumulate new files instead of replacing
			pendingMediaFiles = [...pendingMediaFiles, ...Array.from(input.files)];
			// Clear the input so the same file can be selected again if needed
			input.value = '';
		}
	}

	function removePendingFile(index: number) {
		pendingMediaFiles = pendingMediaFiles.filter((_, i) => i !== index);
	}
</script>

<svelte:head>
	<title>{$t('admin.content.custom.title')} {$t('admin.content.common.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="custom-content">
		{@html $t('admin.content.custom.help_text')}
		<p>{$t('admin.content.custom.help_tip_1')}</p>
	</PageHelp>

	{#if showRecoveryBanner && recoveryData}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

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
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.content.custom.title')}</h1>
		<div class="flex items-center gap-2">
			{#if items.length > 1 && !showForm}
				<button
					class="btn {reorderMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={reorderMode ? cancelReorder : enterReorderMode}
				>
					{reorderMode ? 'Cancel' : 'Reorder'}
				</button>
			{/if}
			{#if items.length > 0 && !reorderMode}
				<button class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}" onclick={toggleSelectMode}>
					{selectMode ? $t('admin.content.common.cancel') : $t('admin.content.common.select')}
				</button>
			{/if}
			{#if !reorderMode}
				<button class="btn btn-primary" onclick={openNewForm}> {$t('admin.content.common.new_button', { values: { type: $t('admin.content.custom.title') } })} </button>
			{/if}
		</div>
	</div>

	{#if !reorderMode}
		<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} {availableTags} />
	{/if}

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.content.common.loading', { values: { type: $t('admin.content.custom.title').toLowerCase() } })}</div>
		</div>
	{:else if reorderMode}
		<!-- Reorder Mode -->
		<div class="card p-4">
			<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
				Drag and drop to reorder. Click "Save Order" when done.
			</p>
			{#if dndLoaded}
				<div
					use:dndzone={{ items: reorderItems, flipDurationMs }}
					onconsider={handleDndConsider}
					onfinalize={handleDndFinalize}
					class="space-y-2"
				>
					{#each reorderItems as item (item.id)}
						<div
							animate:flip={{ duration: flipDurationMs }}
							class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg cursor-grab active:cursor-grabbing"
						>
							<svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M4 8h16M4 16h16"
								/>
							</svg>
							<span class="font-medium text-gray-900 dark:text-white">{item.title}</span>
							{#if item.is_draft}
								<span
									class="px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded"
								>
									Draft
								</span>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
			<div class="flex justify-end gap-2 mt-4">
				<button class="btn btn-secondary" onclick={cancelReorder}>Cancel</button>
				<button class="btn btn-primary" onclick={saveOrder} disabled={savingOrder}>
					{savingOrder ? 'Saving...' : 'Save Order'}
				</button>
			</div>
		</div>
	{:else if showForm}
		<!-- Custom Content Form -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingItem ? 'Edit Custom Content' : 'New Custom Content'}
					</h2>
					<button
						type="button"
						class="text-gray-500 hover:text-gray-700"
						onclick={closeForm}
						aria-label="Close form"
					>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
					<label for="title" class="label">Title / Header *</label>
					<input
						type="text"
						id="title"
						bind:value={title}
						class="input"
						placeholder="About Me, My Philosophy, etc."
						required
					/>
					<p class="text-xs text-gray-500 mt-1">This will be the section header when displayed</p>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="content" class="label mb-0">Content (Markdown)</label>
						<AIContentHelper
							fieldType="content"
							content={content}
							context={{ title }}
							on:apply={(e) => (content = e.detail.content)}
							size="sm"
						/>
					</div>
					<textarea
						id="content"
						bind:value={content}
						class="input min-h-[200px] font-mono text-sm"
						placeholder="Write your content here... (Markdown supported)"
					></textarea>
				</div>

				<div>
					<label for="cover_image" class="label">Cover Image</label>
					{#if editingItem?.cover_image}
						<div class="mb-2 flex items-center gap-2">
							<img
								src={getFileUrl(
									{ id: editingItem.id, collectionName: 'custom_content' },
									editingItem.cover_image
								)}
								alt="Current cover"
								class="w-24 h-24 object-cover rounded"
							/>
							<button type="button" class="btn btn-ghost btn-sm text-red-600" onclick={removeCoverImage}>
								Remove
							</button>
						</div>
					{/if}
					<input
						type="file"
						id="cover_image"
						accept="image/jpeg,image/png,image/webp"
						bind:files={coverImageFile}
						class="input"
					/>
				</div>

				<div>
					<label for="media" class="label">Media Gallery</label>

					<!-- Existing media (when editing) -->
					{#if editingItem && mediaToKeep.length > 0}
						<p class="text-xs text-gray-500 mb-1">Existing images:</p>
						<div class="flex flex-wrap gap-2 mb-3">
							{#each mediaToKeep as filename}
								<div class="relative group">
									<img
										src={getFileUrl(
											{ id: editingItem.id, collectionName: 'custom_content' },
											filename
										)}
										alt="Media"
										class="w-16 h-16 object-cover rounded border border-gray-200 dark:border-gray-700"
									/>
									<button
										type="button"
										class="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white rounded-full text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
										onclick={() => removeMediaFromKeep(filename)}
										title="Remove image"
									>
										×
									</button>
								</div>
							{/each}
						</div>
					{/if}

					<!-- Pending new uploads -->
					{#if pendingMediaFiles.length > 0}
						<p class="text-xs text-gray-500 mb-1">New images to upload:</p>
						<div class="flex flex-wrap gap-2 mb-3">
							{#each pendingMediaFiles as file, index}
								<div class="relative group">
									<img
										src={URL.createObjectURL(file)}
										alt={file.name}
										class="w-16 h-16 object-cover rounded border-2 border-primary-300 dark:border-primary-600"
									/>
									<button
										type="button"
										class="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white rounded-full text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
										onclick={() => removePendingFile(index)}
										title="Remove"
									>
										×
									</button>
								</div>
							{/each}
						</div>
					{/if}

					<input
						type="file"
						id="media"
						accept="image/jpeg,image/png,image/webp,image/svg+xml"
						multiple
						onchange={handleMediaFilesSelected}
						class="input"
					/>
					<p class="text-xs text-gray-500 mt-1">Select images to add (you can select multiple times to build up your gallery)</p>
				</div>

				<div>
					<p class="label">Attached media / embeds</p>
					<p class="text-xs text-gray-500 dark:text-gray-400 mb-2">
						Link media from the library (YouTube, Vimeo, external URLs, or uploaded files).
					</p>
					<div class="flex flex-wrap items-center gap-2 mb-2 text-sm text-gray-600 dark:text-gray-400">
						<input
							class="input w-full md:w-64"
							placeholder="Search media..."
							bind:value={mediaSearch}
							onkeydown={(e) => e.key === 'Enter' && loadMediaOptions(mediaSearch)}
						/>
						<button
							type="button"
							class="btn btn-secondary btn-sm"
							onclick={() => loadMediaOptions(mediaSearch)}
							aria-busy={loadingMedia}
						>
							{loadingMedia ? 'Searching…' : 'Search'}
						</button>
						<button
							type="button"
							class="btn btn-ghost btn-sm"
							onclick={() => {
								mediaSearch = '';
								loadMediaOptions('');
							}}
						>
							Clear
						</button>
					</div>
					{#if loadingMedia}
						<p class="text-sm text-gray-500 dark:text-gray-400">Loading media…</p>
					{:else if mediaOptions.length === 0}
						<p class="text-sm text-gray-500 dark:text-gray-400">
							No media found. Add uploads or external links in the <a
								href="/admin/media"
								class="text-primary-600 dark:text-primary-400 hover:underline">Media Library</a
							>.
						</p>
					{:else}
						<div class="flex flex-col gap-2 max-h-48 overflow-y-auto pr-2">
							{#each mediaOptions as opt}
								<label
									class={`flex items-center gap-2 px-3 py-2 rounded border cursor-pointer ${
										mediaRefs.includes(opt.id)
											? 'bg-primary-50 dark:bg-primary-900/30 border-primary-200 dark:border-primary-700 text-primary-700 dark:text-primary-200'
											: 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700'
									}`}
								>
									<input type="checkbox" class="w-4 h-4" bind:group={mediaRefs} value={opt.id} />
									<div class="flex flex-col min-w-0">
										<span class="text-sm font-medium truncate">{opt.title}</span>
										{#if opt.provider}
											<span class="text-xs text-gray-500 dark:text-gray-400">{opt.provider}</span>
										{/if}
									</div>
								</label>
							{/each}
						</div>
					{/if}
				</div>

				<div>
					<label class="label">Admin Tags</label>
					<AdminTagSelector bind:selectedIds={adminTagIds} />
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Publishing</h2>

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
					d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
				/>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No custom content yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Create custom sections for your facets with flexible content.
			</p>
			<button class="btn btn-primary" onclick={openNewForm}> Create your first content </button>
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
		<!-- Items List -->
		<div class="space-y-4">
			{#each filteredItems as item (item.id)}
				<div
					class="card p-4 hover:shadow-md transition-shadow {selectMode && selectedIds.has(item.id)
						? 'ring-2 ring-primary-500'
						: ''}"
				>
					<div class="flex items-start gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(item.id)}
								onchange={() => toggleSelect(item.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}

						{#if item.cover_image}
							<img
								src={getFileUrl({ id: item.id, collectionName: 'custom_content' }, item.cover_image)}
								alt={item.title}
								class="w-16 h-16 object-cover rounded shrink-0"
							/>
						{/if}

						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1">
								<h3 class="text-lg font-medium text-gray-900 dark:text-white truncate">
									{item.title}
								</h3>
								{#if item.is_draft}
									<span
										class="px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded"
									>
										Draft
									</span>
								{:else}
									<span
										class="px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 rounded"
									>
										Published
									</span>
								{/if}
								<span
									class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded capitalize"
								>
									{item.visibility}
								</span>
							</div>

							{#if item.content}
								<p class="text-sm text-gray-600 dark:text-gray-400 mb-2 line-clamp-2">
									{item.content.replace(/[#*_`]/g, '').slice(0, 150)}...
								</p>
							{/if}

							{#if item.expand?.admin_tags?.length}
								<div class="flex flex-wrap gap-1 mb-2">
									{#each item.expand.admin_tags as tag}
										<AdminTagBadge name={tag.name} color={tag.color} />
									{/each}
								</div>
							{/if}

							{#if memberships[item.id]?.length}
								<div class="flex flex-wrap gap-1 mb-2">
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
								<p class="text-xs text-gray-500 dark:text-gray-400 mb-2">Not in any view</p>
							{/if}
						</div>

						<div class="flex items-center gap-2 shrink-0">
							<button
								class="btn btn-ghost btn-sm"
								title={item.is_draft ? 'Publish' : 'Set as Draft'}
								onclick={() => togglePublish(item)}
							>
								{#if item.is_draft}
									<svg class="w-4 h-4 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M5 13l4 4L19 7"
										/>
									</svg>
								{:else}
									<svg class="w-4 h-4 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
										/>
									</svg>
								{/if}
							</button>
							<button class="btn btn-ghost btn-sm" title="Edit" onclick={() => openEditForm(item)}>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/>
								</svg>
							</button>
							<button
								class="btn btn-ghost btn-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
								title="Delete"
								onclick={() => deleteItem(item)}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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

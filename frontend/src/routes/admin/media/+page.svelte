<script lang="ts">
	import { onMount } from 'svelte';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import { formatDate } from '$lib/utils';
	import { pb } from '$lib/pocketbase';
	import { goto } from '$app/navigation';

	type MediaItem = {
		collection: string;
		collection_id: string;
		record_id: string;
		field: string;
		filename: string;
		url: string;
		size: number;
		mime: string;
		uploaded_at: string;
		relative_path?: string;
		orphan?: boolean;
		display_name?: string;
		record_label?: string;
		thumbnail_url?: string;
		collection_key?: string;
		external?: boolean;
	};

	type MediaStats = {
		referencedFiles: number;
		referencedSize: number;
		orphanFiles: number;
		orphanSize: number;
		totalFiles: number;
		totalSize: number;
		storageFiles: number;
		storageSize: number;
	};

	let items: MediaItem[] = [];
	let loading = true;
	let page = 1;
	let perPage = 50;
	let totalItems = 0;
	let totalPages = 1;
	let search = '';
	let typeFilter: 'all' | 'image' = 'all';
	let statusFilter: 'referenced' | 'all' | 'orphans' = 'referenced';
	let error = '';
	let stats: MediaStats = {
		referencedFiles: 0,
		referencedSize: 0,
		orphanFiles: 0,
		orphanSize: 0,
		totalFiles: 0,
		totalSize: 0,
		storageFiles: 0,
		storageSize: 0
	};
	let selectedOrphans: Set<string> = new Set();
	let newExternal = {
		url: '',
		title: '',
		mime: '',
		thumbnail_url: '',
		saving: false
	};
	let uploadFile: File | null = null;
	let uploadTitle = '';
	let uploading = false;

	const humanSize = (bytes: number) => {
		if (!bytes) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB'];
		let i = 0;
		let size = bytes;
		while (size >= 1024 && i < units.length - 1) {
			size /= 1024;
			i++;
		}
		return `${size.toFixed(size >= 10 ? 0 : 1)} ${units[i]}`;
	};

	const mimeLabel = (mime: string) => {
		if (!mime) return 'Link';
		if (mime.startsWith('image/')) return 'Image';
		if (mime.startsWith('video/')) return 'Video';
		if (mime.startsWith('audio/')) return 'Audio';
		return 'File';
	};

	async function loadMedia() {
		loading = true;
		error = '';
		try {
			const params = new URLSearchParams({
				page: String(page),
				perPage: String(perPage)
			});
			if (search.trim()) params.set('q', search.trim());
			if (typeFilter === 'image') params.set('type', 'image');
			if (statusFilter === 'orphans') {
				params.set('orphans', '1');
			} else if (statusFilter === 'all') {
				params.set('includeOrphans', '1');
			}

			const res = await fetch(`/api/media?${params.toString()}`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!res.ok) {
				if (res.status === 401) {
					toasts.add('error', 'Session expired. Please sign in again.');
					goto('/admin/login');
					return;
				}
				throw new Error(`Failed to load media (${res.status})`);
			}
			const data = await res.json();
			stats = data.stats || {
				referencedFiles: 0,
				referencedSize: 0,
				orphanFiles: 0,
				orphanSize: 0,
				totalFiles: totalItems,
				totalSize: 0,
				storageFiles: 0,
				storageSize: 0
			};
			items = data.items || [];
			// Append external media directly (some environments may not surface them via /api/media)
			const externalRes = await fetch('/api/collections/external_media/records?perPage=200', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (externalRes.ok) {
				const ext = await externalRes.json();
				const externalItems =
					(ext.items || []).map((item: any) => ({
						collection: 'external_media',
						collection_id: item.collectionId,
						record_id: item.id,
						field: 'external',
						filename: item.title || item.url,
						display_name: item.title || item.url,
						record_label: item.title || item.url,
						url: item.url,
						mime: item.mime || '',
						uploaded_at: item.created || new Date().toISOString(),
						external: true,
						collection_key: 'external',
						provider: item.provider || 'external'
					})) || [];
				items = [...items, ...externalItems];
				stats.referencedFiles += externalItems.length;
				stats.totalFiles += externalItems.length;
			}
			totalItems = items.length;
			totalPages = Math.max(1, Math.ceil(totalItems / perPage));
			selectedOrphans = new Set();
		} catch (err) {
			console.error(err);
			error = 'Failed to load media';
			toasts.add('error', 'Failed to load media');
		} finally {
			loading = false;
		}
	}

	function copyUrl(url: string) {
		const absolute = typeof window !== 'undefined' ? new URL(url, window.location.origin).toString() : url;
		navigator.clipboard.writeText(absolute);
		toasts.add('success', 'URL copied');
	}

	async function deleteFile(item: MediaItem) {
		const confirmed = await confirm({
			title: 'Delete Media',
			message: `Delete "${item.filename}"? This cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;
		try {
			if (item.external && item.record_id) {
				const res = await fetch(`/api/media/external/${item.record_id}`, {
					method: 'DELETE',
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				});
				if (!res.ok) {
					const body = await res.json().catch(() => ({}));
					throw new Error(body.error || 'Failed to delete external media');
				}
				toasts.add('success', 'External media deleted');
				await loadMedia();
				return;
			}
			const body =
				item.orphan && item.relative_path
					? { relative_path: item.relative_path }
					: {
							collection_id: item.collection_id,
							record_id: item.record_id,
							field: item.field,
							filename: item.filename
					  };
			const res = await fetch('/api/media', {
				method: 'DELETE',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify(body)
			});
			if (!res.ok) {
				if (res.status === 401) {
					toasts.add('error', 'Session expired. Please sign in again.');
					goto('/admin/login');
					return;
				}
				const body = await res.json().catch(() => ({}));
				throw new Error(body.error || 'Failed to delete file');
			}
			toasts.add('success', 'File deleted');
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : 'Failed to delete file');
		}
	}

	function toggleSelection(item: MediaItem) {
		if (!item.orphan || !item.relative_path) return;
		const next = new Set(selectedOrphans);
		if (next.has(item.relative_path)) {
			next.delete(item.relative_path);
		} else {
			next.add(item.relative_path);
		}
		selectedOrphans = next;
	}

	function selectVisibleOrphans() {
		const next = new Set(selectedOrphans);
		items.forEach((item) => {
			if (item.orphan && item.relative_path) {
				next.add(item.relative_path);
			}
		});
		selectedOrphans = next;
	}

	function clearSelection() {
		selectedOrphans = new Set();
	}

	async function bulkDeleteSelected() {
		if (selectedOrphans.size === 0) return;
		const confirmed = await confirm({
			title: 'Delete Orphan Files',
			message: `Delete ${selectedOrphans.size} orphan file(s)? These files are not referenced anywhere and this action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;

		try {
			const res = await fetch('/api/media/bulk-delete', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({ orphans: Array.from(selectedOrphans) })
			});
			if (!res.ok) {
				if (res.status === 401) {
					toasts.add('error', 'Session expired. Please sign in again.');
					goto('/admin/login');
					return;
				}
				const body = await res.json().catch(() => ({}));
				throw new Error(body.error || 'Failed to delete files');
			}
			const result = await res.json().catch(() => ({}));
			toasts.add('success', `Deleted ${result.deleted ?? selectedOrphans.size} orphan file(s)`);
			selectedOrphans = new Set();
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : 'Failed to delete files');
		}
	}

	async function createExternal() {
		if (!newExternal.url.trim()) {
			toasts.add('error', 'URL is required');
			return;
		}
		newExternal.saving = true;
		try {
			const res = await fetch('/api/media/external', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({
					url: newExternal.url.trim(),
					title: newExternal.title.trim(),
					mime: newExternal.mime.trim(),
					thumbnail_url: newExternal.thumbnail_url.trim()
				})
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				throw new Error(body.error || 'Failed to add external media');
			}
			toasts.add('success', 'External media added');
			newExternal = { url: '', title: '', mime: '', thumbnail_url: '', saving: false };
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : 'Failed to add external media');
		} finally {
			newExternal.saving = false;
		}
	}

	async function uploadMedia() {
		if (!uploadFile) {
			toasts.add('error', 'Choose a file to upload');
			return;
		}
		uploading = true;
		error = '';
		try {
			const form = new FormData();
			form.append('file', uploadFile);
			if (uploadTitle.trim()) {
				form.append('title', uploadTitle.trim());
			}
			const mime = uploadFile.type || '';
			if (mime) {
				form.append('mime', mime);
			}
			const res = await fetch('/api/collections/uploads/records', {
				method: 'POST',
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
				body: form
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				throw new Error(body.message || 'Failed to upload file');
			}
			toasts.add('success', 'File uploaded');
			uploadFile = null;
			uploadTitle = '';
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : 'Upload failed');
			error = 'Upload failed';
		} finally {
			uploading = false;
		}
	}

	function handleFileChange(event: Event) {
		const target = event.currentTarget as HTMLInputElement;
		uploadFile = target.files?.[0] ?? null;
		if (uploadFile && !uploadTitle) {
			uploadTitle = uploadFile.name;
		}
	}

	function resetAndLoad() {
		page = 1;
		loadMedia();
	}

	onMount(loadMedia);
</script>

<svelte:head>
	<title>Media Library | Facet Admin</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Media Library</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400">Browse and manage uploaded files across your profile.</p>
		</div>
		<button class="btn btn-secondary" on:click={loadMedia} aria-busy={loading}>
			{loading ? 'Loading...' : 'Refresh'}
		</button>
	</div>

	<div class="card p-4 mb-4 space-y-3">
		<div class="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-3">
			<div class="flex items-center gap-2">
				<input
					class="input"
					placeholder="Search filename..."
					bind:value={search}
					on:keydown={(e) => e.key === 'Enter' && resetAndLoad()}
				/>
				<button class="btn btn-primary" on:click={resetAndLoad}>Search</button>
			</div>
			<div class="flex items-center gap-2">
				<label class="label mb-0" for="type-filter">Type</label>
				<select id="type-filter" class="input" bind:value={typeFilter} on:change={resetAndLoad}>
					<option value="all">All</option>
					<option value="image">Images</option>
				</select>
			</div>
			<div class="flex items-center gap-2">
				<label class="label mb-0" for="status-filter">Scope</label>
				<select id="status-filter" class="input" bind:value={statusFilter} on:change={resetAndLoad}>
					<option value="referenced">Referenced only</option>
					<option value="all">Referenced + orphans</option>
					<option value="orphans">Orphans only</option>
				</select>
			</div>
			<div class="flex items-center justify-end gap-3 text-sm text-gray-600 dark:text-gray-400 flex-wrap">
				<span>{stats.totalFiles} files • {humanSize(stats.totalSize)}</span>
				<span class="inline-flex items-center gap-1 px-2 py-1 rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">
					{stats.orphanFiles} orphan{stats.orphanFiles === 1 ? '' : 's'}
				</span>
				{#if totalPages > 1}
					<span>Page {page} of {totalPages}</span>
				{/if}
			</div>
		</div>
		<div class="flex flex-col gap-2 text-sm text-gray-600 dark:text-gray-400">
			<div class="flex flex-wrap gap-3">
				<span>Storage: {humanSize(stats.storageSize)} ({stats.storageFiles} files)</span>
				<span>Referenced: {humanSize(stats.referencedSize)}</span>
				<span>Orphans: {humanSize(stats.orphanSize)} ({stats.orphanFiles})</span>
			</div>
			<div class="flex flex-wrap gap-2 items-center">
				{#if selectedOrphans.size > 0}
					<span class="text-gray-700 dark:text-gray-200">{selectedOrphans.size} orphan{selectedOrphans.size === 1 ? '' : 's'} selected</span>
					<button
						class="btn btn-secondary text-red-600 border-red-200 dark:border-red-800 hover:bg-red-50 dark:hover:bg-red-900/30"
						on:click={bulkDeleteSelected}
					>
						Delete selected
					</button>
					<button class="btn btn-ghost btn-sm" on:click={clearSelection}>Clear selection</button>
				{:else if statusFilter !== 'referenced'}
					<button class="btn btn-secondary btn-sm" on:click={selectVisibleOrphans}>
						Select all visible orphans
					</button>
				{/if}
			</div>
		</div>
	</div>

	{#if error}
		<div class="mb-4 p-3 rounded bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-200 text-sm">
			{error}
		</div>
	{/if}

	<div class="card p-4 mb-4 space-y-3">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Upload media</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Add files directly to the media library (images, docs, video up to 20MB).
				</p>
			</div>
			<button class="btn btn-primary" on:click={uploadMedia} aria-busy={uploading}>
				{uploading ? 'Uploading…' : 'Upload'}
			</button>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
			<div class="md:col-span-2">
				<label class="label" for="upload-file">File *</label>
				<input
					id="upload-file"
					type="file"
					class="input"
					on:change={handleFileChange}
				/>
			</div>
			<div>
				<label class="label" for="upload-title">Title (optional)</label>
				<input
					id="upload-title"
					class="input"
					placeholder="Display name"
					bind:value={uploadTitle}
				/>
			</div>
		</div>
	</div>

	<div class="card p-4 mb-4 space-y-3">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Add external media</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">Link to remote assets (YouTube, Vimeo, images, etc.).</p>
			</div>
			<button class="btn btn-primary" on:click={createExternal} aria-busy={newExternal.saving}>
				{newExternal.saving ? 'Saving…' : 'Add'}
			</button>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
			<div class="md:col-span-2">
				<label class="label" for="ext-url">URL *</label>
				<input id="ext-url" class="input" bind:value={newExternal.url} placeholder="https://…" />
			</div>
			<div>
				<label class="label" for="ext-title">Title</label>
				<input id="ext-title" class="input" bind:value={newExternal.title} placeholder="Optional display name" />
			</div>
			<div>
				<label class="label" for="ext-mime">MIME (optional)</label>
				<input id="ext-mime" class="input" bind:value={newExternal.mime} placeholder="image/png, video/mp4…" />
			</div>
			<div class="md:col-span-2">
				<label class="label" for="ext-thumb">Thumbnail URL (optional)</label>
				<input id="ext-thumb" class="input" bind:value={newExternal.thumbnail_url} placeholder="https://example.com/thumb.jpg" />
			</div>
		</div>
	</div>

	{#if loading}
		<div class="card p-6 text-gray-500 dark:text-gray-400">Loading media...</div>
	{:else if items.length === 0}
		<div class="card p-6 text-gray-500 dark:text-gray-400">No media found.</div>
	{:else}
		<div class="card overflow-hidden">
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
						<thead class="bg-gray-50 dark:bg-gray-800">
							<tr>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase w-8"></th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">File</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">Type</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">Size</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">Collection</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">Record</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">Uploaded</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
							{#each items as item}
								<tr class="hover:bg-gray-50 dark:hover:bg-gray-800">
									<td class="px-4 py-3">
										{#if item.orphan && item.relative_path}
											<input
												type="checkbox"
												class="w-4 h-4 text-primary-600 rounded border-gray-300"
												checked={selectedOrphans.has(item.relative_path)}
												on:change={() => toggleSelection(item)}
											/>
										{/if}
									</td>
									<td class="px-4 py-3">
										<div class="flex items-center gap-2">
											{@html icon(item.mime.startsWith('image/') ? 'image' : 'document')}
											<a
												class="text-primary-600 dark:text-primary-300 hover:underline break-all"
												href={item.url}
											target="_blank"
											rel="noopener noreferrer"
										>
											{item.filename}
										</a>
											{#if item.orphan}
												<span class="inline-flex items-center gap-1 px-2 py-1 text-xs rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">
													Orphan
												</span>
											{:else if item.external}
												<span class="inline-flex items-center gap-1 px-2 py-1 text-xs rounded bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200">
													External
												</span>
											{/if}
										</div>
									</td>
								<td class="px-4 py-3">
									<span class="inline-flex items-center gap-1 px-2 py-1 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200">
										{mimeLabel(item.mime)}
									</span>
								</td>
								<td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{humanSize(item.size)}</td>
								<td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{item.collection}</td>
								<td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
									{#if item.field}
										<code class="bg-gray-100 dark:bg-gray-800 px-1 rounded">{item.field}</code>
									{:else}
										<span class="text-gray-500 dark:text-gray-400">—</span>
									{/if}
									{#if item.record_id}
										<span class="text-gray-500 dark:text-gray-400 ml-1">({item.record_id})</span>
									{/if}
								</td>
								<td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
									{formatDate(item.uploaded_at, { month: 'short', day: 'numeric', year: 'numeric' })}
								</td>
								<td class="px-4 py-3">
									<div class="flex items-center gap-2">
										<button class="btn btn-ghost btn-sm" on:click={() => copyUrl(item.url)}>
											{@html icon('copy')}
										</button>
										<button class="btn btn-ghost btn-sm text-red-600" on:click={() => deleteFile(item)}>
											{@html icon('trash')}
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			{#if totalPages > 1}
				<div class="flex items-center justify-between p-4 border-t border-gray-200 dark:border-gray-700 text-sm text-gray-600 dark:text-gray-300">
					<div>Page {page} of {totalPages}</div>
					<div class="flex items-center gap-2">
						<button class="btn btn-ghost btn-sm" on:click={() => { page = Math.max(1, page - 1); loadMedia(); }} disabled={page === 1}>
							Previous
						</button>
						<button class="btn btn-ghost btn-sm" on:click={() => { if (page < totalPages) { page += 1; loadMedia(); } }} disabled={page >= totalPages}>
							Next
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import { formatDate } from '$lib/utils';
	import { pb } from '$lib/pocketbase';
	import { goto } from '$app/navigation';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import { t } from 'svelte-i18n';

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

	let items: MediaItem[] = $state([]);
	let loading = $state(true);
	let page = $state(1);
	let perPage = 50;
	let totalItems = 0;
	let totalPages = $state(1);
	let search = $state('');
	let typeFilter: 'all' | 'image' = $state('all');
	let statusFilter: 'referenced' | 'all' | 'orphans' = $state('referenced');
	let error = $state('');
	let stats: MediaStats = $state({
		referencedFiles: 0,
		referencedSize: 0,
		orphanFiles: 0,
		orphanSize: 0,
		totalFiles: 0,
		totalSize: 0,
		storageFiles: 0,
		storageSize: 0
	});
	let selectedOrphans: Set<string> = $state(new Set());
	let newExternal = $state({
		url: '',
		title: '',
		mime: '',
		thumbnail_url: '',
		saving: false
	});
	let uploadFile: File | null = null;
	let uploadTitle = $state('');
	let uploading = $state(false);

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
		if (!mime) return $t('admin.media.type_link');
		if (mime.startsWith('image/')) return $t('admin.media.type_image');
		if (mime.startsWith('video/')) return $t('admin.media.type_video');
		if (mime.startsWith('audio/')) return $t('admin.media.type_audio');
		return $t('admin.media.type_file');
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
					toasts.add('error', $t('admin.media.toast_session_expired'));
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
			error = $t('admin.media.toast_load_failed');
			toasts.add('error', $t('admin.media.toast_load_failed'));
		} finally {
			loading = false;
		}
	}

	function copyUrl(url: string) {
		const absolute = typeof window !== 'undefined' ? new URL(url, window.location.origin).toString() : url;
		navigator.clipboard.writeText(absolute);
		toasts.add('success', $t('admin.media.toast_url_copied'));
	}

	async function deleteFile(item: MediaItem) {
		const confirmed = await confirm({
			title: $t('admin.media.confirm_delete_title'),
			message: $t('admin.media.confirm_delete_message', { values: { filename: item.filename } }),
			confirmText: $t('admin.media.confirm_delete_button'),
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
				toasts.add('success', $t('admin.media.toast_external_deleted'));
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
					toasts.add('error', $t('admin.media.toast_session_expired'));
					goto('/admin/login');
					return;
				}
				const body = await res.json().catch(() => ({}));
				throw new Error(body.error || 'Failed to delete file');
			}
			toasts.add('success', $t('admin.media.toast_file_deleted'));
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.media.toast_delete_failed'));
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
			title: $t('admin.media.confirm_bulk_title'),
			message: $t('admin.media.confirm_bulk_message', { values: { count: selectedOrphans.size } }),
			confirmText: $t('admin.media.confirm_bulk_button'),
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
					toasts.add('error', $t('admin.media.toast_session_expired'));
					goto('/admin/login');
					return;
				}
				const body = await res.json().catch(() => ({}));
				throw new Error(body.error || 'Failed to delete files');
			}
			const result = await res.json().catch(() => ({}));
			toasts.add('success', $t('admin.media.toast_bulk_deleted', { values: { count: result.deleted ?? selectedOrphans.size } }));
			selectedOrphans = new Set();
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : 'Failed to delete files');
		}
	}

	async function createExternal() {
		if (!newExternal.url.trim()) {
			toasts.add('error', $t('admin.media.toast_url_required'));
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
			toasts.add('success', $t('admin.media.toast_external_added'));
			newExternal = { url: '', title: '', mime: '', thumbnail_url: '', saving: false };
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.media.toast_external_failed'));
		} finally {
			newExternal.saving = false;
		}
	}

	async function uploadMedia() {
		if (!uploadFile) {
			toasts.add('error', $t('admin.media.toast_choose_file'));
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
			toasts.add('success', $t('admin.media.toast_upload_success'));
			uploadFile = null;
			uploadTitle = '';
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.media.toast_upload_failed'));
			error = $t('admin.media.toast_upload_failed');
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
	<title>{$t('admin.media.page_title')}</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
	<PageHelp pageKey="media">
		<p>{@html $t('admin.media.help_intro')}</p>
		<p>{$t('admin.media.help_description')}</p>
		<p>{@html $t('admin.media.help_tip')}</p>
	</PageHelp>

	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.media.title')}</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.media.subtitle')}</p>
		</div>
		<button class="btn btn-secondary" onclick={loadMedia} aria-busy={loading}>
			{loading ? $t('admin.media.loading') : $t('admin.media.refresh')}
		</button>
	</div>

	<div class="card p-4 mb-4 space-y-3">
		<div class="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-3">
			<div class="flex items-center gap-2">
				<input
					class="input"
					placeholder={$t('admin.media.search_placeholder')}
					bind:value={search}
					onkeydown={(e) => e.key === 'Enter' && resetAndLoad()}
				/>
				<button class="btn btn-primary" onclick={resetAndLoad}>{$t('admin.media.search')}</button>
			</div>
			<div class="flex items-center gap-2">
				<label class="label mb-0" for="type-filter">{$t('admin.media.filter_type')}</label>
				<select id="type-filter" class="input" bind:value={typeFilter} onchange={resetAndLoad}>
					<option value="all">{$t('admin.media.filter_type_all')}</option>
					<option value="image">{$t('admin.media.filter_type_images')}</option>
				</select>
			</div>
			<div class="flex items-center gap-2">
				<label class="label mb-0" for="status-filter">{$t('admin.media.filter_scope')}</label>
				<select id="status-filter" class="input" bind:value={statusFilter} onchange={resetAndLoad}>
					<option value="referenced">{$t('admin.media.filter_scope_referenced')}</option>
					<option value="all">{$t('admin.media.filter_scope_all')}</option>
					<option value="orphans">{$t('admin.media.filter_scope_orphans')}</option>
				</select>
			</div>
			<div class="flex items-center justify-end gap-3 text-sm text-gray-600 dark:text-gray-400 flex-wrap">
				<span>{$t('admin.media.stats_files', { values: { count: stats.totalFiles } })} • {humanSize(stats.totalSize)}</span>
				<span class="inline-flex items-center gap-1 px-2 py-1 rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">
					{stats.orphanFiles === 1 ? $t('admin.media.stats_orphan', { values: { count: stats.orphanFiles } }) : $t('admin.media.stats_orphans', { values: { count: stats.orphanFiles } })}
				</span>
				{#if totalPages > 1}
					<span>{$t('admin.media.stats_page', { values: { page, total: totalPages } })}</span>
				{/if}
			</div>
		</div>
		<div class="flex flex-col gap-2 text-sm text-gray-600 dark:text-gray-400">
			<div class="flex flex-wrap gap-3">
				<span>{$t('admin.media.stats_storage', { values: { size: humanSize(stats.storageSize), count: stats.storageFiles } })}</span>
				<span>{$t('admin.media.stats_referenced', { values: { size: humanSize(stats.referencedSize) } })}</span>
				<span>{$t('admin.media.stats_orphan_size', { values: { size: humanSize(stats.orphanSize), count: stats.orphanFiles } })}</span>
			</div>
			<div class="flex flex-wrap gap-2 items-center">
				{#if selectedOrphans.size > 0}
					<span class="text-gray-700 dark:text-gray-200">{$t('admin.media.selected_orphans', { values: { count: selectedOrphans.size } })}</span>
					<button
						class="btn btn-danger"
						onclick={bulkDeleteSelected}
					>
						{$t('admin.media.delete_selected')}
					</button>
					<button class="btn btn-ghost btn-sm" onclick={clearSelection}>{$t('admin.media.clear_selection')}</button>
				{:else if statusFilter !== 'referenced'}
					<button class="btn btn-secondary btn-sm" onclick={selectVisibleOrphans}>
						{$t('admin.media.select_visible_orphans')}
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
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.media.upload_title')}</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					{$t('admin.media.upload_description')}
				</p>
			</div>
			<button class="btn btn-primary" onclick={uploadMedia} aria-busy={uploading}>
				{uploading ? $t('admin.media.uploading') : $t('admin.media.upload_button')}
			</button>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
			<div class="md:col-span-2">
				<label class="label" for="upload-file">{$t('admin.media.file_required')}</label>
				<input
					id="upload-file"
					type="file"
					class="input"
					onchange={handleFileChange}
				/>
			</div>
			<div>
				<label class="label" for="upload-title">{$t('admin.media.title_label')}</label>
				<input
					id="upload-title"
					class="input"
					placeholder={$t('admin.media.title_placeholder')}
					bind:value={uploadTitle}
				/>
			</div>
		</div>
	</div>

	<div class="card p-4 mb-4 space-y-3">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.media.external_title')}</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.media.external_description')}</p>
			</div>
			<button class="btn btn-primary" onclick={createExternal} aria-busy={newExternal.saving}>
				{newExternal.saving ? $t('admin.media.saving') : $t('admin.media.add_button')}
			</button>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
			<div class="md:col-span-2">
				<label class="label" for="ext-url">{$t('admin.media.url_required')}</label>
				<input id="ext-url" class="input" bind:value={newExternal.url} placeholder={$t('admin.media.url_placeholder')} />
			</div>
			<div>
				<label class="label" for="ext-title">{$t('admin.media.external_title_label')}</label>
				<input id="ext-title" class="input" bind:value={newExternal.title} placeholder={$t('admin.media.external_title_placeholder')} />
			</div>
			<div>
				<label class="label" for="ext-mime">{$t('admin.media.mime_label')}</label>
				<input id="ext-mime" class="input" bind:value={newExternal.mime} placeholder={$t('admin.media.mime_placeholder')} />
			</div>
			<div class="md:col-span-2">
				<label class="label" for="ext-thumb">{$t('admin.media.thumbnail_label')}</label>
				<input id="ext-thumb" class="input" bind:value={newExternal.thumbnail_url} placeholder={$t('admin.media.thumbnail_placeholder')} />
			</div>
		</div>
	</div>

	{#if loading}
		<div class="card p-6 text-gray-500 dark:text-gray-400">{$t('admin.media.loading_media')}</div>
	{:else if items.length === 0}
		<div class="card p-6 text-gray-500 dark:text-gray-400">{$t('admin.media.no_media')}</div>
	{:else}
		<div class="card overflow-hidden">
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700 table-fixed">
						<thead class="bg-gray-50 dark:bg-gray-800">
							<tr>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase w-8"></th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase w-1/4">{$t('admin.media.table_file')}</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_type')}</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_size')}</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_collection')}</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_record')}</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_uploaded')}</th>
								<th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_actions')}</th>
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
												onchange={() => toggleSelection(item)}
											/>
										{/if}
									</td>
									<td class="px-4 py-3 max-w-xs">
										<div class="flex items-center gap-2">
											<span class="shrink-0">{@html icon(item.mime.startsWith('image/') ? 'image' : 'document')}</span>
											<div class="min-w-0 flex-1 overflow-hidden">
												<a
													class="text-primary-600 dark:text-primary-300 hover:underline break-words line-clamp-2"
													href={item.url}
													target="_blank"
													rel="noopener noreferrer"
													title={item.display_name || item.filename}
												>
													{item.display_name || item.filename}
												</a>
												{#if item.display_name && item.display_name !== item.filename}
													<span class="text-xs text-gray-500 dark:text-gray-400 break-words line-clamp-1">{item.filename}</span>
												{/if}
											</div>
											{#if item.orphan}
												<span class="inline-flex items-center gap-1 px-2 py-1 text-xs rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200 shrink-0">
													{$t('admin.media.badge_orphan')}
												</span>
											{:else if item.external}
												<span class="inline-flex items-center gap-1 px-2 py-1 text-xs rounded bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200 shrink-0">
													{$t('admin.media.badge_external')}
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
										<button class="btn btn-ghost btn-sm" onclick={() => copyUrl(item.url)}>
											{@html icon('copy')}
										</button>
										<button class="btn btn-danger-ghost btn-sm" onclick={() => deleteFile(item)}>
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
					<div>{$t('admin.media.pagination_page', { values: { page, total: totalPages } })}</div>
					<div class="flex items-center gap-2">
						<button class="btn btn-ghost btn-sm" onclick={() => { page = Math.max(1, page - 1); loadMedia(); }} disabled={page === 1}>
							{$t('admin.media.pagination_previous')}
						</button>
						<button class="btn btn-ghost btn-sm" onclick={() => { if (page < totalPages) { page += 1; loadMedia(); } }} disabled={page >= totalPages}>
							{$t('admin.media.pagination_next')}
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import { formatDate } from '$lib/utils';
	import { pb } from '$lib/pocketbase';
	import { goto } from '$app/navigation';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import DropZone from '$lib/components/admin/DropZone.svelte';
	import MediaPreviewModal from '$lib/components/admin/MediaPreviewModal.svelte';
	import { t } from 'svelte-i18n';

	type MediaUsageItem = {
		collection: string;
		record_id: string;
		title: string;
		slug?: string;
	};

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
		provider?: string;
		embed_url?: string;
		usage_count?: number;
		used_by?: MediaUsageItem[];
	};

	// Get provider icon name for external media
	function getProviderIcon(item: MediaItem): string {
		if (!item.external) {
			if (item.mime?.startsWith('image/')) return 'image';
			if (item.mime?.startsWith('video/')) return 'video';
			if (item.mime?.startsWith('audio/')) return 'music';
			return 'file';
		}
		const provider = (item.provider || '').toLowerCase();
		if (provider === 'youtube') return 'youtube';
		if (provider === 'vimeo') return 'vimeo';
		if (provider === 'spotify') return 'spotify';
		if (provider === 'soundcloud') return 'soundcloud';
		if (item.mime?.startsWith('video/')) return 'video';
		if (item.mime?.startsWith('audio/')) return 'music';
		if (item.mime?.startsWith('image/')) return 'image';
		return 'globe';
	}

	// Check if item has a displayable thumbnail
	function hasThumbnail(item: MediaItem): boolean {
		if (item.thumbnail_url) return true;
		if (item.mime?.startsWith('image/') && item.url) return true;
		return false;
	}

	// Get thumbnail URL for an item
	function getThumbnailUrl(item: MediaItem): string {
		if (item.thumbnail_url) return item.thumbnail_url;
		if (item.mime?.startsWith('image/') && item.url) return item.url;
		return '';
	}

	// Check if item is a video type
	function isVideo(item: MediaItem): boolean {
		if (item.mime?.startsWith('video/')) return true;
		const provider = (item.provider || '').toLowerCase();
		return ['youtube', 'vimeo', 'loom'].includes(provider);
	}

	// Check if item supports editing (all non-orphan media can have display names)
	function isEditable(item: MediaItem): boolean {
		return !item.orphan;
	}

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
	let perPage = $state(50);
	let totalItems = $state(0);
	let totalPages = $state(1);
	let search = $state('');
	let typeFilter: 'all' | 'image' | 'video' | 'audio' | 'document' = $state('all');
	let statusFilter: 'referenced' | 'all' | 'orphans' = $state('referenced');
	let sortField: 'date' | 'name' | 'size' = $state('date');
	let sortOrder: 'asc' | 'desc' = $state('desc');
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
	let uploadFile: File | null = $state(null);
	let uploadTitle = $state('');
	let uploading = $state(false);
	let uploadProgress = $state({ current: 0, total: 0 });

	type FileProgress = { name: string; loaded: number; total: number; status: 'pending' | 'uploading' | 'done' | 'error' };
	let fileProgresses = $state<FileProgress[]>([]);

	let externalPreviewFailed = $state(false);
	let previewItem: MediaItem | null = $state(null);
	let editItem: MediaItem | null = $state(null);
	let editModalMouseDownTarget: EventTarget | null = $state(null);
	let editForm = $state({
		title: '',
		url: '',
		mime: '',
		thumbnail_url: '',
		saving: false
	});

	function openPreview(item: MediaItem) {
		previewItem = item;
	}

	function closePreview() {
		previewItem = null;
	}

	function openEdit(item: MediaItem) {
		editItem = item;
		editForm = {
			title: item.display_name || item.filename || '',
			url: item.url || '',
			mime: item.mime || '',
			thumbnail_url: item.thumbnail_url || '',
			saving: false
		};
	}

	function closeEdit() {
		editItem = null;
		editForm = { title: '', url: '', mime: '', thumbnail_url: '', saving: false };
	}

	async function saveEdit() {
		if (!editItem || !editItem.record_id) return;
		editForm.saving = true;
		try {
			const isExternal = editItem.external || editItem.collection === 'external_media';
			const isUploads = editItem.collection === 'uploads';

			if (isExternal) {
				// External media: update the external_media record directly
				const body = {
					title: editForm.title.trim(),
					url: editForm.url.trim(),
					mime: editForm.mime.trim(),
					thumbnail_url: editForm.thumbnail_url.trim()
				};
				const res = await fetch(`/api/collections/external_media/records/${editItem.record_id}`, {
					method: 'PATCH',
					headers: {
						'Content-Type': 'application/json',
						...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
					},
					body: JSON.stringify(body)
				});
				if (!res.ok) {
					const respBody = await res.json().catch(() => ({}));
					throw new Error(respBody.message || respBody.error || 'Failed to update media');
				}
			} else if (isUploads) {
				// Uploads collection: update the record's title field
				const res = await fetch(`/api/collections/uploads/records/${editItem.record_id}`, {
					method: 'PATCH',
					headers: {
						'Content-Type': 'application/json',
						...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
					},
					body: JSON.stringify({ title: editForm.title.trim() })
				});
				if (!res.ok) {
					const respBody = await res.json().catch(() => ({}));
					throw new Error(respBody.message || respBody.error || 'Failed to update media');
				}
			} else {
				// Other collections: use display-name endpoint to set custom name
				const res = await fetch('/api/media/display-name', {
					method: 'PATCH',
					headers: {
						'Content-Type': 'application/json',
						...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
					},
					body: JSON.stringify({
						collection_id: editItem.collection_id,
						record_id: editItem.record_id,
						filename: editItem.filename,
						display_name: editForm.title.trim()
					})
				});
				if (!res.ok) {
					const respBody = await res.json().catch(() => ({}));
					throw new Error(respBody.message || respBody.error || 'Failed to update display name');
				}
			}

			toasts.add('success', $t('admin.media.toast_media_updated'));
			closeEdit();
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.media.toast_update_failed'));
		} finally {
			editForm.saving = false;
		}
	}

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
			if (typeFilter !== 'all') params.set('type', typeFilter);
			if (statusFilter === 'orphans') {
				params.set('orphans', '1');
			} else if (statusFilter === 'all') {
				params.set('includeOrphans', '1');
			}
			params.set('sort', sortField);
			params.set('order', sortOrder);

			const res = await fetch(`/api/media?${params.toString()}`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
				cache: 'no-store' // Prevent browser caching
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

	async function deleteFile(item: MediaItem, force = false) {
		// Check if item is external media (either by flag or by collection name)
		const isExternal = item.external || item.collection === 'external_media';

		// For non-external or orphan items, show standard confirmation
		if (!isExternal || item.orphan) {
			const confirmed = await confirm({
				title: $t('admin.media.confirm_delete_title'),
				message: $t('admin.media.confirm_delete_message', { values: { filename: item.filename } }),
				confirmText: $t('admin.media.confirm_delete_button'),
				danger: true
			});
			if (!confirmed) return;
		} else if (!force) {
			// For external items without force flag, show standard confirmation first
			const confirmed = await confirm({
				title: $t('admin.media.confirm_delete_title'),
				message: $t('admin.media.confirm_delete_message', { values: { filename: item.filename } }),
				confirmText: $t('admin.media.confirm_delete_button'),
				danger: true
			});
			if (!confirmed) return;
		}

		try {
			if (isExternal && item.record_id) {
				const url = force
					? `/api/media/external/${item.record_id}?force=1`
					: `/api/media/external/${item.record_id}`;
				const res = await fetch(url, {
					method: 'DELETE',
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				});

				// Handle conflict response (media is referenced)
				if (res.status === 409) {
					const body = await res.json().catch(() => ({}));
					const usedBy = body.used_by || [];
					const usageCount = body.usage_count || usedBy.length;

					// Build list of content items using this media
					const usageList = usedBy
						.slice(0, 5)
						.map((u: { collection: string; title: string }) => `• ${u.collection}: ${u.title}`)
						.join('\n');
					const moreText = usedBy.length > 5 ? `\n...and ${usedBy.length - 5} more` : '';

					const forceConfirmed = await confirm({
						title: $t('admin.media.confirm_force_delete_title'),
						message: $t('admin.media.confirm_force_delete_message', { values: { count: usageCount } }) +
							'\n\n' + usageList + moreText,
						confirmText: $t('admin.media.confirm_force_delete_button'),
						danger: true
					});

					if (forceConfirmed) {
						// Retry with force flag
						await deleteFile(item, true);
					}
					return;
				}

				if (!res.ok) {
					const body = await res.json().catch(() => ({}));
					throw new Error(body.message || body.error || 'Failed to delete external media');
				}

				const result = await res.json().catch(() => ({}));
				if (result.references_removed) {
					toasts.add('success', $t('admin.media.toast_references_removed', { values: { count: result.references_removed } }));
				} else {
					toasts.add('success', $t('admin.media.toast_external_deleted'));
				}
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
				const respBody = await res.json().catch(() => ({}));
				throw new Error(respBody.message || respBody.error || 'Failed to delete file');
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
				const respBody = await res.json().catch(() => ({}));
				throw new Error(respBody.message || respBody.error || 'Failed to delete files');
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
				const respBody = await res.json().catch(() => ({}));
				throw new Error(respBody.message || respBody.error || 'Failed to add external media');
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

	/**
	 * Upload a single file with progress tracking using XMLHttpRequest
	 */
	function uploadFileWithProgress(
		file: File,
		onProgress: (loaded: number, total: number) => void
	): Promise<boolean> {
		return new Promise((resolve) => {
			const xhr = new XMLHttpRequest();
			const form = new FormData();
			form.append('file', file);
			form.append('title', file.name);
			if (file.type) form.append('mime', file.type);

			xhr.upload.onprogress = (e) => {
				if (e.lengthComputable) onProgress(e.loaded, e.total);
			};
			xhr.onload = () => resolve(xhr.status >= 200 && xhr.status < 300);
			xhr.onerror = () => resolve(false);

			xhr.open('POST', '/api/collections/uploads/records');
			if (pb.authStore.isValid) xhr.setRequestHeader('Authorization', `Bearer ${pb.authStore.token}`);
			xhr.send(form);
		});
	}

	async function handleDropZoneFiles(event: CustomEvent<{ files: File[] }>) {
		const files = event.detail.files;
		if (files.length === 0) return;

		uploading = true;
		uploadProgress = { current: 0, total: files.length };
		error = '';

		// Initialize progress tracking for all files
		fileProgresses = files.map((f) => ({
			name: f.name,
			loaded: 0,
			total: f.size,
			status: 'pending' as const
		}));

		let successCount = 0;
		let failCount = 0;

		for (let i = 0; i < files.length; i++) {
			const file = files[i];
			uploadProgress.current = i + 1;

			// Mark file as uploading
			fileProgresses[i] = { ...fileProgresses[i], status: 'uploading' };

			const success = await uploadFileWithProgress(file, (loaded, total) => {
				fileProgresses[i] = { ...fileProgresses[i], loaded, total };
			});

			if (success) {
				successCount++;
				fileProgresses[i] = { ...fileProgresses[i], loaded: fileProgresses[i].total, status: 'done' };
			} else {
				failCount++;
				fileProgresses[i] = { ...fileProgresses[i], status: 'error' };
				console.error('Failed to upload:', file.name);
			}
		}

		uploading = false;
		uploadProgress = { current: 0, total: 0 };

		if (successCount > 0) {
			toasts.add('success', $t('admin.media.upload_complete', { values: { count: successCount } }));
		}
		if (failCount > 0) {
			toasts.add('error', $t('admin.media.toast_upload_failed') + ` (${failCount})`);
		}

		// Clear progress after a short delay to show completion
		setTimeout(() => {
			fileProgresses = [];
		}, 2000);

		await loadMedia();
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
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
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
					<option value="video">{$t('admin.media.filter_type_video')}</option>
					<option value="audio">{$t('admin.media.filter_type_audio')}</option>
					<option value="document">{$t('admin.media.filter_type_document')}</option>
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
			<div class="flex items-center gap-2">
				<label class="label mb-0" for="sort-field">{$t('admin.media.sort_by')}</label>
				<select id="sort-field" class="input" bind:value={sortField} onchange={resetAndLoad}>
					<option value="date">{$t('admin.media.sort_date')}</option>
					<option value="name">{$t('admin.media.sort_name')}</option>
					<option value="size">{$t('admin.media.sort_size')}</option>
				</select>
				<button
					class="btn btn-ghost btn-sm px-2"
					onclick={() => { sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; resetAndLoad(); }}
					title={sortOrder === 'asc' ? $t('admin.media.sort_ascending') : $t('admin.media.sort_descending')}
				>
					{sortOrder === 'asc' ? '↑' : '↓'}
				</button>
			</div>
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
			{#if uploadProgress.total > 0}
				<span class="text-sm text-gray-600 dark:text-gray-400">
					{$t('admin.media.upload_progress', { values: { current: uploadProgress.current, total: uploadProgress.total } })}
				</span>
			{/if}
		</div>
		{#if fileProgresses.length > 0}
			<div class="space-y-2">
				{#each fileProgresses as fp}
					<div class="flex items-center gap-2">
						<span class="text-sm truncate w-32 text-gray-700 dark:text-gray-300" title={fp.name}>{fp.name}</span>
						<div class="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded overflow-hidden">
							<div
								class="h-full transition-all duration-200 {fp.status === 'error' ? 'bg-red-500' : fp.status === 'done' ? 'bg-green-500' : 'bg-primary-500'}"
								style="width: {fp.total ? Math.round(fp.loaded / fp.total * 100) : 0}%"
							></div>
						</div>
						<span class="text-xs w-12 text-right text-gray-500 dark:text-gray-400">
							{#if fp.status === 'done'}
								✓
							{:else if fp.status === 'error'}
								✗
							{:else if fp.status === 'pending'}
								—
							{:else}
								{fp.total ? Math.round(fp.loaded / fp.total * 100) : 0}%
							{/if}
						</span>
					</div>
				{/each}
			</div>
		{/if}
		<DropZone
			on:drop={handleDropZoneFiles}
			disabled={uploading}
			multiple={true}
		/>
		<div class="text-xs text-gray-500 dark:text-gray-400 text-center">
			{$t('admin.media.upload_description')}
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
				<input id="ext-url" class="input" bind:value={newExternal.url} placeholder={$t('admin.media.url_placeholder')} oninput={() => { externalPreviewFailed = false; }} />
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
			{#if newExternal.url.trim()}
				{@const previewMime = newExternal.mime.trim()}
				{@const previewUrl = newExternal.url.trim()}
				{@const thumbUrl = newExternal.thumbnail_url.trim()}
				{@const isVideo = previewMime.startsWith('video/') || /youtube|vimeo|loom/i.test(previewUrl)}
				<div class="lg:col-span-4">
					{#key previewUrl}
						<div class="p-4 bg-gray-100 dark:bg-gray-800 rounded-lg">
							{#if isVideo}
								{#if thumbUrl}
									<img src={thumbUrl} alt="Thumbnail" class="max-h-48 mx-auto rounded" />
								{:else}
									<p class="text-sm text-gray-500 dark:text-gray-400 text-center">{$t('admin.media.video_preview_note')}</p>
								{/if}
							{:else}
								<!-- Try to load as image; show fallback on error -->
								{#if !externalPreviewFailed}
									<img
										src={previewUrl}
										alt="Preview"
										class="max-h-48 mx-auto rounded"
										onerror={() => { externalPreviewFailed = true; }}
									/>
								{:else}
									<div class="flex items-center justify-center h-24">
										<a href={previewUrl} target="_blank" rel="noopener noreferrer" class="btn btn-secondary btn-sm">
											{$t('admin.media.open_link')}
										</a>
									</div>
								{/if}
							{/if}
						</div>
					{/key}
				</div>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="card p-6 text-gray-500 dark:text-gray-400">{$t('admin.media.loading_media')}</div>
	{:else if items.length === 0}
		<div class="card p-6 text-gray-500 dark:text-gray-400">{$t('admin.media.no_media')}</div>
	{:else}
		<div class="card overflow-hidden">
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
						<thead class="bg-gray-50 dark:bg-gray-800">
							<tr>
								<th class="px-2 py-2 text-left text-xs font-semibold text-gray-500 uppercase w-8"></th>
								<th class="px-2 py-2 text-left text-xs font-semibold text-gray-500 uppercase w-14">{$t('admin.media.table_preview')}</th>
								<th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase">{$t('admin.media.table_file')}</th>
								<th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden md:table-cell w-20">{$t('admin.media.table_type')}</th>
								<th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden lg:table-cell w-20">{$t('admin.media.table_size')}</th>
								<th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden lg:table-cell w-24">{$t('admin.media.table_uploaded')}</th>
								<th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden md:table-cell w-24">{$t('admin.media.table_used_by')}</th>
								<th class="px-2 py-2 text-right text-xs font-semibold text-gray-500 uppercase w-28">{$t('admin.media.table_actions')}</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
							{#each items as item}
								<tr class="hover:bg-gray-50 dark:hover:bg-gray-800">
									<td class="px-2 py-2">
										{#if item.orphan && item.relative_path}
											<input
												type="checkbox"
												class="w-4 h-4 text-primary-600 rounded border-gray-300"
												checked={selectedOrphans.has(item.relative_path)}
												onchange={() => toggleSelection(item)}
											/>
										{/if}
									</td>
									<td class="px-2 py-2">
										<div class="w-10 h-10 relative rounded overflow-hidden bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
											{#if hasThumbnail(item)}
												<img
													src={getThumbnailUrl(item)}
													alt=""
													class="w-full h-full object-cover"
													loading="lazy"
													onerror={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
												/>
												{#if isVideo(item)}
													<div class="absolute inset-0 flex items-center justify-center bg-black/30">
														<span class="text-white">{@html icon('play')}</span>
													</div>
												{/if}
											{:else}
												<span class="text-gray-400 dark:text-gray-500">
													{@html icon(getProviderIcon(item) as keyof typeof import('$lib/icons').icons)}
												</span>
											{/if}
										</div>
									</td>
									<td class="px-3 py-2">
										<div class="flex items-center gap-2 min-w-0">
											<div class="min-w-0 flex-1 max-w-[200px] lg:max-w-[300px]">
												<a
													class="text-primary-600 dark:text-primary-300 hover:underline break-words line-clamp-2 text-sm"
													href={item.url}
													target="_blank"
													rel="noopener noreferrer"
													title={item.display_name || item.filename}
												>
													{item.display_name || item.filename}
												</a>
												{#if item.display_name && item.display_name !== item.filename}
													<span class="text-xs text-gray-500 dark:text-gray-400 truncate block">{item.filename}</span>
												{/if}
											</div>
											{#if item.orphan}
												<span class="inline-flex items-center px-1.5 py-0.5 text-xs rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200 shrink-0">
													{$t('admin.media.badge_orphan')}
												</span>
											{:else if item.external || item.collection === 'external_media'}
												<span class="inline-flex items-center px-1.5 py-0.5 text-xs rounded bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200 shrink-0">
													{$t('admin.media.badge_external')}
												</span>
											{/if}
										</div>
									</td>
								<td class="px-3 py-2 hidden md:table-cell">
									<span class="inline-flex items-center px-1.5 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200">
										{mimeLabel(item.mime)}
									</span>
								</td>
								<td class="px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hidden lg:table-cell">{humanSize(item.size)}</td>
								<td class="px-3 py-2 text-xs text-gray-700 dark:text-gray-200 hidden lg:table-cell">
									{formatDate(item.uploaded_at, { month: 'short', day: 'numeric' })}
								</td>
								<td class="px-3 py-2 text-sm hidden md:table-cell">
									{#if !item.orphan && item.usage_count && item.usage_count > 0}
										<div class="group relative">
											<span class="inline-flex items-center px-1.5 py-0.5 text-xs rounded bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-200 cursor-help">
												{$t('admin.media.used_by_count', { values: { count: item.usage_count } })}
											</span>
											{#if item.used_by && item.used_by.length > 0}
												<div class="hidden group-hover:block absolute z-10 right-0 top-full mt-1 p-2 bg-white dark:bg-gray-800 rounded shadow-lg border border-gray-200 dark:border-gray-700 min-w-[180px] max-w-[250px]">
													<ul class="text-xs text-gray-600 dark:text-gray-300 space-y-1">
														{#each item.used_by.slice(0, 5) as usage}
															<li class="truncate" title="{usage.collection}: {usage.title}">
																<span class="font-medium">{usage.collection}</span>: {usage.title}
															</li>
														{/each}
														{#if item.used_by.length > 5}
															<li class="text-gray-400 italic">+{item.used_by.length - 5} more</li>
														{/if}
													</ul>
												</div>
											{/if}
										</div>
									{:else if item.orphan}
										<span class="text-amber-600 dark:text-amber-400 text-xs">{$t('admin.media.badge_orphan')}</span>
									{:else}
										<span class="text-gray-400 dark:text-gray-500 text-xs">—</span>
									{/if}
								</td>
								<td class="px-2 py-2">
									<div class="flex items-center justify-end gap-1">
										<button class="btn btn-ghost btn-sm" onclick={() => openPreview(item)} title={$t('admin.media.preview_button')}>
											{@html icon('eye')}
										</button>
										{#if isEditable(item)}
											<button class="btn btn-ghost btn-sm" onclick={() => openEdit(item)} title={$t('admin.media.edit_button')}>
												{@html icon('edit')}
											</button>
										{/if}
										<button class="btn btn-ghost btn-sm" onclick={() => copyUrl(item.url)} title={$t('admin.media.copy_url_button')}>
											{@html icon('copy')}
										</button>
										<button class="btn btn-danger-ghost btn-sm" onclick={() => deleteFile(item)} title={$t('admin.media.delete_button')}>
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

<MediaPreviewModal item={previewItem} on:close={closePreview} />

{#if editItem}
	<div
		class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-fade-in"
		onmousedown={(e) => editModalMouseDownTarget = e.target}
		onclick={(e) => {
			if (e.target === e.currentTarget && editModalMouseDownTarget === e.currentTarget) {
				closeEdit();
			}
			editModalMouseDownTarget = null;
		}}
		onkeydown={(e) => e.key === 'Escape' && closeEdit()}
		role="dialog"
		aria-modal="true"
		aria-labelledby="edit-modal-title"
		tabindex="-1"
	>
		<div class="card bg-white dark:bg-gray-900 shadow-2xl max-w-lg w-full max-h-[90vh] overflow-y-auto animate-scale-in">
			<div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-600">
				<h2 id="edit-modal-title" class="text-lg font-semibold text-gray-900 dark:text-white">
					{$t('admin.media.edit_modal_title')}
				</h2>
				<button
					class="btn btn-ghost btn-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
					onclick={closeEdit}
					aria-label={$t('shared.actions.close')}
				>
					{@html icon('x')}
				</button>
			</div>
			
			<form class="p-4 space-y-4" onsubmit={(e) => { e.preventDefault(); saveEdit(); }}>
				{#if editItem.thumbnail_url || (editItem.mime?.startsWith('image/') && editItem.url)}
					<div class="flex justify-center">
						<div class="w-24 h-24 rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-600">
							<img
								src={editItem.thumbnail_url || editItem.url}
								alt=""
								class="w-full h-full object-cover"
								loading="lazy"
							/>
						</div>
					</div>
				{/if}
				<div>
					<label class="label" for="edit-title">{$t('admin.media.edit_title_label')}</label>
					<input
						id="edit-title"
						class="input transition-shadow focus:ring-2 focus:ring-primary-500/20"
						bind:value={editForm.title}
						placeholder={$t('admin.media.edit_title_placeholder')}
					/>
					<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.media.edit_title_help')}</p>
				</div>
				{#if editItem.external || editItem.collection === 'external_media'}
					<div>
						<label class="label" for="edit-url">{$t('admin.media.edit_url_label')}</label>
						<input
							id="edit-url"
							class="input transition-shadow focus:ring-2 focus:ring-primary-500/20"
							bind:value={editForm.url}
							placeholder={$t('admin.media.url_placeholder')}
						/>
					</div>
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label class="label" for="edit-mime">{$t('admin.media.edit_mime_label')}</label>
							<input
								id="edit-mime"
								class="input transition-shadow focus:ring-2 focus:ring-primary-500/20"
								bind:value={editForm.mime}
								placeholder={$t('admin.media.mime_placeholder')}
							/>
						</div>
						<div>
							<label class="label" for="edit-thumbnail">{$t('admin.media.edit_thumbnail_label')}</label>
							<input
								id="edit-thumbnail"
								class="input transition-shadow focus:ring-2 focus:ring-primary-500/20"
								bind:value={editForm.thumbnail_url}
								placeholder={$t('admin.media.thumbnail_placeholder')}
							/>
						</div>
					</div>
				{/if}
				<div class="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
					<button type="button" class="btn btn-secondary" onclick={closeEdit}>
						{$t('shared.actions.cancel')}
					</button>
					<button
						type="submit"
						class="btn btn-primary transition-transform active:scale-95"
						disabled={editForm.saving}
					>
						{#if editForm.saving}
							<span class="inline-flex items-center gap-2">
								<span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
								{$t('admin.media.saving')}
							</span>
						{:else}
							{$t('shared.actions.save')}
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

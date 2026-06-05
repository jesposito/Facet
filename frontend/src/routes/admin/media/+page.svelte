<script lang="ts">
	import { onMount } from 'svelte';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import {
		formatDate,
		detectExternalMediaProvider,
		getExternalMediaProviderLabel,
		getExternalMediaThumbnail
	} from '$lib/utils';
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

	type MediaTag = {
		id: string;
		name: string;
		color?: string;
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
		width?: number;
		height?: number;
		uploaded_at: string;
		relative_path?: string;
		orphan?: boolean;
		display_name?: string;
		alt_text?: string;
		description?: string;
		record_label?: string;
		thumbnail_url?: string;
		collection_key?: string;
		external?: boolean;
		provider?: string;
		embed_url?: string;
		usage_count?: number;
		used_by?: MediaUsageItem[];
		tags?: MediaTag[];
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
		if (provider === 'loom') return 'loom';
		if (provider === 'codepen') return 'codepen';
		if (provider === 'figma') return 'figma';
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
	let usageFilter: 'all' | 'in_use' | 'not_in_use' = $state('all');
	let collectionFilter = $state('');

	// Available collections for filtering (using i18n keys)
	const availableCollections = [
		{ value: '', labelKey: 'admin.media.filter_collection_all' },
		{ value: 'uploads', labelKey: 'admin.media.filter_collection_uploads' },
		{ value: 'external_media', labelKey: 'admin.media.filter_collection_external_media' },
		{ value: 'profile', labelKey: 'admin.media.filter_collection_profile' },
		{ value: 'experience', labelKey: 'admin.media.filter_collection_experience' },
		{ value: 'projects', labelKey: 'admin.media.filter_collection_projects' },
		{ value: 'education', labelKey: 'admin.media.filter_collection_education' },
		{ value: 'posts', labelKey: 'admin.media.filter_collection_posts' },
		{ value: 'talks', labelKey: 'admin.media.filter_collection_talks' },
		{ value: 'views', labelKey: 'admin.media.filter_collection_views' },
		{ value: 'site_settings', labelKey: 'admin.media.filter_collection_site_settings' },
		{ value: 'custom_content', labelKey: 'admin.media.filter_collection_custom_content' },
		{ value: 'testimonials', labelKey: 'admin.media.filter_collection_testimonials' },
		{ value: 'certifications', labelKey: 'admin.media.filter_collection_certifications' },
		{ value: 'media_library', labelKey: 'admin.media.filter_collection_media_library' },
		{ value: 'view_exports', labelKey: 'admin.media.filter_collection_view_exports' },
		{ value: 'resume_imports', labelKey: 'admin.media.filter_collection_resume_imports' },
	];
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
	// Track selected items for bulk operations (uploads and external_media)
	type SelectedItem = { id: string; collection: string };
	let selectedItems: SelectedItem[] = $state([]);
	let bulkTagModalOpen = $state(false);
	let bulkTagForm = $state({
		addTags: [] as string[],
		removeTags: [] as string[],
		saving: false
	});
	let newExternal = $state({
		url: '',
		title: '',
		mime: '',
		thumbnail_url: '',
		saving: false,
		fetching: false,
		urlTouched: false
	});

	// Detected provider for the currently-typed URL (null when empty or unsupported).
	// Validation only runs once the user has interacted (typed/blurred) so the form
	// doesn't show an error before they've started typing.
	let externalProvider = $derived.by(() => detectExternalMediaProvider(newExternal.url));
	let externalUrlInvalid = $derived.by(() =>
		newExternal.urlTouched && newExternal.url.trim() !== '' && externalProvider === null
	);
	let externalAutoThumbnail = $derived.by(() => getExternalMediaThumbnail(newExternal.url));
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
		alt_text: '',
		description: '',
		url: '',
		mime: '',
		thumbnail_url: '',
		tag_ids: [] as string[],
		saving: false
	});

	// Available admin tags for filtering and editing
	type AdminTag = { id: string; name: string; color?: string };
	let availableTags = $state<AdminTag[]>([]);
	let selectedTagFilter = $state('');

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
			alt_text: item.alt_text || '',
			description: item.description || '',
			url: item.url || '',
			mime: item.mime || '',
			thumbnail_url: item.thumbnail_url || '',
			tag_ids: item.tags?.map(t => t.id) || [],
			saving: false
		};
	}

	function closeEdit() {
		editItem = null;
		editForm = { title: '', alt_text: '', description: '', url: '', mime: '', thumbnail_url: '', tag_ids: [], saving: false };
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
					thumbnail_url: editForm.thumbnail_url.trim(),
					alt_text: editForm.alt_text.trim(),
					description: editForm.description.trim(),
					admin_tags: editForm.tag_ids
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
				// Uploads collection: update the record's title, metadata, and tags
				const res = await fetch(`/api/collections/uploads/records/${editItem.record_id}`, {
					method: 'PATCH',
					headers: {
						'Content-Type': 'application/json',
						...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
					},
					body: JSON.stringify({
						title: editForm.title.trim(),
						alt_text: editForm.alt_text.trim(),
						description: editForm.description.trim(),
						admin_tags: editForm.tag_ids
					})
				});
				if (!res.ok) {
					const respBody = await res.json().catch(() => ({}));
					throw new Error(respBody.message || respBody.error || 'Failed to update media');
				}
			} else {
				// Other collections: use metadata endpoint to set all metadata
				const res = await fetch('/api/media/metadata', {
					method: 'PATCH',
					headers: {
						'Content-Type': 'application/json',
						...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
					},
					body: JSON.stringify({
						collection_id: editItem.collection_id,
						record_id: editItem.record_id,
						filename: editItem.filename,
						display_name: editForm.title.trim(),
						alt_text: editForm.alt_text.trim(),
						description: editForm.description.trim(),
						tag_ids: editForm.tag_ids
					})
				});
				if (!res.ok) {
					const respBody = await res.json().catch(() => ({}));
					throw new Error(respBody.message || respBody.error || 'Failed to update metadata');
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

	async function loadTags() {
		try {
			const res = await fetch('/api/collections/admin_tags/records?perPage=100', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (res.ok) {
				const data = await res.json();
				availableTags = (data.items || []).map((t: { id: string; name: string; color?: string }) => ({
					id: t.id,
					name: t.name,
					color: t.color
				}));
			}
		} catch (e) {
			console.error('Failed to load tags', e);
		}
	}

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
			if (collectionFilter) params.set('collection', collectionFilter);
			if (statusFilter === 'orphans') {
				params.set('orphans', '1');
			} else if (statusFilter === 'all') {
				params.set('includeOrphans', '1');
			}
			if (selectedTagFilter) params.set('tags', selectedTagFilter);
			if (usageFilter !== 'all') params.set('usage', usageFilter);
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

		// If item is in use, show a warning with usage details before confirmation
		if ((item.usage_count ?? 0) > 0 && !force) {
			// Fetch full usage details
			let usageDetails: MediaUsageItem[] = item.used_by || [];

			// If we don't have used_by details, fetch them
			if (usageDetails.length === 0 && item.record_id) {
				try {
					const res = await fetch(`/api/media/usage/${item.record_id}`, {
						headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
					});
					if (res.ok) {
						const data = await res.json();
						usageDetails = data.used_by || [];
					}
				} catch {
					// Continue with what we have
				}
			}

			// Build list of content items using this media
			const usageList = usageDetails
				.slice(0, 5)
				.map((u) => `• ${u.collection}: ${u.title}`)
				.join('\n');
			const moreText = usageDetails.length > 5 ? `\n...${$t('admin.media.usage_warning_more', { values: { count: usageDetails.length - 5 } })}` : '';

			const confirmed = await confirm({
				title: $t('admin.media.confirm_force_delete_title'),
				message: $t('admin.media.usage_warning_message', { values: { filename: item.display_name || item.filename, count: item.usage_count } }) +
					'\n\n' + usageList + moreText,
				confirmText: $t('admin.media.confirm_force_delete_button'),
				danger: true
			});

			if (!confirmed) return;
			// Continue with force delete
			return deleteFile(item, true);
		}

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
		selectedItems = [];
	}

	// Check if an item is selectable for bulk operations (any non-orphan item with a record_id)
	function isSelectableForBulk(item: MediaItem): boolean {
		return !item.orphan && !!item.record_id;
	}

	// Toggle selection for bulk operations
	function toggleBulkSelection(item: MediaItem) {
		if (!isSelectableForBulk(item)) return;
		const idx = selectedItems.findIndex(s => s.id === item.record_id && s.collection === item.collection);
		if (idx >= 0) {
			selectedItems = selectedItems.filter((_, i) => i !== idx);
		} else {
			selectedItems = [...selectedItems, { id: item.record_id, collection: item.collection }];
		}
	}

	// Check if item is selected for bulk operations
	function isSelectedForBulk(item: MediaItem): boolean {
		return selectedItems.some(s => s.id === item.record_id && s.collection === item.collection);
	}

	// Select all visible selectable items
	function selectVisibleForBulk() {
		const newSelections: SelectedItem[] = [];
		items.forEach((item) => {
			if (isSelectableForBulk(item)) {
				if (!selectedItems.some(s => s.id === item.record_id && s.collection === item.collection)) {
					newSelections.push({ id: item.record_id, collection: item.collection });
				}
			}
		});
		selectedItems = [...selectedItems, ...newSelections];
	}

	// Open bulk tag modal
	function openBulkTagModal() {
		bulkTagForm = { addTags: [], removeTags: [], saving: false };
		bulkTagModalOpen = true;
	}

	// Close bulk tag modal
	function closeBulkTagModal() {
		bulkTagModalOpen = false;
		bulkTagForm = { addTags: [], removeTags: [], saving: false };
	}

	// Execute bulk tagging
	async function executeBulkTag() {
		if (selectedItems.length === 0) return;
		if (bulkTagForm.addTags.length === 0 && bulkTagForm.removeTags.length === 0) {
			toasts.add('error', $t('admin.media.bulk_tag_no_changes'));
			return;
		}

		bulkTagForm.saving = true;
		try {
			const res = await fetch('/api/media/bulk-tag', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({
					items: selectedItems,
					add_tags: bulkTagForm.addTags,
					remove_tags: bulkTagForm.removeTags
				})
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				throw new Error(body.message || body.error || 'Failed to update tags');
			}
			const result = await res.json();
			toasts.add('success', $t('admin.media.bulk_tag_success', { values: { count: result.updated } }));
			closeBulkTagModal();
			selectedItems = [];
			await loadMedia();
		} catch (err) {
			console.error(err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.media.bulk_tag_failed'));
		} finally {
			bulkTagForm.saving = false;
		}
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

	// Bulk delete selected items (non-orphans)
	async function bulkDeleteSelectedItems() {
		if (selectedItems.length === 0) return;

		// Get the actual media items for the selected IDs
		const itemsToDelete = items.filter(item =>
			selectedItems.some(s => s.id === item.record_id && s.collection === item.collection)
		);

		// Check if any items are in use
		const inUseItems = itemsToDelete.filter(item => (item.usage_count ?? 0) > 0);
		const notInUseItems = itemsToDelete.filter(item => (item.usage_count ?? 0) === 0);

		// Build confirmation message
		let message = $t('admin.media.confirm_bulk_delete_items', { values: { count: selectedItems.length } });

		if (inUseItems.length > 0) {
			const usageList = inUseItems
				.slice(0, 5)
				.map(item => {
					const usedByText = item.used_by?.slice(0, 2).map(u => `${u.collection}: ${u.title}`).join(', ') || '';
					return `• ${item.display_name || item.filename} (${item.usage_count} uses${usedByText ? ': ' + usedByText : ''})`;
				})
				.join('\n');
			const moreText = inUseItems.length > 5 ? `\n...${$t('admin.media.usage_warning_more', { values: { count: inUseItems.length - 5 } })}` : '';

			message += '\n\n' + $t('admin.media.bulk_delete_in_use_warning', { values: { count: inUseItems.length } }) +
				'\n' + usageList + moreText;
		}

		const confirmed = await confirm({
			title: $t('admin.media.confirm_bulk_delete_title'),
			message,
			confirmText: $t('admin.media.confirm_bulk_delete_button'),
			danger: true
		});

		if (!confirmed) return;

		// Delete items one by one
		let deleted = 0;
		let failed = 0;

		for (const item of itemsToDelete) {
			try {
				const isExternal = item.external || item.collection === 'external_media';

				if (isExternal && item.record_id) {
					// External media - use dedicated endpoint with force flag
					const res = await fetch(`/api/media/external/${item.record_id}?force=1`, {
						method: 'DELETE',
						headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
					});
					if (res.ok) {
						deleted++;
					} else {
						failed++;
					}
				} else {
					// All other items (uploads, profile photos, experience logos, etc.)
					// Use the same pattern as single delete
					const body = {
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
					if (res.ok) {
						deleted++;
					} else {
						failed++;
					}
				}
			} catch {
				failed++;
			}
		}

		if (deleted > 0) {
			toasts.add('success', $t('admin.media.toast_bulk_items_deleted', { values: { count: deleted } }));
		}
		if (failed > 0) {
			toasts.add('error', $t('admin.media.toast_bulk_delete_some_failed', { values: { count: failed } }));
		}

		selectedItems = [];
		await loadMedia();
	}

	async function fetchExternalMetadata() {
		const url = newExternal.url.trim();
		if (!url) return;

		newExternal.fetching = true;
		try {
			const res = await fetch(`/api/media/fetch-metadata?url=${encodeURIComponent(url)}`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (res.ok) {
				const data = await res.json();
				if (data.supported && !data.error) {
					// Only auto-fill if fields are empty (don't override user input)
					if (data.title && !newExternal.title.trim()) {
						newExternal.title = data.title;
					}
					if (data.thumbnail_url && !newExternal.thumbnail_url.trim()) {
						newExternal.thumbnail_url = data.thumbnail_url;
					}
				}
			}
		} catch (err) {
			console.error('Failed to fetch metadata', err);
		} finally {
			newExternal.fetching = false;
		}
	}

	async function createExternal() {
		const trimmed = newExternal.url.trim();
		if (!trimmed) {
			newExternal.urlTouched = true;
			toasts.add('error', $t('admin.media.toast_url_required'));
			return;
		}
		// Enforce the same provider allowlist as the markdown sanitizer.
		// Anything outside this list cannot be embedded in posts/projects, so
		// rejecting it here keeps the library and the renderer in sync.
		const provider = detectExternalMediaProvider(trimmed);
		if (!provider) {
			newExternal.urlTouched = true;
			toasts.add('error', $t('admin.media.toast_unsupported_provider'));
			return;
		}
		// If the user didn't supply a thumbnail, fill in the predictable
		// provider thumbnail (currently YouTube). Other providers fall back to
		// the provider icon in the grid.
		const autoThumb = getExternalMediaThumbnail(trimmed);
		const thumb = newExternal.thumbnail_url.trim() || autoThumb || '';
		newExternal.saving = true;
		try {
			const res = await fetch('/api/media/external', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({
					url: trimmed,
					title: newExternal.title.trim(),
					mime: newExternal.mime.trim(),
					thumbnail_url: thumb,
					provider
				})
			});
			if (!res.ok) {
				const respBody = await res.json().catch(() => ({}));
				throw new Error(respBody.message || respBody.error || 'Failed to add external media');
			}
			toasts.add('success', $t('admin.media.toast_external_added'));
			newExternal = {
				url: '',
				title: '',
				mime: '',
				thumbnail_url: '',
				saving: false,
				fetching: false,
				urlTouched: false
			};
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

	onMount(() => {
		loadTags();
		loadMedia();
	});
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
				<label class="label mb-0" for="collection-filter">{$t('admin.media.filter_collection')}</label>
				<select id="collection-filter" class="input" bind:value={collectionFilter} onchange={resetAndLoad}>
					{#each availableCollections as coll}
						<option value={coll.value}>{$t(coll.labelKey)}</option>
					{/each}
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
				<label class="label mb-0" for="usage-filter">{$t('admin.media.filter_usage')}</label>
				<select id="usage-filter" class="input" bind:value={usageFilter} onchange={resetAndLoad}>
					<option value="all">{$t('admin.media.filter_usage_all')}</option>
					<option value="in_use">{$t('admin.media.filter_usage_in_use')}</option>
					<option value="not_in_use">{$t('admin.media.filter_usage_not_in_use')}</option>
				</select>
			</div>
			{#if availableTags.length > 0}
				<div class="flex items-center gap-2">
					<label class="label mb-0" for="tag-filter">{$t('admin.media.filter_tag')}</label>
					<select id="tag-filter" class="input" bind:value={selectedTagFilter} onchange={resetAndLoad}>
						<option value="">{$t('admin.media.filter_tag_all')}</option>
						{#each availableTags as tag}
							<option value={tag.id}>{tag.name}</option>
						{/each}
					</select>
				</div>
			{/if}
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
				{#if selectedItems.length > 0}
					<span class="text-gray-700 dark:text-gray-200">{$t('admin.media.selected_items', { values: { count: selectedItems.length } })}</span>
					{#if availableTags.length > 0}
						<button class="btn btn-secondary" onclick={openBulkTagModal}>
							{$t('admin.media.bulk_tag_button')}
						</button>
					{/if}
					<button class="btn btn-danger" onclick={bulkDeleteSelectedItems}>
						{$t('admin.media.bulk_delete_button')}
					</button>
					<button class="btn btn-ghost btn-sm" onclick={() => { selectedItems = []; }}>{$t('admin.media.clear_selection')}</button>
				{:else if statusFilter !== 'orphans'}
					<button class="btn btn-secondary btn-sm" onclick={selectVisibleForBulk}>
						{$t('admin.media.select_visible')}
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
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
					{$t('admin.media.external_supported_providers')}
				</p>
			</div>
			<button
				class="btn btn-primary"
				onclick={createExternal}
				aria-busy={newExternal.saving}
				disabled={newExternal.saving || externalUrlInvalid || !newExternal.url.trim()}
			>
				{newExternal.saving ? $t('admin.media.saving') : $t('admin.media.add_button')}
			</button>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
			<div class="md:col-span-2">
				<label class="label" for="ext-url">{$t('admin.media.url_required')}</label>
				<div class="flex gap-2">
					<input
						id="ext-url"
						type="url"
						class="input flex-1 {externalUrlInvalid ? 'border-red-500 focus:border-red-500' : ''}"
						bind:value={newExternal.url}
						placeholder={$t('admin.media.url_placeholder')}
						aria-invalid={externalUrlInvalid}
						aria-describedby="ext-url-help"
						oninput={() => {
							externalPreviewFailed = false;
							newExternal.urlTouched = true;
						}}
						onblur={() => {
							newExternal.urlTouched = true;
							// Auto-fetch metadata when URL is from a supported provider
							if (externalProvider) {
								fetchExternalMetadata();
							}
						}}
					/>
					{#if externalProvider}
						<button
							type="button"
							class="btn btn-secondary btn-sm whitespace-nowrap"
							onclick={fetchExternalMetadata}
							disabled={newExternal.fetching}
						>
							{#if newExternal.fetching}
								<span class="inline-flex items-center gap-1">
									<span class="w-3 h-3 border-2 border-gray-400 border-t-transparent rounded-full animate-spin"></span>
									{$t('admin.media.fetching_metadata')}
								</span>
							{:else}
								{$t('admin.media.fetch_metadata')}
							{/if}
						</button>
					{/if}
				</div>
				{#if externalUrlInvalid}
					<p id="ext-url-help" class="text-xs text-red-600 dark:text-red-400 mt-1" role="alert">
						{$t('admin.media.external_url_invalid')}
					</p>
				{:else if externalProvider}
					<p id="ext-url-help" class="text-xs text-green-600 dark:text-green-400 mt-1">
						{$t('admin.media.external_url_detected', { values: { provider: getExternalMediaProviderLabel(externalProvider) } })}
					</p>
				{:else}
					<p id="ext-url-help" class="sr-only">{$t('admin.media.external_url_help')}</p>
				{/if}
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
			{#if newExternal.url.trim() && externalProvider}
				{@const previewUrl = newExternal.url.trim()}
				{@const userThumb = newExternal.thumbnail_url.trim()}
				{@const thumbUrl = userThumb || externalAutoThumbnail || ''}
				{@const providerIconName = (externalProvider === 'youtube' || externalProvider === 'vimeo' || externalProvider === 'spotify' || externalProvider === 'soundcloud' || externalProvider === 'loom' || externalProvider === 'codepen' || externalProvider === 'figma') ? externalProvider : 'globe'}
				<div class="lg:col-span-4">
					{#key previewUrl}
						<div class="p-4 bg-gray-100 dark:bg-gray-800 rounded-lg">
							{#if thumbUrl && !externalPreviewFailed}
								<img
									src={thumbUrl}
									alt={$t('admin.media.preview_alt', { values: { provider: getExternalMediaProviderLabel(externalProvider) } })}
									class="max-h-48 mx-auto rounded"
									onerror={() => { externalPreviewFailed = true; }}
								/>
							{:else}
								<div class="flex flex-col items-center justify-center h-32 gap-3">
									<span class="text-gray-500 dark:text-gray-400" aria-hidden="true">
										{@html icon(providerIconName, 'w-12 h-12')}
									</span>
									<span class="text-sm text-gray-600 dark:text-gray-400">
										{getExternalMediaProviderLabel(externalProvider)}
									</span>
									<a href={previewUrl} target="_blank" rel="noopener noreferrer" class="btn btn-secondary btn-sm">
										{$t('admin.media.open_link')}
									</a>
								</div>
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
										{:else if isSelectableForBulk(item)}
											<input
												type="checkbox"
												class="w-4 h-4 text-primary-600 rounded border-gray-300"
												checked={isSelectedForBulk(item)}
												onchange={() => toggleBulkSelection(item)}
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
											{#if item.tags && item.tags.length > 0}
												{#each item.tags.slice(0, 2) as tag}
													<span
														class="inline-flex items-center px-1.5 py-0.5 text-xs rounded shrink-0 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200"
													>
														{#if tag.color}
															<span class="w-1.5 h-1.5 rounded-full mr-1" style="background-color: {tag.color}"></span>
														{/if}
														{tag.name}
													</span>
												{/each}
												{#if item.tags.length > 2}
													<span class="text-xs text-gray-500 dark:text-gray-400">+{item.tags.length - 2}</span>
												{/if}
											{/if}
										</div>
									</td>
								<td class="px-3 py-2 hidden md:table-cell">
									<span class="inline-flex items-center px-1.5 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200">
										{mimeLabel(item.mime)}
									</span>
								</td>
								<td class="px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hidden lg:table-cell">
								{humanSize(item.size)}{#if item.width && item.height}<span class="text-gray-400 dark:text-gray-500 mx-1">•</span><span class="text-xs">{item.width}×{item.height}</span>{/if}
							</td>
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
				<!-- Alt text, description, and tags available for all media items -->
				<div>
					<label class="label" for="edit-alt-text">{$t('admin.media.edit_alt_text_label')}</label>
						<input
							id="edit-alt-text"
							class="input transition-shadow focus:ring-2 focus:ring-primary-500/20"
							bind:value={editForm.alt_text}
							placeholder={$t('admin.media.edit_alt_text_placeholder')}
							maxlength="255"
						/>
						<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.media.edit_alt_text_help')}</p>
					</div>
					<div>
						<label class="label" for="edit-description">{$t('admin.media.edit_description_label')}</label>
						<textarea
							id="edit-description"
							class="input min-h-[80px] transition-shadow focus:ring-2 focus:ring-primary-500/20"
							bind:value={editForm.description}
							placeholder={$t('admin.media.edit_description_placeholder')}
							maxlength="1000"
							rows="3"
						></textarea>
						<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.media.edit_description_help')}</p>
					</div>
					{#if availableTags.length > 0}
						<div>
							<span class="label">{$t('admin.media.edit_tags_label')}</span>
							<div class="flex flex-wrap gap-2">
								{#each availableTags as tag}
									<label
										class="inline-flex items-center gap-1.5 px-2 py-1 rounded cursor-pointer transition-all border focus-within:ring-2 focus-within:ring-primary-500 focus-within:ring-offset-1 dark:focus-within:ring-offset-gray-900 {editForm.tag_ids.includes(tag.id) ? 'bg-white dark:bg-gray-900 border-primary-500 dark:border-primary-500 text-primary-800 dark:text-primary-300' : 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
									>
										<input
											type="checkbox"
											class="sr-only peer"
											checked={editForm.tag_ids.includes(tag.id)}
											onchange={() => {
												if (editForm.tag_ids.includes(tag.id)) {
													editForm.tag_ids = editForm.tag_ids.filter(id => id !== tag.id);
												} else {
													editForm.tag_ids = [...editForm.tag_ids, tag.id];
												}
											}}
										/>
										{#if tag.color}
											<span
												class="w-2.5 h-2.5 rounded-full shrink-0"
												style="background-color: {tag.color}"
											></span>
										{/if}
										<span class="text-sm text-gray-900 dark:text-gray-100">{tag.name}</span>
									</label>
								{/each}
							</div>
							<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.media.edit_tags_help')}</p>
						</div>
					{/if}
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

{#if bulkTagModalOpen}
	<div
		class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-fade-in"
		onclick={(e) => e.target === e.currentTarget && closeBulkTagModal()}
		onkeydown={(e) => e.key === 'Escape' && closeBulkTagModal()}
		role="dialog"
		aria-modal="true"
		aria-labelledby="bulk-tag-modal-title"
		tabindex="-1"
	>
		<div class="card bg-white dark:bg-gray-900 shadow-2xl max-w-md w-full max-h-[90vh] overflow-y-auto animate-scale-in">
			<div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-600">
				<h2 id="bulk-tag-modal-title" class="text-lg font-semibold text-gray-900 dark:text-white">
					{$t('admin.media.bulk_tag_title', { values: { count: selectedItems.length } })}
				</h2>
				<button
					class="btn btn-ghost btn-sm hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
					onclick={closeBulkTagModal}
					aria-label={$t('shared.actions.close')}
				>
					{@html icon('x')}
				</button>
			</div>

			<form class="p-4 space-y-4" onsubmit={(e) => { e.preventDefault(); executeBulkTag(); }}>
				<div>
					<span class="label">{$t('admin.media.bulk_tag_add_label')}</span>
					<div class="flex flex-wrap gap-2">
						{#each availableTags as tag}
							<label
								class="inline-flex items-center gap-1.5 px-2 py-1 rounded cursor-pointer transition-all border focus-within:ring-2 focus-within:ring-primary-500 focus-within:ring-offset-1 dark:focus-within:ring-offset-gray-900 {bulkTagForm.addTags.includes(tag.id) ? 'border-green-500 bg-green-50 dark:bg-green-900/30 text-green-700 dark:text-green-200' : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 text-gray-700 dark:text-gray-300'}"
							>
								<input
									type="checkbox"
									class="sr-only peer"
									checked={bulkTagForm.addTags.includes(tag.id)}
									onchange={() => {
										if (bulkTagForm.addTags.includes(tag.id)) {
											bulkTagForm.addTags = bulkTagForm.addTags.filter(id => id !== tag.id);
										} else {
											bulkTagForm.addTags = [...bulkTagForm.addTags, tag.id];
											// Remove from remove if adding
											bulkTagForm.removeTags = bulkTagForm.removeTags.filter(id => id !== tag.id);
										}
									}}
								/>
								{#if tag.color}
									<span
										class="w-2.5 h-2.5 rounded-full shrink-0"
										style="background-color: {tag.color}"
									></span>
								{/if}
								<span class="text-sm">{tag.name}</span>
								{#if bulkTagForm.addTags.includes(tag.id)}
									<span class="text-green-600 dark:text-green-400">+</span>
								{/if}
							</label>
						{/each}
					</div>
				</div>

				<div>
					<span class="label">{$t('admin.media.bulk_tag_remove_label')}</span>
					<div class="flex flex-wrap gap-2">
						{#each availableTags as tag}
							<label
								class="inline-flex items-center gap-1.5 px-2 py-1 rounded cursor-pointer transition-all border focus-within:ring-2 focus-within:ring-primary-500 focus-within:ring-offset-1 dark:focus-within:ring-offset-gray-900 {bulkTagForm.removeTags.includes(tag.id) ? 'border-red-500 bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-200' : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 text-gray-700 dark:text-gray-300'}"
							>
								<input
									type="checkbox"
									class="sr-only peer"
									checked={bulkTagForm.removeTags.includes(tag.id)}
									onchange={() => {
										if (bulkTagForm.removeTags.includes(tag.id)) {
											bulkTagForm.removeTags = bulkTagForm.removeTags.filter(id => id !== tag.id);
										} else {
											bulkTagForm.removeTags = [...bulkTagForm.removeTags, tag.id];
											// Remove from add if removing
											bulkTagForm.addTags = bulkTagForm.addTags.filter(id => id !== tag.id);
										}
									}}
								/>
								{#if tag.color}
									<span
										class="w-2.5 h-2.5 rounded-full shrink-0"
										style="background-color: {tag.color}"
									></span>
								{/if}
								<span class="text-sm">{tag.name}</span>
								{#if bulkTagForm.removeTags.includes(tag.id)}
									<span class="text-red-600 dark:text-red-400">−</span>
								{/if}
							</label>
						{/each}
					</div>
				</div>

				<div class="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
					<button type="button" class="btn btn-secondary" onclick={closeBulkTagModal}>
						{$t('shared.actions.cancel')}
					</button>
					<button
						type="submit"
						class="btn btn-primary transition-transform active:scale-95"
						disabled={bulkTagForm.saving || (bulkTagForm.addTags.length === 0 && bulkTagForm.removeTags.length === 0)}
					>
						{#if bulkTagForm.saving}
							<span class="inline-flex items-center gap-2">
								<span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
								{$t('admin.media.saving')}
							</span>
						{:else}
							{$t('admin.media.bulk_tag_apply')}
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

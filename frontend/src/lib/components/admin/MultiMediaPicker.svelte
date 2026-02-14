<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { toasts } from '$lib/stores';

	export type MediaOption = {
		id: string;
		title: string;
		provider?: string;
		url?: string;
		thumbnail_url?: string;
		mime?: string;
		description?: string;
		alt_text?: string;
		collection?: string;
	};

	interface Props {
		/** Currently selected media IDs (bindable) */
		value?: string[];
		/** Label for the section */
		label?: string;
		/** Help text shown below the input */
		helpText?: string;
		/** Whether to filter to images only */
		imagesOnly?: boolean;
		/** Callback when selection changes */
		onchange?: () => void;
	}

	let {
		value = $bindable([]),
		label = '',
		helpText = '',
		imagesOnly = false,
		onchange
	}: Props = $props();

	const MAX_UPLOAD_SIZE = 20 * 1024 * 1024; // 20MB

	let mediaOptions: MediaOption[] = $state([]);
	let mediaSearch = $state('');
	let loadingMedia = $state(false);
	let showPicker = $state(false);
	let uploading = $state(false);
	let uploadingFileName = $state('');
	// Working selection while modal is open
	let workingSelection: string[] = $state([]);

	/**
	 * Mark a media item as recently used
	 */
	async function markAsUsed(id: string, collection: string) {
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}`, 'Content-Type': 'application/json' }
				: { 'Content-Type': 'application/json' };
			await fetch('/api/media/mark-used', {
				method: 'POST',
				headers,
				body: JSON.stringify({ id, collection })
			});
		} catch (err) {
			console.error('Failed to mark media as used', err);
		}
	}

	/**
	 * Load media options from the API
	 */
	async function loadMediaOptions(searchTerm = '') {
		loadingMedia = true;
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};
			const mediaParams = new URLSearchParams({ perPage: '50' });
			if (searchTerm.trim()) mediaParams.set('q', searchTerm.trim());
			if (imagesOnly) mediaParams.set('type', 'image');
			const externalFilter = searchTerm.trim()
				? `&filter=${encodeURIComponent(`title~"${searchTerm}" || url~"${searchTerm}"`)}`
				: '';

			const [mediaRes, externalRes] = await Promise.all([
				fetch(`/api/media?${mediaParams.toString()}`, { headers }),
				fetch(`/api/collections/external_media/records?perPage=50${externalFilter}`, { headers })
			]);

			const mediaData = mediaRes.ok ? await mediaRes.json() : { items: [] };
			const externalData = externalRes.ok ? await externalRes.json() : { items: [] };

			const options: MediaOption[] = [];
			const urlToExternalId = new Map<string, string>();

			// Helper to normalize URL for comparison
			const normalizeUrl = (url?: string) => {
				if (!url) return '';
				try {
					const parsed = new URL(url, window.location.origin);
					if (parsed.pathname.includes('/api/files/')) {
						return parsed.pathname;
					}
					return url;
				} catch {
					return url;
				}
			};

			// First pass: collect ALL external_media IDs by normalized URL
			for (const item of externalData.items || []) {
				if (item.url) {
					urlToExternalId.set(normalizeUrl(item.url), item.id);
				}
			}

			// Add internal media items
			for (const item of mediaData.items || []) {
				if (imagesOnly && !item.mime?.startsWith('image/')) continue;
				const normalizedUrl = normalizeUrl(item.url);
				const externalId = normalizedUrl ? urlToExternalId.get(normalizedUrl) : null;
				options.push({
					id: externalId || item.record_id || item.relative_path || item.url,
					title: item.display_name || item.filename || item.url,
					provider: item.provider || (item.external ? 'external' : 'upload'),
					url: item.url,
					thumbnail_url: item.thumbnail_url,
					mime: item.mime,
					// Use external_media collection if we're using the external mirror ID
					collection: externalId ? 'external_media' : item.collection,
					description: item.description || '',
					alt_text: item.alt_text || ''
				});
			}

			// Add external_media items that don't have a matching internal URL
			const addedIds = new Set(options.map((opt) => opt.id));
			for (const item of externalData.items || []) {
				if (imagesOnly && item.mime && !item.mime.startsWith('image/')) continue;
				if (!addedIds.has(item.id) && !item.url?.includes('/api/files/')) {
					options.push({
						id: item.id,
						title: item.title || item.url,
						provider: 'external',
						url: item.url,
						thumbnail_url: item.thumbnail_url,
						mime: item.mime,
						collection: 'external_media',
						description: item.description || '',
						alt_text: item.alt_text || ''
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

	/**
	 * Resolve media refs, mirroring uploads to external_media if needed
	 */
	export async function resolveMediaRefs(selected: string[]): Promise<string[]> {
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
			if (!opt) {
				resolved.push(id);
				continue;
			}
			if (opt.provider === 'upload' && opt.url) {
				try {
					const absoluteUrl = toAbsolute(opt.url);
					const filter = encodeURIComponent(`url="${opt.url}" || url="${absoluteUrl}"`);
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
						body: JSON.stringify({ url: absoluteUrl, title: opt.title })
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

	/**
	 * Fetch metadata for IDs that are in value but not in loaded options
	 */
	async function reconcileSelectedItems() {
		if (value.length === 0 || mediaOptions.length === 0) return;

		const loadedIds = new Set(mediaOptions.map((opt) => opt.id));
		const missingIds = value.filter((id) => !loadedIds.has(id));

		if (missingIds.length === 0) return;

		const headers: Record<string, string> = pb.authStore.isValid
			? { Authorization: `Bearer ${pb.authStore.token}` }
			: {};

		const newOptions: MediaOption[] = [];

		for (const id of missingIds) {
			try {
				const res = await fetch(`/api/collections/external_media/records/${id}`, { headers });
				if (res.ok) {
					const item = await res.json();
					const isMirror = item.url?.includes('/api/files/');
					let thumbnailUrl = item.thumbnail_url;
					if (isMirror && item.url && !thumbnailUrl) {
						const match = item.url.match(/\/api\/files\/([^/]+)\/([^/]+)\/(.+)/);
						if (match) {
							const [, collection, recordId, filename] = match;
							thumbnailUrl = `/api/media/thumb/${collection}/${recordId}/${filename}`;
						}
					}

					newOptions.push({
						id: item.id,
						title: item.title || item.url || id,
						provider: isMirror ? 'upload' : 'external',
						url: item.url,
						thumbnail_url: thumbnailUrl,
						mime: item.mime,
						collection: isMirror ? 'uploads' : 'external_media',
						description: item.description || '',
						alt_text: item.alt_text || ''
					});
				}
			} catch (err) {
				console.error('Failed to fetch metadata for', id, err);
			}
		}

		if (newOptions.length > 0) {
			mediaOptions = [...newOptions, ...mediaOptions];
		}
	}

	function handleSearch() {
		loadMediaOptions(mediaSearch);
	}

	function handleClearSearch() {
		mediaSearch = '';
		loadMediaOptions('');
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			handleSearch();
		}
	}

	function handleToggleSelection(option: MediaOption) {
		if (workingSelection.includes(option.id)) {
			workingSelection = workingSelection.filter((id) => id !== option.id);
		} else {
			workingSelection = [...workingSelection, option.id];
			if (option.collection && option.id) {
				markAsUsed(option.id, option.collection);
			}
		}
	}

	function handleOpenPicker() {
		workingSelection = [...value];
		showPicker = true;
		loadMediaOptions();
	}

	function handleClosePicker() {
		showPicker = false;
	}

	function handleConfirmSelection() {
		value = [...workingSelection];
		showPicker = false;
		onchange?.();
	}

	function handleClearAll() {
		value = [];
		onchange?.();
	}

	function handleRemoveItem(id: string) {
		value = value.filter((v) => v !== id);
		onchange?.();
	}

	async function handleFileUpload(event: Event) {
		const target = event.target as HTMLInputElement;
		const files = target.files;
		if (!files || files.length === 0) return;
		target.value = '';

		uploading = true;
		let successCount = 0;
		const failedFiles: string[] = [];

		for (const file of files) {
			uploadingFileName = file.name;

			// Validate file size
			if (file.size > MAX_UPLOAD_SIZE) {
				console.warn(`File ${file.name} exceeds 20MB limit`);
				failedFiles.push(file.name);
				continue;
			}

			try {
				const form = new FormData();
				form.append('file', file);
				form.append('title', file.name);
				if (file.type) form.append('mime', file.type);

				const headers: Record<string, string> = pb.authStore.isValid
					? { Authorization: `Bearer ${pb.authStore.token}` }
					: {};

				const res = await fetch('/api/collections/uploads/records', {
					method: 'POST',
					headers,
					body: form
				});

				if (!res.ok) {
					const body = await res.json().catch(() => ({}));
					throw new Error(body.message || 'Upload failed');
				}

				const record = await res.json();
				const collectionId = record.collectionId || 'uploads';
				const uploadUrl = `/api/files/${collectionId}/${record.id}/${record.file}`;

				const newOption: MediaOption = {
					id: record.id,
					title: file.name,
					provider: 'upload',
					url: uploadUrl,
					mime: file.type,
					collection: 'uploads'
				};

				mediaOptions = [newOption, ...mediaOptions];
				value = [...value, record.id];
				markAsUsed(record.id, 'uploads');
				successCount++;
			} catch (err) {
				console.error(`Failed to upload ${file.name}:`, err);
				failedFiles.push(file.name);
			}
		}

		if (successCount > 0) {
			toasts.add('success', $t('admin.media.toast_upload_success'));
			onchange?.();
		}
		if (failedFiles.length > 0) {
			toasts.add('error', $t('admin.media.toast_upload_failed'));
		}

		uploading = false;
		uploadingFileName = '';
	}

	function isImage(option: MediaOption): boolean {
		return option.mime?.startsWith('image/') || false;
	}

	function getAbsoluteUrl(url: string): string {
		if (!url) return '';
		if (/^https?:\/\//i.test(url)) return url;
		const base = pb.baseUrl.replace(/\/$/, '');
		return `${base}${url.startsWith('/') ? url : `/${url}`}`;
	}

	// Get display info for selected items
	let selectedItems = $derived(
		value
			.map((id) => mediaOptions.find((opt) => opt.id === id))
			.filter((opt): opt is MediaOption => opt !== undefined)
	);

	// Load on mount and reconcile
	$effect(() => {
		loadMediaOptions();
	});

	$effect(() => {
		if (!loadingMedia && mediaOptions.length > 0 && value.length > 0) {
			reconcileSelectedItems();
		}
	});
</script>

<div class="multi-media-picker">
	{#if label}
		<span class="label">{label}</span>
	{/if}

	<div class="space-y-2">
		<!-- Selected items display -->
		{#if selectedItems.length > 0}
			<div class="flex flex-wrap gap-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				{#each selectedItems as item}
					<div class="flex items-center gap-2 px-2 py-1 bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600">
						{#if item.thumbnail_url || (isImage(item) && item.url)}
							<img
								src={item.thumbnail_url || getAbsoluteUrl(item.url || '')}
								alt=""
								class="w-6 h-6 object-cover rounded"
								loading="lazy"
							/>
						{:else}
							<div class="w-6 h-6 flex items-center justify-center rounded bg-gray-100 dark:bg-gray-600">
								<svg class="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
								</svg>
							</div>
						{/if}
						<span class="text-sm text-gray-700 dark:text-gray-300 truncate max-w-[120px]">{item.title}</span>
						<button
							type="button"
							class="text-gray-400 hover:text-red-500 dark:hover:text-red-400"
							onclick={() => handleRemoveItem(item.id)}
							aria-label={$t('admin.media.remove_item')}
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					</div>
				{/each}
				<button
					type="button"
					class="text-xs text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 px-2 py-1"
					onclick={handleClearAll}
				>
					{$t('admin.media.multi_picker_clear_all')}
				</button>
			</div>
		{/if}

		<!-- Action button -->
		<div class="flex flex-wrap items-center gap-2">
			<button
				type="button"
				class="btn btn-secondary btn-sm"
				onclick={handleOpenPicker}
				disabled={uploading}
			>
				{value.length > 0
					? $t('admin.media.multi_picker_modify_selection')
					: $t('admin.media.multi_picker_select_from_library')}
			</button>
			<span class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.media.multi_picker_or')}</span>
			<label class="btn btn-secondary btn-sm cursor-pointer {uploading ? 'opacity-50 pointer-events-none' : ''}">
				{#if uploading}
					{$t('admin.media.uploading')}
				{:else}
					{$t('admin.media.multi_picker_upload_new')}
				{/if}
				<input
					type="file"
					accept={imagesOnly ? 'image/*' : 'image/*,video/*,audio/*,.pdf'}
					multiple
					class="hidden"
					onchange={handleFileUpload}
					disabled={uploading}
				/>
			</label>
			{#if value.length > 0}
				<span class="text-sm text-gray-500 dark:text-gray-400">
					{$t('admin.media.multi_picker_count', { values: { count: value.length } })}
				</span>
			{/if}
		</div>

		{#if helpText}
			<p class="text-xs text-gray-500 dark:text-gray-400">{helpText}</p>
		{/if}
	</div>

	<!-- Media picker modal -->
	{#if showPicker}
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" role="presentation" onclick={handleClosePicker}>
			<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
			<div
				class="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-3xl w-full mx-4 max-h-[80vh] flex flex-col"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
				onclick={(e) => e.stopPropagation()}
			>
				<div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
						{$t('admin.media.multi_picker_modal_title')}
						{#if workingSelection.length > 0}
							<span class="text-sm font-normal text-gray-500 dark:text-gray-400 ml-2">
								({$t('admin.media.multi_picker_selected_count', { values: { count: workingSelection.length } })})
							</span>
						{/if}
					</h3>
					<button
						type="button"
						class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
						onclick={handleClosePicker}
						aria-label={$t('shared.aria.close')}
					>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div class="p-4 border-b border-gray-200 dark:border-gray-700">
					<div class="flex items-center gap-2">
						<input
							class="input flex-1"
							placeholder={$t('admin.media.picker_search_placeholder')}
							bind:value={mediaSearch}
							onkeydown={handleKeydown}
						/>
						<button type="button" class="btn btn-secondary btn-sm" onclick={handleSearch}>
							{loadingMedia ? $t('admin.media.picker_searching') : $t('admin.media.picker_search')}
						</button>
						<button type="button" class="btn btn-ghost btn-sm" onclick={handleClearSearch}>
							{$t('admin.media.picker_clear')}
						</button>
					</div>
				</div>

				<div class="flex-1 overflow-y-auto p-4">
					{#if loadingMedia}
						<p class="text-center text-gray-500 dark:text-gray-400 py-8">
							{$t('admin.media.picker_loading')}
						</p>
					{:else if mediaOptions.length === 0}
						<div class="text-center py-8">
							<p class="text-gray-500 dark:text-gray-400 mb-2">
								{$t('admin.media.picker_empty')}
							</p>
							<a href="/admin/media" class="text-primary-600 dark:text-primary-400 hover:underline text-sm">
								{$t('admin.media.picker_go_to_library')}
							</a>
						</div>
					{:else}
						<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
							{#each mediaOptions as opt}
								{@const isSelected = workingSelection.includes(opt.id)}
								<button
									type="button"
									class="group relative aspect-square rounded-lg overflow-hidden border-2 transition-all {isSelected
										? 'border-primary-500 ring-2 ring-primary-500/30'
										: 'border-gray-200 dark:border-gray-700 hover:border-primary-300'}"
									onclick={() => handleToggleSelection(opt)}
								>
									{#if opt.thumbnail_url || (isImage(opt) && opt.url)}
										<img
											src={opt.thumbnail_url || getAbsoluteUrl(opt.url || '')}
											alt={opt.title}
											class="w-full h-full object-cover"
											loading="lazy"
										/>
									{:else}
										<div class="w-full h-full flex items-center justify-center bg-gray-100 dark:bg-gray-800">
											<svg class="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
											</svg>
										</div>
									{/if}
									<div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent p-2">
										<p class="text-xs text-white truncate">{opt.title}</p>
									</div>
									{#if isSelected}
										<div class="absolute top-2 right-2 w-6 h-6 bg-primary-500 rounded-full flex items-center justify-center">
											<svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
											</svg>
										</div>
									{:else}
										<div class="absolute top-2 right-2 w-6 h-6 border-2 border-white/70 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"></div>
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				</div>

				<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-between gap-2">
					<button type="button" class="btn btn-ghost" onclick={handleClosePicker}>
						{$t('shared.actions.cancel')}
					</button>
					<button type="button" class="btn btn-primary" onclick={handleConfirmSelection}>
						{$t('admin.media.multi_picker_confirm', { values: { count: workingSelection.length } })}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

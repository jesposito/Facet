<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';

	export type MediaOption = {
		id: string;
		title: string;
		provider?: string;
		url?: string;
		thumbnail_url?: string;
		mime?: string;
		collection?: string;
		description?: string;
		alt_text?: string;
	};

	interface Props {
		/** Currently selected media IDs (bindable) */
		value?: string[];
		/** Label for the section */
		label?: string;
		/** Whether to show help text linking to Media Library */
		showHelp?: boolean;
	}

	let {
		value = $bindable([]),
		label = '',
		showHelp = false
	}: Props = $props();

	const dispatch = createEventDispatcher<{
		change: { value: string[] };
	}>();

	let mediaOptions: MediaOption[] = $state([]);
	let recentItems: MediaOption[] = $state([]);
	let mediaSearch = $state('');
	let loadingMedia = $state(false);

	/**
	 * Load recently used media
	 */
	async function loadRecentItems() {
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};
			const res = await fetch('/api/media/recent?limit=10', { headers });
			if (res.ok) {
				const data = await res.json();
				recentItems = (data.items || []).map((item: {
					record_id: string;
					display_name?: string;
					filename?: string;
					url?: string;
					thumbnail_url?: string;
					mime?: string;
					collection?: string;
					external?: boolean;
				}) => ({
					id: item.record_id,
					title: item.display_name || item.filename || item.url || '',
					url: item.url,
					thumbnail_url: item.thumbnail_url,
					mime: item.mime,
					collection: item.collection,
					provider: item.external ? 'external' : 'upload'
				}));
			}
		} catch (err) {
			console.error('Failed to load recent items', err);
		}
	}

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
	 * Load media options from the unified media_library API
	 */
	export async function loadMediaOptions(searchTerm = '') {
		loadingMedia = true;
		try {
			// Debug: log auth state
			console.log('[MediaPicker] loadMediaOptions called', {
				authValid: pb.authStore.isValid,
				hasToken: !!pb.authStore.token,
				tokenPreview: pb.authStore.token ? pb.authStore.token.substring(0, 20) + '...' : 'none'
			});

			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};
			// Filter to media_library only - after migration all files have media_library records
			const mediaParams = new URLSearchParams({ perPage: '50', collection: 'media_library' });
			if (searchTerm.trim()) mediaParams.set('q', searchTerm.trim());

			console.log('[MediaPicker] fetching', `/api/media?${mediaParams.toString()}`, { headers });

			const mediaRes = await fetch(`/api/media?${mediaParams.toString()}`, {
				headers,
				credentials: 'include'  // Ensure cookies are sent for auth
			});

			console.log('[MediaPicker] response', { ok: mediaRes.ok, status: mediaRes.status });
			const mediaData = mediaRes.ok ? await mediaRes.json() : { items: [] };

			const options: MediaOption[] = [];
			const addedIds = new Set<string>();

			for (const item of mediaData.items || []) {
				// Use record_id (media_library ID) as the primary identifier
				const id = item.record_id || item.relative_path || item.url;
				if (addedIds.has(id)) continue;
				addedIds.add(id);

				options.push({
					id,
					title: item.display_name || item.filename || item.url,
					provider: item.provider || (item.external ? 'external' : 'upload'),
					url: item.url,
					thumbnail_url: item.thumbnail_url,
					mime: item.mime,
					collection: item.collection || 'media_library',
					description: item.description || '',
					alt_text: item.alt_text || ''
				});
			}

			mediaOptions = options;
		} catch (err) {
			console.error('Failed to load media options', err);
		} finally {
			loadingMedia = false;
		}
	}

	function handleSearch() {
		loadMediaOptions(mediaSearch);
	}

	function handleClear() {
		mediaSearch = '';
		loadMediaOptions('');
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			handleSearch();
		}
	}

	function handleSelectionChange(id: string, checked: boolean, collection?: string) {
		if (checked) {
			value = [...value, id];
			// Mark as used when selected
			if (collection) {
				markAsUsed(id, collection);
			}
		} else {
			value = value.filter((v) => v !== id);
		}
		dispatch('change', { value });
	}

	function isImage(option: MediaOption): boolean {
		return option.mime?.startsWith('image/') || false;
	}

	/**
	 * Fetch metadata for IDs that are in value but not in loaded options
	 * This ensures pre-selected items from editing are shown as checked
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

		// Fetch metadata for missing IDs from media_library (with fallback to external_media)
		for (const id of missingIds) {
			try {
				// Try media_library first
				let res = await fetch(`/api/collections/media_library/records/${id}`, { headers });

				// Fallback to external_media for backward compatibility during migration
				if (!res.ok) {
					res = await fetch(`/api/collections/external_media/records/${id}`, { headers });
					if (res.ok) {
						const item = await res.json();
						const isMirror = item.url?.includes('/api/files/');
						let thumbnailUrl = item.thumbnail_url;
						let resolvedId = item.id;

						if (isMirror && item.url) {
							// Extract the upload record ID from mirror URL
							// Format: /api/files/{collection_id}/{record_id}/{filename}
							const match = item.url.match(/\/api\/files\/([^/]+)\/([^/]+)\/(.+)/);
							if (match) {
								const [, collection, recordId, filename] = match;
								// Use the upload ID instead of mirror ID for media_library relation
								resolvedId = recordId;
								if (!thumbnailUrl) {
									thumbnailUrl = `/api/media/thumb/${collection}/${recordId}/${filename}`;
								}
							}
						}
						newOptions.push({
							id: resolvedId,
							title: item.title || item.url || id,
							provider: isMirror ? 'upload' : 'external',
							url: item.url,
							thumbnail_url: thumbnailUrl,
							mime: item.mime,
							collection: 'media_library', // Use media_library for relation
							description: item.description || '',
							alt_text: item.alt_text || ''
						});
						continue;
					}
				}

				if (res.ok) {
					const item = await res.json();
					const isUpload = item.type === 'upload';

					// For uploads, construct thumbnail URL from file field
					let thumbnailUrl = item.thumbnail_url;
					if (isUpload && item.file && !thumbnailUrl) {
						thumbnailUrl = `/api/media/thumb/media_library/${item.id}/${item.file}`;
					}

					newOptions.push({
						id: item.id,
						title: item.title || item.url || id,
						provider: isUpload ? 'upload' : 'external',
						url: isUpload ? `/api/files/media_library/${item.id}/${item.file}` : item.url,
						thumbnail_url: thumbnailUrl,
						mime: item.mime,
						collection: 'media_library',
						description: item.description || '',
						alt_text: item.alt_text || ''
					});
				}
			} catch (err) {
				console.error('Failed to fetch metadata for', id, err);
			}
		}

		// Prepend new options so selected items appear at the top
		if (newOptions.length > 0) {
			mediaOptions = [...newOptions, ...mediaOptions];
		}
	}

	// Load media options and recent items on mount
	$effect(() => {
		loadMediaOptions();
		loadRecentItems();
	});

	// Reconcile selected items after options load
	$effect(() => {
		if (!loadingMedia && mediaOptions.length > 0 && value.length > 0) {
			reconcileSelectedItems();
		}
	});
</script>

<div class="media-picker">
	{#if label}
		<p class="label">{label}</p>
	{/if}

	<div class="flex flex-wrap items-center gap-2 mb-2 text-sm text-gray-600 dark:text-gray-400">
		<input
			class="input w-full md:w-64"
			placeholder={$t('admin.media.picker_search_placeholder')}
			bind:value={mediaSearch}
			onkeydown={handleKeydown}
		/>
		<button
			type="button"
			class="btn btn-secondary btn-sm"
			onclick={handleSearch}
			aria-busy={loadingMedia}
		>
			{loadingMedia ? $t('admin.media.picker_searching') : $t('admin.media.picker_search')}
		</button>
		<button type="button" class="btn btn-ghost btn-sm" onclick={handleClear}>
			{$t('admin.media.picker_clear')}
		</button>
	</div>

	{#if recentItems.length > 0 && !mediaSearch}
		<div class="mb-3">
			<p class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">{$t('admin.media.picker_recent')}</p>
			<div class="flex flex-wrap gap-1.5">
				{#each recentItems.slice(0, 6) as opt}
					<button
						type="button"
						class="flex items-center gap-1.5 px-2 py-1 rounded text-xs border transition-colors {value.includes(opt.id)
							? 'bg-white dark:bg-gray-900 border-primary-500 dark:border-primary-500 text-primary-800 dark:text-primary-300'
							: 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
						onclick={() => handleSelectionChange(opt.id, !value.includes(opt.id), opt.collection)}
					>
						{#if opt.thumbnail_url || (isImage(opt) && opt.url)}
							<div class="w-5 h-5 rounded overflow-hidden bg-gray-200 dark:bg-gray-700 shrink-0">
								<img
									src={opt.thumbnail_url || opt.url}
									alt=""
									class="w-full h-full object-cover"
									loading="lazy"
								/>
							</div>
						{/if}
						<span class="truncate max-w-[100px]">{opt.title}</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}

	{#if loadingMedia}
		<p class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.media.picker_loading')}</p>
	{:else if mediaOptions.length === 0}
		<p class="text-sm text-gray-500 dark:text-gray-400">
			{$t('admin.media.picker_empty')}
			{#if showHelp}
				<a href="/admin/media" class="text-primary-600 dark:text-primary-400 hover:underline">
					{$t('admin.media.picker_go_to_library')}
				</a>
			{/if}
		</p>
	{:else}
		<div class="flex flex-col gap-2 max-h-48 overflow-y-auto pr-2">
			{#each mediaOptions as opt}
				{@const isSelected = value.includes(opt.id)}
				<label
					class="flex items-center gap-2 px-3 py-2 rounded border cursor-pointer {isSelected
						? 'bg-white dark:bg-gray-900 border-primary-500 dark:border-primary-500 text-primary-800 dark:text-primary-300'
						: 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700'}"
				>
					<input
						type="checkbox"
						class="w-4 h-4"
						checked={isSelected}
						onchange={(e) => handleSelectionChange(opt.id, (e.target as HTMLInputElement).checked, opt.collection)}
					/>
					{#if opt.thumbnail_url || (isImage(opt) && opt.url)}
						<div class="w-8 h-8 rounded overflow-hidden bg-gray-200 dark:bg-gray-700 shrink-0">
							<img
								src={opt.thumbnail_url || opt.url}
								alt=""
								class="w-full h-full object-cover"
								loading="lazy"
							/>
						</div>
					{/if}
					<div class="flex flex-col min-w-0 flex-1">
						<span class="text-sm font-medium truncate">{opt.title}</span>
						{#if opt.description}
							<span class="text-xs line-clamp-2 {isSelected ? 'text-primary-700 dark:text-primary-300' : 'text-gray-600 dark:text-gray-300'}">{opt.description}</span>
						{:else if opt.provider}
							<span class="text-xs {isSelected ? 'text-primary-600 dark:text-primary-300' : 'text-gray-500 dark:text-gray-400'}">{opt.provider}</span>
						{/if}
					</div>
				</label>
			{/each}
		</div>
	{/if}
</div>

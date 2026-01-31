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
	 * Load media options from the API
	 */
	export async function loadMediaOptions(searchTerm = '') {
		loadingMedia = true;
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};
			const mediaParams = new URLSearchParams({ perPage: '50' });
			if (searchTerm.trim()) mediaParams.set('q', searchTerm.trim());
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

			// Helper to normalize URL for comparison (strip origin for internal files)
			const normalizeUrl = (url?: string) => {
				if (!url) return '';
				try {
					const parsed = new URL(url, window.location.origin);
					// For internal files, use just the pathname
					if (parsed.pathname.includes('/api/files/')) {
						return parsed.pathname;
					}
					return url;
				} catch {
					return url;
				}
			};

			// First pass: collect ALL external_media IDs by normalized URL
			// Include mirrors (URLs with /api/files/) since those are what media_refs stores
			for (const item of externalData.items || []) {
				if (item.url) {
					urlToExternalId.set(normalizeUrl(item.url), item.id);
				}
			}

			// Add internal media items, preferring external_media ID if one exists for same URL
			for (const item of mediaData.items || []) {
				const normalizedUrl = normalizeUrl(item.url);
				const externalId = normalizedUrl ? urlToExternalId.get(normalizedUrl) : null;
				options.push({
					// Use external_media ID if available (since that's what media_refs stores)
					id: externalId || item.record_id || item.relative_path || item.url,
					title: item.display_name || item.filename || item.url,
					provider: externalId ? 'external' : (item.provider || (item.external ? 'external' : 'upload')),
					url: item.url,
					thumbnail_url: item.thumbnail_url,
					mime: item.mime,
					collection: item.collection,
					description: item.description || '',
					alt_text: item.alt_text || ''
				});
			}

			// Add external_media items that don't have a matching internal URL
			// Skip "mirror" entries that point to internal PocketBase files (already handled above)
			const addedIds = new Set(options.map((opt) => opt.id));
			for (const item of externalData.items || []) {
				// Skip if already added (via mirror matching) or if it's an internal mirror
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
				// ID not in current options - likely an existing external_media ID, preserve it
				resolved.push(id);
				continue;
			}
			if (opt.provider === 'upload' && opt.url) {
				// Mirror upload into external_media so it can be stored in media_refs
				try {
					const absoluteUrl = toAbsolute(opt.url);
					// Search for existing external_media with either relative or absolute URL
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
			if (collection === 'uploads' || collection === 'external_media') {
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

		// Fetch metadata for missing IDs from external_media
		for (const id of missingIds) {
			try {
				const res = await fetch(`/api/collections/external_media/records/${id}`, { headers });
				if (res.ok) {
					const item = await res.json();

					// Check if this is a mirror of an internal upload
					const isMirror = item.url?.includes('/api/files/');

					// For mirrors, try to get thumbnail from the internal URL
					let thumbnailUrl = item.thumbnail_url;
					if (isMirror && item.url && !thumbnailUrl) {
						// Extract path parts to construct thumbnail URL
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
							? 'bg-primary-50 dark:bg-primary-900 border-primary-300 dark:border-primary-600 text-primary-700 dark:text-primary-200'
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
						? 'bg-primary-50 dark:bg-primary-900 border-primary-200 dark:border-primary-700 text-primary-700 dark:text-primary-200'
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

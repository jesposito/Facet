<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';

	export type MediaOption = {
		id: string;
		title: string;
		provider?: string;
		url?: string;
		thumbnail_url?: string;
		mime?: string;
		description?: string;
		alt_text?: string;
	};

	interface Props {
		/** Currently selected media URL (bindable) */
		value?: string;
		/** Label for the section */
		label?: string;
		/** Help text shown below the input */
		helpText?: string;
		/** Current file preview URL (for existing uploaded files) */
		currentFileUrl?: string;
		/** Current file name for display */
		currentFileName?: string;
		/** File input for direct upload (bindable) */
		fileInput?: FileList | null;
		/** Accept filter for file input */
		accept?: string;
		/** Whether to filter to images only */
		imagesOnly?: boolean;
		/** Flag to indicate existing file should be cleared (bindable) */
		clearExisting?: boolean;
		/** Callback when selection changes (for triggering form autosave) */
		onchange?: () => void;
		/** Context hint to display instead of media title (e.g., company name for logo) */
		contextHint?: string;
	}

	let {
		value = $bindable(''),
		label = '',
		helpText = '',
		currentFileUrl = '',
		currentFileName = '',
		fileInput = $bindable(null),
		accept = 'image/*',
		imagesOnly = true,
		clearExisting = $bindable(false),
		onchange,
		contextHint = ''
	}: Props = $props();

	let mediaOptions: MediaOption[] = $state([]);
	let mediaSearch = $state('');
	let loadingMedia = $state(false);
	let showPicker = $state(false);

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
			const addedUrls = new Set<string>();

			// Add internal media items
			for (const item of mediaData.items || []) {
				if (imagesOnly && !item.mime?.startsWith('image/')) continue;
				if (item.url && !addedUrls.has(item.url)) {
					addedUrls.add(item.url);
					options.push({
						id: item.record_id || item.relative_path || item.url,
						title: item.display_name || item.filename || item.url,
						provider: item.provider || (item.external ? 'external' : 'upload'),
						url: item.url,
						thumbnail_url: item.thumbnail_url,
						mime: item.mime
					});
				}
			}

			// Add external_media items (skip mirrors that point to internal files)
			for (const item of externalData.items || []) {
				// Skip "mirror" entries that point to internal PocketBase files
				if (item.url?.includes('/api/files/')) continue;
				if (imagesOnly && item.mime && !item.mime.startsWith('image/')) continue;
				if (item.url && !addedUrls.has(item.url)) {
					addedUrls.add(item.url);
					options.push({
						id: item.id,
						title: item.title || item.url,
						provider: 'external',
						url: item.url,
						thumbnail_url: item.thumbnail_url,
						mime: item.mime
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

	function handleSelect(option: MediaOption) {
		value = option.url || '';
		fileInput = null; // Clear file input when selecting from library
		clearExisting = false; // Selecting from library overrides, no need to clear
		showPicker = false;
		onchange?.();
	}

	function handleClearSelection() {
		value = '';
		fileInput = null;
		// Signal to parent that existing file should be cleared
		if (currentFileUrl) {
			clearExisting = true;
		}
		onchange?.();
	}

	function handleOpenPicker() {
		showPicker = true;
		loadMediaOptions();
	}

	function handleClosePicker() {
		showPicker = false;
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

	// Computed: what to display as current selection
	// If clearExisting is set, don't show currentFileUrl
	let effectiveCurrentFileUrl = $derived(clearExisting ? '' : currentFileUrl);
	let displayUrl = $derived(value || effectiveCurrentFileUrl || '');
	let displayTitle = $derived(
		contextHint ||
		(value
			? mediaOptions.find((o) => o.url === value)?.title || value.split('/').pop() || 'Selected'
			: (clearExisting ? '' : currentFileName) || '')
	);
	let hasSelection = $derived(!!value || !!effectiveCurrentFileUrl || (fileInput && fileInput.length > 0));
</script>

<div class="single-media-picker">
	{#if label}
		<span class="label">{label}</span>
	{/if}

	<div class="space-y-2">
		<!-- Current selection display -->
		{#if hasSelection && !showPicker}
			<div class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				{#if displayUrl && !fileInput?.length}
					<img
						src={getAbsoluteUrl(displayUrl)}
						alt=""
						class="w-12 h-12 object-contain rounded bg-white dark:bg-gray-700"
					/>
				{:else if fileInput?.length}
					<div class="w-12 h-12 flex items-center justify-center rounded bg-primary-100 dark:bg-primary-900">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
					</div>
				{/if}
				<div class="flex-1 min-w-0">
					<p class="text-sm font-medium text-gray-900 dark:text-white truncate">
						{#if fileInput?.length}
							{fileInput[0].name}
						{:else}
							{displayTitle}
						{/if}
					</p>
					<p class="text-xs text-gray-500 dark:text-gray-400">
						{#if fileInput?.length}
							{$t('admin.media.single_picker_new_upload')}
						{:else if value}
							{$t('admin.media.single_picker_from_library')}
						{:else}
							{$t('admin.media.single_picker_current')}
						{/if}
					</p>
				</div>
				<button
					type="button"
					class="btn btn-ghost btn-sm text-red-600 hover:text-red-700"
					onclick={handleClearSelection}
				>
					{$t('admin.media.single_picker_remove')}
				</button>
			</div>
		{/if}

		<!-- Action buttons -->
		<div class="flex flex-wrap items-center gap-2">
			<button
				type="button"
				class="btn btn-secondary btn-sm"
				onclick={handleOpenPicker}
			>
				{$t('admin.media.single_picker_select_from_library')}
			</button>
			<span class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.media.single_picker_or')}</span>
			<label class="btn btn-secondary btn-sm cursor-pointer">
				{$t('admin.media.single_picker_upload_new')}
				<input
					type="file"
					{accept}
					bind:files={fileInput}
					class="hidden"
					onchange={() => { value = ''; clearExisting = false; showPicker = false; }}
				/>
			</label>
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
				class="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
				onclick={(e) => e.stopPropagation()}
			>
				<div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
						{$t('admin.media.single_picker_modal_title')}
					</h3>
					<button
						type="button"
						class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
						onclick={handleClosePicker}
						aria-label="Close"
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
								<button
									type="button"
									class="group relative aspect-square rounded-lg overflow-hidden border-2 transition-all {value === opt.url
										? 'border-primary-500 ring-2 ring-primary-500/30'
										: 'border-gray-200 dark:border-gray-700 hover:border-primary-300'}"
									onclick={() => handleSelect(opt)}
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
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
											</svg>
										</div>
									{/if}
									<div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent p-2">
										<p class="text-xs text-white truncate">{opt.title}</p>
									</div>
									{#if value === opt.url}
										<div class="absolute top-2 right-2 w-6 h-6 bg-primary-500 rounded-full flex items-center justify-center">
											<svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
											</svg>
										</div>
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				</div>

				<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
					<button type="button" class="btn btn-ghost" onclick={handleClosePicker}>
						{$t('shared.actions.cancel')}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

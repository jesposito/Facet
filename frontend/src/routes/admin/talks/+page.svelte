<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Talk } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { formatDate } from '$lib/utils';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';

	let talks: Talk[] = [];
	let loading = true;
	let showForm = false;
	let editingTalk: Talk | null = null;

	// Form fields
	let title = '';
let slug = '';
let event = '';
let eventUrl = '';
let date = '';
let location = '';
let description = '';
let slidesUrl = '';
let videoUrl = '';
let visibility = 'public';
let isDraft = false;
let sortOrder = 0;
let mediaRefs: string[] = [];
let mediaOptions: { id: string; title: string; provider?: string; url?: string }[] = [];
let showShortcodes = false;
let saving = false;
let mediaSearch = '';
let loadingMedia = false;

let selectMode = false;
let selectedIds: Set<string> = new Set();

	// Generate slug from title
	function generateSlug(text: string): string {
		return text
			.toLowerCase()
			.replace(/[^a-z0-9\s-]/g, '')
			.replace(/\s+/g, '-')
			.replace(/-+/g, '-')
			.slice(0, 100);
	}

	function handleTitleChange() {
		// Only auto-generate slug for new talks or if slug is empty
		if (!editingTalk || !slug) {
			slug = generateSlug(title);
		}
	}

	onMount(loadTalks);
onMount(loadMediaOptions);

async function loadMediaOptions(searchTerm = '') {
	loadingMedia = true;
	try {
		const headers: Record<string, string> = pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {};
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
	const headers: Record<string, string> = pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {};
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
				const existingRes = await fetch(`/api/collections/external_media/records?perPage=1&filter=${filter}`, {
					headers
				});
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

	async function loadTalks() {
		loading = true;
		try {
			const records = await await collection('talks').getList(1, 100, {
				sort: '-date,-sort_order'
			});
			talks = records.items as unknown as Talk[];
		} catch (err) {
			console.error('Failed to load talks:', err);
			toasts.add('error', 'Failed to load talks');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		title = '';
		slug = '';
		event = '';
		eventUrl = '';
		date = '';
		location = '';
		description = '';
		slidesUrl = '';
		videoUrl = '';
		visibility = 'public';
		isDraft = false;
		sortOrder = 0;
		mediaRefs = [];
		editingTalk = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function toggleShortcodes() {
		showShortcodes = !showShortcodes;
	}

	function openEditForm(talk: Talk) {
		editingTalk = talk;
		title = talk.title;
		slug = talk.slug || '';
		event = talk.event || '';
		eventUrl = talk.event_url || '';
		date = talk.date ? talk.date.split('T')[0] : '';
		location = talk.location || '';
		description = talk.description || '';
		slidesUrl = talk.slides_url || '';
		videoUrl = talk.video_url || '';
		visibility = talk.visibility;
		isDraft = talk.is_draft;
		sortOrder = talk.sort_order;
		mediaRefs = (talk as any).media_refs || [];
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

		saving = true;
		try {
			const resolvedRefs = await resolveMediaRefs(mediaRefs);
			const data = {
				title: title.trim(),
				slug: slug.trim(),
				event: event.trim(),
				event_url: eventUrl.trim(),
				date: date ? new Date(date).toISOString() : null,
				location: location.trim(),
				description: description,
				slides_url: slidesUrl.trim(),
				video_url: videoUrl.trim(),
				media_refs: resolvedRefs,
				visibility,
				is_draft: isDraft,
				sort_order: sortOrder
			};

			if (editingTalk) {
				await await collection('talks').update(editingTalk.id, data);
				toasts.add('success', 'Talk updated successfully');
			} else {
				await await collection('talks').create(data);
				toasts.add('success', 'Talk created successfully');
			}

			closeForm();
			await loadTalks();
		} catch (err) {
			console.error('Failed to save talk:', err);
			toasts.add('error', 'Failed to save talk');
		} finally {
			saving = false;
		}
	}

	async function deleteTalk(talk: Talk) {
		const confirmed = await confirm({
			title: 'Delete Talk',
			message: `Are you sure you want to delete "${talk.title}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await await collection('talks').delete(talk.id);
			toasts.add('success', 'Talk deleted');
			await loadTalks();
		} catch (err) {
			console.error('Failed to delete talk:', err);
			toasts.add('error', 'Failed to delete talk');
		}
	}

	async function togglePublish(talk: Talk) {
		try {
			await await collection('talks').update(talk.id, {
				is_draft: !talk.is_draft
			});
			toasts.add('success', talk.is_draft ? 'Talk published' : 'Talk unpublished');
			await loadTalks();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update talk');
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

	function selectAll() { selectedIds = new Set(talks.map(e => e.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('talks').update(id, { visibility });
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadTalks();
		} catch (err) {
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: 'Delete Talks',
			message: `Are you sure you want to delete ${ids.length} talk(s)? This action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;
		try {
			for (const id of ids) await collection('talks').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadTalks();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}
</script>

<svelte:head>
	<title>Talks | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={talks.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Talks & Presentations</h1>
		<div class="flex items-center gap-2">
			{#if talks.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					on:click={toggleSelectMode}
				>
					{selectMode ? 'Cancel' : 'Select'}
				</button>
			{/if}
			<button class="btn btn-primary" on:click={openNewForm}>
				+ New Talk
			</button>
		</div>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading talks...</div>
		</div>
	{:else if showForm}
		<!-- Talk Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingTalk ? 'Edit Talk' : 'New Talk'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" on:click={closeForm}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="title" class="label">Title *</label>
					<input
						type="text"
						id="title"
						bind:value={title}
						on:input={handleTitleChange}
						class="input"
						placeholder="Building Scalable APIs with Go"
						required
					/>
				</div>

				<div>
					<label for="slug" class="label">
						Slug
						<span class="font-normal text-gray-500">(for URL /talks/your-slug)</span>
					</label>
					<div class="flex gap-2">
						<input
							type="text"
							id="slug"
							bind:value={slug}
							class="input flex-1"
							placeholder="building-scalable-apis-with-go"
						/>
						<button
							type="button"
							class="btn btn-secondary text-sm"
							on:click={() => { slug = generateSlug(title); }}
							title="Generate from title"
						>
							Generate
						</button>
					</div>
					<p class="text-xs text-gray-500 mt-1">Leave empty to hide from public talks listing</p>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="event" class="label">Event</label>
						<input
							type="text"
							id="event"
							bind:value={event}
							class="input"
							placeholder="GopherCon 2024"
						/>
					</div>

					<div>
						<label for="event_url" class="label">Event URL</label>
						<input
							type="url"
							id="event_url"
							bind:value={eventUrl}
							class="input"
							placeholder="https://gophercon.com"
						/>
					</div>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="date" class="label">Date</label>
						<input
							type="date"
							id="date"
							bind:value={date}
							class="input"
						/>
					</div>

					<div>
						<label for="location" class="label">Location</label>
						<input
							type="text"
							id="location"
							bind:value={location}
							class="input"
							placeholder="San Francisco, CA"
						/>
					</div>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="description" class="label mb-0">Description</label>
						<AIContentHelper
							fieldType="description"
							content={description}
							context={{ title, event, location }}
							on:apply={(e) => (description = e.detail.content)}
						/>
					</div>
					<textarea
						id="description"
						bind:value={description}
						class="input min-h-[150px]"
						placeholder="A brief description of your talk... (Markdown + media shortcodes)"
					></textarea>
					<div class="mt-2 flex items-center gap-3 text-xs text-gray-600 dark:text-gray-400">
						<button type="button" class="btn btn-ghost btn-sm" on:click={toggleShortcodes}>Media shortcodes</button>
						<span>Use {'{{provider:url}}'} for embeds (YouTube, Vimeo, SoundCloud, images, etc.).</span>
					</div>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Media & Links</h2>

				<div>
					<label for="video_url" class="label">Video URL</label>
					<input
						type="url"
						id="video_url"
						bind:value={videoUrl}
						class="input"
						placeholder="https://youtube.com/watch?v=..."
					/>
					<p class="text-xs text-gray-500 mt-1">YouTube and Vimeo links will be embedded automatically</p>
				</div>

				<div>
					<label for="slides_url" class="label">Slides URL</label>
					<input
						type="url"
						id="slides_url"
						bind:value={slidesUrl}
						class="input"
						placeholder="https://speakerdeck.com/... or https://slides.com/..."
					/>
				</div>

				<div>
					<p class="label">Attached media / embeds</p>
					<div class="flex flex-wrap items-center gap-2 mb-2 text-sm text-gray-600 dark:text-gray-400">
						<input
							class="input w-full md:w-64"
							placeholder="Search media..."
							bind:value={mediaSearch}
							on:keydown={(e) => e.key === 'Enter' && loadMediaOptions(mediaSearch)}
						/>
						<button type="button" class="btn btn-secondary btn-sm" on:click={() => loadMediaOptions(mediaSearch)} aria-busy={loadingMedia}>
							{loadingMedia ? 'Searching…' : 'Search'}
						</button>
						<button type="button" class="btn btn-ghost btn-sm" on:click={() => { mediaSearch = ''; loadMediaOptions(''); }}>
							Clear
						</button>
					</div>
					{#if loadingMedia}
						<p class="text-sm text-gray-500 dark:text-gray-400">Loading media…</p>
					{:else if mediaOptions.length === 0}
						<p class="text-sm text-gray-500 dark:text-gray-400">No media found. Add uploads or external links in the Media Library.</p>
					{:else}
						<div class="flex flex-col gap-2 max-h-48 overflow-y-auto pr-2">
							{#each mediaOptions as opt}
								<label
									class={`flex items-center gap-2 px-3 py-2 rounded border cursor-pointer ${
										mediaRefs.includes(opt.id)
											? 'bg-primary-50 border-primary-200 text-primary-700'
											: 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700'
									}`}
								>
									<input
										type="checkbox"
										class="w-4 h-4"
										bind:group={mediaRefs}
										value={opt.id}
									/>
									<div class="flex flex-col">
										<span class="text-sm font-medium">{opt.title}</span>
										{#if opt.provider}
											<span class="text-xs text-gray-500">{opt.provider}</span>
										{/if}
									</div>
								</label>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Settings</h2>

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
						<input
							type="number"
							id="sort_order"
							bind:value={sortOrder}
							class="input"
							min="0"
						/>
						<p class="text-xs text-gray-500 mt-1">Higher numbers appear first</p>
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
				<button type="button" class="btn btn-secondary" on:click={closeForm}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingTalk ? 'Update Talk' : 'Create Talk'}
				</button>
			</div>
		</form>
		{#if showShortcodes}
			<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
				<div class="bg-white dark:bg-gray-900 rounded-lg shadow-lg max-w-2xl w-full p-6 space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Media shortcodes</h3>
						<button class="btn btn-ghost btn-sm" on:click={toggleShortcodes}>Close</button>
					</div>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						Embed media in Markdown using <code>{'{{provider:url}}'}</code>. Paste URLs from Media Library (uploads or external) or any supported provider.
					</p>
					<div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm text-gray-700 dark:text-gray-200">
						<div>
							<p class="font-semibold">Video</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{youtube:https://youtu.be/ID}}'}</code></li>
								<li><code>{'{{vimeo:https://vimeo.com/ID}}'}</code></li>
								<li><code>{'{{loom:https://www.loom.com/share/ID}}'}</code></li>
								<li><code>{'{{video:https://example.com/video.mp4}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">Audio</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{soundcloud:https://soundcloud.com/...}}'}</code></li>
								<li><code>{'{{spotify:https://open.spotify.com/track/...}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">Images / Docs</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{image:https://.../image.jpg}}'}</code></li>
								<li><code>{'{{pdf:https://.../file.pdf}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">Design / Code</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{figma:https://www.figma.com/file/...}}'}</code></li>
								<li><code>{'{{codepen:https://codepen.io/user/pen/...}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">Immich / Other</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{immich:https://immich.example.com/...}}'}</code></li>
								<li><code>{'{{embed:https://any-link}}'}</code></li>
							</ul>
						</div>
					</div>
				</div>
			</div>
		{/if}
	{:else if talks.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No talks yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your conference talks, presentations, and speaking engagements.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>
				+ Add Your First Talk
			</button>
		</div>
	{:else}
		<!-- Talks List -->
		<div class="space-y-4">
			{#each talks as talk (talk.id)}
				<div class="card p-4 {selectMode && selectedIds.has(talk.id) ? 'ring-2 ring-primary-500' : ''}">
					<div class="flex items-start justify-between gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(talk.id)}
								on:change={() => toggleSelect(talk.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<h3 class="font-medium text-gray-900 dark:text-white truncate">
									{talk.title}
								</h3>
								{#if talk.is_draft}
									<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
										Draft
									</span>
								{/if}
								{#if talk.visibility !== 'public'}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{talk.visibility}
									</span>
								{/if}
							</div>

							<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
								{#if talk.event}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
										</svg>
										{talk.event}
									</span>
								{/if}
								{#if talk.date}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
										</svg>
										{formatDate(talk.date)}
									</span>
								{/if}
								{#if talk.location}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
										</svg>
										{talk.location}
									</span>
								{/if}
							</div>

							<div class="flex gap-3 mt-2">
								{#if talk.slug}
									<a href="/talks/{talk.slug}" target="_blank" rel="noopener noreferrer" class="text-xs text-gray-600 dark:text-gray-400 hover:underline flex items-center gap-1">
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
										</svg>
										/talks/{talk.slug}
									</a>
								{/if}
								{#if talk.video_url}
									<a href={talk.video_url} target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 dark:text-primary-400 hover:underline flex items-center gap-1">
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
										Video
									</a>
								{/if}
								{#if talk.slides_url}
									<a href={talk.slides_url} target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 dark:text-primary-400 hover:underline flex items-center gap-1">
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
										</svg>
										Slides
									</a>
								{/if}
							</div>
						</div>

						<div class="flex items-center gap-2">
							<button
								class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
								on:click={() => togglePublish(talk)}
								title={talk.is_draft ? 'Publish' : 'Unpublish'}
							>
								{#if talk.is_draft}
									<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
									</svg>
								{:else}
									<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.542 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
									</svg>
								{/if}
							</button>
							<button
								class="p-2 text-gray-500 hover:text-blue-600"
								on:click={() => openEditForm(talk)}
								title="Edit"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="p-2 text-gray-500 hover:text-red-600"
								on:click={() => deleteTalk(talk)}
								title="Delete"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
								</svg>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

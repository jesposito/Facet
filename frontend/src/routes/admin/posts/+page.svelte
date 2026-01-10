<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
import { pb, type Post } from '$lib/pocketbase';
import { collection } from '$lib/stores/demo';
import { toasts, confirm } from '$lib/stores';
import { formatDate } from '$lib/utils';
import AIContentHelper from '$components/admin/AIContentHelper.svelte';
import BulkActionBar from '$components/admin/BulkActionBar.svelte';

let posts: Post[] = $state([]);
let loading = $state(true);
let showForm = $state(false);
let editingPost: Post | null = $state(null);
let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});
let mediaRefs: string[] = $state([]);
let mediaOptions: { id: string; title: string; provider?: string; url?: string }[] = $state([]);
let mediaSearch = $state('');
let loadingMedia = $state(false);
let showShortcodes = $state(false);

let selectMode = $state(false);
let selectedIds: Set<string> = $state(new Set());

	// Form fields
	let title = $state('');
	let slug = $state('');
	let excerpt = $state('');
	let content = $state('');
	let tags: string[] = $state([]);
	let tagInput = $state('');
	let visibility = $state('public');
	let isDraft = $state(true);
	let publishedAt = $state('');
	let saving = $state(false);

	// Simple pattern - admin layout handles auth
onMount(loadPosts);
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
			// Mirror upload into external_media so it can be stored in media_refs
			try {
				const filter = encodeURIComponent(`url="${opt.url}"`);
				const existingRes = await fetch(`/api/collections/external_media/records?perPage=1&filter=${filter}`, { headers });
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

async function loadPosts() {
	loading = true;
	try {
		const [records, membershipResp] = await Promise.all([
			await collection('posts').getList(1, 100, {
				sort: '-published_at'
			}),
			fetch('/api/admin/view-memberships?collection=posts', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			}).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed memberships'))))
		]);

		posts = records.items as unknown as Post[];
		memberships = (membershipResp.memberships as typeof memberships) || {};
	} catch (err) {
		console.error('Failed to load posts:', err);
		toasts.add('error', 'Failed to load posts');
	} finally {
		loading = false;
		}
	}

function resetForm() {
	title = '';
	slug = '';
	excerpt = '';
	content = '';
	mediaRefs = [];
	tags = [];
	tagInput = '';
	visibility = 'public';
	isDraft = true;
	publishedAt = '';
		editingPost = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

function openEditForm(post: Post) {
	editingPost = post;
	title = post.title;
	slug = post.slug || '';
	excerpt = post.excerpt || '';
	content = post.content || '';
	mediaRefs = (post as any).media_refs || [];
	tags = post.tags || [];
	visibility = post.visibility;
	isDraft = post.is_draft;
	publishedAt = post.published_at ? post.published_at.split('T')[0] : '';
	showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	function toggleShortcodes() {
		showShortcodes = !showShortcodes;
	}

	function generateSlug() {
		slug = title
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	function addTag() {
		const tag = tagInput.trim();
		if (tag && !tags.includes(tag)) {
			tags = [...tags, tag];
		}
		tagInput = '';
	}

	function removeTag(tag: string) {
		tags = tags.filter(t => t !== tag);
	}

	function handleTagKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			addTag();
		}
	}

	async function handleSubmit() {
		if (!title.trim()) {
			toasts.add('error', 'Title is required');
			return;
		}
		if (!slug.trim()) {
			toasts.add('error', 'Slug is required');
			return;
		}

		saving = true;
		try {
			const resolvedRefs = await resolveMediaRefs(mediaRefs);
			const data = {
				title: title.trim(),
				slug: slug.trim(),
				excerpt: excerpt.trim(),
				content: content,
				media_refs: resolvedRefs,
				tags: tags,
				visibility,
				is_draft: isDraft,
				published_at: publishedAt ? new Date(publishedAt).toISOString() : null
			};

			if (editingPost) {
				await await collection('posts').update(editingPost.id, data);
				toasts.add('success', 'Post updated successfully');
			} else {
				await await collection('posts').create(data);
				toasts.add('success', 'Post created successfully');
			}

			closeForm();
			await loadPosts();
		} catch (err: unknown) {
			console.error('Failed to save post:', err);
			const error = err as { data?: { data?: { slug?: { message?: string } } } };
			if (error.data?.data?.slug?.message) {
				toasts.add('error', 'Slug already exists');
			} else {
				toasts.add('error', 'Failed to save post');
			}
		} finally {
			saving = false;
		}
	}

	async function deletePost(post: Post) {
		const confirmed = await confirm({
			title: 'Delete Post',
			message: `Are you sure you want to delete "${post.title}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await await collection('posts').delete(post.id);
			toasts.add('success', 'Post deleted');
			await loadPosts();
		} catch (err) {
			console.error('Failed to delete post:', err);
			toasts.add('error', 'Failed to delete post');
		}
	}

	async function togglePublish(post: Post) {
		try {
			const newDraftState = !post.is_draft;
			await await collection('posts').update(post.id, {
				is_draft: newDraftState,
				published_at: newDraftState ? null : (post.published_at || new Date().toISOString())
			});
			toasts.add('success', newDraftState ? 'Post unpublished' : 'Post published');
			await loadPosts();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update post');
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

	function selectAll() { selectedIds = new Set(posts.map(e => e.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('posts').update(id, { visibility });
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadPosts();
		} catch (err) {
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: 'Delete Posts',
			message: `Are you sure you want to delete ${ids.length} post(s)? This action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;
		try {
			for (const id of ids) await collection('posts').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadPosts();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}
</script>

<svelte:head>
	<title>Posts | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={posts.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Posts</h1>
		<div class="flex items-center gap-2">
			{#if posts.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={toggleSelectMode}
				>
					{selectMode ? 'Cancel' : 'Select'}
				</button>
			{/if}
			<button class="btn btn-primary" onclick={openNewForm}>
				+ New Post
			</button>
		</div>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading posts...</div>
		</div>
	{:else if showForm}
		<!-- Post Form -->
		<form onsubmit={preventDefault(handleSubmit)} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingPost ? 'Edit Post' : 'New Post'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" onclick={closeForm} aria-label="Close form">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
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
						class="input"
						placeholder="My Awesome Post"
						required
						onblur={() => !slug && generateSlug()}
					/>
				</div>

				<div>
					<label for="slug" class="label">
						Slug *
						<button type="button" class="text-xs text-primary-600 ml-2" onclick={generateSlug}>
							Generate from title
						</button>
					</label>
					<input
						type="text"
						id="slug"
						bind:value={slug}
						class="input"
						placeholder="my-awesome-post"
						required
					/>
					<p class="text-xs text-gray-500 mt-1">URL: /posts/{slug || 'my-awesome-post'}</p>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="excerpt" class="label mb-0">Excerpt</label>
						<AIContentHelper
							fieldType="summary"
							content={excerpt}
							context={{ title, tags: tags.join(', ') }}
							on:apply={(e) => (excerpt = e.detail.content)}
						/>
					</div>
					<textarea
						id="excerpt"
						bind:value={excerpt}
						class="input"
						rows="2"
						placeholder="A brief summary of the post..."
					></textarea>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="content" class="label mb-0">Content (Markdown)</label>
						<AIContentHelper
							fieldType="content"
							content={content}
							context={{ title, excerpt, tags: tags.join(', ') }}
							on:apply={(e) => (content = e.detail.content)}
							size="sm"
						/>
					</div>
					<textarea
						id="content"
						bind:value={content}
						class="input min-h-[300px] font-mono text-sm"
						placeholder="Write your post content here... (Markdown + media shortcodes)"
					></textarea>
					<div class="mt-2 flex items-center gap-3 text-xs text-gray-600 dark:text-gray-400">
						<button type="button" class="btn btn-ghost btn-sm" onclick={toggleShortcodes}>Media shortcodes</button>
						<span>Use {'{{provider:url}}'} (youtube, vimeo, soundcloud, spotify, image, video, pdf, figma, codepen)</span>
					</div>
				</div>

			<div>
				<p class="label">Attached media / embeds</p>
				<div class="flex flex-wrap items-center gap-2 mb-2 text-sm text-gray-600 dark:text-gray-400">
					<input
						class="input w-full md:w-64"
						placeholder="Search media..."
						bind:value={mediaSearch}
						onkeydown={(e) => e.key === 'Enter' && loadMediaOptions(mediaSearch)}
					/>
					<button type="button" class="btn btn-secondary btn-sm" onclick={() => loadMediaOptions(mediaSearch)} aria-busy={loadingMedia}>
						{loadingMedia ? 'Searching…' : 'Search'}
					</button>
					<button type="button" class="btn btn-ghost btn-sm" onclick={() => { mediaSearch = ''; loadMediaOptions(''); }}>
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

				<div>
					<span class="label">Tags</span>
					<div class="flex flex-wrap gap-2 mb-2">
						{#each tags as tag}
							<span class="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm">
								{tag}
								<button type="button" class="text-gray-500 hover:text-red-500" onclick={() => removeTag(tag)} aria-label="Remove tag {tag}">
									<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</span>
						{/each}
					</div>
					<div class="flex gap-2">
						<input
							type="text"
							bind:value={tagInput}
							class="input flex-1"
							placeholder="Add a tag..."
							onkeydown={handleTagKeydown}
						/>
						<button type="button" class="btn btn-secondary" onclick={addTag}>Add</button>
					</div>
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
						<label for="published_at" class="label">Publish Date</label>
						<input
							type="date"
							id="published_at"
							bind:value={publishedAt}
							class="input"
						/>
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
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingPost ? 'Update Post' : 'Create Post'}
				</button>
			</div>
		</form>
		{#if showShortcodes}
			<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
				<div class="bg-white dark:bg-gray-900 rounded-lg shadow-lg max-w-2xl w-full p-6 space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Media shortcodes</h3>
						<button class="btn btn-ghost btn-sm" onclick={toggleShortcodes}>Close</button>
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
	{:else if posts.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No posts yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Start writing to share your thoughts and ideas.
			</p>
			<button class="btn btn-primary" onclick={openNewForm}>
				Write your first post
			</button>
		</div>
	{:else}
		<!-- Posts List -->
		<div class="space-y-4">
			{#each posts as post (post.id)}
				<div class="card p-4 hover:shadow-md transition-shadow {selectMode && selectedIds.has(post.id) ? 'ring-2 ring-primary-500' : ''}">
					<div class="flex items-start gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(post.id)}
								onchange={() => toggleSelect(post.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1">
								<h3 class="text-lg font-medium text-gray-900 dark:text-white truncate">
									{post.title}
								</h3>
								{#if post.is_draft}
									<span class="px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">
										Draft
									</span>
								{:else}
									<span class="px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 rounded">
										Published
									</span>
								{/if}
								<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded capitalize">
									{post.visibility}
								</span>
							</div>

							{#if post.excerpt}
								<p class="text-sm text-gray-600 dark:text-gray-400 mb-2 line-clamp-2">
									{post.excerpt}
								</p>
							{/if}

							{#if memberships[post.id]?.length}
								<div class="flex flex-wrap gap-1 mb-2">
									{#each memberships[post.id].slice(0, 3) as viewRef}
										<a
											href={`/admin/views/${viewRef.id}`}
											target="_blank"
											class="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
											title={`Open view: ${viewRef.name}`}
										>
											{viewRef.slug || viewRef.name}
										</a>
									{/each}
									{#if memberships[post.id].length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-400">
											+{memberships[post.id].length - 3}
										</span>
									{/if}
								</div>
							{:else}
								<p class="text-xs text-gray-500 dark:text-gray-400 mb-2">Not in any view</p>
							{/if}

							<div class="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
								{#if post.slug}
									<span>/posts/{post.slug}</span>
								{/if}
								{#if post.published_at}
									<span>Published: {formatDate(post.published_at, { month: 'short', day: 'numeric', year: 'numeric' })}</span>
								{/if}
								{#if post.tags && post.tags.length > 0}
									<span class="flex items-center gap-1">
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
										</svg>
										{post.tags.length} {post.tags.length === 1 ? 'tag' : 'tags'}
									</span>
								{/if}
							</div>
						</div>

						<div class="flex items-center gap-2 shrink-0">
							{#if post.slug && !post.is_draft && post.visibility === 'public'}
								<a
									href="/posts/{post.slug}"
									target="_blank"
									class="btn btn-ghost btn-sm"
									title="View post"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
									</svg>
								</a>
							{/if}
							<button
								class="btn btn-ghost btn-sm"
								title={post.is_draft ? 'Publish' : 'Unpublish'}
								onclick={() => togglePublish(post)}
							>
								{#if post.is_draft}
									<svg class="w-4 h-4 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
								{:else}
									<svg class="w-4 h-4 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
									</svg>
								{/if}
							</button>
							<button
								class="btn btn-ghost btn-sm"
								title="Edit"
								onclick={() => openEditForm(post)}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="btn btn-ghost btn-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
								title="Delete"
								onclick={() => deletePost(post)}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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

<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
import { pb, type Post, getFileUrl } from '$lib/pocketbase';
import { t } from 'svelte-i18n';
import { collection } from '$lib/stores/demo';
import { toasts, confirm } from '$lib/stores';
import { createAutosave } from '$lib/stores/autosave';
import { createFilterState } from '$lib/admin/filterState.svelte';
import { formatDate, toDateInputValue } from '$lib/utils';
import AIContentHelper from '$components/admin/AIContentHelper.svelte';
import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
import BulkActionBar from '$components/admin/BulkActionBar.svelte';
import MediaPicker from '$lib/components/admin/MediaPicker.svelte';
import PageHelp from '$components/admin/PageHelp.svelte';
import AdminFilters from '$components/admin/AdminFilters.svelte';

let posts: Post[] = $state([]);
let loading = $state(true);
let showForm = $state(false);
let editingPost: Post | null = $state(null);
let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});
let mediaRefs: string[] = $state([]);
let mediaPickerRef: MediaPicker | undefined = $state();
let showShortcodes = $state(false);

let selectMode = $state(false);
let selectedIds: Set<string> = $state(new Set());

const filterStore = createFilterState({
	enableSearch: true,
	enableVisibilityFilter: true,
	enableDraftFilter: true,
	enableTagFilter: false,
	searchPlaceholder: 'Search posts...'
});
let showAdvancedFilters = $state(false);

let filteredPosts = $derived(
	filterStore.filterItems(
		posts,
		(post) => `${post.title} ${post.excerpt || ''} ${post.content || ''} ${(post.tags || []).join(' ')}`,
		(post) => post.visibility,
		(post) => post.is_draft,
		() => []
	)
);

const autosave = createAutosave('admin-posts', { saveDelay: 1500 });
let showRecoveryBanner = $state(false);
let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

function getFormData() {
	return { title, slug, excerpt, content, tags, visibility, isDraft, featured, publishedAt, mediaRefs };
}

function restoreFromDraft(data: Record<string, any>) {
	title = data.title || '';
	slug = data.slug || '';
	excerpt = data.excerpt || '';
	content = data.content || '';
	tags = data.tags || [];
	visibility = data.visibility || 'public';
	isDraft = data.isDraft !== false;
	featured = data.featured === true;
	publishedAt = data.publishedAt || '';
	mediaRefs = data.mediaRefs || [];
}

function handleFormChange() {
	if (showForm) {
		autosave.save(getFormData(), !!editingPost, editingPost?.id);
	}
}

afterNavigate(() => {
	showForm = false;
	editingPost = null;
	selectMode = false;
	selectedIds = new Set();
	
	const draft = autosave.loadDraft();
	if (draft?.data && Object.values(draft.data).some(v => v !== '' && v !== false && v !== 0 && !(Array.isArray(v) && v.length === 0))) {
		recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
		showRecoveryBanner = true;
	} else {
		showRecoveryBanner = false;
		recoveryData = null;
	}
});

	// Form fields
	let title = $state('');
	let slug = $state('');
	let excerpt = $state('');
	let content = $state('');
	let tags: string[] = $state([]);
	let tagInput = $state('');
	let visibility = $state('public');
	let isDraft = $state(true);
	let featured = $state(false);
	let publishedAt = $state('');
	let saving = $state(false);
	let coverImageFile: FileList | null = $state(null);

	// Simple pattern - admin layout handles auth
onMount(loadPosts);

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
	featured = false;
	publishedAt = '';
	coverImageFile = null;
		editingPost = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function handleRestoreDraft() {
		const draft = autosave.loadDraft();
		if (draft?.data) {
			restoreFromDraft(draft.data);
			if (draft.isEditing && draft.editingId) {
				const post = posts.find(p => p.id === draft.editingId);
				if (post) editingPost = post;
			}
			showForm = true;
		}
		showRecoveryBanner = false;
	}

	function handleDismissDraft() {
		autosave.clearDraft();
		showRecoveryBanner = false;
		recoveryData = null;
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
	featured = post.featured || false;
	publishedAt = toDateInputValue(post.published_at);
	coverImageFile = null;
	showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
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

	function getCoverImageUrl(post: Post): string {
		if (!(post as any).cover_image) return '';
		return getFileUrl(post, (post as any).cover_image);
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
			const resolvedRefs = mediaPickerRef ? await mediaPickerRef.resolveMediaRefs(mediaRefs) : mediaRefs;

			// Use FormData to support file uploads
			const formData = new FormData();
			formData.append('title', title.trim());
			formData.append('slug', slug.trim());
			formData.append('excerpt', excerpt.trim());
			formData.append('content', content);
			formData.append('media_refs', JSON.stringify(resolvedRefs));
			formData.append('tags', JSON.stringify(tags));
			formData.append('visibility', visibility);
			formData.append('is_draft', String(isDraft));
			formData.append('featured', String(featured));
			if (publishedAt) {
				formData.append('published_at', new Date(publishedAt).toISOString());
			}

			if (coverImageFile && coverImageFile.length > 0) {
				formData.append('cover_image', coverImageFile[0]);
			}

			if (editingPost) {
				await collection('posts').update(editingPost.id, formData);
				toasts.add('success', 'Post updated successfully');
			} else {
				await collection('posts').create(formData);
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
	<title>{$t('admin.content.posts.title')} {$t('admin.content.common.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="posts">
		{@html $t('admin.content.posts.help_text')}
		{$t('admin.content.posts.help_tip_1')}
		{@html $t('admin.content.posts.help_tip_2')}
	</PageHelp>

	{#if showRecoveryBanner && recoveryData}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

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
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.content.posts.title')}</h1>
		<div class="flex items-center gap-2">
			{#if posts.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={toggleSelectMode}
				>
					{selectMode ? $t('admin.content.common.cancel') : $t('admin.content.common.select')}
				</button>
			{/if}
			<button class="btn btn-primary" onclick={openNewForm}>
				+ {$t('admin.content.common.new_button', { values: { type: $t('admin.content.posts.title').slice(0, -1) } })}
			</button>
		</div>
	</div>

	<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} availableTags={[]} />

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.content.common.loading', { values: { type: $t('admin.content.posts.title').toLowerCase() } })}</div>
		</div>
	{:else if showForm}
		<!-- Post Form -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
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

			<MediaPicker
				bind:this={mediaPickerRef}
				bind:value={mediaRefs}
				label={$t('admin.content.common.attached_media')}
				showHelp={true}
			/>

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

				<div>
					<label for="cover_image" class="label">Cover Image</label>
					<input
						type="file"
						id="cover_image"
						accept="image/*"
						bind:files={coverImageFile}
						class="input"
					/>
					{#if editingPost && getCoverImageUrl(editingPost) && !coverImageFile}
						<div class="mt-2 flex items-center gap-2">
							<img
								src={getCoverImageUrl(editingPost)}
								alt="Current cover"
								class="w-20 h-20 object-cover rounded"
							/>
							<span class="text-sm text-gray-500">Current image</span>
						</div>
					{/if}
					<p class="text-xs text-gray-500 mt-1">Displayed in post grids and as the header image</p>
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
							type="text"
							id="published_at"
							bind:value={publishedAt}
							placeholder="2024-01-15"
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

				<div class="flex items-center gap-2">
					<input
						type="checkbox"
						id="featured"
						bind:checked={featured}
						class="w-4 h-4 text-primary-600 rounded border-gray-300"
					/>
					<label for="featured" class="text-sm text-gray-700 dark:text-gray-300">
						Featured post (highlight in featured layout)
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
	{:else if filteredPosts.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No posts match your filters</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Try adjusting your search or filter criteria.
			</p>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				Clear All Filters
			</button>
		</div>
	{:else}
		<!-- Posts List -->
		<div class="space-y-4">
			{#each filteredPosts as post (post.id)}
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
								{#if post.featured}
									<span class="px-2 py-0.5 text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200 rounded">
										Featured
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
							{#if post.slug}
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
								class="btn btn-danger-ghost btn-sm"
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

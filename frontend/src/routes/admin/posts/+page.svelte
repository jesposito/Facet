<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Post } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { formatDate } from '$lib/utils';

	let posts: Post[] = [];
	let loading = true;
	let showForm = false;
	let editingPost: Post | null = null;

	// Form fields
	let title = '';
	let slug = '';
	let excerpt = '';
	let content = '';
	let tags: string[] = [];
	let tagInput = '';
	let visibility = 'public';
	let isDraft = true;
	let publishedAt = '';
	let saving = false;

	// Simple pattern - admin layout handles auth
	onMount(loadPosts);

	async function loadPosts() {
		loading = true;
		try {
			const records = await pb.collection('posts').getList(1, 100, {
				sort: '-created'
			});
			posts = records.items as unknown as Post[];
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
			const data = {
				title: title.trim(),
				slug: slug.trim(),
				excerpt: excerpt.trim(),
				content: content,
				tags: tags,
				visibility,
				is_draft: isDraft,
				published_at: publishedAt ? new Date(publishedAt).toISOString() : null
			};

			if (editingPost) {
				await pb.collection('posts').update(editingPost.id, data);
				toasts.add('success', 'Post updated successfully');
			} else {
				await pb.collection('posts').create(data);
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
		if (!confirm(`Are you sure you want to delete "${post.title}"?`)) {
			return;
		}

		try {
			await pb.collection('posts').delete(post.id);
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
			await pb.collection('posts').update(post.id, {
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
</script>

<svelte:head>
	<title>Posts | Me.yaml Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Posts</h1>
		<button class="btn btn-primary" on:click={openNewForm}>
			+ New Post
		</button>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading posts...</div>
		</div>
	{:else if showForm}
		<!-- Post Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingPost ? 'Edit Post' : 'New Post'}
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
						class="input"
						placeholder="My Awesome Post"
						required
						on:blur={() => !slug && generateSlug()}
					/>
				</div>

				<div>
					<label for="slug" class="label">
						Slug *
						<button type="button" class="text-xs text-primary-600 ml-2" on:click={generateSlug}>
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
					<label for="excerpt" class="label">Excerpt</label>
					<textarea
						id="excerpt"
						bind:value={excerpt}
						class="input"
						rows="2"
						placeholder="A brief summary of the post..."
					></textarea>
				</div>

				<div>
					<label for="content" class="label">Content</label>
					<textarea
						id="content"
						bind:value={content}
						class="input min-h-[300px] font-mono text-sm"
						placeholder="Write your post content here... (Markdown supported)"
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown formatting is supported</p>
				</div>

				<div>
					<span class="label">Tags</span>
					<div class="flex flex-wrap gap-2 mb-2">
						{#each tags as tag}
							<span class="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm">
								{tag}
								<button type="button" class="text-gray-500 hover:text-red-500" on:click={() => removeTag(tag)}>
									<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
							on:keydown={handleTagKeydown}
						/>
						<button type="button" class="btn btn-secondary" on:click={addTag}>Add</button>
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
				<button type="button" class="btn btn-secondary" on:click={closeForm}>Cancel</button>
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
	{:else if posts.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No posts yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Start writing to share your thoughts and ideas.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>
				Write your first post
			</button>
		</div>
	{:else}
		<!-- Posts List -->
		<div class="space-y-4">
			{#each posts as post (post.id)}
				<div class="card p-4 hover:shadow-md transition-shadow">
					<div class="flex items-start gap-4">
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
								on:click={() => togglePublish(post)}
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
								on:click={() => openEditForm(post)}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="btn btn-ghost btn-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
								title="Delete"
								on:click={() => deletePost(post)}
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

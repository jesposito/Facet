<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Project, getFileUrl } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import AIImproveButton from '$components/admin/AIImproveButton.svelte';

	let projects: Project[] = [];
	let loading = true;
	let showForm = false;
	let editingProject: Project | null = null;

	// Form fields
	let title = '';
	let slug = '';
	let summary = '';
	let description = '';
	let techStackText = '';
	let links: Array<{ type: string; url: string }> = [];
	let categoriesText = '';
	let visibility = 'public';
	let isDraft = false;
	let isFeatured = false;
	let sortOrder = 0;
	let coverImageFile: FileList | null = null;
	let saving = false;

	onMount(loadProjects);

	async function loadProjects() {
		loading = true;
		try {
			const records = await pb.collection('projects').getList(1, 100, {
				sort: '-is_featured,-sort_order'
			});
			projects = records.items as unknown as Project[];
		} catch (err) {
			console.error('Failed to load projects:', err);
			toasts.add('error', 'Failed to load projects');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		title = '';
		slug = '';
		summary = '';
		description = '';
		techStackText = '';
		links = [];
		categoriesText = '';
		visibility = 'public';
		isDraft = false;
		isFeatured = false;
		sortOrder = 0;
		coverImageFile = null;
		editingProject = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(project: Project) {
		editingProject = project;
		title = project.title;
		slug = project.slug || '';
		summary = project.summary || '';
		description = project.description || '';
		techStackText = (project.tech_stack || []).join(', ');
		links = project.links || [];
		categoriesText = (project.categories || []).join(', ');
		visibility = project.visibility;
		isDraft = project.is_draft;
		isFeatured = project.is_featured;
		sortOrder = project.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	function addLink() {
		links = [...links, { type: 'website', url: '' }];
	}

	function removeLink(index: number) {
		links = links.filter((_, i) => i !== index);
	}

	function generateSlug() {
		slug = title
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	async function handleSubmit() {
		if (!title.trim()) {
			toasts.add('error', 'Project title is required');
			return;
		}

		saving = true;
		try {
			// Parse tech stack from comma-separated
			const techStack = techStackText
				.split(',')
				.map((s) => s.trim())
				.filter((s) => s);

			// Parse categories from comma-separated
			const categories = categoriesText
				.split(',')
				.map((s) => s.trim())
				.filter((s) => s);

			// Filter out empty links
			const validLinks = links.filter((l) => l.url.trim());

			const formData = new FormData();
			formData.append('title', title.trim());
			if (slug.trim()) formData.append('slug', slug.trim());
			formData.append('summary', summary.trim());
			formData.append('description', description.trim());
			formData.append('tech_stack', JSON.stringify(techStack));
			formData.append('links', JSON.stringify(validLinks));
			formData.append('categories', JSON.stringify(categories));
			formData.append('visibility', visibility);
			formData.append('is_draft', String(isDraft));
			formData.append('is_featured', String(isFeatured));
			formData.append('sort_order', String(sortOrder));

			if (coverImageFile && coverImageFile.length > 0) {
				formData.append('cover_image', coverImageFile[0]);
			}

			if (editingProject) {
				await pb.collection('projects').update(editingProject.id, formData);
				toasts.add('success', 'Project updated successfully');
			} else {
				await pb.collection('projects').create(formData);
				toasts.add('success', 'Project created successfully');
			}

			closeForm();
			await loadProjects();
		} catch (err: any) {
			console.error('Failed to save project:', err);
			const message = err?.response?.data?.slug?.message || 'Failed to save project';
			toasts.add('error', message);
		} finally {
			saving = false;
		}
	}

	async function deleteProject(project: Project) {
		if (!confirm(`Are you sure you want to delete "${project.title}"?`)) {
			return;
		}

		try {
			await pb.collection('projects').delete(project.id);
			toasts.add('success', 'Project deleted');
			await loadProjects();
		} catch (err) {
			console.error('Failed to delete project:', err);
			toasts.add('error', 'Failed to delete project');
		}
	}

	async function togglePublish(project: Project) {
		try {
			await pb.collection('projects').update(project.id, {
				is_draft: !project.is_draft
			});
			toasts.add('success', project.is_draft ? 'Project published' : 'Project unpublished');
			await loadProjects();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update project');
		}
	}

	async function toggleFeatured(project: Project) {
		try {
			await pb.collection('projects').update(project.id, {
				is_featured: !project.is_featured
			});
			toasts.add('success', project.is_featured ? 'Project unfeatured' : 'Project featured');
			await loadProjects();
		} catch (err) {
			console.error('Failed to toggle featured:', err);
			toasts.add('error', 'Failed to update project');
		}
	}

	function getCoverImageUrl(project: Project): string {
		if (!project.cover_image) return '';
		return getFileUrl(
			{ id: project.id, collectionName: 'projects' },
			project.cover_image
		);
	}

	const linkTypes = [
		{ value: 'website', label: 'Website' },
		{ value: 'github', label: 'GitHub' },
		{ value: 'demo', label: 'Demo' },
		{ value: 'docs', label: 'Documentation' },
		{ value: 'npm', label: 'NPM' },
		{ value: 'other', label: 'Other' }
	];
</script>

<svelte:head>
	<title>Projects | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Projects</h1>
		<div class="flex gap-2">
			<a href="/admin/import" class="btn btn-secondary">
				Import from GitHub
			</a>
			<button class="btn btn-primary" on:click={openNewForm}>
				+ New Project
			</button>
		</div>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading projects...</div>
		</div>
	{:else if showForm}
		<!-- Project Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingProject ? 'Edit Project' : 'New Project'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" on:click={closeForm}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="title" class="label">Project Title *</label>
					<input
						type="text"
						id="title"
						bind:value={title}
						class="input"
						placeholder="My Awesome Project"
						required
						on:blur={() => !slug && generateSlug()}
					/>
				</div>

				<div>
					<label for="slug" class="label">URL Slug</label>
					<div class="flex gap-2">
						<input
							type="text"
							id="slug"
							bind:value={slug}
							class="input flex-1"
							placeholder="my-awesome-project"
						/>
						<button type="button" class="btn btn-secondary" on:click={generateSlug}>
							Generate
						</button>
					</div>
					<p class="text-xs text-gray-500 mt-1">Used in the URL: /projects/{slug || 'my-project'}</p>
				</div>

				<div>
					<div class="flex items-center justify-between">
						<label for="summary" class="label mb-0">Summary</label>
						<AIImproveButton
							contentType="summary"
							content={summary}
							context={{ project: title, technologies: techStackText }}
							action={summary ? 'improve' : 'generate'}
							label={summary ? 'Improve' : 'Generate'}
							on:result={(e) => (summary = e.detail.content)}
						/>
					</div>
					<textarea
						id="summary"
						bind:value={summary}
						class="input mt-1"
						rows="2"
						placeholder="A brief one-liner about the project"
					></textarea>
				</div>

				<div>
					<div class="flex items-center justify-between">
						<label for="description" class="label mb-0">Description</label>
						<AIImproveButton
							contentType="description"
							content={description}
							context={{ project: title, summary, technologies: techStackText }}
							action={description ? 'improve' : 'generate'}
							label={description ? 'Improve' : 'Generate'}
							on:result={(e) => (description = e.detail.content)}
						/>
					</div>
					<textarea
						id="description"
						bind:value={description}
						class="input min-h-[150px] mt-1"
						placeholder="Full project description with details about features, architecture, etc."
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown supported</p>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Technical Details</h2>

				<div>
					<label for="tech_stack" class="label">Tech Stack</label>
					<input
						type="text"
						id="tech_stack"
						bind:value={techStackText}
						class="input"
						placeholder="Go, TypeScript, Docker, PostgreSQL"
					/>
					<p class="text-xs text-gray-500 mt-1">Comma-separated list of technologies</p>
				</div>

				<div>
					<label for="categories" class="label">Categories</label>
					<input
						type="text"
						id="categories"
						bind:value={categoriesText}
						class="input"
						placeholder="web, devtools, open-source"
					/>
					<p class="text-xs text-gray-500 mt-1">Comma-separated list of categories</p>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Links</h2>
					<button type="button" class="btn btn-secondary btn-sm" on:click={addLink}>
						+ Add Link
					</button>
				</div>

				{#if links.length === 0}
					<p class="text-sm text-gray-500 dark:text-gray-400">
						No links added yet. Add links to your project's repo, demo, or documentation.
					</p>
				{:else}
					<div class="space-y-3">
						{#each links as link, i}
							<div class="flex gap-2">
								<select bind:value={link.type} class="input w-32">
									{#each linkTypes as lt}
										<option value={lt.value}>{lt.label}</option>
									{/each}
								</select>
								<input
									type="url"
									bind:value={link.url}
									class="input flex-1"
									placeholder="https://..."
								/>
								<button type="button" class="btn btn-secondary" on:click={() => removeLink(i)}>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Media</h2>

				<div>
					<label for="cover_image" class="label">Cover Image</label>
					<input
						type="file"
						id="cover_image"
						accept="image/*"
						bind:files={coverImageFile}
						class="input"
					/>
					{#if editingProject?.cover_image && !coverImageFile}
						<div class="mt-2 flex items-center gap-2">
							<img
								src={getCoverImageUrl(editingProject)}
								alt="Current cover"
								class="w-20 h-20 object-cover rounded"
							/>
							<span class="text-sm text-gray-500">Current image</span>
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
					</div>
				</div>

				<div class="space-y-2">
					<div class="flex items-center gap-2">
						<input
							type="checkbox"
							id="is_draft"
							bind:checked={isDraft}
							class="w-4 h-4 text-primary-600 rounded border-gray-300"
						/>
						<label for="is_draft" class="text-sm text-gray-700 dark:text-gray-300">
							Save as draft
						</label>
					</div>

					<div class="flex items-center gap-2">
						<input
							type="checkbox"
							id="is_featured"
							bind:checked={isFeatured}
							class="w-4 h-4 text-primary-600 rounded border-gray-300"
						/>
						<label for="is_featured" class="text-sm text-gray-700 dark:text-gray-300">
							Featured project (appears first)
						</label>
					</div>
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
					{editingProject ? 'Update Project' : 'Create Project'}
				</button>
			</div>
		</form>
	{:else if projects.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No projects yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add projects to showcase your work, or import them from GitHub.
			</p>
			<div class="flex gap-2 justify-center">
				<a href="/admin/import" class="btn btn-secondary">
					Import from GitHub
				</a>
				<button class="btn btn-primary" on:click={openNewForm}>
					+ Add Manually
				</button>
			</div>
		</div>
	{:else}
		<!-- Projects Grid -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each projects as project (project.id)}
				<div class="card overflow-hidden">
					{#if project.cover_image}
						<div class="h-32 bg-gray-100 dark:bg-gray-800">
							<img
								src={getCoverImageUrl(project)}
								alt={project.title}
								class="w-full h-full object-cover"
							/>
						</div>
					{/if}
					<div class="p-4">
						<div class="flex items-start justify-between gap-2">
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 flex-wrap">
									<h3 class="font-medium text-gray-900 dark:text-white truncate">
										{project.title}
									</h3>
									{#if project.is_featured}
										<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
											Featured
										</span>
									{/if}
									{#if project.is_draft}
										<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
											Draft
										</span>
									{/if}
									{#if project.visibility !== 'public'}
										<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
											{project.visibility}
										</span>
									{/if}
								</div>

								{#if project.summary}
									<p class="text-sm text-gray-500 dark:text-gray-400 mt-1 line-clamp-2">
										{project.summary}
									</p>
								{/if}

								{#if project.tech_stack && project.tech_stack.length > 0}
									<div class="flex flex-wrap gap-1 mt-2">
										{#each project.tech_stack.slice(0, 4) as tech}
											<span class="px-2 py-0.5 text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded">
												{tech}
											</span>
										{/each}
										{#if project.tech_stack.length > 4}
											<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
												+{project.tech_stack.length - 4}
											</span>
										{/if}
									</div>
								{/if}
							</div>
						</div>

						<div class="flex items-center justify-between mt-4 pt-3 border-t border-gray-100 dark:border-gray-700">
							<div class="flex gap-1">
								{#if project.slug}
									<a
										href="/projects/{project.slug}"
										target="_blank"
										class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
										title="View"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
										</svg>
									</a>
								{/if}
							</div>

							<div class="flex items-center gap-1">
								<button
									class="p-2 text-gray-500 hover:text-yellow-600"
									on:click={() => toggleFeatured(project)}
									title={project.is_featured ? 'Unfeature' : 'Feature'}
								>
									<svg class="w-4 h-4" fill={project.is_featured ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
									</svg>
								</button>
								<button
									class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
									on:click={() => togglePublish(project)}
									title={project.is_draft ? 'Publish' : 'Unpublish'}
								>
									{#if project.is_draft}
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
										</svg>
									{:else}
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.542 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
										</svg>
									{/if}
								</button>
								<button
									class="p-2 text-gray-500 hover:text-blue-600"
									on:click={() => openEditForm(project)}
									title="Edit"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
									</svg>
								</button>
								<button
									class="p-2 text-gray-500 hover:text-red-600"
									on:click={() => deleteProject(project)}
									title="Delete"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
									</svg>
								</button>
							</div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

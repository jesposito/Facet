<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { browser } from '$app/environment';
	import { flip } from 'svelte/animate';
	import { pb, type Project, getFileUrl } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import MultiMediaPicker from '$lib/components/admin/MultiMediaPicker.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import AdminTagSelector from '$components/admin/AdminTagSelector.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import SingleMediaPicker from '$lib/components/admin/SingleMediaPicker.svelte';
	import VisibilitySelector from '$components/admin/VisibilitySelector.svelte';

	let projects: Project[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingProject: Project | null = $state(null);

	// Form fields
	let title = $state('');
	let slug = $state('');
	let summary = $state('');
	let description = $state('');
	let techStackText = $state('');
	let links: Array<{ type: string; url: string }> = $state([]);
let categoriesText = $state('');
let visibility = $state('public');
let isDraft = $state(false);
let isFeatured = $state(false);
let sortOrder = $state(0);
let coverImageFile: FileList | null = $state(null);
let coverImageLibraryUrl = $state('');
let clearCoverImage = $state(false);
let mediaRefs: string[] = $state([]);
let mediaPickerRef: MultiMediaPicker | undefined = $state();
let showShortcodes = $state(false);
let saving = $state(false);
let commentsEnabled = $state(false);
let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});
let adminTagIds: string[] = $state([]);

let selectMode = $state(false);
let selectedIds: Set<string> = $state(new Set());

let reorderMode = $state(false);
let reorderItems: Array<{ id: string; title: string; is_featured: boolean; is_draft: boolean }> = $state([]);
let savingOrder = $state(false);
const flipDurationMs = 200;

let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
let dndLoaded = $state(false);

const filterStore = createFilterState({
	enableSearch: true,
	enableVisibilityFilter: true,
	enableDraftFilter: true,
	enableTagFilter: true,
	searchPlaceholder: 'Search projects...'
});
let availableTags: Array<{ id: string; name: string; color: string }> = $state([]);
let showAdvancedFilters = $state(false);

const autosave = createAutosave('admin-projects', { saveDelay: 1500 });
let showRecoveryBanner = $state(false);
let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

function getFormData() {
	return { title, slug, summary, description, techStackText, links, categoriesText, visibility, isDraft, isFeatured, sortOrder, mediaRefs, coverImageLibraryUrl, commentsEnabled };
}

function restoreFromDraft(data: Record<string, any>) {
	title = data.title || '';
	slug = data.slug || '';
	summary = data.summary || '';
	description = data.description || '';
	techStackText = data.techStackText || '';
	links = data.links || [];
	categoriesText = data.categoriesText || '';
	visibility = data.visibility || 'public';
	isDraft = data.isDraft || false;
	isFeatured = data.isFeatured || false;
	sortOrder = data.sortOrder || 0;
	mediaRefs = data.mediaRefs || [];
	coverImageLibraryUrl = data.coverImageLibraryUrl || '';
	commentsEnabled = data.commentsEnabled === true;
}

function handleFormChange() {
	if (showForm) {
		autosave.save(getFormData(), !!editingProject, editingProject?.id);
	}
}

afterNavigate(() => {
	showForm = false;
	editingProject = null;
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

onMount(loadProjects);
onMount(loadAvailableTags);
onMount(async () => {
	if (browser) {
		const { dndzone: dnd } = await import('svelte-dnd-action');
		dndzone = dnd;
		dndLoaded = true;
	}
});

async function loadAvailableTags() {
	try {
		const records = await collection('admin_tags').getList(1, 100, {
			sort: 'sort_order,name'
		});
		availableTags = records.items.map(tag => ({
			id: tag.id,
			name: tag.name,
			color: tag.color
		}));
	} catch (err) {
		console.error('Failed to load available tags:', err);
	}
}

let filteredProjects = $derived(
	filterStore.filterItems(
		projects,
		(proj) => `${proj.title} ${proj.summary || ''} ${proj.description || ''} ${(proj.tech_stack || []).join(' ')}`,
		(proj) => proj.visibility,
		(proj) => proj.is_draft,
		(proj) => (proj as any).admin_tags || []
	)
);

	async function loadProjects() {
	loading = true;
	try {
		const [projectResp, membershipResp] = await Promise.all([
			await collection('projects').getList(1, 100, {
				sort: '-is_featured,-sort_order',
				expand: 'admin_tags'
			}),
			fetch('/api/admin/view-memberships?collection=projects', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			}).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed memberships'))))
		]);
		projects = projectResp.items as unknown as Project[];
		memberships = (membershipResp.memberships as typeof memberships) || {};
	} catch (err) {
		console.error('Failed to load projects:', err);
		toasts.add('error', $t('admin.content.common.toast_load_error', { values: { type: $t('admin.content.projects.title').toLowerCase() } }));
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
		coverImageLibraryUrl = '';
		clearCoverImage = false;
	editingProject = null;
	mediaRefs = [];
	adminTagIds = [];
	commentsEnabled = false;
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
				const proj = projects.find(p => p.id === draft.editingId);
				if (proj) editingProject = proj;
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

	function toggleShortcodes() {
		showShortcodes = !showShortcodes;
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
	mediaRefs = project.media_refs || [];
	visibility = project.visibility;
	isDraft = project.is_draft;
	isFeatured = project.is_featured;
	sortOrder = project.sort_order;
	adminTagIds = project.admin_tags || [];
	coverImageLibraryUrl = (project as any).cover_image_library_url || '';
	clearCoverImage = false;
	commentsEnabled = (project as any).comments_enabled ?? false;
	showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
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
			toasts.add('error', $t('admin.content.projects.required_error'));
			return;
		}

		saving = true;
		try {
			const resolvedRefs = mediaPickerRef ? await mediaPickerRef.resolveMediaRefs(mediaRefs) : mediaRefs;
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
			formData.append('media_refs', JSON.stringify(resolvedRefs));
			formData.append('visibility', visibility);
			formData.append('is_draft', String(isDraft));
			formData.append('is_featured', String(isFeatured));
			formData.append('sort_order', String(sortOrder));
			formData.append('admin_tags', JSON.stringify(adminTagIds));
			formData.append('comments_enabled', String(commentsEnabled));

			if (coverImageFile && coverImageFile.length > 0) {
				formData.append('cover_image', coverImageFile[0]);
				// Clear library URL when uploading a new file
				formData.append('cover_image_library_url', '');
			} else if (clearCoverImage && editingProject?.cover_image) {
				// Clear existing file - PocketBase accepts empty string to remove file
				formData.append('cover_image', '');
				formData.append('cover_image_library_url', coverImageLibraryUrl || '');
			} else {
				// Set library URL when not uploading a file
				formData.append('cover_image_library_url', coverImageLibraryUrl || '');
			}

			if (editingProject) {
				await await collection('projects').update(editingProject.id, formData);
				toasts.add('success', $t('admin.content.common.toast_updated', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }));
			} else {
				await await collection('projects').create(formData);
				toasts.add('success', $t('admin.content.common.toast_created', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }));
			}

			closeForm();
			await loadProjects();
		} catch (err: any) {
			console.error('Failed to save project:', err);
			const message = err?.response?.data?.slug?.message || $t('admin.content.common.toast_save_error', { values: { type: $t('admin.content.projects.title').slice(0, -1).toLowerCase() } });
			toasts.add('error', message);
		} finally {
			saving = false;
		}
	}

	async function deleteProject(project: Project) {
		const confirmed = await confirm({
			title: $t('admin.content.common.confirm_delete_title', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }),
			message: $t('admin.content.common.confirm_delete_message', { values: { name: project.title } }),
			confirmText: $t('admin.content.common.confirm_delete_button'),
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await await collection('projects').delete(project.id);
			toasts.add('success', $t('admin.content.common.toast_deleted', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }));
			await loadProjects();
		} catch (err) {
			console.error('Failed to delete project:', err);
			toasts.add('error', $t('admin.content.common.toast_delete_error', { values: { type: $t('admin.content.projects.title').slice(0, -1).toLowerCase() } }));
		}
	}

	async function togglePublish(project: Project) {
		try {
			await await collection('projects').update(project.id, {
				is_draft: !project.is_draft
			});
			toasts.add('success', project.is_draft ? $t('admin.content.common.toast_published', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }) : $t('admin.content.common.toast_unpublished', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }));
			await loadProjects();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', $t('admin.content.common.toast_update_error', { values: { type: $t('admin.content.projects.title').slice(0, -1).toLowerCase() } }));
		}
	}

	async function toggleFeatured(project: Project) {
		try {
			await await collection('projects').update(project.id, {
				is_featured: !project.is_featured
			});
			toasts.add('success', project.is_featured ? $t('admin.content.projects.toast_unfeatured') : $t('admin.content.projects.toast_featured'));
			await loadProjects();
		} catch (err) {
			console.error('Failed to toggle featured:', err);
			toasts.add('error', $t('admin.content.common.toast_update_error', { values: { type: $t('admin.content.projects.title').slice(0, -1).toLowerCase() } }));
		}
	}

	function getCoverImageUrl(project: Project): string {
		// Prefer library URL over direct upload
		const libraryUrl = (project as any).cover_image_library_url;
		if (libraryUrl) return libraryUrl;
		if (!project.cover_image) return '';
		return getFileUrl(project, project.cover_image);
	}

	function hasCoverImage(project: Project): boolean {
		return !!(project.cover_image || (project as any).cover_image_library_url);
	}

	let linkTypes = $derived([
		{ value: 'website', label: $t('admin.content.projects.link_type_website') },
		{ value: 'github', label: $t('admin.content.projects.link_type_github') },
		{ value: 'demo', label: $t('admin.content.projects.link_type_demo') },
		{ value: 'docs', label: $t('admin.content.projects.link_type_docs') },
		{ value: 'npm', label: $t('admin.content.projects.link_type_npm') },
		{ value: 'other', label: $t('admin.content.projects.link_type_other') }
	]);

	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		if (selectedIds.has(id)) selectedIds.delete(id);
		else selectedIds.add(id);
		selectedIds = selectedIds;
	}

	function selectAll() { selectedIds = new Set(projects.map(e => e.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('projects').update(id, { visibility });
			toasts.add('success', $t('admin.content.common.bulk_updated', { values: { count: ids.length, visibility } }));
			selectedIds = new Set();
			selectMode = false;
			await loadProjects();
		} catch (err) {
			toasts.add('error', $t('admin.content.common.bulk_error'));
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: $t('admin.content.common.confirm_bulk_delete_title', { values: { type: $t('admin.content.projects.title') } }),
			message: $t('admin.content.common.confirm_bulk_delete_message', { values: { count: ids.length, type: $t('admin.content.projects.title').toLowerCase() } }),
			confirmText: $t('admin.content.common.confirm_bulk_delete_button'),
			danger: true
		});
		if (!confirmed) return;
		try {
			for (const id of ids) await collection('projects').delete(id);
			toasts.add('success', $t('admin.content.common.bulk_deleted', { values: { count: ids.length } }));
			selectedIds = new Set();
			selectMode = false;
			await loadProjects();
		} catch (err) {
			toasts.add('error', $t('admin.content.common.bulk_error'));
		}
	}

	function enterReorderMode() {
		reorderItems = projects.map(p => ({
			id: p.id,
			title: p.title,
			is_featured: p.is_featured,
			is_draft: p.is_draft
		}));
		reorderMode = true;
		selectMode = false;
	}

	function cancelReorder() {
		reorderMode = false;
		reorderItems = [];
	}

	function handleReorderConsider(e: CustomEvent<{ items: typeof reorderItems }>) {
		reorderItems = e.detail.items;
	}

	function handleReorderFinalize(e: CustomEvent<{ items: typeof reorderItems }>) {
		reorderItems = e.detail.items;
	}

	async function saveReorder() {
		savingOrder = true;
		try {
			const updates = reorderItems.map((item, index) => 
				collection('projects').update(item.id, { sort_order: index })
			);
			await Promise.all(updates);
			toasts.add('success', $t('admin.content.projects.toast_order_saved'));
			reorderMode = false;
			reorderItems = [];
			await loadProjects();
		} catch (err) {
			console.error('Failed to save order:', err);
			toasts.add('error', $t('admin.content.projects.toast_order_error'));
		} finally {
			savingOrder = false;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.content.projects.title')} {$t('admin.content.common.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="projects">
		<p>{@html $t('admin.content.projects.help_text')}</p>
		<p>{$t('admin.content.projects.help_tip_1')}</p>
		<p>{@html $t('admin.content.projects.help_tip_2')}</p>
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
			totalCount={projects.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.content.projects.title')}</h1>
		<div class="flex items-center gap-2">
			{#if projects.length > 0 && !reorderMode}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={toggleSelectMode}
				>
					{selectMode ? $t('admin.content.common.cancel') : $t('admin.content.common.select')}
				</button>
				<button
					class="btn btn-ghost"
					onclick={enterReorderMode}
				>
					{$t('admin.content.projects.reorder_button')}
				</button>
			{/if}
			{#if !reorderMode}
				<a href="/admin/import" class="btn btn-secondary">
					{$t('admin.content.common.import_from_github')}
				</a>
				<button class="btn btn-primary" onclick={openNewForm}>
					{$t('admin.content.common.new_button', { values: { type: $t('admin.content.projects.title').slice(0, -1) } })}
				</button>
			{/if}
		</div>
	</div>

	<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} {availableTags} />

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.content.common.loading', { values: { type: $t('admin.content.projects.title').toLowerCase() } })}</div>
		</div>
	{:else if reorderMode}
		<div class="card p-6">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.common.reorder_drag_hint')}</h2>
				<div class="flex gap-2">
					<button class="btn btn-ghost" onclick={cancelReorder} disabled={savingOrder}>
						{$t('shared.actions.cancel')}
					</button>
					<button class="btn btn-primary" onclick={saveReorder} disabled={savingOrder}>
						{#if savingOrder}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
						{/if}
						{$t('admin.content.common.reorder_save')}
					</button>
				</div>
			</div>
			{#key dndLoaded}
				<div
					class="space-y-2"
					use:dndzone={{ items: reorderItems, flipDurationMs, type: 'projects' }}
					onconsider={handleReorderConsider}
					onfinalize={handleReorderFinalize}
				>
					{#each reorderItems as item (item.id)}
						<div
							class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
							animate:flip={{ duration: flipDurationMs }}
						>
							<div class="cursor-grab active:cursor-grabbing p-2 rounded hover:bg-gray-200 dark:hover:bg-gray-700">
								<svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
								</svg>
							</div>
							<span class="flex-1 font-medium text-gray-900 dark:text-white">{item.title}</span>
							{#if item.is_featured}
								<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">{$t('admin.content.common.badge_featured')}</span>
							{/if}
							{#if item.is_draft}
								<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">{$t('admin.content.common.badge_draft')}</span>
							{/if}
						</div>
					{/each}
				</div>
			{/key}
		</div>
	{:else if showForm}
		<!-- Project Form -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingProject ? $t('admin.content.common.form_title_edit', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }) : $t('admin.content.common.form_title_new', { values: { type: $t('admin.content.projects.title').slice(0, -1) } })}
					</h2>
				<button type="button" class="text-gray-500 hover:text-gray-700" onclick={closeForm} aria-label={$t('admin.content.common.close_form')}>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
				</div>

				<div>
					<label for="title" class="label">{$t('admin.content.projects.title_label')} {$t('admin.content.common.required_field')}</label>
					<input
						type="text"
						id="title"
						bind:value={title}
						class="input"
						placeholder={$t('admin.content.projects.title_placeholder')}
						required
						onblur={() => !slug && generateSlug()}
					/>
				</div>

				<div>
					<label for="slug" class="label">{$t('admin.content.projects.slug_label')}</label>
					<div class="flex gap-2">
						<input
							type="text"
							id="slug"
							bind:value={slug}
							class="input flex-1"
							placeholder={$t('admin.content.projects.slug_placeholder')}
						/>
						<button type="button" class="btn btn-secondary" onclick={generateSlug}>
							{$t('admin.content.projects.generate_slug')}
						</button>
					</div>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.projects.slug_help', { values: { slug: slug || 'my-project' } })}</p>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="summary" class="label mb-0">{$t('admin.content.projects.summary_label')}</label>
						<AIContentHelper
							fieldType="summary"
							content={summary}
							context={{ project: title, technologies: techStackText }}
							on:apply={(e) => (summary = e.detail.content)}
						/>
					</div>
					<textarea
						id="summary"
						bind:value={summary}
						class="input mt-1"
						rows="2"
						placeholder={$t('admin.content.projects.summary_placeholder')}
					></textarea>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="description" class="label mb-0">{$t('admin.content.common.description_label')}</label>
						<AIContentHelper
							fieldType="description"
							content={description}
							context={{ project: title, summary, technologies: techStackText }}
							on:apply={(e) => (description = e.detail.content)}
						/>
					</div>
					<textarea
						id="description"
						bind:value={description}
						class="input min-h-[150px] mt-1"
						placeholder={$t('admin.content.projects.description_placeholder')}
					></textarea>
					<div class="mt-2 flex items-center gap-3 text-xs text-gray-600 dark:text-gray-400">
						<button type="button" class="btn btn-ghost btn-sm" onclick={toggleShortcodes}>{$t('admin.content.projects.media_toggle_shortcodes')}</button>
						<span>{$t('admin.content.projects.shortcodes_hint')}</span>
					</div>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.projects.technical_details_heading')}</h2>

				<div>
					<label for="tech_stack" class="label">{$t('admin.content.projects.tech_stack_label')}</label>
					<input
						type="text"
						id="tech_stack"
						bind:value={techStackText}
						class="input"
						placeholder={$t('admin.content.projects.tech_stack_placeholder')}
					/>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.projects.tech_stack_help')}</p>
				</div>

				<div>
					<label for="categories" class="label">{$t('admin.content.projects.categories_label')}</label>
					<input
						type="text"
						id="categories"
						bind:value={categoriesText}
						class="input"
						placeholder={$t('admin.content.projects.categories_placeholder')}
					/>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.projects.categories_help')}</p>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.projects.links_label')}</h2>
					<button type="button" class="btn btn-secondary btn-sm" onclick={addLink}>
						{$t('admin.content.projects.add_link')}
					</button>
				</div>

				{#if links.length === 0}
					<p class="text-sm text-gray-500 dark:text-gray-400">
						{$t('admin.content.projects.no_links_yet')}
					</p>
				{:else}
					<div class="space-y-3">
						{#each links as link, i}
							<div class="flex flex-col sm:flex-row gap-2">
								<select bind:value={link.type} class="input w-full sm:w-32">
									{#each linkTypes as lt}
										<option value={lt.value}>{lt.label}</option>
									{/each}
								</select>
								<div class="flex gap-2 flex-1">
									<input
										type="url"
										bind:value={link.url}
										class="input flex-1"
										placeholder="https://..."
									/>
									<button type="button" class="btn btn-secondary" onclick={() => removeLink(i)} aria-label={$t('admin.content.projects.remove_link')}>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.projects.media_label')}</h2>

				<div>
					<SingleMediaPicker
						label={$t('admin.content.projects.cover_image_label')}
						helpText={$t('admin.content.projects.cover_image_help')}
						bind:value={coverImageLibraryUrl}
						bind:fileInput={coverImageFile}
						bind:clearExisting={clearCoverImage}
						currentFileUrl={editingProject?.cover_image ? getCoverImageUrl(editingProject) : ''}
						currentFileName={editingProject?.cover_image || ''}
						imagesOnly={true}
						onchange={handleFormChange}
					/>
				</div>

			<MultiMediaPicker
				bind:this={mediaPickerRef}
				bind:value={mediaRefs}
				label={$t('admin.content.common.attached_media')}
				helpText={$t('admin.content.projects.media_attach_help')}
			/>
		</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.projects.publishing_heading')}</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<VisibilitySelector bind:value={visibility} />

					<div>
						<label for="sort_order" class="label">{$t('admin.content.common.sort_order_label')}</label>
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
							{$t('admin.content.common.is_draft_label')}
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
							{$t('admin.content.projects.is_featured_label')}
						</label>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<input
						type="checkbox"
						id="comments_enabled"
						bind:checked={commentsEnabled}
						class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
					/>
					<label for="comments_enabled" class="text-sm text-gray-700 dark:text-gray-300">
						{$t('admin.content.projects.enable_comments')}
					</label>
				</div>

				<div>
					<span id="admin-tags-label" class="label">{$t('admin.content.common.admin_tags_label')}</span>
					<AdminTagSelector bind:selectedIds={adminTagIds} labelledBy="admin-tags-label" />
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.common.admin_tags_help')}</p>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<button type="button" class="btn btn-secondary" onclick={closeForm}>{$t('shared.actions.cancel')}</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingProject ? $t('admin.content.common.update_button', { values: { type: $t('admin.content.projects.title').slice(0, -1) } }) : $t('admin.content.common.create_button', { values: { type: $t('admin.content.projects.title').slice(0, -1) } })}
				</button>
			</div>
		</form>
		{#if showShortcodes}
			<div
				class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
				role="dialog"
				aria-modal="true"
				aria-labelledby="shortcodes-title"
				onkeydown={(e) => e.key === 'Escape' && toggleShortcodes()}
			>
				<div class="bg-white dark:bg-gray-900 rounded-lg shadow-lg max-w-2xl w-full p-6 space-y-4">
					<div class="flex items-center justify-between">
						<h3 id="shortcodes-title" class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.projects.shortcodes_title')}</h3>
						<button class="btn btn-ghost btn-sm" onclick={toggleShortcodes}>{$t('shared.actions.close')}</button>
					</div>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						{$t('admin.content.projects.shortcodes_description')}
					</p>
					<div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm text-gray-700 dark:text-gray-200">
						<div>
							<p class="font-semibold">{$t('admin.content.projects.shortcodes_video')}</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{youtube:https://youtu.be/ID}}'}</code></li>
								<li><code>{'{{vimeo:https://vimeo.com/ID}}'}</code></li>
								<li><code>{'{{loom:https://www.loom.com/share/ID}}'}</code></li>
								<li><code>{'{{video:https://example.com/video.mp4}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">{$t('admin.content.projects.shortcodes_audio')}</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{soundcloud:https://soundcloud.com/...}}'}</code></li>
								<li><code>{'{{spotify:https://open.spotify.com/track/...}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">{$t('admin.content.projects.shortcodes_images')}</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{image:https://.../image.jpg}}'}</code></li>
								<li><code>{'{{pdf:https://.../file.pdf}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">{$t('admin.content.projects.shortcodes_design')}</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{figma:https://www.figma.com/file/...}}'}</code></li>
								<li><code>{'{{codepen:https://codepen.io/user/pen/...}}'}</code></li>
							</ul>
						</div>
						<div>
							<p class="font-semibold">{$t('admin.content.projects.shortcodes_other')}</p>
							<ul class="list-disc list-inside space-y-1">
								<li><code>{'{{immich:https://immich.example.com/...}}'}</code></li>
								<li><code>{'{{embed:https://any-link}}'}</code></li>
							</ul>
						</div>
					</div>
				</div>
			</div>
		{/if}
	{:else if projects.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.content.common.empty_state_no_items', { values: { type: $t('admin.content.projects.title').toLowerCase() } })}</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				{$t('admin.content.projects.empty_state_message')}
			</p>
			<div class="flex gap-2 justify-center">
				<a href="/admin/import" class="btn btn-secondary">
					{$t('admin.content.common.import_from_github')}
				</a>
				<button class="btn btn-primary" onclick={openNewForm}>
					{$t('admin.content.common.add_manually')}
				</button>
			</div>
		</div>
	{:else if filteredProjects.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.content.common.empty_state_no_matches', { values: { type: $t('admin.content.projects.title').toLowerCase() } })}</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				{$t('admin.content.common.empty_state_adjust_filters')}
			</p>
		<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
			{$t('admin.content.common.clear_filters')}
		</button>
		</div>
	{:else}
		<!-- Projects Grid -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each filteredProjects as project (project.id)}
				<div class="card overflow-hidden {selectMode && selectedIds.has(project.id) ? 'ring-2 ring-primary-500' : ''}">
					{#if hasCoverImage(project)}
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
							{#if selectMode}
								<input
									type="checkbox"
									checked={selectedIds.has(project.id)}
									onchange={() => toggleSelect(project.id)}
									class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
								/>
							{/if}
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 flex-wrap">
									<h3 class="font-medium text-gray-900 dark:text-white truncate">
										{project.title}
									</h3>
									{#if project.is_featured}
										<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
											{$t('admin.content.common.badge_featured')}
										</span>
									{/if}
									{#if project.is_draft}
										<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
											{$t('admin.content.common.badge_draft')}
										</span>
									{/if}
								{#if project.visibility !== 'public'}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{project.visibility}
									</span>
								{/if}
								{#if project.expand?.admin_tags && project.expand.admin_tags.length > 0}
									{#each project.expand.admin_tags as tag (tag.id)}
										<AdminTagBadge name={tag.name} color={tag.color} />
									{/each}
								{/if}
								</div>

								{#if memberships[project.id]?.length}
									<div class="flex flex-wrap gap-1 mt-2">
										{#each memberships[project.id].slice(0, 3) as viewRef}
											<a
												href={`/admin/views/${viewRef.id}`}
												target="_blank"
												class="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
												title={`Open view: ${viewRef.name}`}
											>
												{viewRef.slug || viewRef.name}
											</a>
										{/each}
										{#if memberships[project.id].length > 3}
											<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-400">
												+{memberships[project.id].length - 3}
											</span>
										{/if}
									</div>
								{:else}
									<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.content.common.not_in_view')}</p>
								{/if}

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
										class="p-3 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
										title={$t('shared.actions.view')}
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
										</svg>
									</a>
								{/if}
							</div>

							<div class="flex items-center gap-1">
								<button
									class="p-3 text-gray-500 hover:text-yellow-600"
									onclick={() => toggleFeatured(project)}
									title={project.is_featured ? $t('admin.content.projects.action_unfeature') : $t('admin.content.projects.action_feature')}
								>
									<svg class="w-4 h-4" fill={project.is_featured ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
									</svg>
								</button>
								<button
									class="p-3 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
									onclick={() => togglePublish(project)}
									title={project.is_draft ? $t('shared.actions.publish') : $t('shared.actions.unpublish')}
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
									class="p-3 text-gray-500 hover:text-blue-600"
									onclick={() => openEditForm(project)}
									title={$t('shared.actions.edit')}
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
									</svg>
								</button>
								<button
									class="p-3 text-gray-500 hover:text-red-600"
									onclick={() => deleteProject(project)}
									title={$t('shared.actions.delete')}
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

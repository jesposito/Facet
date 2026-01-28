<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { browser } from '$app/environment';
	import { flip } from 'svelte/animate';
	import { pb, type Skill } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';

	let skills: Skill[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingSkill: Skill | null = $state(null);
	let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});

	// Form fields
	let name = $state('');
	let category = $state('');
	let proficiency: 'expert' | 'proficient' | 'familiar' = $state('proficient');
	let visibility = $state('public');
	let sortOrder = $state(0);
	let saving = $state(false);

	// Available categories (derived from existing skills)
	let availableCategories: string[] = $state([]);

let selectMode = $state(false);
let selectedIds: Set<string> = $state(new Set());

let reorderMode = $state(false);
let reorderData: Map<string, Array<{ id: string; name: string }>> = $state(new Map());
let categoryOrder: Array<{ id: string; name: string }> = $state([]);
let savingOrder = $state(false);
const flipDurationMs = 200;

let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
let dndLoaded = $state(false);
let SHADOW_PLACEHOLDER_ITEM_ID: string = $state('');

// Load saved category order from site settings
let savedCategoryOrder: string[] = $state([]);

	const autosave = createAutosave('admin-skills', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: true,
		enableDraftFilter: false,
		enableTagFilter: false,
		searchPlaceholder: 'Search skills...'
	});
	let showAdvancedFilters = $state(false);

	let filteredSkills = $derived(
		filterStore.filterItems(
			skills,
			(skill) => `${skill.name} ${skill.category || ''}`,
			(skill) => skill.visibility
		)
	);

	function getFormData() {
	return { name, category, proficiency, visibility, sortOrder };
}

function restoreFromDraft(data: Record<string, any>) {
	name = data.name || '';
	category = data.category || '';
	proficiency = data.proficiency || 'proficient';
	visibility = data.visibility || 'public';
	sortOrder = data.sortOrder || 0;
}

function handleFormChange() {
	if (showForm) {
		autosave.save(getFormData(), !!editingSkill, editingSkill?.id);
	}
}

function handleRestoreDraft() {
	const draft = autosave.loadDraft();
	if (draft?.data) {
		restoreFromDraft(draft.data);
		if (draft.isEditing && draft.editingId) {
			const skill = skills.find(s => s.id === draft.editingId);
			if (skill) editingSkill = skill;
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

afterNavigate(() => {
	showForm = false;
	editingSkill = null;
	selectMode = false;
	selectedIds = new Set();
	
	const draft = autosave.loadDraft();
	if (draft?.data && Object.values(draft.data).some(v => v !== '' && v !== false && v !== 0)) {
		recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
		showRecoveryBanner = true;
	} else {
		showRecoveryBanner = false;
		recoveryData = null;
	}
});

onMount(loadSkills);
onMount(loadCategoryOrder);
onMount(async () => {
	if (browser) {
		const { dndzone: dnd, SHADOW_PLACEHOLDER_ITEM_ID: shadowId } = await import('svelte-dnd-action');
		dndzone = dnd;
		SHADOW_PLACEHOLDER_ITEM_ID = shadowId;
		dndLoaded = true;
	}
});

async function loadCategoryOrder() {
	try {
		const response = await fetch('/api/site-settings');
		if (response.ok) {
			const data = await response.json();
			savedCategoryOrder = data.skills_category_order || [];
		}
	} catch (err) {
		console.error('Failed to load category order:', err);
	}
}

	async function loadSkills() {
		loading = true;
		try {
			const [records, membershipResp] = await Promise.all([
				await collection('skills').getList(1, 200, {
					sort: 'category,sort_order,name'
				}),
				fetch('/api/admin/view-memberships?collection=skills', {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				}).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed memberships'))))
			]);
			skills = records.items as unknown as Skill[];
			memberships = (membershipResp.memberships as typeof memberships) || {};

			// Extract unique categories
			availableCategories = [...new Set(skills.map(s => s.category).filter(Boolean))] as string[];
		} catch (err) {
			console.error('Failed to load skills:', err);
			toasts.add('error', 'Failed to load skills');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		name = '';
		category = '';
		proficiency = 'proficient';
		visibility = 'public';
		sortOrder = 0;
		editingSkill = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function openEditForm(skill: Skill) {
		editingSkill = skill;
		name = skill.name;
		category = skill.category || '';
		proficiency = skill.proficiency || 'proficient';
		visibility = skill.visibility;
		sortOrder = skill.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
	}

	async function handleSubmit() {
		if (!name.trim()) {
			toasts.add('error', 'Skill name is required');
			return;
		}

		saving = true;
		try {
			const data = {
				name: name.trim(),
				category: category.trim(),
				proficiency,
				visibility,
				sort_order: sortOrder
			};

			if (editingSkill) {
				await await collection('skills').update(editingSkill.id, data);
				toasts.add('success', 'Skill updated successfully');
			} else {
				await await collection('skills').create(data);
				toasts.add('success', 'Skill created successfully');
			}

			closeForm();
			await loadSkills();
		} catch (err) {
			console.error('Failed to save skill:', err);
			toasts.add('error', 'Failed to save skill');
		} finally {
			saving = false;
		}
	}

	async function deleteSkill(skill: Skill) {
		const confirmed = await confirm({
			title: 'Delete Skill',
			message: `Are you sure you want to delete "${skill.name}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await await collection('skills').delete(skill.id);
			toasts.add('success', 'Skill deleted');
			await loadSkills();
		} catch (err) {
			console.error('Failed to delete skill:', err);
			toasts.add('error', 'Failed to delete skill');
		}
	}

	// Group skills by category, respecting saved category order
	function groupByCategory(skillList: Skill[]): Map<string, Skill[]> {
		const groups = new Map<string, Skill[]>();
		for (const skill of skillList) {
			const categoryKey = skill.category || 'Other';
			if (!groups.has(categoryKey)) {
				groups.set(categoryKey, []);
			}
			groups.get(categoryKey)!.push(skill);
		}

		// If we have a saved category order, reorder the map
		if (savedCategoryOrder.length > 0) {
			const orderedGroups = new Map<string, Skill[]>();
			// First add categories in saved order
			for (const cat of savedCategoryOrder) {
				if (groups.has(cat)) {
					orderedGroups.set(cat, groups.get(cat)!);
				}
			}
			// Then add any categories not in saved order
			for (const [cat, skills] of groups) {
				if (!orderedGroups.has(cat)) {
					orderedGroups.set(cat, skills);
				}
			}
			return orderedGroups;
		}

		return groups;
	}

	function getProficiencyColor(level: string | undefined): string {
		switch (level) {
			case 'expert':
				return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200';
			case 'proficient':
				return 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200';
			case 'familiar':
				return 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400';
			default:
				return 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400';
		}
	}

	function getProficiencyLabel(level: string | undefined): string {
		switch (level) {
			case 'expert':
				return 'Expert';
			case 'proficient':
				return 'Proficient';
			case 'familiar':
				return 'Familiar';
			default:
				return '';
		}
	}

	let groupedSkills = $derived(groupByCategory(filteredSkills));

	// Default categories to suggest
	const suggestedCategories = [
		'Languages',
		'Frameworks',
		'Databases',
		'DevOps',
		'Cloud',
		'Tools',
		'Soft Skills'
	];

	let categoryOptions = $derived([...new Set([...suggestedCategories, ...availableCategories])].sort());

	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		if (selectedIds.has(id)) selectedIds.delete(id);
		else selectedIds.add(id);
		selectedIds = selectedIds;
	}

	function selectAll() { selectedIds = new Set(skills.map(e => e.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('skills').update(id, { visibility });
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadSkills();
		} catch (err) {
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: 'Delete Skills',
			message: `Are you sure you want to delete ${ids.length} skill(s)? This action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;
		try {
			for (const id of ids) await collection('skills').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadSkills();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}

	function enterReorderMode() {
		const grouped = groupByCategory(skills);
		reorderData = new Map();
		const catNames: string[] = [];
		for (const [cat, catSkills] of grouped) {
			reorderData.set(cat, catSkills.map(s => ({ id: s.id, name: s.name })));
			catNames.push(cat);
		}
		// Set up category order for drag-drop (use name-based IDs for stability)
		categoryOrder = catNames.map(name => ({ id: `cat-${name}`, name }));
		reorderMode = true;
		selectMode = false;
	}

	function cancelReorder() {
		reorderMode = false;
		reorderData = new Map();
		categoryOrder = [];
	}

	// Handlers for skill reordering within a category
	function handleSkillReorderConsider(cat: string, e: CustomEvent<{ items: Array<{ id: string; name: string }> }>) {
		reorderData.set(cat, e.detail.items);
		reorderData = new Map(reorderData);
	}

	function handleSkillReorderFinalize(cat: string, e: CustomEvent<{ items: Array<{ id: string; name: string }> }>) {
		const filteredItems = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
		reorderData.set(cat, filteredItems);
		reorderData = new Map(reorderData);
	}

	// Handlers for category reordering
	function handleCategoryOrderConsider(e: CustomEvent<{ items: Array<{ id: string; name: string }> }>) {
		categoryOrder = e.detail.items;
	}

	function handleCategoryOrderFinalize(e: CustomEvent<{ items: Array<{ id: string; name: string }> }>) {
		// Filter out the shadow placeholder item
		const filteredItems = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
		categoryOrder = filteredItems;

		// Rebuild reorderData in new category order
		const oldData = new Map(reorderData);
		reorderData = new Map();
		for (const cat of categoryOrder) {
			if (oldData.has(cat.name)) {
				reorderData.set(cat.name, oldData.get(cat.name)!);
			}
		}

		// Preserve any categories that might have been missed (defensive)
		for (const [catName, skills] of oldData) {
			if (!reorderData.has(catName)) {
				reorderData.set(catName, skills);
			}
		}
	}

	async function saveReorder() {
		savingOrder = true;
		try {
			// Save skill sort orders within categories
			const updates: Promise<any>[] = [];
			for (const [_, catItems] of reorderData) {
				catItems.forEach((item, index) => {
					updates.push(collection('skills').update(item.id, { sort_order: index }));
				});
			}
			await Promise.all(updates);

			// Save category order to site settings
			const newCategoryOrder = categoryOrder.map(c => c.name);
			await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					skills_category_order: newCategoryOrder
				})
			});
			savedCategoryOrder = newCategoryOrder;

			toasts.add('success', 'Order saved');
			reorderMode = false;
			reorderData = new Map();
			categoryOrder = [];
			await loadSkills();
		} catch (err) {
			console.error('Failed to save order:', err);
			toasts.add('error', 'Failed to save order');
		} finally {
			savingOrder = false;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.content.skills.title')} {$t('admin.content.common.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="skills">
		<p>{@html $t('admin.content.skills.help_text')}</p>
		<p>{$t('admin.content.skills.help_tip_1')}</p>
		<p>{@html $t('admin.content.skills.help_tip_2')}</p>
	</PageHelp>

	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={skills.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.content.skills.title')}</h1>
		<div class="flex items-center gap-2">
			{#if skills.length > 0 && !reorderMode}
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
					{$t('admin.content.skills.reorder_skills')}
				</button>
			{/if}
			{#if !reorderMode}
				<button class="btn btn-primary" onclick={openNewForm}>
					{$t('admin.content.common.new_button', { values: { type: $t('admin.content.skills.title').slice(0, -1) } })}
				</button>
			{/if}
		</div>
	</div>

	{#if showRecoveryBanner && recoveryData}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

	<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} />

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.content.common.loading', { values: { type: $t('admin.content.skills.title').toLowerCase() } })}</div>
		</div>
	{:else if reorderMode}
		<div class="space-y-6">
			<!-- Category Order -->
			<div class="card p-6">
				<div class="flex items-center justify-between mb-4">
					<div>
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Category Order</h2>
						<p class="text-sm text-gray-500">Drag categories to change their display order across the site.</p>
					</div>
				</div>
				{#key dndLoaded}
					<div
						class="flex flex-wrap gap-2"
						use:dndzone={{ items: categoryOrder, flipDurationMs, type: 'category-order' }}
						onconsider={(e: any) => handleCategoryOrderConsider(e)}
						onfinalize={(e: any) => handleCategoryOrderFinalize(e)}
					>
						{#each categoryOrder as cat (cat.id)}
							<div
								class="flex items-center gap-2 px-3 py-2 bg-gray-100 dark:bg-gray-700 rounded-lg cursor-grab active:cursor-grabbing"
								animate:flip={{ duration: flipDurationMs }}
							>
								<svg class="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
								</svg>
								<span class="font-medium text-gray-800 dark:text-gray-200">{cat.name}</span>
							</div>
						{/each}
					</div>
				{/key}
			</div>

			<!-- Skills within Categories -->
			<div class="card p-6">
				<div class="flex items-center justify-between mb-4">
					<div>
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Skill Order</h2>
						<p class="text-sm text-gray-500">Drag skills to reorder within each category.</p>
					</div>
					<div class="flex gap-2">
						<button class="btn btn-ghost" onclick={cancelReorder} disabled={savingOrder}>
							Cancel
						</button>
						<button class="btn btn-primary" onclick={saveReorder} disabled={savingOrder}>
							{#if savingOrder}
								<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
							{/if}
							Save Order
						</button>
					</div>
				</div>
				{#key dndLoaded}
					<div class="space-y-6">
						{#each categoryOrder as cat (cat.id)}
							{@const categoryItems = reorderData.get(cat.name) || []}
							<div>
								<h3 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
									{cat.name}
								</h3>
								<div
									class="space-y-2"
									use:dndzone={{ items: categoryItems, flipDurationMs, type: `skills-${cat.name}` }}
									onconsider={(e: any) => handleSkillReorderConsider(cat.name, e)}
									onfinalize={(e: any) => handleSkillReorderFinalize(cat.name, e)}
							>
								{#each categoryItems as item (item.id)}
									<div
										class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
										animate:flip={{ duration: flipDurationMs }}
									>
										<div class="cursor-grab active:cursor-grabbing p-2 rounded hover:bg-gray-200 dark:hover:bg-gray-700">
											<svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
												<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
											</svg>
										</div>
										<span class="flex-1 font-medium text-gray-900 dark:text-white">{item.name}</span>
									</div>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			{/key}
			</div>
		</div>
	{:else if showForm}
		<!-- Skill Form -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingSkill ? $t('admin.content.common.form_title_edit', { values: { type: $t('admin.content.skills.title').slice(0, -1) } }) : $t('admin.content.common.form_title_new', { values: { type: $t('admin.content.skills.title').slice(0, -1) } })}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" onclick={closeForm} aria-label={$t('admin.content.common.close_form')}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="name" class="label">{$t('admin.content.skills.name_label')} {$t('admin.content.common.required_field')}</label>
					<input
						type="text"
						id="name"
						bind:value={name}
						class="input"
						placeholder={$t('admin.content.skills.name_placeholder')}
						required
					/>
				</div>

				<div>
					<label for="category" class="label">{$t('admin.content.skills.category_label')}</label>
					<input
						type="text"
						id="category"
						bind:value={category}
						list="category-options"
						class="input"
						placeholder={$t('admin.content.skills.category_placeholder')}
					/>
					<datalist id="category-options">
						{#each categoryOptions as cat}
							<option value={cat}></option>
						{/each}
					</datalist>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.skills.category_help')}</p>
				</div>

				<div>
					<label for="proficiency" class="label">{$t('admin.content.skills.proficiency_label')}</label>
					<select id="proficiency" bind:value={proficiency} class="input">
						<option value="expert">{$t('admin.content.skills.proficiency_expert')}</option>
						<option value="proficient">{$t('admin.content.skills.proficiency_proficient')}</option>
						<option value="familiar">{$t('admin.content.skills.proficiency_familiar')}</option>
					</select>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="visibility" class="label">{$t('admin.content.common.visibility_label')}</label>
						<select id="visibility" bind:value={visibility} class="input">
							<option value="public">{$t('admin.content.common.visibility_public')}</option>
							<option value="unlisted">{$t('admin.content.common.visibility_unlisted')}</option>
							<option value="private">{$t('admin.content.common.visibility_private')}</option>
						</select>
					</div>

					<div>
						<label for="sort_order" class="label">{$t('admin.content.common.sort_order_label')}</label>
						<input
							type="number"
							id="sort_order"
							bind:value={sortOrder}
							class="input"
							min="0"
						/>
						<p class="text-xs text-gray-500 mt-1">{$t('admin.content.common.sort_order_help')}</p>
					</div>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<button type="button" class="btn btn-secondary" onclick={closeForm}>{$t('admin.content.common.cancel')}</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingSkill ? $t('admin.content.common.update_button', { values: { type: $t('admin.content.skills.title').slice(0, -1) } }) : $t('admin.content.common.create_button', { values: { type: $t('admin.content.skills.title').slice(0, -1) } })}
				</button>
			</div>
		</form>
	{:else if skills.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.content.common.empty_state_no_items', { values: { type: $t('admin.content.skills.title').toLowerCase() } })}</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				{$t('admin.content.skills.empty_state_message')}
			</p>
			<button class="btn btn-primary" onclick={openNewForm}>
				{$t('admin.content.common.add_first_button', { values: { type: $t('admin.content.skills.title').slice(0, -1) } })}
			</button>
		</div>
	{:else if filteredSkills.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.content.common.empty_state_no_matches', { values: { type: $t('admin.content.skills.title').toLowerCase() } })}</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				{$t('admin.content.common.empty_state_adjust_filters')}
			</p>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				{$t('admin.content.common.clear_filters')}
			</button>
		</div>
	{:else}
		<!-- Skills List - Grouped by Category -->
		<div class="space-y-6">
			{#each [...groupedSkills] as [categoryName, categorySkills] (categoryName)}
				<div class="card p-4">
					<h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-3">
						{categoryName}
					</h2>
					<div class="flex flex-wrap gap-2">
						{#each categorySkills as skill (skill.id)}
							<div class="group relative inline-flex items-center gap-1 px-3 py-1.5 bg-gray-100 dark:bg-gray-700 rounded-lg text-sm {selectMode && selectedIds.has(skill.id) ? 'ring-2 ring-primary-500' : ''}">
								{#if selectMode}
									<input
										type="checkbox"
										checked={selectedIds.has(skill.id)}
										onchange={() => toggleSelect(skill.id)}
										class="w-4 h-4 text-primary-600 rounded border-gray-300"
									/>
								{/if}
								<span class="font-medium text-gray-800 dark:text-gray-200">{skill.name}</span>
								{#if skill.proficiency}
									<span class="px-1.5 py-0.5 text-xs rounded {getProficiencyColor(skill.proficiency)}">
										{getProficiencyLabel(skill.proficiency)}
									</span>
								{/if}
								{#if skill.visibility !== 'public'}
									<span class="px-1.5 py-0.5 text-xs rounded bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200">
										{skill.visibility}
									</span>
								{/if}
								{#if memberships[skill.id]?.length}
									<span 
										class="px-1.5 py-0.5 text-xs rounded bg-purple-50 dark:bg-purple-900/30 text-purple-600 dark:text-purple-300 ml-1 cursor-help"
										title="Used in: {memberships[skill.id].map(v => v.name).join(', ')}"
									>
										👁 {memberships[skill.id].length}
									</span>
								{/if}

								<!-- Action buttons on hover -->
								<div class="hidden group-hover:flex items-center gap-1 ml-1 pl-1 border-l border-gray-300 dark:border-gray-600">
									<button
										class="p-1 text-gray-500 hover:text-blue-600 rounded"
										onclick={() => openEditForm(skill)}
										title="Edit"
									>
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
										</svg>
									</button>
									<button
										class="p-1 text-gray-500 hover:text-red-600 rounded"
										onclick={() => deleteSkill(skill)}
										title="Delete"
									>
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
										</svg>
									</button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>

		<!-- Legend -->
		<div class="mt-6 card p-4">
			<h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Proficiency Levels</h3>
			<div class="flex flex-wrap gap-3 text-xs">
				<div class="flex items-center gap-1">
					<span class="px-2 py-0.5 rounded {getProficiencyColor('expert')}">Expert</span>
					<span class="text-gray-500">Deep expertise</span>
				</div>
				<div class="flex items-center gap-1">
					<span class="px-2 py-0.5 rounded {getProficiencyColor('proficient')}">Proficient</span>
					<span class="text-gray-500">Strong knowledge</span>
				</div>
				<div class="flex items-center gap-1">
					<span class="px-2 py-0.5 rounded {getProficiencyColor('familiar')}">Familiar</span>
					<span class="text-gray-500">Basic understanding</span>
				</div>
			</div>
		</div>
	{/if}
</div>

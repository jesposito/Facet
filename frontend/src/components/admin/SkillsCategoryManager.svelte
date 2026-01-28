<script lang="ts">
	/**
	 * SkillsCategoryManager - Hybrid category/skill management for skills section
	 *
	 * Shows skill categories as expandable rows with:
	 * - Category-level: drag to reorder, toggle to enable/disable entire category
	 * - Skill-level (when expanded): checkboxes to include/exclude, drag to reorder
	 */
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';
	import { flip } from 'svelte/animate';
	import { t } from 'svelte-i18n';

	// Import DnD safely - only in browser
	let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
	let TRIGGERS: any = $state({});
	let SHADOW_PLACEHOLDER_ITEM_ID: string = $state('');
	let dndLoaded = $state(false);

	onMount(async () => {
		if (browser) {
			const { dndzone: dnd, TRIGGERS: trig, SHADOW_PLACEHOLDER_ITEM_ID: shadow } = await import('svelte-dnd-action');
			dndzone = dnd;
			TRIGGERS = trig;
			SHADOW_PLACEHOLDER_ITEM_ID = shadow;
			dndLoaded = true;
		}
	});

	const flipDurationMs = 200;

	interface SkillItem {
		id: string;
		label: string;
		visibility: string;
		is_draft?: boolean;
		data: Record<string, unknown>;
	}

	// Props from parent
	let {
		items = $bindable(),
		selectedItems = $bindable(),
		categoryOrder = $bindable(),
		disabledCategories = $bindable(),
		categoryDisplayModes = $bindable(),
		sectionLayout = 'grouped',
		onUpdate
	}: {
		items: SkillItem[];
		selectedItems?: string[];
		categoryOrder?: string[];
		disabledCategories?: string[];
		categoryDisplayModes?: Record<string, string>;
		sectionLayout?: string;
		onUpdate: () => void;
	} = $props();

	// Ensure we have non-undefined values for internal use
	let _selectedItems = $derived(selectedItems ?? []);
	let _categoryOrder = $derived(categoryOrder ?? []);
	let _disabledCategories = $derived(disabledCategories ?? []);
	let _categoryDisplayModes = $derived(categoryDisplayModes ?? {});

	// Track which categories are expanded
	let expandedCategories: Set<string> = $state(new Set());

	// Group skills by category
	let groupedSkills = $derived.by(() => {
		const groups: Record<string, SkillItem[]> = {};
		for (const item of items) {
			const category = (item.data.category as string) || 'Other';
			if (!groups[category]) {
				groups[category] = [];
			}
			groups[category].push(item);
		}
		return groups;
	});

	// Get all categories in proper order
	let orderedCategories = $derived.by(() => {
		const allCategories = Object.keys(groupedSkills);
		if (!_categoryOrder.length) {
			return allCategories.sort();
		}
		const ordered: string[] = [];
		// First add categories in saved order
		for (const cat of _categoryOrder) {
			if (allCategories.includes(cat)) {
				ordered.push(cat);
			}
		}
		// Then add any new categories not in saved order
		for (const cat of allCategories) {
			if (!ordered.includes(cat)) {
				ordered.push(cat);
			}
		}
		return ordered;
	});

	// Category items for drag-and-drop - must be $state for svelte-dnd-action to work
	let categoryItems: Array<{ id: string; name: string }> = $state([]);

	// Sync categoryItems from orderedCategories when source data changes
	$effect(() => {
		const newItems = orderedCategories.map(name => ({ id: `cat-${name}`, name }));
		// Only update if the underlying categories changed (not during DnD)
		const currentNames = categoryItems.map(c => c.name).join(',');
		const newNames = newItems.map(c => c.name).join(',');
		if (currentNames !== newNames || categoryItems.length === 0) {
			categoryItems = newItems;
		}
	});

	// Check if a category is disabled
	function isCategoryDisabled(category: string): boolean {
		return _disabledCategories.includes(category);
	}

	// Check if a category is expanded
	function isCategoryExpanded(category: string): boolean {
		return expandedCategories.has(category);
	}

	// Get count of selected skills in a category
	function getSelectedCount(category: string): number {
		const skills = groupedSkills[category] || [];
		const selectedSet = new Set(_selectedItems);
		return skills.filter(s => selectedSet.has(s.id)).length;
	}

	// Get total skills in a category (excluding drafts and private)
	function getPublicSkillCount(category: string): number {
		const skills = groupedSkills[category] || [];
		return skills.filter(s => s.visibility !== 'private' && !s.is_draft).length;
	}

	// Toggle category expanded state
	function toggleCategoryExpand(category: string) {
		if (expandedCategories.has(category)) {
			expandedCategories.delete(category);
		} else {
			expandedCategories.add(category);
		}
		expandedCategories = new Set(expandedCategories);
	}

	// Toggle category enabled/disabled
	function toggleCategoryEnabled(category: string) {
		if (isCategoryDisabled(category)) {
			// Enable: remove from disabled list
			disabledCategories = _disabledCategories.filter(c => c !== category);
		} else {
			// Disable: add to disabled list
			disabledCategories = [..._disabledCategories, category];
		}
		onUpdate();
	}

	// Get the display mode for a category (empty string means use section default)
	function getCategoryDisplayMode(category: string): string {
		return _categoryDisplayModes[category] || '';
	}

	// Update display mode for a category
	function updateCategoryDisplayMode(category: string, mode: string) {
		if (mode === '') {
			// Remove custom mode, use section default
			const { [category]: _, ...rest } = _categoryDisplayModes;
			categoryDisplayModes = rest;
		} else {
			categoryDisplayModes = { ..._categoryDisplayModes, [category]: mode };
		}
		onUpdate();
	}

	// Toggle individual skill selection
	function toggleSkill(skillId: string) {
		const idx = _selectedItems.indexOf(skillId);
		if (idx === -1) {
			selectedItems = [..._selectedItems, skillId];
		} else {
			selectedItems = _selectedItems.filter(id => id !== skillId);
		}
		onUpdate();
	}

	// Select all skills in a category
	function selectAllInCategory(category: string) {
		const skills = groupedSkills[category] || [];
		const publicSkillIds = skills
			.filter(s => s.visibility !== 'private' && !s.is_draft)
			.map(s => s.id);
		const currentSet = new Set(_selectedItems);
		for (const id of publicSkillIds) {
			currentSet.add(id);
		}
		selectedItems = Array.from(currentSet);
		onUpdate();
	}

	// Clear all skills in a category
	function clearAllInCategory(category: string) {
		const skills = groupedSkills[category] || [];
		const skillIds = new Set(skills.map(s => s.id));
		selectedItems = _selectedItems.filter(id => !skillIds.has(id));
		onUpdate();
	}

	// Handle category drag-and-drop
	function handleCategoryDndConsider(e: CustomEvent<{ items: Array<{ id: string; name: string }> }>) {
		// Update categoryItems during drag for visual feedback
		categoryItems = e.detail.items;
	}

	function handleCategoryDndFinalize(e: CustomEvent<{ items: Array<{ id: string; name: string }> }>) {
		// Filter out placeholder and update both categoryItems and categoryOrder
		const filteredItems = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
		categoryItems = filteredItems;
		categoryOrder = filteredItems.map(c => c.name);
		onUpdate();
	}

	// Handle skill drag-and-drop within a category
	function handleSkillDndConsider(category: string, e: CustomEvent<{ items: SkillItem[] }>) {
		// Update the items array during drag for visual feedback
		const otherCategorySkills = items.filter(s => (s.data.category || 'Other') !== category);
		const reorderedSkills = e.detail.items;
		items = [...otherCategorySkills, ...reorderedSkills];
	}

	function handleSkillDndFinalize(category: string, e: CustomEvent<{ items: SkillItem[]; info: { trigger: string } }>) {
		const trigger = e.detail.info?.trigger;
		const finalSkills = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);

		// Update items with new order
		const otherCategorySkills = items.filter(s => (s.data.category || 'Other') !== category);
		items = [...otherCategorySkills, ...finalSkills];

		// Update selected items order to match display order
		if (trigger === TRIGGERS.DROPPED_INTO_ZONE || trigger === TRIGGERS.DROPPED_INTO_ANOTHER) {
			const displayOrder = finalSkills.map(s => s.id);
			const selectedSet = new Set(_selectedItems);
			const categorySelectedInOrder = displayOrder.filter(id => selectedSet.has(id));
			const otherSelected = _selectedItems.filter(id => !displayOrder.includes(id));
			selectedItems = [...otherSelected, ...categorySelectedInOrder];
			onUpdate();
		}
	}

	// Get skills for a specific category (for display in expanded view)
	function getCategorySkills(category: string): SkillItem[] {
		return groupedSkills[category] || [];
	}

	// Get public skills for a category (excluding private and drafts)
	function getPublicCategorySkills(category: string): SkillItem[] {
		return (groupedSkills[category] || []).filter(s => s.visibility !== 'private' && !s.is_draft);
	}
</script>

<div class="space-y-2">
	<div class="flex items-center justify-between mb-3">
		<p class="text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.view_editor.skills_categories.description')}
		</p>
	</div>

	{#if dndLoaded && categoryItems.length > 0}
		<div
			class="space-y-2"
			use:dndzone={{ items: categoryItems, flipDurationMs, type: 'skill-categories' }}
			onconsider={handleCategoryDndConsider}
			onfinalize={handleCategoryDndFinalize}
		>
			{#each categoryItems as cat (cat.id)}
				{@const category = cat.name}
				{@const isDisabled = isCategoryDisabled(category)}
				{@const isExpanded = isCategoryExpanded(category)}
				{@const publicCount = getPublicSkillCount(category)}
				{@const selectedCount = getSelectedCount(category)}
				{@const categorySkills = getPublicCategorySkills(category)}

				<div
					class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900 {isDisabled ? 'opacity-60' : ''}"
					animate:flip={{ duration: flipDurationMs }}
				>
					<!-- Category Header -->
					<div class="flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800/50">
						<!-- Drag Handle -->
						<div
							class="cursor-grab active:cursor-grabbing p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 flex-shrink-0"
							title={$t('admin.view_editor.sections.drag_to_reorder')}
						>
							<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
							</svg>
						</div>

						<!-- Enable/Disable Toggle -->
						<button
							type="button"
							class="w-9 h-5 rounded-full transition-colors relative flex-shrink-0
								{isDisabled ? 'bg-gray-300 dark:bg-gray-600' : 'bg-primary-600'}"
							onclick={() => toggleCategoryEnabled(category)}
							aria-label={$t('admin.view_editor.skills_categories.toggle_category', { values: { category } })}
						>
							<span
								class="absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform shadow-sm
									{isDisabled ? 'left-0.5' : 'left-4'}"
							></span>
						</button>

						<!-- Category Name -->
						<span class="font-medium text-gray-900 dark:text-white flex-1 min-w-0 truncate">
							{category}
						</span>

						<!-- Skill Count -->
						<span class="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
							{#if selectedCount > 0 && selectedCount < publicCount}
								{selectedCount}/{publicCount} {$t('admin.view_editor.skills_categories.selected')}
							{:else if selectedCount === publicCount && publicCount > 0}
								{$t('admin.view_editor.skills_categories.all_selected')}
							{:else}
								{publicCount} {$t('admin.view_editor.skills_categories.skills')}
							{/if}
						</span>

						<!-- Per-Category Layout Selector -->
						{#if !isDisabled && (sectionLayout === 'grouped' || sectionLayout === 'bars')}
							<select
								class="text-xs border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 rounded px-2 py-1 flex-shrink-0"
								value={getCategoryDisplayMode(category)}
								onchange={(e) => updateCategoryDisplayMode(category, e.currentTarget.value)}
								onclick={(e) => e.stopPropagation()}
								aria-label={$t('admin.view_editor.skills_categories.display_mode_label', { values: { category } })}
							>
								<option value="">{$t('admin.view_editor.skills_categories.use_section_default')}</option>
								<option value="grouped">{$t('admin.view_editor.skills_categories.layout_grouped')}</option>
								<option value="bars">{$t('admin.view_editor.skills_categories.layout_bars')}</option>
							</select>
						{/if}

						<!-- Expand/Collapse Button -->
						{#if publicCount > 0 && !isDisabled}
							<button
								type="button"
								class="p-1.5 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-700 flex-shrink-0"
								onclick={() => toggleCategoryExpand(category)}
								aria-label={isExpanded
									? $t('admin.view_editor.skills_categories.collapse', { values: { category } })
									: $t('admin.view_editor.skills_categories.expand', { values: { category } })}
							>
								<svg
									class="w-5 h-5 transition-transform {isExpanded ? 'rotate-180' : ''}"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
								>
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</button>
						{/if}
					</div>

					<!-- Expanded Skills List -->
					{#if isExpanded && !isDisabled && categorySkills.length > 0}
						<div class="p-3 border-t border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/30">
							<div class="flex items-center justify-between mb-2">
								<p class="text-xs text-gray-500">
									{selectedCount === 0
										? $t('admin.view_editor.skills_categories.none_selected_hint')
										: $t('admin.view_editor.skills_categories.drag_hint')}
								</p>
								<div class="flex gap-2">
									<button
										type="button"
										class="text-xs text-primary-600 hover:underline"
										onclick={() => selectAllInCategory(category)}
									>
										{$t('admin.view_editor.sections.select_all')}
									</button>
									<button
										type="button"
										class="text-xs text-gray-500 hover:underline"
										onclick={() => clearAllInCategory(category)}
									>
										{$t('admin.view_editor.sections.clear')}
									</button>
								</div>
							</div>

							<div
								class="space-y-1 max-h-48 overflow-y-auto"
								use:dndzone={{
									items: categorySkills,
									flipDurationMs,
									type: `skills-${category}`,
									dragDisabled: false,
									dropFromOthersDisabled: true
								}}
								onconsider={(e: any) => handleSkillDndConsider(category, e)}
								onfinalize={(e: any) => handleSkillDndFinalize(category, e)}
							>
								{#each categorySkills as skill (skill.id)}
									{@const isSelected = _selectedItems.includes(skill.id)}
									<div
										class="flex items-center gap-2 p-2 rounded bg-white dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800"
										animate:flip={{ duration: flipDurationMs }}
									>
										<!-- Drag Handle -->
										<div
											class="cursor-grab active:cursor-grabbing p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
											title={$t('admin.view_editor.sections.drag_to_reorder')}
										>
											<svg class="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
												<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
											</svg>
										</div>

										<!-- Checkbox + Label -->
										<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
										<label
											class="flex items-center gap-2 flex-1 cursor-pointer min-w-0"
											onpointerdown={(e) => e.stopPropagation()}
											onmousedown={(e) => e.stopPropagation()}
											ontouchstart={(e) => e.stopPropagation()}
										>
											<input
												type="checkbox"
												checked={isSelected}
												onchange={() => toggleSkill(skill.id)}
												onclick={(e) => e.stopPropagation()}
												class="w-4 h-4 text-primary-600 rounded border-gray-300"
											/>
											<span class="text-sm text-gray-700 dark:text-gray-300 truncate">
												{skill.label}
											</span>
										</label>
									</div>
								{/each}
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{:else if categoryItems.length === 0}
		<p class="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
			{$t('admin.view_editor.skills_categories.no_skills')}
		</p>
	{:else}
		<div class="animate-pulse text-sm text-gray-500 text-center py-4">
			{$t('admin.layout.loading')}
		</div>
	{/if}
</div>

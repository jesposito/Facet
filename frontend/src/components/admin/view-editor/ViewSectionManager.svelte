<script lang="ts">
	import { preventDefault, createBubbler, stopPropagation, self } from 'svelte/legacy';
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';
	import { flip } from 'svelte/animate';
	import { icon } from '$lib/icons';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import {
		OVERRIDABLE_FIELDS,
		VALID_LAYOUTS,
		VALID_WIDTHS,
		getValidWidthsForLayout,
		isWidthValidForLayout,
		type ViewSection,
		type SectionWidth,
		type ItemConfig
	} from '$lib/pocketbase';
	import { t } from 'svelte-i18n';

	// Import DnD safely - only in browser
	let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
	let TRIGGERS: any = $state({});
	let SHADOW_PLACEHOLDER_ITEM_ID: string = $state('');
	let dndLoaded = $state(false);

	// Load DnD functionality when in browser
	onMount(async () => {
		if (browser) {
			const { dndzone: dnd, TRIGGERS: trig, SHADOW_PLACEHOLDER_ITEM_ID: shadow } = await import('svelte-dnd-action');
			dndzone = dnd;
			TRIGGERS = trig;
			SHADOW_PLACEHOLDER_ITEM_ID = shadow;
			dndLoaded = true;
			
			// Apply saved item order after dndzone is loaded
			// This ensures selected items appear at top in saved order
			applySavedItemOrder();
		}
	});

	// Apply saved item order to sectionItems
	// Selected items appear first (in saved order), then unselected items (in original order)
	function applySavedItemOrder() {
		let didUpdate = false;
		for (const key of Object.keys(sections)) {
			const savedOrder = sections[key]?.items || [];
			const allItems = sectionItems[key] || [];

			// Skip if no saved order or no items
			if (savedOrder.length === 0 || allItems.length === 0) continue;

			// Create a map for quick lookup
			const itemMap = new Map(allItems.map(item => [item.id, item]));

			// Build reordered array: selected items first (in saved order), then unselected
			const selectedItems: typeof allItems = [];
			for (const id of savedOrder) {
				const item = itemMap.get(id);
				if (item) {
					selectedItems.push(item);
				}
			}

			const selectedSet = new Set(savedOrder);
			const unselectedItems = allItems.filter(item => !selectedSet.has(item.id));

			sectionItems[key] = [...selectedItems, ...unselectedItems];
			didUpdate = true;
		}
		// Trigger Svelte 5 reactivity with full object reassignment
		if (didUpdate) {
			updateSectionItems();
		}
	}

	const bubble = createBubbler();

	// Default section definitions - used to initialize and provide label keys
	const SECTION_DEFS: Record<string, { labelKey: string; collection: string }> = {
		experience: { labelKey: 'admin.view_editor.sections.experience', collection: 'experience' },
		projects: { labelKey: 'admin.view_editor.sections.projects', collection: 'projects' },
		education: { labelKey: 'admin.view_editor.sections.education', collection: 'education' },
		certifications: { labelKey: 'admin.view_editor.sections.certifications', collection: 'certifications' },
		awards: { labelKey: 'admin.view_editor.sections.awards', collection: 'awards' },
		skills: { labelKey: 'admin.view_editor.sections.skills', collection: 'skills' },
		posts: { labelKey: 'admin.view_editor.sections.posts', collection: 'posts' },
		talks: { labelKey: 'admin.view_editor.sections.talks', collection: 'talks' },
		contacts: { labelKey: 'admin.view_editor.sections.contacts', collection: 'contact_methods' },
		testimonials: { labelKey: 'admin.view_editor.sections.testimonials', collection: 'testimonials' }
	};

	// Helper to check if a section key is for custom content
	function isCustomSection(sectionKey: string): boolean {
		return sectionKey.startsWith('custom:');
	}

	// Get display label key for a section (handles custom content)
	function getSectionLabelKey(sectionKey: string): string | null {
		if (isCustomSection(sectionKey)) {
			return null; // Custom sections use title directly
		}
		return SECTION_DEFS[sectionKey]?.labelKey || null;
	}

	// Get custom section title
	function getCustomSectionTitle(sectionKey: string): string {
		const customId = sectionKey.replace('custom:', '');
		const title = customContentTitleMap.get(customId);
		return title ? `Custom: ${title}` : `Custom: ${customId.slice(0, 8)}...`;
	}

	const flipDurationMs = 200;

	// Props from parent
	let {
		sections = $bindable(),
		sectionOrder = $bindable(),
		sectionItems = $bindable(),
		customContentItems = [],
		viewId,
		onOpenOverrideEditor
	}: {
		sections: Record<string, {
			enabled: boolean;
			items: string[];
			expanded: boolean;
			layout: string;
			width: SectionWidth;
			itemConfig: Record<string, ItemConfig>;
			categoryOrder?: string[];
		}>;
		sectionOrder: Array<{ id: string; key: string }>;
		sectionItems: Record<string, Array<{
			id: string;
			label: string;
			visibility: string;
			is_draft?: boolean;
			data: Record<string, unknown>;
			expand?: {
				admin_tags?: Array<{ id: string; name: string; color: string }>;
			};
		}>>;
		customContentItems?: Array<{ id: string; title: string; is_draft: boolean; visibility: string }>;
		viewId: string;
		onOpenOverrideEditor: (sectionKey: string, itemId: string) => void;
	} = $props();

	// Build a map for quick custom content title lookup
	$effect(() => {
		customContentTitleMap = new Map(customContentItems.map(item => [item.id, item.title]));
	});
	let customContentTitleMap: Map<string, string> = $state(new Map());

	// Helper to trigger reactivity by creating a new object reference
	function updateSections() {
		sections = { ...sections };
	}

	// Helper to trigger reactivity for sectionItems
	function updateSectionItems() {
		sectionItems = { ...sectionItems };
	}

	function toggleSection(key: string) {
		sections[key].enabled = !sections[key].enabled;
		updateSections();
	}

	function toggleSectionExpand(key: string) {
		sections[key].expanded = !sections[key].expanded;
		updateSections();
	}

	function toggleItem(sectionKey: string, itemId: string) {
		const idx = sections[sectionKey].items.indexOf(itemId);
		if (idx === -1) {
			sections[sectionKey].items.push(itemId);
		} else {
			sections[sectionKey].items.splice(idx, 1);
		}
		updateSections();
	}

	function selectAllItems(sectionKey: string) {
		sections[sectionKey].items = sectionItems[sectionKey]?.map((i) => i.id) || [];
		updateSections();
	}

	function clearAllItems(sectionKey: string) {
		sections[sectionKey].items = [];
		updateSections();
	}

	function updateSectionWidth(sectionKey: string, width: string) {
		sections[sectionKey].width = width as SectionWidth;
		updateSections();
	}

	function updateSectionLayout(sectionKey: string, layout: string) {
		sections[sectionKey].layout = layout;
		// Auto-reset width to 'full' if current width is not valid for new layout
		if (!isWidthValidForLayout(sectionKey, layout, sections[sectionKey].width)) {
			sections[sectionKey].width = 'full';
		}
		updateSections();
	}

	function isEnabledForView(viewVisibility: unknown, viewId: string): boolean {
		if (!viewVisibility || typeof viewVisibility !== 'object') return false;
		return (viewVisibility as Record<string, boolean>)[viewId] === true;
	}

	function hasOverrides(sectionKey: string, itemId: string): boolean {
		const config = sections[sectionKey]?.itemConfig?.[itemId];
		return !!(config?.overrides && Object.keys(config.overrides).length > 0);
	}

	function getOverrideCount(sectionKey: string, itemId: string): number {
		const config = sections[sectionKey]?.itemConfig?.[itemId];
		return config?.overrides ? Object.keys(config.overrides).length : 0;
	}

	// Drag-drop handlers for section reordering
	function handleSectionDndConsider(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items;
	}

	function handleSectionDndFinalize(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items;
	}

	// Drag-drop handlers for item reordering within a section
	function handleItemDndConsider(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }>; info: { trigger: string } }>) {
		sectionItems[sectionKey] = e.detail.items;
		updateSectionItems();
	}

	function handleItemDndFinalize(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }>; info: { trigger: string } }>) {
		const trigger = e.detail.info?.trigger;
		const finalItems = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
		sectionItems[sectionKey] = finalItems;
		updateSectionItems();
		
		// Only update selection order if this was an actual drag operation, not just a click
		if (trigger === TRIGGERS.DROPPED_INTO_ZONE || trigger === TRIGGERS.DROPPED_INTO_ANOTHER) {
			updateItemsOrderFromDisplay(sectionKey);
		}
	}

	// Update section.items to preserve order based on current display order
	function updateItemsOrderFromDisplay(sectionKey: string) {
		const displayOrder = sectionItems[sectionKey]?.map(i => i.id) || [];
		const selectedSet = new Set(sections[sectionKey].items);
		// Reorder selected items to match display order
		sections[sectionKey].items = displayOrder.filter(id => selectedSet.has(id));
		updateSections();
	}

	// Skills category ordering
	let showCategoryReorder = $state(false);
	let categoryItems: Array<{ id: string; name: string }> = $state([]);

	function getUniqueCategories(): string[] {
		const skills = sectionItems['skills'] || [];
		const categories = new Set<string>();
		for (const skill of skills) {
			const cat = (skill.data.category as string) || 'Other';
			categories.add(cat);
		}
		return Array.from(categories).sort();
	}

	function initCategoryOrder() {
		const allCategories = getUniqueCategories();
		const savedOrder = sections['skills']?.categoryOrder || [];
		
		const orderedCategories: string[] = [];
		for (const cat of savedOrder) {
			if (allCategories.includes(cat)) {
				orderedCategories.push(cat);
			}
		}
		for (const cat of allCategories) {
			if (!orderedCategories.includes(cat)) {
				orderedCategories.push(cat);
			}
		}
		
		categoryItems = orderedCategories.map(name => ({ id: `cat-${name}`, name }));
		showCategoryReorder = true;
	}

	function closeCategoryReorder() {
		showCategoryReorder = false;
		categoryItems = [];
	}

	function handleCategoryDndConsider(e: CustomEvent<{ items: typeof categoryItems }>) {
		categoryItems = e.detail.items;
	}

	function handleCategoryDndFinalize(e: CustomEvent<{ items: typeof categoryItems }>) {
		categoryItems = e.detail.items;
		sections['skills'].categoryOrder = categoryItems.map(c => c.name);
		updateSections();
	}

	function layoutUsesCategoryOrder(layout: string): boolean {
		return layout === 'grouped' || layout === 'bars';
	}
</script>

<!-- Sections -->
<div class="card p-4 sm:p-6 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.view_editor.sections.title')}</h2>
			<p class="text-sm text-gray-500">{$t('admin.view_editor.sections.description')}</p>
		</div>
	</div>

	{#key dndLoaded}
	<div
		class="space-y-3"
		use:dndzone={{ items: sectionOrder, flipDurationMs, type: 'sections' }}
		onconsider={handleSectionDndConsider}
		onfinalize={handleSectionDndFinalize}
	>
		{#each sectionOrder as sectionItem (sectionItem.id)}
			{@const sectionKey = sectionItem.key}
			{@const sectionDef = SECTION_DEFS[sectionKey]}
			{@const isCustom = isCustomSection(sectionKey)}
			{@const sectionLabelKey = getSectionLabelKey(sectionKey)}
			{@const sectionLabel = sectionLabelKey ? $t(sectionLabelKey) : getCustomSectionTitle(sectionKey)}
			{@const sectionConfig = sections[sectionKey] || { enabled: false, items: [], expanded: false, itemConfig: {} }}
			{@const items = isCustom ? [] : (sectionItems[sectionKey] || [])}
			{@const publicItems = items.filter(i => i.visibility !== 'private' && !i.is_draft)}
			{@const publicItemIds = new Set(publicItems.map(i => i.id))}
			{@const selectedPublicCount = sectionConfig.items.filter(id => publicItemIds.has(id)).length}
			{@const layoutKey = isCustom ? 'custom' : sectionKey}

			<div
				class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900"
				animate:flip={{ duration: flipDurationMs }}
			>
				<!-- Section Header -->
				<div class="flex flex-wrap items-center justify-between p-3 bg-gray-50 dark:bg-gray-800/50">
					<div class="flex items-center gap-3">
						<!-- Drag Handle -->
						<div class="cursor-grab active:cursor-grabbing p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700" title={$t('admin.view_editor.sections.drag_to_reorder')}>
							<svg class="w-6 h-6 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
							</svg>
						</div>
						<button
							type="button"
							class="w-10 h-6 rounded-full transition-colors relative
								{sectionConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
							onclick={() => toggleSection(sectionKey)}
							aria-label={$t('admin.view_editor.sections.toggle_section', { values: { section: sectionLabel } })}
						>
							<span
								class="absolute top-1 w-4 h-4 bg-white rounded-full transition-transform shadow-sm
									{sectionConfig.enabled ? 'left-5' : 'left-1'}"
							></span>
						</button>
						<span class="font-medium text-gray-900 dark:text-white">{sectionLabel}</span>
						{#if !isCustom}
							<span class="text-xs text-gray-500">
								{#if selectedPublicCount > 0}
									{selectedPublicCount} {$t('admin.view_editor.sections.selected')}
								{:else if sectionConfig.enabled}
									{$t('admin.view_editor.sections.all_items', { values: { count: publicItems.length } })}
								{:else}
									{publicItems.length} {$t('admin.view_editor.sections.available')}
								{/if}
							</span>
						{/if}
					</div>

					<div class="flex items-center gap-2">
						<!-- Desktop Width/Layout Selectors -->
						<div class="hidden lg:flex items-center gap-2">
							<!-- Width Selector with visual indicator -->
							{#if sectionConfig.enabled}
								{@const validWidths = getValidWidthsForLayout(layoutKey, sectionConfig.layout)}
								{#if validWidths.length > 1}
									<div class="flex items-center gap-1" title="Section width - controls side-by-side layout">
										<!-- Width icon indicator -->
										<div class="flex gap-0.5">
											{#if sectionConfig.width === 'half'}
												<div class="w-2 h-4 bg-primary-500 rounded-sm"></div>
												<div class="w-2 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"></div>
											{:else if sectionConfig.width === 'third'}
												<div class="w-1.5 h-4 bg-primary-500 rounded-sm"></div>
												<div class="w-1.5 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"></div>
												<div class="w-1.5 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"></div>
											{:else}
												<div class="w-5 h-4 bg-primary-500 rounded-sm"></div>
											{/if}
										</div>
										<select
											class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
											value={sectionConfig.width}
											onchange={(e) => updateSectionWidth(sectionKey, e.currentTarget.value)}
											onclick={stopPropagation(bubble('click'))}
										>
											{#each validWidths as widthOption}
												<option value={widthOption.value}>{widthOption.label}</option>
											{/each}
										</select>
									</div>
								{/if}
							{/if}

							<!-- Layout Selector -->
							{#if sectionConfig.enabled && VALID_LAYOUTS[layoutKey]}
								{@const layoutConfig = VALID_LAYOUTS[layoutKey]}
								<select
									class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
									value={sectionConfig.layout}
									onchange={(e) => updateSectionLayout(sectionKey, e.currentTarget.value)}
									onclick={stopPropagation(bubble('click'))}
									title="Section layout"
								>
									{#each layoutConfig.layouts as layoutOption}
										<option value={layoutOption}>{layoutConfig.labels[layoutOption] || layoutOption}</option>
									{/each}
								</select>
							{/if}
						</div>

						{#if sectionConfig.enabled && items.length > 0 && !isCustom}
							<button
								type="button"
								class="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 min-w-[44px] min-h-[44px] flex items-center justify-center"
								onclick={() => toggleSectionExpand(sectionKey)}
								aria-label={sectionConfig.expanded ? $t('admin.view_editor.sections.collapse_section', { values: { section: sectionLabel } }) : $t('admin.view_editor.sections.expand_section', { values: { section: sectionLabel } })}
							>
								<svg
									class="w-5 h-5 transition-transform {sectionConfig.expanded ? 'rotate-180' : ''}"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									aria-hidden="true"
								>
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
								</svg>
							</button>
						{/if}
					</div>
				</div>

			<!-- Section Items (only for non-custom sections) -->
				{#if sectionConfig.enabled && sectionConfig.expanded && items.length > 0 && !isCustom}
					<div class="p-3 border-t border-gray-200 dark:border-gray-700">
						<!-- Mobile Layout/Width Controls -->
						<div class="lg:hidden grid grid-cols-1 gap-3 mb-4 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-100 dark:border-gray-700">
							{#if sectionConfig.enabled}
								{@const validWidths = getValidWidthsForLayout(layoutKey, sectionConfig.layout)}
								{#if validWidths.length > 1}
									<div>
										<label for="width-mobile-{sectionKey}" class="text-xs font-medium text-gray-500 uppercase mb-1 block">{$t('admin.view_editor.sections.width_label')}</label>
										<div class="flex items-center gap-2">
											<div class="flex gap-0.5">
												{#if sectionConfig.width === 'half'}
													<div class="w-2 h-4 bg-primary-500 rounded-sm"></div>
													<div class="w-2 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"></div>
												{:else if sectionConfig.width === 'third'}
													<div class="w-1.5 h-4 bg-primary-500 rounded-sm"></div>
													<div class="w-1.5 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"></div>
													<div class="w-1.5 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"></div>
												{:else}
													<div class="w-5 h-4 bg-primary-500 rounded-sm"></div>
												{/if}
											</div>
											<select
												id="width-mobile-{sectionKey}"
												class="text-sm border border-gray-300 dark:border-gray-600 rounded px-2 py-1.5 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 w-full"
												value={sectionConfig.width}
												onchange={(e) => updateSectionWidth(sectionKey, e.currentTarget.value)}
											>
												{#each validWidths as widthOption}
													<option value={widthOption.value}>{widthOption.label}</option>
												{/each}
											</select>
										</div>
									</div>
								{/if}

								{#if VALID_LAYOUTS[layoutKey]}
									{@const layoutConfig = VALID_LAYOUTS[layoutKey]}
									<div>
										<label for="layout-mobile-{sectionKey}" class="text-xs font-medium text-gray-500 uppercase mb-1 block">{$t('admin.view_editor.sections.layout_label')}</label>
										<select
											id="layout-mobile-{sectionKey}"
											class="text-sm border border-gray-300 dark:border-gray-600 rounded px-2 py-1.5 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 w-full"
											value={sectionConfig.layout}
											onchange={(e) => updateSectionLayout(sectionKey, e.currentTarget.value)}
										>
											{#each layoutConfig.layouts as layoutOption}
												<option value={layoutOption}>{layoutConfig.labels[layoutOption] || layoutOption}</option>
											{/each}
										</select>
									</div>
								{/if}
							{/if}
						</div>

						<div class="flex items-center justify-between mb-2">
							<p class="text-xs text-gray-500">
								{selectedPublicCount === 0
									? $t('admin.view_editor.sections.select_all_hint')
									: $t('admin.view_editor.sections.items_selected', { values: { count: selectedPublicCount, total: publicItems.length } })}
							</p>
							<div class="flex gap-2">
								<button
									type="button"
									class="text-xs text-primary-600 hover:underline"
									onclick={() => selectAllItems(sectionKey)}
								>
									{$t('admin.view_editor.sections.select_all')}
								</button>
								<button
									type="button"
									class="text-xs text-gray-500 hover:underline"
									onclick={() => clearAllItems(sectionKey)}
								>
									{$t('admin.view_editor.sections.clear')}
								</button>
							</div>
						</div>

						<div
							class="space-y-1 max-h-64 overflow-y-auto"
							use:dndzone={{ 
								items: sectionItems[sectionKey] || [], 
								flipDurationMs, 
								type: `items-${sectionKey}`,
								dragDisabled: false,
								dropFromOthersDisabled: true
							}}
							onconsider={(e: any) => handleItemDndConsider(sectionKey, e)}
							onfinalize={(e: any) => handleItemDndFinalize(sectionKey, e)}
						>
							{#each items as item (item.id)}
								{@const isSelected = sectionConfig.items.includes(item.id)}
								{@const itemHasOverrides = hasOverrides(sectionKey, item.id)}
								{@const overrideCount = getOverrideCount(sectionKey, item.id)}
								{@const canOverride = OVERRIDABLE_FIELDS[sectionKey]?.length > 0}
						<div
							class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 bg-white dark:bg-gray-900"
							animate:flip={{ duration: flipDurationMs }}
						>
							<div
								class="cursor-grab active:cursor-grabbing p-1.5 min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700"
								title={$t('admin.view_editor.sections.drag_to_reorder')}
								role="presentation"
							>
								<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
								</svg>
							</div>
							<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
							<label
								class="flex items-center gap-2 flex-1 cursor-pointer"
								onpointerdown={(e) => e.stopPropagation()}
								onmousedown={(e) => e.stopPropagation()}
								ontouchstart={(e) => e.stopPropagation()}
							>
								<input
									type="checkbox"
									checked={isSelected}
									onchange={() => toggleItem(sectionKey, item.id)}
									onclick={(e) => e.stopPropagation()}
									class="w-4 h-4 text-primary-600 rounded border-gray-300"
								/>
								<div class="flex-1 min-w-0">
									<span class="block text-sm text-gray-700 dark:text-gray-300 truncate">
										{item.label}
									</span>
									{#if item.expand?.admin_tags && item.expand.admin_tags.length > 0}
										<div class="flex gap-1 mt-1 flex-wrap">
											{#each item.expand.admin_tags as tag (tag.id)}
												<AdminTagBadge name={tag.name} color={tag.color} />
											{/each}
										</div>
									{/if}
								</div>
							</label>
									{#if itemHasOverrides}
										<span class="px-1.5 py-0.5 text-xs bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300 rounded flex items-center gap-1">
											{@html icon('zap')}
											{overrideCount} {overrideCount > 1 ? $t('admin.view_editor.sections.overrides') : $t('admin.view_editor.sections.override')}
										</span>
									{/if}
								{#if item.visibility === 'private'}
									{@const enabledForThisView = isEnabledForView(item.data.view_visibility, viewId)}
										<span class="px-1.5 py-0.5 text-xs rounded {enabledForThisView ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'}">
											{enabledForThisView ? $t('admin.view_editor.sections.view_only') : $t('admin.view_editor.sections.private')}
										</span>
									{:else if item.visibility !== 'public'}
										<span class="px-1.5 py-0.5 text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300 rounded">
											{item.visibility}
										</span>
									{/if}
									{#if item.is_draft}
										<span class="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											{$t('admin.view_editor.sections.draft')}
										</span>
									{/if}
									{#if isSelected && canOverride}
										<button
											type="button"
											class="text-xs text-primary-600 hover:text-primary-700 hover:underline whitespace-nowrap"
											onclick={stopPropagation(() => onOpenOverrideEditor(sectionKey, item.id))}
										>
											{itemHasOverrides ? $t('admin.view_editor.sections.edit') : $t('admin.view_editor.sections.customize')}
										</button>
									{/if}
								</div>
					{/each}
				</div>

				{#if sectionKey === 'skills' && layoutUsesCategoryOrder(sectionConfig.layout)}
					<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
						<div class="flex items-center justify-between mb-2">
							<p class="text-xs text-gray-500">
								{sectionConfig.categoryOrder?.length
									? $t('admin.view_editor.sections.category_order_custom', { values: { count: sectionConfig.categoryOrder.length } })
									: $t('admin.view_editor.sections.category_order_alphabetical')}
							</p>
							<button
								type="button"
								class="text-xs text-primary-600 hover:underline"
								onclick={initCategoryOrder}
							>
								{sectionConfig.categoryOrder?.length ? $t('admin.view_editor.sections.edit_order') : $t('admin.view_editor.sections.customize_order')}
							</button>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/each}
</div>
{/key}

<!-- Category Reorder Modal -->
{#if showCategoryReorder}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
		<div class="bg-white dark:bg-gray-900 rounded-lg shadow-lg w-full max-w-md p-6 m-4">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.view_editor.category_reorder.title')}</h3>
				<button
					type="button"
					class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 p-1"
					onclick={closeCategoryReorder}
					aria-label={$t('shared.actions.close')}
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
			<p class="text-sm text-gray-500 mb-4">{$t('admin.view_editor.category_reorder.description')}</p>

			{#if dndLoaded && categoryItems.length > 0}
				<div
					class="space-y-2 max-h-64 overflow-y-auto"
					use:dndzone={{ items: categoryItems, flipDurationMs, type: 'categories' }}
					onconsider={handleCategoryDndConsider}
					onfinalize={handleCategoryDndFinalize}
				>
					{#each categoryItems as cat (cat.id)}
						<div
							class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
							animate:flip={{ duration: flipDurationMs }}
						>
							<div class="cursor-grab active:cursor-grabbing p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700">
								<svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
								</svg>
							</div>
							<span class="flex-1 text-sm font-medium text-gray-900 dark:text-white">{cat.name}</span>
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-sm text-gray-500">{$t('admin.view_editor.category_reorder.no_categories')}</p>
			{/if}

			<div class="flex justify-end gap-2 mt-4">
				<button
					type="button"
					class="btn btn-ghost"
					onclick={() => {
						sections['skills'].categoryOrder = undefined;
						updateSections();
						closeCategoryReorder();
					}}
				>
					{$t('admin.view_editor.category_reorder.reset_to_default')}
				</button>
				<button
					type="button"
					class="btn btn-primary"
					onclick={closeCategoryReorder}
				>
					{$t('admin.view_editor.category_reorder.done')}
				</button>
			</div>
		</div>
	</div>
{/if}
</div>
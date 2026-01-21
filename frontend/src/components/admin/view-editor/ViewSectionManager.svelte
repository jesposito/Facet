<script lang="ts">
	import { preventDefault, createBubbler, stopPropagation, self } from 'svelte/legacy';
	import { dndzone, TRIGGERS, SHADOW_PLACEHOLDER_ITEM_ID } from 'svelte-dnd-action';
	import { flip } from 'svelte/animate';
	import { icon } from '$lib/icons';
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

	const bubble = createBubbler();

	// Default section definitions - used to initialize and provide labels
	const SECTION_DEFS: Record<string, { label: string; collection: string }> = {
		experience: { label: 'Experience', collection: 'experience' },
		projects: { label: 'Projects', collection: 'projects' },
		education: { label: 'Education', collection: 'education' },
		certifications: { label: 'Certifications', collection: 'certifications' },
		awards: { label: 'Awards', collection: 'awards' },
		skills: { label: 'Skills', collection: 'skills' },
		posts: { label: 'Posts', collection: 'posts' },
		talks: { label: 'Talks', collection: 'talks' },
		contacts: { label: 'Contact Methods', collection: 'contact_methods' },
		testimonials: { label: 'Testimonials', collection: 'testimonials' }
	};

	const flipDurationMs = 200;

	// Props from parent
	let { 
		sections = $bindable(),
		sectionOrder = $bindable(),
		sectionItems = $bindable(),
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
		}>;
		sectionOrder: Array<{ id: string; key: string }>;
		sectionItems: Record<string, Array<{
			id: string;
			label: string;
			visibility: string;
			is_draft?: boolean;
			data: Record<string, unknown>;
		}>>;
		viewId: string;
		onOpenOverrideEditor: (sectionKey: string, itemId: string) => void;
	} = $props();

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
</script>

<!-- Sections -->
<div class="card p-4 sm:p-6 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Content Sections</h2>
			<p class="text-sm text-gray-500">Choose which sections to show and drag to reorder.</p>
		</div>
	</div>

	<div
		class="space-y-3"
		use:dndzone={{ items: sectionOrder, flipDurationMs, type: 'sections' }}
		onconsider={handleSectionDndConsider}
		onfinalize={handleSectionDndFinalize}
	>
		{#each sectionOrder as sectionItem (sectionItem.id)}
			{@const sectionKey = sectionItem.key}
			{@const sectionDef = SECTION_DEFS[sectionKey]}
			{@const sectionConfig = sections[sectionKey] || { enabled: false, items: [], expanded: false, itemConfig: {} }}
			{@const items = sectionItems[sectionKey] || []}
			{@const publicItems = items.filter(i => i.visibility !== 'private' && !i.is_draft)}

			<div
				class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900"
				animate:flip={{ duration: flipDurationMs }}
			>
				<!-- Section Header -->
				<div class="flex flex-wrap items-center justify-between p-3 bg-gray-50 dark:bg-gray-800/50">
					<div class="flex items-center gap-3">
						<!-- Drag Handle -->
						<div class="cursor-grab active:cursor-grabbing p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700" title="Drag to reorder">
							<svg class="w-6 h-6 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
							</svg>
						</div>
						<button
							type="button"
							class="w-10 h-6 rounded-full transition-colors relative
								{sectionConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
							onclick={() => toggleSection(sectionKey)}
							aria-label="Toggle {sectionDef?.label || sectionKey} section"
						>
							<span
								class="absolute top-1 w-4 h-4 bg-white rounded-full transition-transform shadow-sm
									{sectionConfig.enabled ? 'left-5' : 'left-1'}"
							></span>
						</button>
						<span class="font-medium text-gray-900 dark:text-white">{sectionDef?.label || sectionKey}</span>
						<span class="text-xs text-gray-500">
							{#if sectionConfig.items.length > 0}
								{sectionConfig.items.length} selected
							{:else if sectionConfig.enabled}
								all items ({publicItems.length})
							{:else}
								{publicItems.length} available
							{/if}
						</span>
					</div>

					<div class="flex items-center gap-2">
						<!-- Desktop Width/Layout Selectors -->
						<div class="hidden lg:flex items-center gap-2">
							<!-- Width Selector with visual indicator -->
							{#if sectionConfig.enabled}
								{@const validWidths = getValidWidthsForLayout(sectionKey, sectionConfig.layout)}
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
							{#if sectionConfig.enabled && VALID_LAYOUTS[sectionKey]}
								{@const layoutConfig = VALID_LAYOUTS[sectionKey]}
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

						{#if sectionConfig.enabled && items.length > 0}
							<button
								type="button"
								class="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 min-w-[44px] min-h-[44px] flex items-center justify-center"
								onclick={() => toggleSectionExpand(sectionKey)}
								aria-label="{sectionConfig.expanded ? 'Collapse' : 'Expand'} {sectionDef?.label || sectionKey} section"
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

			<!-- Section Items -->
				{#if sectionConfig.enabled && sectionConfig.expanded && items.length > 0}
					<div class="p-3 border-t border-gray-200 dark:border-gray-700">
						<!-- Mobile Layout/Width Controls -->
						<div class="lg:hidden grid grid-cols-1 gap-3 mb-4 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-100 dark:border-gray-700">
							{#if sectionConfig.enabled}
								{@const validWidths = getValidWidthsForLayout(sectionKey, sectionConfig.layout)}
								{#if validWidths.length > 1}
									<div>
										<label for="width-mobile-{sectionKey}" class="text-xs font-medium text-gray-500 uppercase mb-1 block">Width</label>
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

								{#if VALID_LAYOUTS[sectionKey]}
									{@const layoutConfig = VALID_LAYOUTS[sectionKey]}
									<div>
										<label for="layout-mobile-{sectionKey}" class="text-xs font-medium text-gray-500 uppercase mb-1 block">Layout</label>
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
								{sectionConfig.items.length === 0
									? 'All public items will be shown. Select and drag items to customize order.'
									: `${sectionConfig.items.length} of ${publicItems.length} items selected. Drag to reorder.`}
							</p>
							<div class="flex gap-2">
								<button
									type="button"
									class="text-xs text-primary-600 hover:underline"
									onclick={() => selectAllItems(sectionKey)}
								>
									Select All
								</button>
								<button
									type="button"
									class="text-xs text-gray-500 hover:underline"
									onclick={() => clearAllItems(sectionKey)}
								>
									Clear
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
							onconsider={(e) => handleItemDndConsider(sectionKey, e)}
							onfinalize={(e) => handleItemDndFinalize(sectionKey, e)}
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
								title="Drag to reorder"
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
										<span class="flex-1 text-sm text-gray-700 dark:text-gray-300 truncate">
											{item.label}
										</span>
									</label>
									{#if itemHasOverrides}
										<span class="px-1.5 py-0.5 text-xs bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300 rounded flex items-center gap-1">
											{@html icon('zap')}
											{overrideCount} override{overrideCount > 1 ? 's' : ''}
										</span>
									{/if}
								{#if item.visibility === 'private'}
									{@const enabledForThisView = isEnabledForView(item.data.view_visibility, viewId)}
										<span class="px-1.5 py-0.5 text-xs rounded {enabledForThisView ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'}" title={enabledForThisView ? 'Private globally, but visible in this view' : 'Private - will be auto-enabled when saved'}>
											{enabledForThisView ? 'view-only' : 'private'}
										</span>
									{:else if item.visibility !== 'public'}
										<span class="px-1.5 py-0.5 text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300 rounded">
											{item.visibility}
										</span>
									{/if}
									{#if item.is_draft}
										<span class="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											draft
										</span>
									{/if}
									{#if isSelected && canOverride}
										<button
											type="button"
											class="text-xs text-primary-600 hover:text-primary-700 hover:underline whitespace-nowrap"
											onclick={stopPropagation(() => onOpenOverrideEditor(sectionKey, item.id))}
										>
											{itemHasOverrides ? 'Edit' : 'Customize'}
										</button>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>
</div>
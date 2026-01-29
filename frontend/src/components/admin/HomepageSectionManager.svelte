<script lang="ts">
	/**
	 * HomepageSectionManager - Identical to ViewSectionManager but for the homepage
	 *
	 * Allows:
	 * - Drag-and-drop reordering of sections
	 * - Expanding sections to see/select individual items
	 * - Per-item selection (when none selected, shows all public items)
	 * - Drag-and-drop reordering of items within sections
	 */
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';
	import { flip } from 'svelte/animate';
	import type { CustomContent } from '$lib/pocketbase';
	import { VALID_LAYOUTS, getValidWidthsForLayout, isWidthValidForLayout, type SectionWidth } from '$lib/pocketbase';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import SkillsCategoryManager from '$components/admin/SkillsCategoryManager.svelte';

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
			applySavedItemOrder();
		}
	});

	const flipDurationMs = 200;

	// Section definitions - same as ViewSectionManager
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

	// Props
	let {
		sections = $bindable(),
		sectionOrder = $bindable(),
		sectionItems = $bindable(),
		customContentItems = [],
		loading = false
	}: {
		sections: Record<string, {
			enabled: boolean;
			items: string[];
			expanded: boolean;
			layout: string;
			width: string;
			categoryOrder?: string[];
			disabledCategories?: string[];
			categoryDisplayModes?: Record<string, string>;
			featuredId?: string;
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
		customContentItems: CustomContent[];
		loading?: boolean;
	} = $props();

	// Build a map for quick custom content title lookup
	$effect(() => {
		customContentTitleMap = new Map(customContentItems.map(item => [item.id, item.title]));
	});
	let customContentTitleMap: Map<string, string> = $state(new Map());

	// Apply saved item order after dndzone loads
	function applySavedItemOrder() {
		let didUpdate = false;
		for (const key of Object.keys(sections)) {
			const savedOrder = sections[key]?.items || [];
			const allItems = sectionItems[key] || [];

			if (savedOrder.length === 0 || allItems.length === 0) continue;

			const itemMap = new Map(allItems.map(item => [item.id, item]));
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
		if (didUpdate) {
			updateSectionItems();
		}
	}

	function updateSections() {
		sections = { ...sections };
	}

	function updateSectionItems() {
		sectionItems = { ...sectionItems };
	}

	function isCustomSection(sectionKey: string): boolean {
		return sectionKey.startsWith('custom:');
	}

	function getSectionLabel(sectionKey: string): string {
		if (isCustomSection(sectionKey)) {
			const customId = sectionKey.replace('custom:', '');
			const title = customContentTitleMap.get(customId);
			return title ? `Custom: ${title}` : `Custom: ${customId.slice(0, 8)}...`;
		}
		return SECTION_DEFS[sectionKey]?.label || sectionKey;
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
		// Only select public, non-draft items
		const publicItems = (sectionItems[sectionKey] || []).filter(i => i.visibility === 'public' && !i.is_draft);
		sections[sectionKey].items = publicItems.map((i) => i.id);
		updateSections();
	}

	function clearAllItems(sectionKey: string) {
		sections[sectionKey].items = [];
		updateSections();
	}

	function updateSectionWidth(sectionKey: string, width: string) {
		sections[sectionKey].width = width;
		updateSections();
	}

	function updateSectionLayout(sectionKey: string, layout: string) {
		sections[sectionKey].layout = layout;
		if (!isWidthValidForLayout(sectionKey, layout, sections[sectionKey].width as SectionWidth)) {
			sections[sectionKey].width = 'full';
		}
		updateSections();
	}

	// Drag-drop handlers for section reordering
	function handleSectionDndConsider(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items;
	}

	function handleSectionDndFinalize(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items;
	}

	// Drag-drop handlers for item reordering within a section
	// Note: dndzone now works with publicItems (filtered list)
	function handleItemDndConsider(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }>; info: { trigger: string } }>) {
		// Update sectionItems with the reordered public items
		const publicItems = e.detail.items;
		// Non-public items = private, unlisted, or draft - keep them at the end (not shown in UI)
		const nonPublicItems = (sectionItems[sectionKey] || []).filter(i => i.visibility !== 'public' || i.is_draft);
		sectionItems[sectionKey] = [...publicItems, ...nonPublicItems];
		updateSectionItems();
	}

	function handleItemDndFinalize(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }>; info: { trigger: string } }>) {
		const trigger = e.detail.info?.trigger;
		const finalPublicItems = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
		// Non-public items = private, unlisted, or draft - keep them at the end (not shown in UI)
		const nonPublicItems = (sectionItems[sectionKey] || []).filter(i => i.visibility !== 'public' || i.is_draft);
		sectionItems[sectionKey] = [...finalPublicItems, ...nonPublicItems];
		updateSectionItems();

		if (trigger === TRIGGERS.DROPPED_INTO_ZONE || trigger === TRIGGERS.DROPPED_INTO_ANOTHER) {
			updateItemsOrderFromDisplay(sectionKey);
		}
	}

	function updateItemsOrderFromDisplay(sectionKey: string) {
		// Get the order from the public items
		const publicItems = (sectionItems[sectionKey] || []).filter(i => i.visibility === 'public' && !i.is_draft);
		const displayOrder = publicItems.map(i => i.id);
		const selectedSet = new Set(sections[sectionKey].items);
		sections[sectionKey].items = displayOrder.filter(id => selectedSet.has(id));
		updateSections();
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Content Sections</h2>
			<p class="text-sm text-gray-500">Choose which sections to show and drag to reorder. Expand to select specific items.</p>
		</div>
	</div>

	{#if loading}
		<div class="animate-pulse text-center py-8">Loading sections...</div>
	{:else}
		{#key dndLoaded}
		<div
			class="space-y-3"
			use:dndzone={{ items: sectionOrder, flipDurationMs, type: 'homepage-sections' }}
			onconsider={handleSectionDndConsider}
			onfinalize={handleSectionDndFinalize}
		>
			{#each sectionOrder as sectionItem (sectionItem.id)}
				{@const sectionKey = sectionItem.key}
				{@const isCustom = isCustomSection(sectionKey)}
				{@const sectionLabel = getSectionLabel(sectionKey)}
				{@const sectionConfig = sections[sectionKey] || { enabled: false, items: [], expanded: false, layout: 'default', width: 'full' }}
				{@const items = isCustom ? [] : (sectionItems[sectionKey] || [])}
				{@const publicItems = items.filter(i => i.visibility === 'public' && !i.is_draft)}
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
							<div class="cursor-grab active:cursor-grabbing p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700" title="Drag to reorder">
								<svg class="w-6 h-6 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
								</svg>
							</div>
							<button
								type="button"
								class="w-10 h-6 rounded-full transition-colors relative
									{sectionConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
								onclick={() => toggleSection(sectionKey)}
								aria-label="Toggle {sectionLabel} section"
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
										{selectedPublicCount} selected
									{:else if sectionConfig.enabled}
										all items ({publicItems.length})
									{:else}
										{publicItems.length} available
									{/if}
								</span>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<!-- Layout/Width Selectors -->
							<div class="hidden lg:flex items-center gap-2">
								{#if sectionConfig.enabled}
									{@const validWidths = getValidWidthsForLayout(layoutKey, sectionConfig.layout)}
									{#if validWidths.length > 1}
										<div class="flex items-center gap-1" title="Section width">
											<select
												class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
												value={sectionConfig.width}
												onchange={(e) => updateSectionWidth(sectionKey, e.currentTarget.value)}
												onclick={(e) => e.stopPropagation()}
											>
												{#each validWidths as widthOption}
													<option value={widthOption.value}>{widthOption.label}</option>
												{/each}
											</select>
										</div>
									{/if}
								{/if}

								{#if sectionConfig.enabled && VALID_LAYOUTS[layoutKey]}
									{@const layoutConfig = VALID_LAYOUTS[layoutKey]}
									<select
										class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
										value={sectionConfig.layout}
										onchange={(e) => updateSectionLayout(sectionKey, e.currentTarget.value)}
										onclick={(e) => e.stopPropagation()}
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
									aria-label="{sectionConfig.expanded ? 'Collapse' : 'Expand'} {sectionLabel} section"
								>
									<svg
										class="w-5 h-5 transition-transform {sectionConfig.expanded ? 'rotate-180' : ''}"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
									</svg>
								</button>
							{/if}
						</div>
					</div>

					<!-- Section Items (expanded) -->
					{#if sectionConfig.enabled && sectionConfig.expanded && publicItems.length > 0 && !isCustom}
						<div class="p-3 border-t border-gray-200 dark:border-gray-700">
							<!-- Mobile Layout/Width Controls -->
							<div class="lg:hidden grid grid-cols-1 gap-3 mb-4 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-100 dark:border-gray-700">
								{#if sectionConfig.enabled}
									{@const validWidths = getValidWidthsForLayout(layoutKey, sectionConfig.layout)}
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

									{#if VALID_LAYOUTS[layoutKey]}
										{@const layoutConfig = VALID_LAYOUTS[layoutKey]}
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

							{#if sectionKey === 'skills'}
								<!-- Skills use category-based management -->
								<SkillsCategoryManager
									items={sectionItems['skills'] || []}
									bind:selectedItems={sections['skills'].items}
									bind:categoryOrder={sections['skills'].categoryOrder}
									bind:disabledCategories={sections['skills'].disabledCategories}
									bind:categoryDisplayModes={sections['skills'].categoryDisplayModes}
									sectionLayout={sections['skills'].layout || 'grouped'}
									onUpdate={updateSections}
								/>
							{:else}
								<!-- Standard items list for other sections -->
								<div class="flex items-center justify-between mb-2">
									<p class="text-xs text-gray-500">
										{selectedPublicCount === 0
											? 'All public items will be shown. Select items to customize.'
											: `${selectedPublicCount} of ${publicItems.length} items selected. Drag to reorder.`}
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
										items: publicItems,
										flipDurationMs,
										type: `items-${sectionKey}`,
										dragDisabled: false,
										dropFromOthersDisabled: true
									}}
									onconsider={(e: any) => handleItemDndConsider(sectionKey, e)}
									onfinalize={(e: any) => handleItemDndFinalize(sectionKey, e)}
								>
									{#each publicItems as item (item.id)}
										{@const isSelected = sectionConfig.items.includes(item.id)}
										<div
											class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 bg-white dark:bg-gray-900"
											animate:flip={{ duration: flipDurationMs }}
										>
											<div
												class="cursor-grab active:cursor-grabbing p-1.5 min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700"
												title="Drag to reorder"
											>
												<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
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
										</div>
									{/each}
								</div>

								{#if sectionKey === 'testimonials' && (sectionConfig.layout === 'featured' || sectionConfig.layout === 'carousel')}
									{@const selectedTestimonials = (sectionItems['testimonials'] || []).filter(t => sectionConfig.items.includes(t.id))}
									{#if selectedTestimonials.length > 0}
										<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
											<label for="homepage-featured-testimonial" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
												{sectionConfig.layout === 'featured' ? 'Featured Testimonial' : 'Primary Testimonial (shown first in carousel)'}
											</label>
											<select
												id="homepage-featured-testimonial"
												class="w-full text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
												value={sectionConfig.featuredId || ''}
												onchange={(e) => {
													sections['testimonials'].featuredId = e.currentTarget.value || undefined;
													updateSections();
												}}
											>
												<option value="">Use global featured (or first)</option>
												{#each selectedTestimonials as testimonial}
													<option value={testimonial.id}>{testimonial.label}</option>
												{/each}
											</select>
										</div>
									{/if}
								{/if}
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
		{/key}
	{/if}
</div>

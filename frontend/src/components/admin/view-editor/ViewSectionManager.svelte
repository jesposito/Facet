<script lang="ts">
	import { preventDefault, createBubbler, stopPropagation, self } from 'svelte/legacy';
	import { browser } from '$app/environment';
	import { onMount, tick } from 'svelte';
	import { flip } from 'svelte/animate';
	import { icon } from '$lib/icons';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import SkillsCategoryManager from '$components/admin/SkillsCategoryManager.svelte';
	import ReorderAnnouncer from '$lib/components/admin/ReorderAnnouncer.svelte';
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

	// Shared screen-reader announcer for keyboard + drag reorder (DRY).
	let reorderAnnouncer = $state<ReorderAnnouncer | undefined>(undefined);

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
		testimonials: { labelKey: 'admin.view_editor.sections.testimonials', collection: 'testimonials' },
		courses: { labelKey: 'admin.view_editor.sections.courses', collection: 'courses' }
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

	// Build a disambiguating subtitle for items in the selection list.
	function getItemSubtitle(sectionKey: string, item: { data: Record<string, unknown> }): string {
		const d = item.data;
		const parts: string[] = [];

		// Date range (experience, education)
		const startDate = d.start_date as string | undefined;
		const endDate = d.end_date as string | undefined;
		if (startDate) {
			const start = formatShortDate(startDate);
			const end = endDate ? formatShortDate(endDate) : $t('admin.view_editor.sections.subtitle_present');
			parts.push(`${start} – ${end}`);
		}

		// Description preview (projects, posts, testimonials)
		if (!startDate) {
			const desc = (d.description || d.summary || d.text || d.content) as string | undefined;
			if (desc) {
				const plain = desc.replace(/[#*_`\[\]]/g, '').trim();
				if (plain.length > 0) {
					parts.push(plain.length > 55 ? plain.slice(0, 55) + '…' : plain);
				}
			}
		}

		// Field of study (education)
		if (sectionKey === 'education') {
			const field = d.field_of_study as string | undefined;
			if (field) parts.push(field);
		}

		return parts.join(' · ');
	}

	function getImportSource(item: { data: Record<string, unknown> }): string | null {
		const importFile = item.data.import_filename as string | undefined;
		if (!importFile) return null;
		return importFile.replace(/\.[^.]+$/, '');
	}

	function formatShortDate(dateStr: string): string {
		try {
			const date = new Date(dateStr);
			if (isNaN(date.getTime())) return dateStr;
			return new Intl.DateTimeFormat('en', { month: 'short', year: 'numeric' }).format(date);
		} catch {
			return dateStr;
		}
	}

	// --- Search & Tag Filter ---
	let searchQueries: Record<string, string> = $state({});
	let activeTagFilters: Record<string, string | null> = $state({});

	function getSearchQuery(sectionKey: string): string {
		return searchQueries[sectionKey] || '';
	}

	function setSearchQuery(sectionKey: string, value: string) {
		searchQueries[sectionKey] = value;
		searchQueries = { ...searchQueries };
	}

	function getActiveTagFilter(sectionKey: string): string | null {
		return activeTagFilters[sectionKey] || null;
	}

	function setActiveTagFilter(sectionKey: string, tagId: string | null) {
		activeTagFilters[sectionKey] = tagId;
		activeTagFilters = { ...activeTagFilters };
	}

	function clearFilters(sectionKey: string) {
		searchQueries[sectionKey] = '';
		activeTagFilters[sectionKey] = null;
		searchQueries = { ...searchQueries };
		activeTagFilters = { ...activeTagFilters };
	}

	function getSectionTags(sectionKey: string): Array<{ id: string; name: string; color: string }> {
		const items = sectionItems[sectionKey] || [];
		const tagMap = new Map<string, { id: string; name: string; color: string }>();
		for (const item of items) {
			for (const tag of item.expand?.admin_tags || []) {
				if (!tagMap.has(tag.id)) tagMap.set(tag.id, tag);
			}
		}
		return Array.from(tagMap.values());
	}

	function filterItems(
		sectionKey: string,
		items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown>; expand?: { admin_tags?: Array<{ id: string; name: string; color: string }> } }>
	) {
		const query = getSearchQuery(sectionKey).toLowerCase().trim();
		const tagFilter = getActiveTagFilter(sectionKey);

		if (!query && !tagFilter) return items;

		return items.filter(item => {
			if (query) {
				const label = item.label.toLowerCase();
				const subtitle = getItemSubtitle(sectionKey, item).toLowerCase();
				const importSource = (getImportSource(item) || '').toLowerCase();
				if (!label.includes(query) && !subtitle.includes(query) && !importSource.includes(query)) {
					return false;
				}
			}
			if (tagFilter) {
				const itemTags = item.expand?.admin_tags || [];
				if (!itemTags.some(t => t.id === tagFilter)) return false;
			}
			return true;
		});
	}

	const SEARCH_THRESHOLD = 5;

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

	function selectAllItems(sectionKey: string, filteredOnly?: { id: string }[]) {
		const source = filteredOnly ?? sectionItems[sectionKey] ?? [];
		sections[sectionKey].items = source.map((i) => i.id);
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

	// Keyboard-accessible section reordering
	function moveSection(sectionId: string, direction: 'up' | 'down') {
		const idx = sectionOrder.findIndex(s => s.id === sectionId);
		if (idx === -1) return;
		if (direction === 'up' && idx === 0) return;
		if (direction === 'down' && idx === sectionOrder.length - 1) return;
		const newOrder = [...sectionOrder];
		const targetIdx = direction === 'up' ? idx - 1 : idx + 1;
		[newOrder[idx], newOrder[targetIdx]] = [newOrder[targetIdx], newOrder[idx]];
		sectionOrder = newOrder;
		updateSections();
	}

	// Drag-drop handlers for section reordering
	function handleSectionDndConsider(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items.filter((item) => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
	}

	function handleSectionDndFinalize(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items.filter((item) => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
	}

	// Drag-drop handlers for item reordering within a section
	function handleItemDndConsider(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }>; info: { trigger: string } }>) {
		sectionItems[sectionKey] = e.detail.items;
		updateSectionItems();
	}

	function handleItemDndFinalize(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }>; info: { trigger: string; id?: string } }>) {
		const trigger = e.detail.info?.trigger;
		const finalItems = e.detail.items.filter(item => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
		sectionItems[sectionKey] = finalItems;
		updateSectionItems();

		// Only update selection order if this was an actual drag operation, not just a click
		if (trigger === TRIGGERS.DROPPED_INTO_ZONE || trigger === TRIGGERS.DROPPED_INTO_ANOTHER) {
			updateItemsOrderFromDisplay(sectionKey);
			// Announce the dropped item's new position for screen-reader users.
			const droppedId = e.detail.info?.id;
			const droppedIdx = finalItems.findIndex(i => i.id === droppedId);
			if (droppedIdx !== -1) {
				announceItemMove(finalItems[droppedIdx].label, droppedIdx + 1, finalItems.length);
			}
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

	// Announce a reorder change for screen-reader users (position is 1-based).
	function announceItemMove(label: string, position: number, total: number) {
		reorderAnnouncer?.announce(
			$t('admin.view_editor.sections.item_reorder_announce', {
				values: { item: label, position, total }
			})
		);
	}

	// Keyboard-accessible item reordering within a section (complements drag).
	// Operates on the full sectionItems list (matches the non-filtered drag path)
	// and reuses updateItemsOrderFromDisplay for persistence.
	async function moveSectionItem(
		sectionKey: string,
		index: number,
		direction: 'up' | 'down',
		btn: HTMLButtonElement
	) {
		const list = sectionItems[sectionKey] || [];
		const targetIdx = direction === 'up' ? index - 1 : index + 1;
		if (targetIdx < 0 || targetIdx >= list.length) return;

		const moved = list[index];
		const next = [...list];
		[next[index], next[targetIdx]] = [next[targetIdx], next[index]];
		sectionItems[sectionKey] = next;
		updateSectionItems();
		updateItemsOrderFromDisplay(sectionKey);

		announceItemMove(moved.label, targetIdx + 1, list.length);

		// Keyed {#each} keeps focus on the moving button; redirect at a boundary
		// where the pressed button becomes disabled and would blur to <body>.
		await tick();
		if (btn.disabled) {
			const opposite = direction === 'up' ? 'down' : 'up';
			btn.parentElement
				?.querySelector<HTMLButtonElement>(`[data-move="${opposite}"]`)
				?.focus();
		}
	}

</script>

<!-- Sections -->
<div class="card p-4 sm:p-6 space-y-4">
	<!-- Polite live region for keyboard + drag reorder announcements -->
	<ReorderAnnouncer bind:this={reorderAnnouncer} />
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
				{@const sectionIdx = sectionOrder.findIndex(s => s.id === sectionItem.id)}
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
						<!-- Drag Handle + Keyboard reorder -->
						<div class="flex items-center gap-1">
							<div class="cursor-grab active:cursor-grabbing p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700" title={$t('admin.view_editor.sections.drag_to_reorder')}>
								<svg class="w-6 h-6 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
								</svg>
							</div>
							<button
								type="button"
								disabled={sectionIdx === 0}
								onclick={() => moveSection(sectionItem.id, 'up')}
								aria-label={$t('admin.view_editor.sections.move_up', { values: { section: sectionLabel } })}
								class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed"
							>
								<svg class="w-4 h-4 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7"/></svg>
							</button>
							<button
								type="button"
								disabled={sectionIdx === sectionOrder.length - 1}
								onclick={() => moveSection(sectionItem.id, 'down')}
								aria-label={$t('admin.view_editor.sections.move_down', { values: { section: sectionLabel } })}
								class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed"
							>
								<svg class="w-4 h-4 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"/></svg>
							</button>
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

						{#if sectionKey === 'skills'}
							<!-- Skills use category-based management -->
							<SkillsCategoryManager
								items={sectionItems['skills'] || []}
								bind:selectedItems={sections['skills'].items}
								bind:categoryOrder={sections['skills'].categoryOrder}
								bind:disabledCategories={sections['skills'].disabledCategories}
								bind:categoryDisplayModes={sections['skills'].categoryDisplayModes}
								sectionLayout={sections['skills'].layout || 'grouped'}
								showAllVisibilities={true}
								onUpdate={updateSections}
							/>
						{:else}
							<!-- Standard items list for other sections -->
							{@const filteredItems = filterItems(sectionKey, items)}
							{@const isFiltering = !!(getSearchQuery(sectionKey) || getActiveTagFilter(sectionKey))}
							{@const sectionTags = getSectionTags(sectionKey)}

							<!-- Search & Tag Filter (progressive disclosure: only for 5+ items) -->
							{#if items.length >= SEARCH_THRESHOLD}
								<div class="mb-3 space-y-2">
									<div class="flex items-center gap-2">
										<div class="relative flex-1">
											<input
												type="text"
												placeholder={$t('admin.view_editor.sections.search_placeholder')}
												value={getSearchQuery(sectionKey)}
												oninput={(e) => setSearchQuery(sectionKey, e.currentTarget.value)}
												onkeydown={(e) => { if (e.key === 'Escape') clearFilters(sectionKey); }}
												class="w-full text-sm border border-gray-300 dark:border-gray-600 rounded-lg pl-8 pr-8 py-1.5 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 placeholder:text-gray-400"
											/>
											<svg class="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
											</svg>
											{#if getSearchQuery(sectionKey)}
												<button
													type="button"
													class="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
													onclick={() => setSearchQuery(sectionKey, '')}
													aria-label={$t('admin.view_editor.sections.clear_search')}
												>
													<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
														<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
													</svg>
												</button>
											{/if}
										</div>
										{#if sectionTags.length > 0}
											<select
												class="text-xs border border-gray-300 dark:border-gray-600 rounded-lg px-2 py-1.5 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
												value={getActiveTagFilter(sectionKey) || ''}
												onchange={(e) => setActiveTagFilter(sectionKey, e.currentTarget.value || null)}
											>
												<option value="">{$t('admin.view_editor.sections.all_tags')}</option>
												{#each sectionTags as tag (tag.id)}
													<option value={tag.id}>{tag.name}</option>
												{/each}
											</select>
										{/if}
									</div>
								</div>
							{/if}

							<div class="flex items-center justify-between mb-2">
								<p class="text-xs text-gray-500">
									{#if isFiltering}
										{$t('admin.view_editor.sections.showing_filtered', { values: { shown: filteredItems.length, total: items.length } })}
									{:else if selectedPublicCount === 0}
										{$t('admin.view_editor.sections.select_all_hint')}
									{:else}
										{$t('admin.view_editor.sections.items_selected', { values: { count: selectedPublicCount, total: publicItems.length } })}
									{/if}
								</p>
								<div class="flex gap-2">
									{#if isFiltering}
										<button
											type="button"
											class="text-xs text-gray-500 hover:underline"
											onclick={() => clearFilters(sectionKey)}
										>
											{$t('admin.view_editor.sections.clear_filters')}
										</button>
									{/if}
									<button
										type="button"
										class="text-xs text-primary-600 hover:underline"
										onclick={() => selectAllItems(sectionKey, isFiltering ? filteredItems : undefined)}
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

							{#if isFiltering && filteredItems.length === 0}
								<div class="py-6 text-center">
									<p class="text-sm text-gray-500">{$t('admin.view_editor.sections.no_results')}</p>
									<button
										type="button"
										class="mt-2 text-xs text-primary-600 hover:underline"
										onclick={() => clearFilters(sectionKey)}
									>
										{$t('admin.view_editor.sections.clear_filters')}
									</button>
								</div>
							{:else}
							<div
								class="space-y-1 max-h-64 overflow-y-auto"
								use:dndzone={{
									items: isFiltering ? filteredItems : (sectionItems[sectionKey] || []),
									flipDurationMs,
									type: `items-${sectionKey}`,
									dragDisabled: isFiltering,
									dropFromOthersDisabled: true
								}}
								onconsider={(e: any) => { if (!isFiltering) handleItemDndConsider(sectionKey, e); }}
								onfinalize={(e: any) => { if (!isFiltering) handleItemDndFinalize(sectionKey, e); }}
							>
								{#each isFiltering ? filteredItems : items as item, itemIdx (item.id)}
									{@const isSelected = sectionConfig.items.includes(item.id)}
									{@const itemHasOverrides = hasOverrides(sectionKey, item.id)}
									{@const overrideCount = getOverrideCount(sectionKey, item.id)}
									{@const canOverride = OVERRIDABLE_FIELDS[sectionKey]?.length > 0}
									{@const subtitle = getItemSubtitle(sectionKey, item)}
									{@const importSource = getImportSource(item)}
									<div
										class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 bg-white dark:bg-gray-900"
										animate:flip={{ duration: flipDurationMs }}
									>
										<div
											class="p-1.5 min-w-[44px] min-h-[44px] flex items-center justify-center rounded {isFiltering ? 'opacity-30 pointer-events-none' : 'cursor-grab active:cursor-grabbing hover:bg-gray-200 dark:hover:bg-gray-700'}"
											title={isFiltering ? '' : $t('admin.view_editor.sections.drag_to_reorder')}
											role="presentation"
										>
											<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
											</svg>
										</div>
										<!-- Keyboard-accessible reorder controls (complement drag; disabled while filtering, like drag) -->
										<div class="flex flex-col flex-shrink-0">
											<button
												type="button"
												data-move="up"
												class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed"
												disabled={isFiltering || itemIdx === 0}
												onclick={(e) => moveSectionItem(sectionKey, itemIdx, 'up', e.currentTarget)}
												aria-label={$t('admin.view_editor.sections.item_move_up', { values: { item: item.label } })}
											>
												<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" />
												</svg>
											</button>
											<button
												type="button"
												data-move="down"
												class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed"
												disabled={isFiltering || itemIdx === (isFiltering ? filteredItems : items).length - 1}
												onclick={(e) => moveSectionItem(sectionKey, itemIdx, 'down', e.currentTarget)}
												aria-label={$t('admin.view_editor.sections.item_move_down', { values: { item: item.label } })}
											>
												<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
												</svg>
											</button>
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
												{#if subtitle || importSource}
													<span class="flex items-center gap-1.5 mt-0.5">
														{#if subtitle}
															<span class="text-xs text-gray-500 dark:text-gray-400 truncate" title={subtitle}>
																{subtitle}
															</span>
														{/if}
														{#if importSource}
															<span class="inline-flex items-center shrink-0 text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400" title={$t('admin.view_editor.sections.subtitle_from_import', { values: { filename: importSource } })}>
																{importSource}
															</span>
														{/if}
													</span>
												{/if}
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
							{/if}

							{#if sectionKey === 'testimonials' && (sectionConfig.layout === 'featured' || sectionConfig.layout === 'carousel')}
								{@const selectedTestimonials = (sectionItems['testimonials'] || []).filter(t => sectionConfig.items.includes(t.id))}
								{#if selectedTestimonials.length > 0}
									<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
										<label for="testimonial-featured-select" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
											{sectionConfig.layout === 'featured' ? $t('admin.view_editor.sections.featured_testimonial_label') : $t('admin.view_editor.sections.primary_testimonial_label')}
										</label>
										<select
											id="testimonial-featured-select"
											class="w-full text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
											value={sectionConfig.featuredId || ''}
											onchange={(e) => {
												sections['testimonials'].featuredId = e.currentTarget.value || undefined;
												updateSections();
											}}
										>
											<option value="">{$t('admin.view_editor.sections.use_global_featured')}</option>
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

</div>
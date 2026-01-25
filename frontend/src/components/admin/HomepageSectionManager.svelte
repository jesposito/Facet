<script lang="ts">
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';
	import { flip } from 'svelte/animate';
	import type { CustomContent } from '$lib/pocketbase';

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
		}
	});

	const flipDurationMs = 200;

	// Standard sections that can be ordered
	const SECTION_DEFS: Record<string, { label: string; description: string }> = {
		experience: { label: 'Experience', description: 'Work history and professional experience' },
		projects: { label: 'Projects', description: 'Portfolio projects and work samples' },
		education: { label: 'Education', description: 'Academic background and degrees' },
		certifications: { label: 'Certifications', description: 'Professional certifications and credentials' },
		awards: { label: 'Awards', description: 'Honors and achievements' },
		skills: { label: 'Skills', description: 'Technical and professional skills' },
		posts: { label: 'Posts', description: 'Blog posts and articles' },
		talks: { label: 'Talks', description: 'Presentations and speaking engagements' },
		testimonials: { label: 'Testimonials', description: 'Recommendations and endorsements' },
		contacts: { label: 'Contacts', description: 'Contact methods and social links' }
	};

	// Props
	let {
		sectionOrder = $bindable(),
		customContentItems = [],
		enabledCustomContent = $bindable(),
		loading = false
	}: {
		sectionOrder: Array<{ id: string; key: string }>;
		customContentItems: CustomContent[];
		enabledCustomContent: Set<string>;
		loading?: boolean;
	} = $props();

	// Track which sections are expanded
	let expandedSections: Set<string> = $state(new Set());

	// Helper to check if a section key is for custom content
	function isCustomSection(sectionKey: string): boolean {
		return sectionKey.startsWith('custom:');
	}

	// Get custom content ID from section key
	function getCustomContentId(sectionKey: string): string {
		return sectionKey.replace('custom:', '');
	}

	// Get display label for a section
	function getSectionLabel(sectionKey: string): string {
		if (isCustomSection(sectionKey)) {
			const customId = getCustomContentId(sectionKey);
			const item = customContentItems.find(c => c.id === customId);
			return item?.title || 'Custom Content';
		}
		return SECTION_DEFS[sectionKey]?.label || sectionKey;
	}

	// Get description for a section
	function getSectionDescription(sectionKey: string): string {
		if (isCustomSection(sectionKey)) {
			return 'Custom content block';
		}
		return SECTION_DEFS[sectionKey]?.description || '';
	}

	// Toggle custom content enabled state
	function toggleCustomContent(id: string) {
		const newSet = new Set(enabledCustomContent);
		if (newSet.has(id)) {
			newSet.delete(id);
			// Also remove from section order
			sectionOrder = sectionOrder.filter(s => s.key !== `custom:${id}`);
		} else {
			newSet.add(id);
			// Add to section order at end
			sectionOrder = [...sectionOrder, { id: `custom-${id}`, key: `custom:${id}` }];
		}
		enabledCustomContent = newSet;
	}

	// Toggle section expansion
	function toggleExpand(sectionKey: string) {
		const newSet = new Set(expandedSections);
		if (newSet.has(sectionKey)) {
			newSet.delete(sectionKey);
		} else {
			newSet.add(sectionKey);
		}
		expandedSections = newSet;
	}

	// Drag-drop handlers for section reordering
	function handleDndConsider(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items;
	}

	function handleDndFinalize(e: CustomEvent<{ items: typeof sectionOrder; info: { trigger: string } }>) {
		sectionOrder = e.detail.items;
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Section Order</h2>
			<p class="text-sm text-gray-500">Drag sections to reorder how they appear on your homepage.</p>
		</div>
	</div>

	<!-- Custom Content Toggle Panel -->
	{#if customContentItems.length > 0}
		<div class="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
			<h3 class="text-sm font-medium text-gray-900 dark:text-white mb-2">
				Custom Content Blocks
			</h3>
			<p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
				Toggle on to include in your homepage. Enabled blocks appear in the section list below.
			</p>
			<div class="space-y-2">
				{#each customContentItems as item (item.id)}
					{@const isEnabled = enabledCustomContent.has(item.id)}
					<div class="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700/50">
						<button
							type="button"
							class="w-10 h-6 rounded-full transition-colors relative shrink-0
								{isEnabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
							onclick={() => toggleCustomContent(item.id)}
							aria-label="Toggle {item.title}"
						>
							<span
								class="absolute top-1 w-4 h-4 bg-white rounded-full transition-transform shadow-sm
									{isEnabled ? 'left-5' : 'left-1'}"
							></span>
						</button>
						<span class="text-sm text-gray-700 dark:text-gray-300 {isEnabled ? 'font-medium' : ''}">
							{item.title}
						</span>
						<a
							href="/admin/custom"
							class="ml-auto text-xs text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
						>
							Edit
						</a>
					</div>
				{/each}
			</div>
		</div>
	{:else}
		<div class="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700 text-center">
			<p class="text-sm text-gray-500 dark:text-gray-400 mb-2">
				No custom content blocks yet.
			</p>
			<a href="/admin/custom" class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400">
				Create custom content →
			</a>
		</div>
	{/if}

	<!-- Section Order List with Drag-and-Drop -->
	{#key dndLoaded}
	<div
		class="space-y-2"
		use:dndzone={{ items: sectionOrder, flipDurationMs, type: 'homepage-sections' }}
		onconsider={handleDndConsider}
		onfinalize={handleDndFinalize}
	>
		{#each sectionOrder as sectionItem (sectionItem.id)}
			{@const sectionKey = sectionItem.key}
			{@const isCustom = isCustomSection(sectionKey)}
			{@const sectionLabel = getSectionLabel(sectionKey)}
			{@const sectionDesc = getSectionDescription(sectionKey)}
			{@const isExpanded = expandedSections.has(sectionKey)}

			<div
				class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900
					{isCustom ? 'border-primary-300 dark:border-primary-700' : ''}"
				animate:flip={{ duration: flipDurationMs }}
			>
				<!-- Section Header -->
				<div class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800/50">
					<!-- Drag Handle -->
					<div
						class="cursor-grab active:cursor-grabbing p-2 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
						title="Drag to reorder"
					>
						<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
						</svg>
					</div>

					<!-- Section Info -->
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<span class="font-medium text-gray-900 dark:text-white">{sectionLabel}</span>
							{#if isCustom}
								<span class="px-1.5 py-0.5 text-xs bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300 rounded">
									custom
								</span>
							{/if}
						</div>
						{#if sectionDesc && !isExpanded}
							<p class="text-xs text-gray-500 dark:text-gray-400 truncate">{sectionDesc}</p>
						{/if}
					</div>

					<!-- Expand Button (for non-custom sections) -->
					{#if !isCustom}
						<button
							type="button"
							class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
							onclick={() => toggleExpand(sectionKey)}
							aria-label="{isExpanded ? 'Collapse' : 'Expand'} {sectionLabel} section"
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
					{:else}
						<!-- Remove button for custom content -->
						<button
							type="button"
							class="p-2 text-gray-400 hover:text-red-500 rounded hover:bg-red-50 dark:hover:bg-red-900/20"
							onclick={() => toggleCustomContent(getCustomContentId(sectionKey))}
							title="Remove from homepage"
						>
							<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>

				<!-- Expanded Content (for standard sections) -->
				{#if isExpanded && !isCustom}
					<div class="p-3 border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
						<p class="text-sm text-gray-600 dark:text-gray-400">
							{sectionDesc}
						</p>
						<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
							All public items in this section will be displayed on your homepage.
							To customize which items appear, use <a href="/admin/views" class="text-primary-600 hover:underline">Facets</a> to create targeted views.
						</p>
					</div>
				{/if}
			</div>
		{/each}
	</div>
	{/key}

	{#if sectionOrder.length === 0}
		<div class="text-center py-8 text-gray-500 dark:text-gray-400">
			<p>No sections configured. Standard sections will be displayed in default order.</p>
		</div>
	{/if}
</div>

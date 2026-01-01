<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pb, type ViewSection, type Profile, type SectionWidth, VALID_LAYOUTS, VALID_WIDTHS } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { dndzone } from 'svelte-dnd-action';
	import { flip } from 'svelte/animate';
	import ViewPreview from '$components/admin/ViewPreview.svelte';

	// Default section definitions - used to initialize and provide labels
	const SECTION_DEFS: Record<string, { label: string; collection: string }> = {
		experience: { label: 'Experience', collection: 'experience' },
		projects: { label: 'Projects', collection: 'projects' },
		education: { label: 'Education', collection: 'education' },
		certifications: { label: 'Certifications', collection: 'certifications' },
		skills: { label: 'Skills', collection: 'skills' },
		posts: { label: 'Posts', collection: 'posts' },
		talks: { label: 'Talks', collection: 'talks' }
	};

	// Default section order
	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'skills', 'posts', 'talks'];

	let loading = true;
	let saving = false;

	// Profile data for preview
	let profile: Profile | null = null;

	// Preview panel state
	let showPreview = true;

	// Form fields
	let name = '';
	let slug = '';
	let description = '';
	let visibility: 'public' | 'unlisted' | 'private' | 'password' = 'public';
	let heroHeadline = '';
	let heroSummary = '';
	let ctaText = '';
	let ctaUrl = '';
	let isActive = true;
	let isDefault = false;

	// Sections configuration with layout and width support (itemConfig empty for new views)
	let sections: Record<string, { enabled: boolean; items: string[]; expanded: boolean; layout: string; width: SectionWidth; itemConfig: Record<string, { overrides?: Record<string, string | string[]> }> }> = {};

	// Section order for drag-drop
	let sectionOrder: Array<{ id: string; key: string }> = [];
	const flipDurationMs = 200;

	// Available items for each section (with full data for preview)
	let sectionItems: Record<string, Array<{
		id: string;
		label: string;
		visibility: string;
		is_draft?: boolean;
		data: Record<string, unknown>;
	}>> = {};

	// Simple pattern - admin layout handles auth
	onMount(async () => {
		initializeSections();
		await Promise.all([
			loadSectionItems(),
			loadProfile()
		]);
		loading = false;
	});

	async function loadProfile() {
		try {
			const records = await pb.collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				const record = records.items[0] as unknown as Profile & { collectionId: string };
				if (record.avatar) {
					const avatarUrl = `/api/files/${record.collectionId}/${record.id}/${record.avatar}`;
					profile = { ...record, avatar: avatarUrl };
				} else {
					profile = record;
				}
			}
		} catch (err) {
			console.error('Failed to load profile:', err);
		}
	}

	function initializeSections() {
		// Start with all sections enabled by default for new views, with default layout and full width
		for (const key of DEFAULT_SECTION_ORDER) {
			const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
			sections[key] = { enabled: true, items: [], expanded: false, layout: defaultLayout, width: 'full', itemConfig: {} };
		}
		// Initialize section order
		sectionOrder = DEFAULT_SECTION_ORDER.map(key => ({ id: `section-${key}`, key }));
	}

	async function loadSectionItems() {
		for (const key of DEFAULT_SECTION_ORDER) {
			const def = SECTION_DEFS[key];
			try {
				const records = await pb.collection(def.collection).getList(1, 100, {
					sort: '-id'
				});

				sectionItems[key] = records.items.map((item) => ({
					id: item.id,
					label: getItemLabel(key, item),
					visibility: (item as Record<string, unknown>).visibility as string || 'public',
					is_draft: (item as Record<string, unknown>).is_draft as boolean || false,
					data: item as Record<string, unknown>
				}));
			} catch (err) {
				console.error(`Failed to load ${key} items:`, err);
				sectionItems[key] = [];
			}
		}
	}

	function getItemLabel(sectionKey: string, item: Record<string, unknown>): string {
		switch (sectionKey) {
			case 'experience':
				return `${item.title} at ${item.company}`;
			case 'projects':
				return item.title as string;
			case 'education':
				return `${item.degree || 'Degree'} - ${item.institution}`;
			case 'certifications':
				return `${item.name} (${item.issuer || 'Unknown issuer'})`;
			case 'skills':
				return `${item.name}${item.category ? ` (${item.category})` : ''}`;
			case 'posts':
				return item.title as string;
			case 'talks':
				return `${item.title}${item.event ? ` @ ${item.event}` : ''}`;
			default:
				return item.title as string || item.name as string || item.id as string;
		}
	}

	// Helper to trigger reactivity by creating a new object reference
	function updateSections() {
		sections = { ...sections };
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
		updateSections();
	}

	function generateSlug(value: string): string {
		return value
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 50);
	}

	function handleNameInput() {
		slug = generateSlug(name);
	}

	// Drag-drop handlers for section reordering
	function handleSectionDndConsider(e: CustomEvent<{ items: typeof sectionOrder }>) {
		sectionOrder = e.detail.items;
	}

	function handleSectionDndFinalize(e: CustomEvent<{ items: typeof sectionOrder }>) {
		sectionOrder = e.detail.items;
	}

	// Drag-drop handlers for item reordering within a section
	function handleItemDndConsider(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }> }>) {
		sectionItems[sectionKey] = e.detail.items;
		updateItemsOrderFromDisplay(sectionKey);
	}

	function handleItemDndFinalize(sectionKey: string, e: CustomEvent<{ items: Array<{ id: string; label: string; visibility: string; is_draft?: boolean; data: Record<string, unknown> }> }>) {
		sectionItems[sectionKey] = e.detail.items;
		updateItemsOrderFromDisplay(sectionKey);
	}

	function updateItemsOrderFromDisplay(sectionKey: string) {
		const displayOrder = sectionItems[sectionKey]?.map(i => i.id) || [];
		const selectedSet = new Set(sections[sectionKey].items);
		sections[sectionKey].items = displayOrder.filter(id => selectedSet.has(id));
		updateSections();
	}

	async function handleSubmit() {
		if (!name.trim()) {
			toasts.add('error', 'Name is required');
			return;
		}
		if (!slug.trim()) {
			toasts.add('error', 'Slug is required');
			return;
		}

		// Check for reserved slugs
		const reservedSlugs = [
			'admin', 'api', 's', 'v', '_app', '_', 'assets', 'static',
			'favicon.ico', 'robots.txt', 'sitemap.xml',
			'health', 'healthz', 'ready', 'login', 'logout',
			'auth', 'oauth', 'callback', 'home', 'index', 'default', 'profile',
			'projects', 'posts', 'new'
		];
		if (reservedSlugs.includes(slug.toLowerCase())) {
			toasts.add('error', `"${slug}" is a reserved slug. Please choose another.`);
			return;
		}

		saving = true;
		try {
			// Build sections array in current order with layout and width
			const sectionsData: ViewSection[] = sectionOrder
				.map(({ key }) => ({
					section: key,
					enabled: sections[key]?.enabled || false,
					items: sections[key]?.items || [],
					layout: sections[key]?.layout || VALID_LAYOUTS[key]?.default || 'default',
					width: sections[key]?.width || 'full'
				}));

			const data = {
				name: name.trim(),
				slug: slug.trim(),
				description: description.trim(),
				visibility,
				hero_headline: heroHeadline.trim() || null,
				hero_summary: heroSummary.trim() || null,
				cta_text: ctaText.trim() || null,
				cta_url: ctaUrl.trim() || null,
				is_active: isActive,
				is_default: isDefault,
				sections: sectionsData
			};

			const newView = await pb.collection('views').create(data);

			// If setting as default, clear other defaults
			if (isDefault) {
				const currentDefaults = await pb.collection('views').getList(1, 100, {
					filter: `is_default = true && id != "${newView.id}"`
				});
				for (const v of currentDefaults.items) {
					await pb.collection('views').update(v.id, { is_default: false });
				}
			}

			toasts.add('success', 'View created successfully');
			goto('/admin/views');
		} catch (err) {
			console.error('Failed to create view:', err);
			const message = err instanceof Error ? err.message : 'Failed to create view';
			if (message.includes('slug')) {
				toasts.add('error', 'A view with this slug already exists');
			} else {
				toasts.add('error', message);
			}
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Create View | Me.yaml</title>
</svelte:head>

<div class="view-editor-container">
	{#if loading}
		<div class="card p-8 text-center max-w-4xl mx-auto">
			<div class="animate-pulse">Loading...</div>
		</div>
	{:else}
		<!-- Header -->
		<div class="flex items-center justify-between mb-6 px-4">
			<div class="flex items-center gap-4">
				<a href="/admin/views" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</a>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Create View</h1>
			</div>
			<div class="flex items-center gap-2">
				<!-- Preview Toggle -->
				<button
					type="button"
					class="btn btn-ghost flex items-center gap-2"
					on:click={() => showPreview = !showPreview}
					title={showPreview ? 'Hide preview' : 'Show preview'}
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						{#if showPreview}
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
						{:else}
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						{/if}
					</svg>
					<span class="hidden sm:inline">{showPreview ? 'Hide' : 'Show'} Preview</span>
				</button>
				<button type="button" class="btn btn-primary" on:click={handleSubmit} disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					Create View
				</button>
			</div>
		</div>

		<!-- Split Pane Layout -->
		<div class="editor-layout" class:with-preview={showPreview}>
			<!-- Editor Pane -->
			<div class="editor-pane">
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<!-- Basic Info -->
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h2>

				<div>
					<label for="name" class="label">Name *</label>
					<input
						type="text"
						id="name"
						bind:value={name}
						on:input={handleNameInput}
						class="input"
						placeholder="Recruiter View"
						required
					/>
					<p class="text-xs text-gray-500 mt-1">Internal name for this view</p>
				</div>

				<div>
					<label for="slug" class="label">URL Slug *</label>
					<div class="flex items-center gap-2">
						<span class="text-gray-500 text-sm">/</span>
						<input
							type="text"
							id="slug"
							bind:value={slug}
							class="input flex-1"
							placeholder="recruiter"
							required
						/>
					</div>
					<p class="text-xs text-gray-500 mt-1">Public URL will be: /{slug}</p>
				</div>

				<div>
					<label for="description" class="label">Description</label>
					<textarea
						id="description"
						bind:value={description}
						class="input min-h-[80px]"
						placeholder="Internal notes about this view..."
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Private notes (not shown publicly)</p>
				</div>
			</div>

			<!-- Hero Overrides -->
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Hero Overrides</h2>
				<p class="text-sm text-gray-500 -mt-2">Override your profile headline and summary for this view</p>

				<div>
					<label for="hero_headline" class="label">Custom Headline</label>
					<input
						type="text"
						id="hero_headline"
						bind:value={heroHeadline}
						class="input"
						placeholder="Leave empty to use profile headline"
					/>
				</div>

				<div>
					<label for="hero_summary" class="label">Custom Summary</label>
					<textarea
						id="hero_summary"
						bind:value={heroSummary}
						class="input min-h-[120px]"
						placeholder="Leave empty to use profile summary (Markdown supported)"
					></textarea>
				</div>
			</div>

			<!-- Call to Action -->
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Call to Action</h2>
				<p class="text-sm text-gray-500 -mt-2">Add a prominent button to this view</p>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="cta_text" class="label">Button Text</label>
						<input
							type="text"
							id="cta_text"
							bind:value={ctaText}
							class="input"
							placeholder="Download Resume"
						/>
					</div>
					<div>
						<label for="cta_url" class="label">Button URL</label>
						<input
							type="url"
							id="cta_url"
							bind:value={ctaUrl}
							class="input"
							placeholder="https://..."
						/>
					</div>
				</div>
			</div>

			<!-- Settings -->
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Settings</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="visibility" class="label">Visibility</label>
						<select id="visibility" bind:value={visibility} class="input">
							<option value="public">Public - Anyone can access</option>
							<option value="unlisted">Unlisted - Only with share token</option>
							<option value="password">Password - Requires password</option>
							<option value="private">Private - Admin only</option>
						</select>
					</div>
				</div>

				<div class="flex flex-col gap-3 pt-2">
					<label class="flex items-center gap-3">
						<input
							type="checkbox"
							bind:checked={isActive}
							class="w-4 h-4 text-primary-600 rounded border-gray-300"
						/>
						<div>
							<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Active</span>
							<p class="text-xs text-gray-500">Inactive views are not accessible publicly</p>
						</div>
					</label>

					<label class="flex items-center gap-3">
						<input
							type="checkbox"
							bind:checked={isDefault}
							class="w-4 h-4 text-primary-600 rounded border-gray-300"
						/>
						<div>
							<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Default View</span>
							<p class="text-xs text-gray-500">Show this view on the homepage (/)</p>
						</div>
					</label>
				</div>
			</div>

			<!-- Sections -->
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Content Sections</h2>
						<p class="text-sm text-gray-500">Choose which sections to show and drag to reorder.</p>
					</div>
				</div>

				<div
					class="space-y-3"
					use:dndzone={{ items: sectionOrder, flipDurationMs, type: 'sections' }}
					on:consider={handleSectionDndConsider}
					on:finalize={handleSectionDndFinalize}
				>
					{#each sectionOrder as sectionItem (sectionItem.id)}
						{@const sectionKey = sectionItem.key}
						{@const sectionDef = SECTION_DEFS[sectionKey]}
						{@const sectionConfig = sections[sectionKey] || { enabled: false, items: [], expanded: false }}
						{@const items = sectionItems[sectionKey] || []}
						{@const publicItems = items.filter(i => i.visibility !== 'private' && !i.is_draft)}

						<div
							class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900"
							animate:flip={{ duration: flipDurationMs }}
						>
							<!-- Section Header -->
							<div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800/50">
								<div class="flex items-center gap-3">
									<!-- Drag Handle -->
									<div class="cursor-grab active:cursor-grabbing p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700" title="Drag to reorder">
										<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
										</svg>
									</div>
									<button
										type="button"
										class="w-10 h-6 rounded-full transition-colors relative
											{sectionConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
										on:click={() => toggleSection(sectionKey)}
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
									<!-- Width Selector with visual indicator -->
									{#if sectionConfig.enabled}
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
												on:change={(e) => updateSectionWidth(sectionKey, e.currentTarget.value)}
												on:click|stopPropagation
											>
												{#each VALID_WIDTHS as widthOption}
													<option value={widthOption.value}>{widthOption.label}</option>
												{/each}
											</select>
										</div>
									{/if}

									<!-- Layout Selector -->
									{#if sectionConfig.enabled && VALID_LAYOUTS[sectionKey]}
										{@const layoutConfig = VALID_LAYOUTS[sectionKey]}
										<select
											class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
											value={sectionConfig.layout}
											on:change={(e) => updateSectionLayout(sectionKey, e.currentTarget.value)}
											on:click|stopPropagation
											title="Section layout"
										>
											{#each layoutConfig.layouts as layoutOption}
												<option value={layoutOption}>{layoutConfig.labels[layoutOption] || layoutOption}</option>
											{/each}
										</select>
									{/if}

									{#if sectionConfig.enabled && items.length > 0}
										<button
											type="button"
											class="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
											on:click={() => toggleSectionExpand(sectionKey)}
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

							<!-- Section Items -->
							{#if sectionConfig.enabled && sectionConfig.expanded && items.length > 0}
								<div class="p-3 border-t border-gray-200 dark:border-gray-700">
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
												on:click={() => selectAllItems(sectionKey)}
											>
												Select All
											</button>
											<button
												type="button"
												class="text-xs text-gray-500 hover:underline"
												on:click={() => clearAllItems(sectionKey)}
											>
												Clear
											</button>
										</div>
									</div>

									<div
										class="space-y-1 max-h-48 overflow-y-auto"
										use:dndzone={{ items: sectionItems[sectionKey] || [], flipDurationMs, type: `items-${sectionKey}` }}
										on:consider={(e) => handleItemDndConsider(sectionKey, e)}
										on:finalize={(e) => handleItemDndFinalize(sectionKey, e)}
									>
										{#each items as item (item.id)}
											{@const isSelected = sectionConfig.items.includes(item.id)}
											<div
												class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 bg-white dark:bg-gray-900"
												animate:flip={{ duration: flipDurationMs }}
											>
												<!-- Drag Handle for Items -->
												<div class="cursor-grab active:cursor-grabbing p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700" title="Drag to reorder">
													<svg class="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
														<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
													</svg>
												</div>
												<label class="flex items-center gap-2 flex-1 cursor-pointer">
													<input
														type="checkbox"
														checked={isSelected}
														on:change={() => toggleItem(sectionKey, item.id)}
														class="w-4 h-4 text-primary-600 rounded border-gray-300"
													/>
													<span class="flex-1 text-sm text-gray-700 dark:text-gray-300 truncate">
														{item.label}
													</span>
												</label>
												{#if item.visibility !== 'public'}
													<span class="px-1.5 py-0.5 text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300 rounded">
														{item.visibility}
													</span>
												{/if}
												{#if item.is_draft}
													<span class="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
														draft
													</span>
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

			</form>
			</div>

			<!-- Preview Pane -->
			{#if showPreview}
				<div class="preview-pane">
					<div class="sticky top-4">
						<div class="flex items-center justify-between mb-3 px-1">
							<h2 class="text-sm font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide">Live Preview</h2>
							<span class="text-xs text-gray-400">Updates as you edit</span>
						</div>
						<ViewPreview
							{profile}
							{heroHeadline}
							{heroSummary}
							{ctaText}
							{ctaUrl}
							{sections}
							{sectionOrder}
							{sectionItems}
						/>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.view-editor-container {
		max-width: 100%;
		padding: 0 1rem;
	}

	.editor-layout {
		display: flex;
		gap: 1.5rem;
		align-items: flex-start;
	}

	.editor-pane {
		flex: 1;
		min-width: 0;
		max-width: 48rem;
	}

	.editor-layout.with-preview .editor-pane {
		flex: 3;
		max-width: none;
	}

	.preview-pane {
		flex: 2;
		min-width: 320px;
		max-width: 480px;
	}

	@media (max-width: 1024px) {
		.editor-layout {
			flex-direction: column;
		}

		.editor-pane {
			max-width: 100%;
		}

		.preview-pane {
			width: 100%;
			max-width: 100%;
			order: -1;
			margin-bottom: 1rem;
		}
	}

	@media (min-width: 1280px) {
		.view-editor-container {
			padding: 0 2rem;
		}

		.preview-pane {
			max-width: 560px;
		}
	}
</style>

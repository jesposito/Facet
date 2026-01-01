<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { pb, type View, type ViewSection, type ItemConfig, OVERRIDABLE_FIELDS } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';

	// Available sections - must match backend
	const AVAILABLE_SECTIONS = [
		{ key: 'experience', label: 'Experience', collection: 'experience' },
		{ key: 'projects', label: 'Projects', collection: 'projects' },
		{ key: 'education', label: 'Education', collection: 'education' },
		{ key: 'certifications', label: 'Certifications', collection: 'certifications' },
		{ key: 'skills', label: 'Skills', collection: 'skills' },
		{ key: 'posts', label: 'Posts', collection: 'posts' },
		{ key: 'talks', label: 'Talks', collection: 'talks' }
	];

	let loading = true;
	let saving = false;
	let view: View | null = null;

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

	// Sections configuration with itemConfig support
	let sections: Record<string, {
		enabled: boolean;
		items: string[];
		expanded: boolean;
		itemConfig: Record<string, ItemConfig>;
	}> = {};

	// Available items for each section (full data for override editing)
	let sectionItems: Record<string, Array<{
		id: string;
		label: string;
		visibility: string;
		is_draft?: boolean;
		data: Record<string, unknown>;
	}>> = {};

	// Override editor state
	let showOverrideEditor = false;
	let editingOverride: {
		sectionKey: string;
		itemId: string;
		itemLabel: string;
		originalData: Record<string, unknown>;
		overrides: Record<string, string | string[]>;
	} | null = null;

	$: viewId = $page.params.id as string;

	// Simple pattern - admin layout handles auth
	onMount(async () => {
		if (!viewId) {
			toasts.add('error', 'Invalid view ID');
			goto('/admin/views');
			return;
		}
		await loadView();
		await loadSectionItems();
	});

	async function loadView() {
		if (!viewId) return;
		loading = true;
		try {
			const record = await pb.collection('views').getOne(viewId);
			view = record as unknown as View;

			// Populate form fields
			name = view.name;
			slug = view.slug;
			description = view.description || '';
			visibility = view.visibility;
			heroHeadline = view.hero_headline || '';
			heroSummary = view.hero_summary || '';
			ctaText = view.cta_text || '';
			ctaUrl = view.cta_url || '';
			isActive = view.is_active;

			// Check if this is the default view
			const defaultViews = await pb.collection('views').getList(1, 1, {
				filter: 'is_default = true'
			});
			isDefault = defaultViews.items.length > 0 && defaultViews.items[0].id === viewId;

			// Initialize sections from view data
			initializeSections(view.sections);
		} catch (err) {
			console.error('Failed to load view:', err);
			toasts.add('error', 'Failed to load view');
			goto('/admin/views');
		} finally {
			loading = false;
		}
	}

	function initializeSections(viewSections?: ViewSection[]) {
		// Start with all sections disabled
		for (const section of AVAILABLE_SECTIONS) {
			sections[section.key] = { enabled: false, items: [], expanded: false, itemConfig: {} };
		}

		// Apply saved section configuration
		if (viewSections) {
			for (const vs of viewSections) {
				if (sections[vs.section]) {
					sections[vs.section].enabled = vs.enabled;
					sections[vs.section].items = vs.items || [];
					sections[vs.section].itemConfig = vs.itemConfig || {};
				}
			}
		}
	}

	async function loadSectionItems() {
		for (const section of AVAILABLE_SECTIONS) {
			try {
				const records = await pb.collection(section.collection).getList(1, 100, {
					sort: '-id'
				});

				sectionItems[section.key] = records.items.map((item) => ({
					id: item.id,
					label: getItemLabel(section.key, item),
					visibility: (item as Record<string, unknown>).visibility as string || 'public',
					is_draft: (item as Record<string, unknown>).is_draft as boolean || false,
					data: item as Record<string, unknown>
				}));
			} catch (err) {
				console.error(`Failed to load ${section.key} items:`, err);
				sectionItems[section.key] = [];
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

	function toggleSection(key: string) {
		sections[key].enabled = !sections[key].enabled;
		sections = sections; // Trigger reactivity
	}

	function toggleSectionExpand(key: string) {
		sections[key].expanded = !sections[key].expanded;
		sections = sections;
	}

	function toggleItem(sectionKey: string, itemId: string) {
		const idx = sections[sectionKey].items.indexOf(itemId);
		if (idx === -1) {
			sections[sectionKey].items.push(itemId);
		} else {
			sections[sectionKey].items.splice(idx, 1);
		}
		sections = sections;
	}

	function selectAllItems(sectionKey: string) {
		sections[sectionKey].items = sectionItems[sectionKey]?.map((i) => i.id) || [];
		sections = sections;
	}

	function clearAllItems(sectionKey: string) {
		sections[sectionKey].items = [];
		sections = sections;
	}

	function generateSlug(value: string): string {
		return value
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 50);
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

		saving = true;
		try {
			// Build sections array with itemConfig
			const sectionsData: ViewSection[] = AVAILABLE_SECTIONS
				.filter((s) => sections[s.key].enabled)
				.map((s) => {
					const sectionData: ViewSection = {
						section: s.key,
						enabled: true,
						items: sections[s.key].items
					};
					// Only include itemConfig if there are overrides
					const itemConfig = sections[s.key].itemConfig;
					if (itemConfig && Object.keys(itemConfig).length > 0) {
						// Filter out empty configs
						const filteredConfig: Record<string, ItemConfig> = {};
						for (const [itemId, config] of Object.entries(itemConfig)) {
							if (config.overrides && Object.keys(config.overrides).length > 0) {
								filteredConfig[itemId] = config;
							}
						}
						if (Object.keys(filteredConfig).length > 0) {
							sectionData.itemConfig = filteredConfig;
						}
					}
					return sectionData;
				});

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
				sections: sectionsData
			};

			await pb.collection('views').update(viewId, data);

			// Handle default view setting
			if (isDefault) {
				// Clear other defaults first
				const currentDefaults = await pb.collection('views').getList(1, 100, {
					filter: `is_default = true && id != "${viewId}"`
				});
				for (const v of currentDefaults.items) {
					await pb.collection('views').update(v.id, { is_default: false });
				}
				await pb.collection('views').update(viewId, { is_default: true });
			}

			toasts.add('success', 'View updated successfully');
		} catch (err) {
			console.error('Failed to save view:', err);
			toasts.add('error', 'Failed to save view');
		} finally {
			saving = false;
		}
	}

	function previewView() {
		window.open(`/${slug}`, '_blank');
	}

	// Override editor functions
	function openOverrideEditor(sectionKey: string, itemId: string) {
		const item = sectionItems[sectionKey]?.find(i => i.id === itemId);
		if (!item) return;

		const existingConfig = sections[sectionKey]?.itemConfig?.[itemId];
		editingOverride = {
			sectionKey,
			itemId,
			itemLabel: item.label,
			originalData: item.data,
			overrides: existingConfig?.overrides
				? { ...existingConfig.overrides }
				: {}
		};
		showOverrideEditor = true;
	}

	function closeOverrideEditor() {
		showOverrideEditor = false;
		editingOverride = null;
	}

	function saveOverrides() {
		if (!editingOverride) return;

		const { sectionKey, itemId, overrides } = editingOverride;

		// Clean up empty overrides
		const cleanedOverrides: Record<string, string | string[]> = {};
		for (const [field, value] of Object.entries(overrides)) {
			if (value && (typeof value === 'string' ? value.trim() : value.length > 0)) {
				cleanedOverrides[field] = value;
			}
		}

		// Update itemConfig
		if (!sections[sectionKey].itemConfig) {
			sections[sectionKey].itemConfig = {};
		}

		if (Object.keys(cleanedOverrides).length > 0) {
			sections[sectionKey].itemConfig[itemId] = { overrides: cleanedOverrides };
		} else {
			delete sections[sectionKey].itemConfig[itemId];
		}

		sections = sections; // Trigger reactivity
		closeOverrideEditor();
		toasts.add('success', 'Overrides saved');
	}

	function clearOverride(field: string) {
		if (!editingOverride) return;
		delete editingOverride.overrides[field];
		editingOverride = editingOverride; // Trigger reactivity
	}

	function hasOverrides(sectionKey: string, itemId: string): boolean {
		const config = sections[sectionKey]?.itemConfig?.[itemId];
		return !!(config?.overrides && Object.keys(config.overrides).length > 0);
	}

	function getOverrideCount(sectionKey: string, itemId: string): number {
		const config = sections[sectionKey]?.itemConfig?.[itemId];
		return config?.overrides ? Object.keys(config.overrides).length : 0;
	}

	function formatFieldValue(value: unknown): string {
		if (Array.isArray(value)) {
			return value.join('\n');
		}
		return String(value || '');
	}

	function parseFieldValue(field: string, value: string): string | string[] {
		// bullets field should be an array
		if (field === 'bullets') {
			return value.split('\n').filter(line => line.trim());
		}
		return value;
	}

	function handleOverrideInput(field: string, event: Event) {
		if (!editingOverride) return;
		const target = event.target as HTMLInputElement | HTMLTextAreaElement;
		editingOverride.overrides[field] = parseFieldValue(field, target.value);
	}
</script>

<svelte:head>
	<title>Edit View | Me.yaml</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading view...</div>
		</div>
	{:else}
		<div class="flex items-center justify-between mb-6">
			<div class="flex items-center gap-4">
				<a href="/admin/views" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</a>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Edit View</h1>
			</div>
			<div class="flex items-center gap-2">
				<button type="button" class="btn btn-secondary" on:click={previewView}>
					Preview
				</button>
				<button type="button" class="btn btn-primary" on:click={handleSubmit} disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					Save Changes
				</button>
			</div>
		</div>

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
						on:input={() => { if (!view?.slug) slug = generateSlug(name); }}
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
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Content Sections</h2>
				<p class="text-sm text-gray-500 -mt-2">Choose which sections to show in this view. Expand each section to select specific items.</p>

				<div class="space-y-3">
					{#each AVAILABLE_SECTIONS as section}
						{@const sectionConfig = sections[section.key] || { enabled: false, items: [], expanded: false }}
						{@const items = sectionItems[section.key] || []}
						{@const publicItems = items.filter(i => i.visibility !== 'private' && !i.is_draft)}

						<div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
							<!-- Section Header -->
							<div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800/50">
								<div class="flex items-center gap-3">
									<button
										type="button"
										class="w-10 h-6 rounded-full transition-colors relative
											{sectionConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
										on:click={() => toggleSection(section.key)}
									>
										<span
											class="absolute top-1 w-4 h-4 bg-white rounded-full transition-transform shadow-sm
												{sectionConfig.enabled ? 'left-5' : 'left-1'}"
										></span>
									</button>
									<span class="font-medium text-gray-900 dark:text-white">{section.label}</span>
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

								{#if sectionConfig.enabled && items.length > 0}
									<button
										type="button"
										class="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
										on:click={() => toggleSectionExpand(section.key)}
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

							<!-- Section Items -->
							{#if sectionConfig.enabled && sectionConfig.expanded && items.length > 0}
								<div class="p-3 border-t border-gray-200 dark:border-gray-700">
									<div class="flex items-center justify-between mb-2">
										<p class="text-xs text-gray-500">
											{sectionConfig.items.length === 0
												? 'All public items will be shown. Select specific items below to customize.'
												: `${sectionConfig.items.length} of ${publicItems.length} items selected`}
										</p>
										<div class="flex gap-2">
											<button
												type="button"
												class="text-xs text-primary-600 hover:underline"
												on:click={() => selectAllItems(section.key)}
											>
												Select All
											</button>
											<button
												type="button"
												class="text-xs text-gray-500 hover:underline"
												on:click={() => clearAllItems(section.key)}
											>
												Clear
											</button>
										</div>
									</div>

									<div class="space-y-1 max-h-64 overflow-y-auto">
										{#each items as item}
											{@const isSelected = sectionConfig.items.includes(item.id)}
											{@const itemHasOverrides = hasOverrides(section.key, item.id)}
											{@const overrideCount = getOverrideCount(section.key, item.id)}
											{@const canOverride = OVERRIDABLE_FIELDS[section.key]?.length > 0}
											<div class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800">
												<label class="flex items-center gap-2 flex-1 cursor-pointer">
													<input
														type="checkbox"
														checked={isSelected}
														on:change={() => toggleItem(section.key, item.id)}
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
												{#if isSelected && canOverride}
													<button
														type="button"
														class="text-xs text-primary-600 hover:text-primary-700 hover:underline whitespace-nowrap"
														on:click|stopPropagation={() => openOverrideEditor(section.key, item.id)}
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

		</form>
	{/if}
</div>

<!-- Override Editor Modal -->
{#if showOverrideEditor && editingOverride}
	{@const overridableFields = OVERRIDABLE_FIELDS[editingOverride.sectionKey] || []}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="card w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
			<div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
				<div>
					<h2 class="text-lg font-bold text-gray-900 dark:text-white">Customize for this View</h2>
					<p class="text-sm text-gray-500">{editingOverride.itemLabel}</p>
				</div>
				<button type="button" class="btn btn-ghost" on:click={closeOverrideEditor}>
					{@html icon('x')}
				</button>
			</div>

			<div class="p-4 space-y-6 overflow-y-auto flex-1">
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Override fields below to customize how this item appears in this view. Leave fields empty to use the original value.
				</p>

				{#each overridableFields as field}
					{@const originalValue = editingOverride.originalData[field]}
					{@const hasOverride = field in editingOverride.overrides}
					{@const isArrayField = field === 'bullets'}

					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<label for="override_{field}" class="label capitalize">{field.replace('_', ' ')}</label>
							{#if hasOverride}
								<button
									type="button"
									class="text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
									on:click={() => clearOverride(field)}
								>
									Reset to original
								</button>
							{/if}
						</div>

						<!-- Original value (collapsed) -->
						<details class="text-sm">
							<summary class="text-gray-500 cursor-pointer hover:text-gray-700 dark:hover:text-gray-300">
								View original value
							</summary>
							<div class="mt-2 p-2 bg-gray-50 dark:bg-gray-800 rounded text-gray-600 dark:text-gray-400 whitespace-pre-wrap text-xs">
								{formatFieldValue(originalValue) || '(empty)'}
							</div>
						</details>

						<!-- Override input -->
						{#if isArrayField}
							<textarea
								id="override_{field}"
								class="input min-h-[100px] font-mono text-sm"
								placeholder="Enter one item per line..."
								value={hasOverride ? formatFieldValue(editingOverride.overrides[field]) : ''}
								on:input={(e) => handleOverrideInput(field, e)}
							></textarea>
							<p class="text-xs text-gray-500">Enter one bullet point per line</p>
						{:else if field === 'description' || field === 'summary'}
							<textarea
								id="override_{field}"
								class="input min-h-[100px]"
								placeholder="Enter override value or leave empty for original..."
								value={hasOverride ? String(editingOverride.overrides[field]) : ''}
								on:input={(e) => handleOverrideInput(field, e)}
							></textarea>
						{:else}
							<input
								type="text"
								id="override_{field}"
								class="input"
								placeholder="Enter override value or leave empty for original..."
								value={hasOverride ? String(editingOverride.overrides[field]) : ''}
								on:input={(e) => handleOverrideInput(field, e)}
							/>
						{/if}
					</div>
				{/each}

				{#if overridableFields.length === 0}
					<p class="text-gray-500 text-center py-8">
						This section type does not support field overrides.
					</p>
				{/if}
			</div>

			<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
				<button type="button" class="btn btn-ghost" on:click={closeOverrideEditor}>
					Cancel
				</button>
				<button type="button" class="btn btn-primary" on:click={saveOverrides}>
					Save Overrides
				</button>
			</div>
		</div>
	</div>
{/if}

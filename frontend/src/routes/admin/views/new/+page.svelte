<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pb, type ViewSection } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';

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

	// Sections configuration
	let sections: Record<string, { enabled: boolean; items: string[]; expanded: boolean }> = {};

	// Available items for each section
	let sectionItems: Record<string, Array<{ id: string; label: string; visibility: string; is_draft?: boolean }>> = {};

	// Simple pattern - admin layout handles auth
	onMount(async () => {
		initializeSections();
		await loadSectionItems();
		loading = false;
	});

	function initializeSections() {
		// Start with all sections enabled by default for new views
		for (const section of AVAILABLE_SECTIONS) {
			sections[section.key] = { enabled: true, items: [], expanded: false };
		}
	}

	async function loadSectionItems() {
		for (const section of AVAILABLE_SECTIONS) {
			try {
				const records = await pb.collection(section.collection).getList(1, 100, {
					sort: '-created'
				});

				sectionItems[section.key] = records.items.map((item) => ({
					id: item.id,
					label: getItemLabel(section.key, item),
					visibility: (item as Record<string, unknown>).visibility as string || 'public',
					is_draft: (item as Record<string, unknown>).is_draft as boolean || false
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
		sections = sections;
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

	function handleNameInput() {
		slug = generateSlug(name);
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
			// Build sections array
			const sectionsData: ViewSection[] = AVAILABLE_SECTIONS
				.filter((s) => sections[s.key].enabled)
				.map((s) => ({
					section: s.key,
					enabled: true,
					items: sections[s.key].items
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

<div class="max-w-4xl mx-auto">
	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading...</div>
		</div>
	{:else}
		<div class="flex items-center justify-between mb-6">
			<div class="flex items-center gap-4">
				<a href="/admin/views" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</a>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Create View</h1>
			</div>
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

									<div class="space-y-1 max-h-48 overflow-y-auto">
										{#each items as item}
											{@const isSelected = sectionConfig.items.includes(item.id)}
											<label class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer">
												<input
													type="checkbox"
													checked={isSelected}
													on:change={() => toggleItem(section.key, item.id)}
													class="w-4 h-4 text-primary-600 rounded border-gray-300"
												/>
												<span class="flex-1 text-sm text-gray-700 dark:text-gray-300 truncate">
													{item.label}
												</span>
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
											</label>
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

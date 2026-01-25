<script lang="ts">
	import { preventDefault, run } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { pb, type View, type ViewSection, type SectionWidth, type ItemConfig, VALID_LAYOUTS } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import ViewSectionManager from '$components/admin/view-editor/ViewSectionManager.svelte';
	import type { DndEvent } from 'svelte-dnd-action';

	// Dnd
	let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
	let SHADOW_PLACEHOLDER_ITEM_ID = $state('');
	
	onMount(async () => {
		if (browser) {
			const dnd = await import('svelte-dnd-action');
			dndzone = dnd.dndzone;
			SHADOW_PLACEHOLDER_ITEM_ID = dnd.SHADOW_PLACEHOLDER_ITEM_ID;
		}
		await Promise.all([loadSettings(), loadProfile()]);
	});

	// Homepage visibility settings
	let settingsLoading = $state(true);
	let homepageEnabled = $state(true);
	let landingPageMessage = $state('This profile is being set up.');
	let hideLoginButton = $state(false);

	// Site navigation settings
	interface NavItem {
		id: string; // for dnd
		view_id: string;
		label: string;
		visible: boolean;
	}
	interface PublicView {
		id: string;
		name: string;
		slug: string;
	}
	let navEnabled = $state(false);
	let navHomeLabel = $state('Home');
	let navItems = $state<NavItem[]>([]);
	let publicViews = $state<PublicView[]>([]);

	// Homepage Sections (ViewSectionManager state)
	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'awards', 'skills', 'posts', 'talks', 'testimonials', 'custom', 'contacts'];
	
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
		testimonials: { label: 'Testimonials', collection: 'testimonials' },
		custom: { label: 'Custom Content', collection: 'custom' }
	};

	let sections: Record<string, {
		enabled: boolean;
		items: string[];
		expanded: boolean;
		layout: string;
		width: SectionWidth;
		itemConfig: Record<string, ItemConfig>;
		categoryOrder?: string[];
	}> = $state({});

	let sectionOrder: Array<{ id: string; key: string }> = $state([]);

	let sectionItems: Record<string, Array<{
		id: string;
		label: string;
		visibility: string;
		is_draft?: boolean;
		data: Record<string, unknown>;
		expand?: {
			admin_tags?: Array<{ id: string; name: string; color: string }>;
		};
	}>> = $state({});

	// Profile data
	let profile: Record<string, unknown> | null = null;
	let profileLoading = $state(true);
	let profileSaving = $state(false);

	// Profile Form fields
	let name = $state('');
	let headline = $state('');
	let location = $state('');
	let summary = $state('');
	let contactEmail = $state('');
	let contactLinks: Array<{ type: string; url: string; label: string }> = $state([]);
	let ctaText = $state('');
	let ctaUrl = $state('');
	let ctaButtonText = $state('');

	// Image fields
	let avatarUrl: string | null = $state(null);
	let heroImageUrl: string | null = $state(null);
	let avatarFile: File | null = null;
	let heroImageFile: File | null = null;

	async function loadSettings() {
		try {
			settingsLoading = true;
			// Load views for navigation
			const viewsResult = await collection('views').getList(1, 100, {
				filter: 'visibility = "public" && is_active = true',
				sort: 'sort_order,created',
				requestKey: null
			});
			publicViews = viewsResult.items.map(v => ({ id: v.id, name: v.name, slug: v.slug }));

			// Load settings
			const settings = await pb.send('/api/site-settings', { method: 'GET', requestKey: null });
			
			// Homepage Visibility
			homepageEnabled = settings.homepage_enabled !== false;
			landingPageMessage = settings.landing_page_message || '';
			hideLoginButton = settings.hide_login_button === true;

			// Site Nav
			if (settings.site_nav) {
				navEnabled = settings.site_nav.enabled === true;
				navHomeLabel = settings.site_nav.home_label || 'Home';
				
				// Map backend items to frontend items
				// We add 'id' for dnd using view_id
				const savedItems = (settings.site_nav.items || []).map((item: any) => ({
					...item,
					id: item.view_id // Use view_id as unique key for dnd
				}));
				navItems = savedItems;
				
				// Sync with public views (add missing ones)
				syncNavItemsWithViews();
			} else {
				syncNavItemsWithViews();
			}

			// Homepage Sections
			initializeSections(settings.homepage_sections);
			await loadSectionItems();

		} catch (err) {
			console.error('Failed to load settings:', err);
			toasts.add('error', 'Failed to load settings');
		} finally {
			settingsLoading = false;
		}
	}

	function syncNavItemsWithViews() {
		const existingIds = new Set(navItems.map(item => item.view_id));
		const viewIds = new Set(publicViews.map(v => v.id));
		
		// Remove items for views that no longer exist or are not public
		navItems = navItems.filter(item => viewIds.has(item.view_id));
		
		// Add new views
		for (const view of publicViews) {
			if (!existingIds.has(view.id)) {
				navItems = [...navItems, { 
					id: view.id,
					view_id: view.id, 
					label: '', 
					visible: true 
				}];
			}
		}
	}

	function getViewName(id: string) {
		return publicViews.find(v => v.id === id)?.name || 'Unknown View';
	}

	// --- Homepage Sections Logic ---

	function initializeSections(savedSections?: ViewSection[]) {
		// Initialize all sections
		for (const key of DEFAULT_SECTION_ORDER) {
			const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
			sections[key] = { 
				enabled: false, 
				items: [], 
				expanded: false, 
				layout: defaultLayout, 
				width: 'full', 
				itemConfig: {} 
			};
		}

		if (savedSections && savedSections.length > 0) {
			// Apply saved config
			const savedOrder = savedSections.map(s => s.section);
			const remaining = DEFAULT_SECTION_ORDER.filter(k => !savedOrder.includes(k));
			const fullOrder = [...savedOrder, ...remaining];
			
			sectionOrder = fullOrder.map(key => ({ id: `section-${key}`, key }));

			for (const s of savedSections) {
				if (sections[s.section]) {
					sections[s.section].enabled = s.enabled;
					sections[s.section].items = s.items || [];
					sections[s.section].layout = s.layout || VALID_LAYOUTS[s.section]?.default || 'default';
					sections[s.section].width = s.width || 'full';
					sections[s.section].itemConfig = s.itemConfig || {};
					sections[s.section].categoryOrder = s.categoryOrder;
				}
			}
		} else {
			// Default order
			sectionOrder = DEFAULT_SECTION_ORDER.map(key => ({ id: `section-${key}`, key }));
		}
	}

	async function loadSectionItems() {
		for (const key of DEFAULT_SECTION_ORDER) {
			const def = SECTION_DEFS[key];
			try {
				const filter = key === 'testimonials' ? 'status = "approved"' : '';
				const records = await collection(def.collection).getList(1, 100, {
					sort: key === 'testimonials' ? '-featured,-sort_order' : '-id',
					filter,
					expand: 'admin_tags',
					requestKey: null
				});

				sectionItems[key] = records.items.map((item) => ({
					id: item.id,
					label: getItemLabel(key, item),
					visibility: (item as Record<string, unknown>).visibility as string || 'public',
					is_draft: (item as Record<string, unknown>).is_draft as boolean || false,
					data: item as Record<string, unknown>,
					expand: (item as any).expand || {}
				}));
			} catch (err) {
				console.error(`Failed to load ${key} items:`, err);
				sectionItems[key] = [];
			}
		}
		// Trigger reactivity
		sectionItems = { ...sectionItems };
	}

	function getItemLabel(sectionKey: string, item: Record<string, unknown>): string {
		switch (sectionKey) {
			case 'experience': return `${item.title} at ${item.company}`;
			case 'projects': return item.title as string;
			case 'education': return `${item.degree || 'Degree'} - ${item.institution}`;
			case 'certifications': return `${item.name} (${item.issuer || 'Unknown issuer'})`;
			case 'awards': return `${item.title}${item.issuer ? ` (${item.issuer})` : ''}`;
			case 'skills': return `${item.name}${item.category ? ` (${item.category})` : ''}`;
			case 'posts': return item.title as string;
			case 'talks': return `${item.title}${item.event ? ` @ ${item.event}` : ''}`;
			case 'contacts': return `${item.label || item.type} - ${item.value}`;
			case 'testimonials': return `${item.author_name}${item.author_company ? ` - ${item.author_company}` : ''}`;
			case 'custom': return item.title as string;
			default: return item.title as string || item.name as string || item.id as string;
		}
	}

	// --- Autosave Logic ---
	let saveTimeout: ReturnType<typeof setTimeout>;
	let isSaving = $state(false);

	function triggerAutosave() {
		if (settingsLoading) return;
		clearTimeout(saveTimeout);
		saveTimeout = setTimeout(saveSettingsToBackend, 1000);
	}

	async function saveSettingsToBackend() {
		if (isSaving) return;
		isSaving = true;

		// Build sections data
		const sectionsData: ViewSection[] = sectionOrder
			.map(({ key }) => ({
				section: key,
				enabled: sections[key]?.enabled || false,
				items: sections[key]?.items || [],
				layout: sections[key]?.layout || 'default',
				width: sections[key]?.width || 'full',
				itemConfig: sections[key]?.itemConfig,
				categoryOrder: sections[key]?.categoryOrder
			}));

		// Build nav data
		const navData = {
			enabled: navEnabled,
			home_label: navHomeLabel,
			items: navItems.map(item => ({
				view_id: item.view_id,
				label: item.label,
				visible: item.visible
			}))
		};

		const data = {
			homepage_enabled: homepageEnabled,
			landing_page_message: landingPageMessage,
			hide_login_button: hideLoginButton,
			homepage_sections: sectionsData,
			site_nav: navData
		};

		try {
			await pb.send('/api/site-settings', {
				method: 'PUT',
				body: data
			});
			// Success - quiet save
		} catch (err) {
			console.error('Failed to save settings:', err);
			toasts.add('error', 'Failed to save settings');
		} finally {
			isSaving = false;
		}
	}

	// Watchers for autosave
	run(() => {
		// Dependencies that trigger autosave
		const _ = {
			homepageEnabled,
			landingPageMessage,
			hideLoginButton,
			navEnabled,
			navHomeLabel,
			navItems, 
			sections, 
			sectionOrder
		};
		// Trigger if loaded
		if (!settingsLoading) {
			triggerAutosave();
		}
	});

	// --- Site Nav Dnd ---
	function handleNavDndConsider(e: CustomEvent<DndEvent<NavItem>>) {
		navItems = e.detail.items;
	}
	function handleNavDndFinalize(e: CustomEvent<DndEvent<NavItem>>) {
		navItems = e.detail.items;
		triggerAutosave();
	}

	// --- Profile Logic ---
	async function loadProfile() {
		try {
			profileLoading = true;
			const records = await collection('profile').getList(1, 1, { requestKey: null });
			if (records.items.length > 0) {
				const p = records.items[0];
				profile = p;
				name = p.name;
				headline = p.headline;
				location = p.location;
				summary = p.summary;
				contactEmail = p.contact_email;
				contactLinks = p.contact_links || [];
				ctaText = p.cta_text;
				ctaUrl = p.cta_url;
				ctaButtonText = p.cta_button_text;
				
				if (p.avatar) avatarUrl = pb.files.getUrl(p, p.avatar);
				if (p.hero_image) heroImageUrl = pb.files.getUrl(p, p.hero_image);
			}
		} catch (err) {
			console.error('Failed to load profile:', err);
		} finally {
			profileLoading = false;
		}
	}

	async function saveProfile() {
		profileSaving = true;
		try {
			const data = {
				name, headline, location, summary,
				contact_email: contactEmail,
				contact_links: contactLinks,
				cta_text: ctaText,
				cta_url: ctaUrl,
				cta_button_text: ctaButtonText
			};

			let record: any;
			if (profile) {
				record = await collection('profile').update(profile.id as string, data);
			} else {
				record = await collection('profile').create({ ...data, visibility: 'public' });
			}
			
			// Handle images
			if (avatarFile) {
				const formData = new FormData();
				formData.append('avatar', avatarFile);
				record = await collection('profile').update(record.id, formData);
			}
			if (heroImageFile) {
				const formData = new FormData();
				formData.append('hero_image', heroImageFile);
				record = await collection('profile').update(record.id, formData);
			}

			profile = record;
			toasts.add('success', 'Profile saved successfully');
		} catch (err) {
			console.error('Failed to save profile:', err);
			toasts.add('error', 'Failed to save profile');
		} finally {
			profileSaving = false;
		}
	}

    // Image handlers
	function handleAvatarChange(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files?.length) {
			avatarFile = input.files[0];
			avatarUrl = URL.createObjectURL(avatarFile);
		}
	}

	function removeAvatar() {
		avatarFile = null;
		avatarUrl = null;
        if (profile?.avatar) {
             collection('profile').update(profile.id as string, { avatar: null });
        }
	}

	function handleHeroImageChange(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files?.length) {
			heroImageFile = input.files[0];
			heroImageUrl = URL.createObjectURL(heroImageFile);
		}
	}

	function removeHeroImage() {
		heroImageFile = null;
		heroImageUrl = null;
        if (profile?.hero_image) {
             collection('profile').update(profile.id as string, { hero_image: null });
        }
	}
</script>

<svelte:head>
	<title>Homepage | Facet</title>
</svelte:head>

<div class="max-w-4xl mx-auto pb-20">
	<PageHelp pageKey="homepage">
		<p><strong>Homepage</strong> controls what visitors see at your root URL.</p>
		<p>Enable or disable public access, customize the landing page message for when your site is hidden, and edit your core profile information that appears across all facets.</p>
		<p><strong>Site Navigation</strong> adds a navigation bar letting visitors browse between your homepage and public facets. Great when you have multiple facets for different audiences.</p>
	</PageHelp>

	<div class="mb-6 flex justify-between items-center">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Homepage</h1>
			<p class="text-gray-600 dark:text-gray-400 mt-1">
				Manage your public profile and control what visitors see at <code class="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-sm">/</code>
			</p>
		</div>
		<div class="text-sm text-gray-500">
			{#if isSaving}
				Saving...
			{:else}
				All changes autosaved
			{/if}
		</div>
	</div>

	<!-- Homepage Visibility -->
	<div class="card p-6 mb-6">
		<div class="flex items-start justify-between gap-4 mb-4">
			<div class="flex-1">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
					Homepage Visibility
				</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					{#if homepageEnabled}
						Your homepage is <span class="font-medium text-green-600 dark:text-green-400">visible</span>.
					{:else}
						Your homepage is <span class="font-medium text-amber-600 dark:text-amber-400">hidden</span>.
					{/if}
				</p>
			</div>
			<label class="relative inline-flex items-center cursor-pointer">
				<input
					type="checkbox"
					class="sr-only peer"
					bind:checked={homepageEnabled}
					disabled={settingsLoading}
				/>
				<div class="w-14 h-7 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
			</label>
		</div>

		{#if !homepageEnabled}
			<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
				<label class="label" for="landing-message">Landing Message</label>
				<textarea
					id="landing-message"
					class="input min-h-[80px] w-full"
					bind:value={landingPageMessage}
					placeholder="This profile is being set up."
					maxlength="2000"
				></textarea>
			</div>
		{/if}

		<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
			<label class="flex items-center gap-3 cursor-pointer">
				<input type="checkbox" bind:checked={hideLoginButton} class="checkbox" />
				<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Hide Login Button from public view</span>
			</label>
		</div>
	</div>

	<!-- Site Navigation -->
	<div class="card p-6 mb-6">
		<div class="flex items-start justify-between gap-4 mb-4">
			<div class="flex-1">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
					Navigation
				</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Show a navigation bar to help visitors browse your public facets.
				</p>
			</div>
			<label class="relative inline-flex items-center cursor-pointer">
				<input
					type="checkbox"
					class="sr-only peer"
					bind:checked={navEnabled}
					disabled={settingsLoading}
				/>
				<div class="w-14 h-7 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
			</label>
		</div>

		{#if navEnabled}
			<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 space-y-4">
				
				<!-- Help Panel -->
				<div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-100 dark:border-blue-800 rounded-lg p-4">
					<h3 class="text-sm font-semibold text-blue-900 dark:text-blue-100 mb-2">How do you want to use Facet?</h3>
					<div class="grid gap-4 md:grid-cols-3 text-sm">
						<div>
							<strong class="block text-blue-800 dark:text-blue-200 mb-1">🔗 Connected Site</strong>
							<p class="text-blue-700 dark:text-blue-300">Turn navigation ON. Your homepage becomes a hub linking to your public facets.</p>
						</div>
						<div>
							<strong class="block text-blue-800 dark:text-blue-200 mb-1">📄 Standalone Pages</strong>
							<p class="text-blue-700 dark:text-blue-300">Turn navigation OFF. Share specific facets via links or tokens.</p>
						</div>
						<div>
							<strong class="block text-blue-800 dark:text-blue-200 mb-1">🔀 Hybrid</strong>
							<p class="text-blue-700 dark:text-blue-300">Enable nav for public facets, keep others private/unlisted.</p>
						</div>
					</div>
				</div>

				<div class="grid gap-4">
					<div>
						<label for="nav-home-label" class="label">Home Button Label</label>
						<input
							type="text"
							id="nav-home-label"
							bind:value={navHomeLabel}
							class="input max-w-xs"
							maxlength="50"
						/>
					</div>

					<div>
						<div class="label mb-2">Navigation Items</div>
						{#if navItems.length > 0}
							<div 
								use:dndzone={{items: navItems, flipDurationMs: 200, dropTargetStyle: {}}} 
								onconsider={handleNavDndConsider} 
								onfinalize={handleNavDndFinalize}
								class="space-y-2"
							>
								{#each navItems as item (item.id)}
									<div class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 group">
										<div class="text-gray-400 cursor-move" aria-label="Drag to reorder">
											<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
										</div>
										<div class="flex-1 min-w-0">
											<div class="text-sm font-medium text-gray-900 dark:text-white truncate">
												{getViewName(item.view_id)}
											</div>
										</div>
										<input
											type="text"
											bind:value={item.label}
											placeholder={getViewName(item.view_id)}
											class="input input-sm w-32"
											maxlength="50"
											title="Custom label"
										/>
										<label class="relative inline-flex items-center cursor-pointer">
											<input
												type="checkbox"
												class="sr-only peer"
												bind:checked={item.visible}
											/>
											<div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
										</label>
									</div>
								{/each}
							</div>
						{:else}
							<p class="text-sm text-gray-500 italic">No public facets available.</p>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- Homepage Sections -->
	<div class="card p-6 mb-6">
		<div class="mb-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
				Homepage Sections
			</h2>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Choose which sections appear on your homepage and arrange their order.
			</p>
		</div>

		<ViewSectionManager
			bind:sections
			bind:sectionOrder
			bind:sectionItems
			viewId="homepage"
			onOpenOverrideEditor={() => {}}
		/>
	</div>

	<!-- Profile Edit Section (Manual Save) -->
	<div class="card p-6">
		<div class="flex items-center justify-between mb-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Profile Information</h2>
			<button 
				type="button" 
				class="btn btn-primary"
				onclick={saveProfile}
				disabled={profileSaving}
			>
				{profileSaving ? 'Saving...' : 'Save Profile'}
			</button>
		</div>

		<div class="space-y-6">
			<!-- Images -->
			<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
				<div>
					<span class="label">Avatar</span>
					<div class="flex items-start gap-4">
						<div class="relative">
							{#if avatarUrl}
								<img src={avatarUrl} alt="Avatar" class="w-24 h-24 rounded-full object-cover border-2 border-gray-200 dark:border-gray-700" />
								<button type="button" onclick={removeAvatar} class="absolute -top-2 -right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600">{@html icon('x')}</button>
							{:else}
								<div class="w-24 h-24 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400">{@html icon('image')}</div>
							{/if}
						</div>
						<div class="flex-1">
							<input type="file" id="avatar" accept="image/*" onchange={handleAvatarChange} class="hidden" />
							<label for="avatar" class="btn btn-secondary btn-sm cursor-pointer">{avatarUrl ? 'Change' : 'Upload'} Avatar</label>
						</div>
					</div>
				</div>
				<div>
					<span class="label">Hero Image</span>
					<div class="space-y-3">
						{#if heroImageUrl}
							<div class="relative">
								<img src={heroImageUrl} alt="Hero" class="w-full h-32 object-cover rounded-lg border border-gray-200 dark:border-gray-700" />
								<button type="button" onclick={removeHeroImage} class="absolute top-2 right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600">{@html icon('x')}</button>
							</div>
						{:else}
							<div class="w-full h-32 bg-gray-100 dark:bg-gray-800 flex items-center justify-center rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400">{@html icon('image')}</div>
						{/if}
						<div>
							<input type="file" id="hero_image" accept="image/*" onchange={handleHeroImageChange} class="hidden" />
							<label for="hero_image" class="btn btn-secondary btn-sm cursor-pointer">{heroImageUrl ? 'Change' : 'Upload'} Hero Image</label>
						</div>
					</div>
				</div>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				<div>
					<label class="label" for="name">Display Name</label>
					<input id="name" type="text" class="input" bind:value={name} />
				</div>
				<div>
					<label class="label" for="location">Location</label>
					<input id="location" type="text" class="input" bind:value={location} />
				</div>
			</div>

			<div>
				<label class="label" for="headline">Headline</label>
				<input id="headline" type="text" class="input" bind:value={headline} />
				<AIContentHelper 
					content={headline} 
					onapply={(c) => headline = c}
					fieldType="headline" 
					context={{ name, location }} 
				/>
			</div>

			<div>
				<label class="label" for="summary">Summary</label>
				<textarea id="summary" class="input min-h-[100px]" bind:value={summary}></textarea>
				<AIContentHelper 
					content={summary} 
					onapply={(c) => summary = c}
					fieldType="summary" 
					context={{ name, headline, location }} 
				/>
			</div>

			<!-- CTA Section -->
			<div class="pt-4 border-t border-gray-200 dark:border-gray-700">
				<h3 class="text-md font-medium text-gray-900 dark:text-white mb-4">Call to Action</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="label" for="cta_button_text">Button Text</label>
						<input id="cta_button_text" type="text" class="input" bind:value={ctaButtonText} placeholder="Contact Me" />
					</div>
					<div>
						<label class="label" for="cta_url">Button URL</label>
						<input id="cta_url" type="text" class="input" bind:value={ctaUrl} placeholder="mailto:..." />
					</div>
					<div class="md:col-span-2">
						<label class="label" for="cta_text">Description Text (Hidden if Navigation Enabled)</label>
						<input id="cta_text" type="text" class="input" bind:value={ctaText} placeholder="Hiring? Let's talk." />
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

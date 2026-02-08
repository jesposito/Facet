<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { browser } from '$app/environment';
	import { onMount, onDestroy } from 'svelte';
	import { flip } from 'svelte/animate';
	import { pb, type View, type CustomContent, VALID_LAYOUTS } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import HomepageSectionManager from '$components/admin/HomepageSectionManager.svelte';

	// Import DnD safely - only in browser (for site navigation reordering)
	let navDndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
	let NAV_DND_TRIGGERS: any = $state({});
	let navDndLoaded = $state(false);
	const navFlipDurationMs = 200;

	// Section definitions for loading items - same as ViewSectionManager
	const SECTION_DEFS: Record<string, { label: string; collection: string }> = {
		experience: { label: 'Experience', collection: 'experience' },
		projects: { label: 'Projects', collection: 'projects' },
		education: { label: 'Education', collection: 'education' },
		certifications: { label: 'Certifications', collection: 'certifications' },
		awards: { label: 'Awards', collection: 'awards' },
		skills: { label: 'Skills', collection: 'skills' },
		posts: { label: 'Posts', collection: 'posts' },
		talks: { label: 'Talks', collection: 'talks' },
		testimonials: { label: 'Testimonials', collection: 'testimonials' },
		contacts: { label: 'Contact Methods', collection: 'contact_methods' }
	};

	// Default section order
	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'awards', 'skills', 'posts', 'talks', 'testimonials', 'contacts'];

	// Homepage visibility settings
	let settingsLoading = $state(true);
	let settingsSaving = $state(false);
	let homepageEnabled = $state(true);
	let landingPageMessage = $state('This profile is being set up.');
	let hideLoginButton = $state(false);

	// Site navigation settings
	let siteNavEnabled = $state(false);
	let siteCtaEnabled = $state(true); // Global CTA toggle (defaults to true for existing behavior)
	let siteNavItems: Array<{ viewId: string; enabled: boolean; label: string }> = $state([]);
	let publicViews: View[] = $state([]);
	let publicViewsLoading = $state(true);

	// Section ordering - array of {id, key} objects for dnd compatibility
	let sectionOrder: Array<{ id: string; key: string }> = $state([]);

	// Sections configuration - per-section settings
	let sections: Record<string, {
		enabled: boolean;
		items: string[];
		expanded: boolean;
		layout: string;
		width: string;
		// Skills-specific settings
		categoryOrder?: string[];
		disabledCategories?: string[];
		categoryDisplayModes?: Record<string, string>;
	}> = $state({});

	// Available items for each section
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

	// Custom content for homepage
	let customContentItems: CustomContent[] = $state([]);
	let customContentLoading = $state(true);

	// Profile data
	let profile: Record<string, unknown> | null = null;
	let profileLoading = $state(true);
	let saving = $state(false);

	// Form fields
	let name = $state('');
	let headline = $state('');
	let location = $state('');
	let summary = $state('');
	let contactEmail = $state('');
	let contactLinks: Array<{ type: string; url: string; label: string }> = $state([]);
	let visibility = $state('public');
	let ctaText = $state('');
	let ctaUrl = $state('');
	let ctaButtonText = $state('');

	// Image fields
	let avatarUrl: string | null = $state(null);
	let heroImageUrl: string | null = $state(null);
	let avatarFile: File | null = null;
	let heroImageFile: File | null = null;

	// Views that override headline/summary
	let viewsOverridingHeadline: View[] = $state([]);
	let viewsOverridingSummary: View[] = $state([]);

	onMount(async () => {
		// Load DnD library for site navigation reordering
		if (browser) {
			const { dndzone: dnd, TRIGGERS: trig } = await import('svelte-dnd-action');
			navDndzone = dnd;
			NAV_DND_TRIGGERS = trig;
			navDndLoaded = true;
		}
		await Promise.all([loadSettings(), loadProfile(), loadCustomContent(), loadSectionItems(), loadPublicViews()]);
	});

	async function loadPublicViews() {
		try {
			const records = await pb.collection('views').getList(1, 100, {
				filter: 'is_active = true && visibility = "public"',
				sort: 'name',
				$cancelKey: 'homepage-public-views'
			});
			publicViews = [...records.items] as unknown as View[];
		} catch (err) {
			console.error('[SITE NAV] Failed to load public views:', err);
		} finally {
			publicViewsLoading = false;
		}
	}

	async function loadSettings() {
		try {
			const response = await fetch('/api/site-settings');
			if (response.ok) {
				const data = await response.json();
				homepageEnabled = data.homepage_enabled !== false;
				landingPageMessage = data.landing_page_message || '';
				hideLoginButton = data.hide_login_button === true;

				// Load site navigation settings
				siteNavEnabled = data.site_nav_enabled === true;
				siteCtaEnabled = data.site_cta_enabled !== false; // Default to true
				siteNavItems = data.site_nav_items || [];

				// Initialize sections from homepage_sections or default
				const savedSections = data.homepage_sections || {};
				initializeSections(savedSections);

				// Parse section order into dnd format
				const rawOrder: string[] = data.homepage_section_order || [];
				if (rawOrder.length > 0) {
					// Use saved order but include any new sections that aren't in the order
					const orderedKeys = new Set(rawOrder);
					const allKeys = [...DEFAULT_SECTION_ORDER];

					// Add custom content sections from saved config
					for (const key of Object.keys(savedSections)) {
						if (key.startsWith('custom:') && !allKeys.includes(key)) {
							allKeys.push(key);
						}
					}

					// Start with saved order
					const finalOrder: string[] = [...rawOrder];

					// Add any missing standard sections at the end
					for (const key of allKeys) {
						if (!orderedKeys.has(key)) {
							finalOrder.push(key);
						}
					}

					sectionOrder = finalOrder.map((key, i) => ({ id: `section-${i}`, key }));
				} else {
					// Default order: standard sections
					sectionOrder = DEFAULT_SECTION_ORDER.map((key, i) => ({ id: `section-${i}`, key }));
				}
			}
		} catch (err) {
			console.error('Failed to load settings:', err);
		} finally {
			settingsLoading = false;
		}
	}

	function initializeSections(savedSections: Record<string, any>) {
		// Start with all standard sections with default values
		for (const key of DEFAULT_SECTION_ORDER) {
			const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
			sections[key] = {
				enabled: true, // Default enabled for homepage (shows all public items)
				items: [],
				expanded: false,
				layout: defaultLayout,
				width: 'full'
			};
		}

		// Apply saved configuration
		for (const [key, config] of Object.entries(savedSections)) {
			if (config && typeof config === 'object') {
				const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
				sections[key] = {
					enabled: (config as any).enabled !== false,
					items: (config as any).items || [],
					expanded: false,
					layout: (config as any).layout || defaultLayout,
					width: (config as any).width || 'full',
					// Skills-specific settings
					categoryOrder: (config as any).categoryOrder || undefined,
					disabledCategories: (config as any).disabledCategories || undefined,
					categoryDisplayModes: (config as any).categoryDisplayModes || undefined
				};
			}
		}
	}

	async function loadCustomContent() {
		try {
			const records = await collection('custom_content').getList(1, 100, {
				sort: 'sort_order',
				filter: 'visibility = "public" && is_draft = false'
			});
			customContentItems = records.items as unknown as CustomContent[];

			// Add custom content items to sectionOrder if they exist
			const existingKeys = new Set(sectionOrder.map(s => s.key));
			let nextId = sectionOrder.length;
			for (const item of customContentItems) {
				const key = `custom:${item.id}`;
				if (!existingKeys.has(key)) {
					sectionOrder = [...sectionOrder, { id: `section-${nextId++}`, key }];
					// Initialize section config for new custom content
					if (!sections[key]) {
						sections[key] = {
							enabled: true,
							items: [],
							expanded: false,
							layout: 'default',
							width: 'full'
						};
					}
				}
			}
		} catch (err) {
			console.error('Failed to load custom content:', err);
		} finally {
			customContentLoading = false;
		}
	}

	async function loadSectionItems() {
		for (const key of DEFAULT_SECTION_ORDER) {
			const def = SECTION_DEFS[key];
			try {
				// Testimonials only show approved ones
				const filter = key === 'testimonials' ? 'status = "approved"' : '';
				const records = await collection(def.collection).getList(1, 500, {
					sort: key === 'testimonials' ? '-featured,-sort_order' : '-id',
					filter,
					expand: 'admin_tags'
				});

				sectionItems[key] = records.items.map((item) => ({
					id: item.id,
					label: getItemLabel(key, item as Record<string, unknown>),
					// Default to 'private' if visibility is not set - safer than assuming public
					visibility: (item as Record<string, unknown>).visibility as string || 'private',
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
			case 'experience':
				return `${item.title} at ${item.company}`;
			case 'projects':
				return item.title as string;
			case 'education':
				return `${item.degree || 'Degree'} - ${item.institution}`;
			case 'certifications':
				return `${item.name} (${item.issuer || 'Unknown issuer'})`;
			case 'awards':
				return `${item.title}${item.issuer ? ` (${item.issuer})` : ''}`;
			case 'skills':
				return `${item.name}${item.category ? ` (${item.category})` : ''}`;
			case 'posts':
				return item.title as string;
			case 'talks':
				return `${item.title}${item.event ? ` @ ${item.event}` : ''}`;
			case 'contacts':
				return `${item.label || item.type} - ${item.value}`;
			case 'testimonials':
				return `${item.author_name}${item.author_company ? ` - ${item.author_company}` : ''}`;
			default:
				return item.title as string || item.name as string || item.id as string;
		}
	}

	async function saveSettings() {
		settingsSaving = true;
		try {
			const response = await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					homepage_enabled: homepageEnabled,
					landing_page_message: landingPageMessage,
					hide_login_button: hideLoginButton,
					site_nav_enabled: siteNavEnabled,
					site_cta_enabled: siteCtaEnabled,
					site_nav_items: siteNavItems
				})
			});

			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || 'Failed to save settings');
				return;
			}

			homepageEnabled = result.homepage_enabled !== false;
			landingPageMessage = result.landing_page_message || '';
			hideLoginButton = result.hide_login_button === true;
			siteNavEnabled = result.site_nav_enabled === true;
			siteCtaEnabled = result.site_cta_enabled !== false;
			siteNavItems = result.site_nav_items || [];
			toasts.add('success', 'Homepage settings saved');
		} catch (err) {
			console.error('Failed to save settings:', err);
			toasts.add('error', 'Failed to save homepage settings');
		} finally {
			settingsSaving = false;
		}
	}

	async function loadProfile() {
		try {
			const records = await collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				profile = records.items[0];
				name = (profile.name as string) || '';
				headline = (profile.headline as string) || '';
				location = (profile.location as string) || '';
				summary = (profile.summary as string) || '';
				contactEmail = (profile.contact_email as string) || '';
				contactLinks = (profile.contact_links as typeof contactLinks) || [];
				visibility = (profile.visibility as string) || 'public';
				ctaText = (profile.cta_text as string) || '';
				ctaUrl = (profile.cta_url as string) || '';
				ctaButtonText = (profile.cta_button_text as string) || '';

				if (profile.avatar) {
					avatarUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.avatar}`;
				}
				if (profile.hero_image) {
					heroImageUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.hero_image}`;
				}
			}

			// Check for views with overrides
			const views = await collection('views').getList(1, 100, {
				$cancelKey: 'homepage-views-overrides'
			});
			viewsOverridingHeadline = (views.items as unknown as View[]).filter(v => v.hero_headline);
			viewsOverridingSummary = (views.items as unknown as View[]).filter(v => v.hero_summary);
		} catch (err) {
			console.error('Failed to load profile:', err);
		} finally {
			profileLoading = false;
		}
	}

	async function handleSubmit() {
		saving = true;
		try {
			// Save profile data
			const formData = new FormData();
			formData.append('name', name);
			formData.append('headline', headline);
			formData.append('location', location);
			formData.append('summary', summary);
			formData.append('contact_email', contactEmail);
			formData.append('contact_links', JSON.stringify(contactLinks));
			formData.append('visibility', visibility);
			formData.append('cta_text', ctaText);
			formData.append('cta_url', ctaUrl);
			formData.append('cta_button_text', ctaButtonText);

			if (avatarFile) {
				formData.append('avatar', avatarFile);
			}
			if (heroImageFile) {
				formData.append('hero_image', heroImageFile);
			}

			if (profile) {
				await collection('profile').update(profile.id as string, formData);
			} else {
				try {
					await collection('profile').create(formData);
				} catch (createErr: unknown) {
					// Profile may already exist (e.g., created by setup wizard after loadProfile ran)
					const records = await collection('profile').getList(1, 1);
					if (records.items.length > 0) {
						profile = records.items[0];
						await collection('profile').update(profile.id as string, formData);
					} else {
						throw createErr;
					}
				}
			}

			// Save section order and section configuration
			const orderKeys = sectionOrder.map(s => s.key);

			// Build homepage_sections for API (without expanded state, only persistent config)
			const homepageSections: Record<string, {
				enabled: boolean;
				items: string[];
				layout: string;
				width: string;
				categoryOrder?: string[];
				disabledCategories?: string[];
				categoryDisplayModes?: Record<string, string>;
			}> = {};
			for (const [key, config] of Object.entries(sections)) {
				homepageSections[key] = {
					enabled: config.enabled,
					items: config.items,
					layout: config.layout,
					width: config.width
				};
				// Include skills-specific settings if present
				if (config.categoryOrder?.length) {
					homepageSections[key].categoryOrder = config.categoryOrder;
				}
				if (config.disabledCategories?.length) {
					homepageSections[key].disabledCategories = config.disabledCategories;
				}
				if (config.categoryDisplayModes && Object.keys(config.categoryDisplayModes).length > 0) {
					homepageSections[key].categoryDisplayModes = config.categoryDisplayModes;
				}
			}

			// Also build homepage_custom_content for backwards compatibility
			const customContentConfig = customContentItems
				.filter(item => {
					const sectionKey = `custom:${item.id}`;
					return sections[sectionKey]?.enabled !== false;
				})
				.map(item => ({ id: item.id, enabled: true }));

			await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					homepage_section_order: orderKeys,
					homepage_sections: homepageSections,
					homepage_custom_content: customContentConfig,
					site_nav_enabled: siteNavEnabled,
					site_cta_enabled: siteCtaEnabled,
					site_nav_items: siteNavItems
				})
			});

			toasts.add('success', 'Homepage saved successfully');

			avatarFile = null;
			heroImageFile = null;

			const records = await collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				profile = records.items[0];
				if (profile.avatar) {
					avatarUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.avatar}?${Date.now()}`;
				}
				if (profile.hero_image) {
					heroImageUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.hero_image}?${Date.now()}`;
				}
			}
		} catch (err) {
			console.error('Failed to save homepage:', err);
			toasts.add('error', 'Failed to save homepage');
		} finally {
			saving = false;
		}
	}

	function addContactLink() {
		contactLinks = [...contactLinks, { type: 'website', url: '', label: '' }];
	}

	function removeContactLink(index: number) {
		contactLinks = contactLinks.filter((_, i) => i !== index);
	}

	let avatarBlobUrl: string | null = null;
	let heroBlobUrl: string | null = null;

	function handleAvatarChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files?.[0]) {
			if (avatarBlobUrl) URL.revokeObjectURL(avatarBlobUrl);
			avatarFile = input.files[0];
			avatarBlobUrl = URL.createObjectURL(avatarFile);
			avatarUrl = avatarBlobUrl;
		}
	}

	function handleHeroImageChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files?.[0]) {
			if (heroBlobUrl) URL.revokeObjectURL(heroBlobUrl);
			heroImageFile = input.files[0];
			heroBlobUrl = URL.createObjectURL(heroImageFile);
			heroImageUrl = heroBlobUrl;
		}
	}

	onDestroy(() => {
		if (avatarBlobUrl) URL.revokeObjectURL(avatarBlobUrl);
		if (heroBlobUrl) URL.revokeObjectURL(heroBlobUrl);
	});

	async function removeAvatar() {
		if (!profile) return;
		const confirmed = await confirm({
			title: 'Remove Avatar',
			message: 'Are you sure you want to remove your avatar image?',
			confirmText: 'Remove',
			danger: true
		});
		if (!confirmed) return;
		try {
			await collection('profile').update(profile.id as string, { avatar: null });
			avatarUrl = null;
			avatarFile = null;
			toasts.add('success', 'Avatar removed');
		} catch (err) {
			console.error('Failed to remove avatar:', err);
			toasts.add('error', 'Failed to remove avatar');
		}
	}

	async function removeHeroImage() {
		if (!profile) return;
		const confirmed = await confirm({
			title: 'Remove Hero Image',
			message: 'Are you sure you want to remove your hero image?',
			confirmText: 'Remove',
			danger: true
		});
		if (!confirmed) return;
		try {
			await collection('profile').update(profile.id as string, { hero_image: null });
			heroImageUrl = null;
			heroImageFile = null;
			toasts.add('success', 'Hero image removed');
		} catch (err) {
			console.error('Failed to remove hero image:', err);
			toasts.add('error', 'Failed to remove hero image');
		}
	}

	// Site navigation helpers - use derived state for reactivity
	let navItemsByViewId = $derived(
		new Map(siteNavItems.map(item => [item.viewId, item]))
	);

	// Ordered nav items for DnD - this is a $state array that DnD can mutate directly
	type OrderedNavItem = { id: string; viewId: string; view: View };
	let orderedNavItems: OrderedNavItem[] = $state([]);
	let navItemsInitialized = false;

	// Build the ordered nav items list from current state
	function buildOrderedNavItems(): OrderedNavItem[] {
		const viewsById = new Map(publicViews.map(v => [v.id, v]));
		const result: OrderedNavItem[] = [];
		const seenViewIds = new Set<string>();

		// First, add items in siteNavItems order
		for (const navItem of siteNavItems) {
			const view = viewsById.get(navItem.viewId);
			if (view) {
				result.push({ id: `nav-${navItem.viewId}`, viewId: navItem.viewId, view });
				seenViewIds.add(navItem.viewId);
			}
		}

		// Then add any publicViews not yet in siteNavItems (new views)
		for (const view of publicViews) {
			if (!seenViewIds.has(view.id)) {
				result.push({ id: `nav-${view.id}`, viewId: view.id, view });
			}
		}

		return result;
	}

	// Initialize orderedNavItems once when publicViews first loads
	$effect(() => {
		if (publicViews.length > 0 && !navItemsInitialized) {
			orderedNavItems = buildOrderedNavItems();
			navItemsInitialized = true;
		}
	});

	function isNavItemEnabled(viewId: string): boolean {
		return navItemsByViewId.get(viewId)?.enabled ?? false;
	}

	function getNavItemLabel(viewId: string): string {
		return navItemsByViewId.get(viewId)?.label || '';
	}

	// DnD handlers for site navigation reordering
	function handleNavDndConsider(e: CustomEvent<{ items: OrderedNavItem[]; info: { trigger: string } }>) {
		// Just update the display array during drag - don't touch siteNavItems yet
		orderedNavItems = e.detail.items;
	}

	function handleNavDndFinalize(e: CustomEvent<{ items: OrderedNavItem[]; info: { trigger: string } }>) {
		// Update the display array
		orderedNavItems = e.detail.items;

		// Now sync back to siteNavItems and save
		syncNavItemsFromDisplay();
		saveSiteNavSettings();
	}

	function syncNavItemsFromDisplay() {
		// Rebuild siteNavItems in the new order from orderedNavItems, preserving enabled/label state
		const currentNavItemsByViewId = new Map(siteNavItems.map(item => [item.viewId, item]));
		const newNavItems: typeof siteNavItems = [];
		for (const item of orderedNavItems) {
			const existing = currentNavItemsByViewId.get(item.viewId);
			if (existing) {
				// Keep existing state
				newNavItems.push({ ...existing });
			} else {
				// New item - not yet in siteNavItems, add with defaults
				newNavItems.push({ viewId: item.viewId, enabled: false, label: '' });
			}
		}
		siteNavItems = newNavItems;
	}

	// Save site nav settings immediately to database
	async function saveSiteNavSettings() {
		try {
			const response = await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					site_nav_enabled: siteNavEnabled,
					site_cta_enabled: siteCtaEnabled,
					site_nav_items: siteNavItems
				})
			});

			if (!response.ok) {
				const result = await response.json();
				toasts.add('error', result.error || 'Failed to save site nav settings');
			}
		} catch (err) {
			console.error('Failed to save site nav settings:', err);
			toasts.add('error', 'Failed to save site nav settings');
		}
	}

	async function toggleSiteNavEnabled() {
		siteNavEnabled = !siteNavEnabled;
		await saveSiteNavSettings();
	}

	async function toggleSiteCtaEnabled() {
		siteCtaEnabled = !siteCtaEnabled;
		await saveSiteNavSettings();
	}

	async function toggleNavItem(viewId: string) {
		const existing = siteNavItems.find(item => item.viewId === viewId);
		if (existing) {
			existing.enabled = !existing.enabled;
			siteNavItems = [...siteNavItems];
		} else {
			siteNavItems = [...siteNavItems, { viewId, enabled: true, label: '' }];
		}
		await saveSiteNavSettings();
	}

	async function updateNavItemLabel(viewId: string, label: string) {
		const existing = siteNavItems.find(item => item.viewId === viewId);
		if (existing) {
			existing.label = label;
			siteNavItems = [...siteNavItems];
		} else {
			siteNavItems = [...siteNavItems, { viewId, enabled: false, label }];
		}
		// Debounce label updates - don't save on every keystroke
	}
</script>

<svelte:head>
	<title>Homepage | Facet</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<PageHelp pageKey="homepage">
		<p><strong>Homepage</strong> controls what visitors see at your root URL.</p>
		<p>Enable or disable public access, customize the landing page message for when your site is hidden, and edit your core profile information that appears across all facets.</p>
		<p><strong>Tip:</strong> Hide your homepage while building your profile, then enable it when you're ready to go live.</p>
	</PageHelp>

	<div class="mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Homepage</h1>
		<p class="text-gray-600 dark:text-gray-400 mt-1">
			Manage your public profile and control what visitors see at <code class="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-sm">/</code>
		</p>
	</div>

	<!-- Homepage Visibility Section -->
	<div class="card p-6 mb-6">
		<div class="flex items-start justify-between gap-4 mb-4">
			<div class="flex-1">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
					Homepage Visibility
				</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					{#if homepageEnabled}
						Your homepage is <span class="font-medium text-green-600 dark:text-green-400">visible</span>.
						Visitors can see your public profile and content.
					{:else}
						Your homepage is <span class="font-medium text-amber-600 dark:text-amber-400">hidden</span>.
						Visitors see a custom message instead.
					{/if}
				</p>
			</div>
			<label class="relative inline-flex items-center cursor-pointer">
				<input
					type="checkbox"
					class="sr-only peer"
					bind:checked={homepageEnabled}
					disabled={settingsSaving || settingsLoading}
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
					disabled={settingsSaving}
					maxlength="2000"
				></textarea>
				<p class="text-xs text-gray-500 mt-1">{landingPageMessage.length}/2000 characters</p>
			</div>
		{/if}

		<!-- Hide Login Button Setting -->
		<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
			<div class="flex items-start justify-between gap-4">
				<div class="flex-1">
					<h3 class="text-sm font-medium text-gray-900 dark:text-white mb-1">
						Hide Login Button
					</h3>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						{#if hideLoginButton}
							The login button is <span class="font-medium text-amber-600 dark:text-amber-400">hidden</span> from public visitors.
							You can still access <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">/admin/login</code> directly.
						{:else}
							The login button is <span class="font-medium text-green-600 dark:text-green-400">visible</span> on your homepage.
						{/if}
					</p>
				</div>
				<label class="relative inline-flex items-center cursor-pointer">
					<input
						type="checkbox"
						class="sr-only peer"
						bind:checked={hideLoginButton}
						disabled={settingsSaving || settingsLoading}
					/>
					<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-amber-500"></div>
				</label>
			</div>
		</div>

		<div class="flex justify-end mt-4">
			<button
				type="button"
				class="btn btn-primary btn-sm"
				onclick={saveSettings}
				disabled={settingsSaving || settingsLoading}
			>
				{settingsSaving ? 'Saving...' : 'Save Visibility'}
			</button>
		</div>
	</div>

	<!-- Profile Section -->
	{#if profileLoading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading profile...</div>
		</div>
	{:else}
		<form onsubmit={preventDefault(handleSubmit)} class="space-y-6">
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Images</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div>
						<span class="label">Avatar</span>
						<div class="flex items-start gap-4">
							<div class="relative">
								{#if avatarUrl}
									<img
										src={avatarUrl}
										alt="Avatar"
										class="w-24 h-24 rounded-full object-cover border-2 border-gray-200 dark:border-gray-700"
									/>
								<button
									type="button"
									onclick={removeAvatar}
									class="absolute -top-2 -right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600"
									title="Remove avatar"
								>
									{@html icon('x')}
								</button>
							{:else}
								<div class="w-24 h-24 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400">
									{@html icon('image')}
								</div>
								{/if}
							</div>
							<div class="flex-1">
								<input
									type="file"
									id="avatar"
									accept="image/jpeg,image/png,image/webp,image/svg+xml"
									onchange={handleAvatarChange}
									class="hidden"
								/>
								<label for="avatar" class="btn btn-secondary btn-sm cursor-pointer">
									{avatarUrl ? 'Change' : 'Upload'} Avatar
								</label>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									JPG, PNG, WebP or SVG. Max 5MB.
								</p>
							</div>
						</div>
					</div>

					<div>
						<span class="label">Hero Image</span>
						<div class="space-y-3">
							{#if heroImageUrl}
								<div class="relative">
									<img
										src={heroImageUrl}
										alt="Hero"
										class="w-full h-32 object-cover rounded-lg border border-gray-200 dark:border-gray-700"
									/>
								<button
									type="button"
									onclick={removeHeroImage}
									class="absolute top-2 right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600"
									title="Remove hero image"
								>
									{@html icon('x')}
								</button>
							</div>
						{:else}
							<div class="w-full h-32 bg-gray-100 dark:bg-gray-800 flex items-center justify-center rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400">
								{@html icon('image')}
							</div>
							{/if}
							<div>
								<input
									type="file"
									id="hero_image"
									accept="image/jpeg,image/png,image/webp,image/gif"
									onchange={handleHeroImageChange}
									class="hidden"
								/>
								<label for="hero_image" class="btn btn-secondary btn-sm cursor-pointer">
									{heroImageUrl ? 'Change' : 'Upload'} Hero Image
								</label>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									JPG, PNG, WebP or GIF. Max 10MB.
								</p>
							</div>
						</div>
					</div>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h2>

				<div>
					<label for="name" class="label">Name *</label>
					<input type="text" id="name" bind:value={name} class="input" required />
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="headline" class="label mb-0">Headline</label>
						<AIContentHelper
							fieldType="headline"
							content={headline}
							context={{ name, location }}
							on:apply={(e) => (headline = e.detail.content)}
						/>
					</div>
					<input
						type="text"
						id="headline"
						bind:value={headline}
						class="input mt-1"
						placeholder="e.g., Senior Software Engineer at Company"
					/>
					{#if viewsOverridingHeadline.length > 0}
						<div class="mt-2 p-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md">
							<p class="text-sm text-amber-800 dark:text-amber-200">
								<strong>Note:</strong> {viewsOverridingHeadline.length === 1 ? 'This view has' : 'These views have'} a custom headline that overrides this value:
								{#each viewsOverridingHeadline as view, i}
									<a href="/admin/views/{view.id}" class="underline hover:no-underline">{view.name}</a>{i < viewsOverridingHeadline.length - 1 ? ', ' : ''}
								{/each}
							</p>
						</div>
					{/if}
				</div>

				<div>
					<label for="location" class="label">Location</label>
					<input
						type="text"
						id="location"
						bind:value={location}
						class="input"
						placeholder="e.g., San Francisco, CA"
					/>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="summary" class="label mb-0">Summary</label>
						<AIContentHelper
							fieldType="summary"
							content={summary}
							context={{ name, headline, location }}
							on:apply={(e) => (summary = e.detail.content)}
						/>
					</div>
					<textarea
						id="summary"
						bind:value={summary}
						class="input min-h-[150px] mt-1"
						placeholder="Tell your story... (Markdown supported)"
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown formatting is supported</p>
					{#if viewsOverridingSummary.length > 0}
						<div class="mt-2 p-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md">
							<p class="text-sm text-amber-800 dark:text-amber-200">
								<strong>Note:</strong> {viewsOverridingSummary.length === 1 ? 'This view has' : 'These views have'} a custom summary that overrides this value:
								{#each viewsOverridingSummary as view, i}
									<a href="/admin/views/{view.id}" class="underline hover:no-underline">{view.name}</a>{i < viewsOverridingSummary.length - 1 ? ', ' : ''}
								{/each}
							</p>
						</div>
					{/if}
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Contact Information</h2>

				<div>
					<label for="email" class="label">Contact Email</label>
					<input type="email" id="email" bind:value={contactEmail} class="input" />
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<span class="label mb-0">Contact Links</span>
						<button type="button" class="btn btn-sm btn-secondary" onclick={addContactLink}>
							+ Add Link
						</button>
					</div>

					{#if contactLinks.length === 0}
						<p class="text-gray-500 dark:text-gray-400 text-sm">Add links to help people reach you.</p>
					{:else}
						<div class="space-y-3">
							{#each contactLinks as link, i}
								<div class="flex flex-col sm:flex-row items-stretch sm:items-start gap-2">
									<select bind:value={link.type} class="input w-full sm:w-32">
										<option value="github">GitHub</option>
										<option value="linkedin">LinkedIn</option>
										<option value="twitter">Twitter</option>
										<option value="email">Email</option>
										<option value="website">Website</option>
										<option value="other">Other</option>
									</select>
									<input
										type="url"
										bind:value={link.url}
										class="input w-full sm:flex-1"
										placeholder="https://..."
									/>
									<div class="flex gap-2">
										<input
											type="text"
											bind:value={link.label}
											class="input flex-1 sm:w-32"
											placeholder="Label"
										/>
										<button
											type="button"
											class="btn btn-ghost text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
											onclick={() => removeContactLink(i)}
											title="Remove link"
										>
											{@html icon('x')}
										</button>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<div class="flex items-start justify-between gap-4">
				<div class="flex-1">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">Call to Action</h2>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						{#if siteCtaEnabled}
							Add a prominent banner to your homepage and public facets.
						{:else}
							The call-to-action is <span class="font-medium text-amber-600 dark:text-amber-400">hidden</span> site-wide.
						{/if}
					</p>
				</div>
				<label class="relative inline-flex items-center cursor-pointer">
					<input
						type="checkbox"
						class="sr-only peer"
						checked={siteCtaEnabled}
						onchange={toggleSiteCtaEnabled}
						disabled={settingsLoading}
					/>
					<div class="w-14 h-7 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
				</label>
			</div>

			{#if siteCtaEnabled}
				<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 space-y-4">
					<div>
						<label for="cta_text" class="label">Description</label>
					<input
						type="text"
						id="cta_text"
						bind:value={ctaText}
						placeholder="Ready to work together?"
						class="input"
					/>
					<p class="text-xs text-gray-500 mt-1">Text shown next to the button</p>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="cta_button_text" class="label">Button Label</label>
						<input
							type="text"
							id="cta_button_text"
							bind:value={ctaButtonText}
							placeholder="Get in touch"
							class="input"
						/>
					</div>

					<div>
						<label for="cta_url" class="label">Button URL</label>
						<input
							type="url"
							id="cta_url"
							bind:value={ctaUrl}
							placeholder="https://calendly.com/..."
							class="input"
						/>
					</div>
				</div>
			</div>
			{/if}
		</div>

		<!-- Site Navigation -->
			<div class="card p-6 space-y-4">
				<div class="flex items-start justify-between gap-4">
					<div class="flex-1">
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">Site Navigation</h2>
						<p class="text-sm text-gray-600 dark:text-gray-400">
							Turn your Facet into a multi-page website. When enabled, navigation buttons appear on your homepage and all public facets.
						</p>
					</div>
					<label class="relative inline-flex items-center cursor-pointer">
						<input
							type="checkbox"
							class="sr-only peer"
							checked={siteNavEnabled}
							onchange={toggleSiteNavEnabled}
							disabled={settingsLoading}
						/>
						<div class="w-14 h-7 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
					</label>
				</div>

				{#if siteNavEnabled}
					<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
						{#if publicViewsLoading}
							<div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
								<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
								Loading facets...
							</div>
						{:else if publicViews.length === 0}
							<p class="text-sm text-gray-500 dark:text-gray-400">
								No public facets available. Create a public facet to add navigation buttons.
							</p>
						{:else}
							<p class="text-sm text-gray-600 dark:text-gray-400 mb-3">
								Select which facets to show in navigation. Drag to reorder. You can customize the button labels.
							</p>
							{#key navDndLoaded}
							<div
								class="space-y-3"
								use:navDndzone={{ items: orderedNavItems, flipDurationMs: navFlipDurationMs, type: 'site-nav-items' }}
								onconsider={handleNavDndConsider}
								onfinalize={handleNavDndFinalize}
							>
								{#each orderedNavItems as item (item.id)}
									<div
										class="flex flex-col sm:flex-row sm:items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg"
										animate:flip={{ duration: navFlipDurationMs }}
									>
										<div class="flex items-center gap-3 flex-1 min-w-0">
											<!-- Drag Handle -->
											<div class="cursor-grab active:cursor-grabbing p-1.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 shrink-0" title="Drag to reorder">
												<svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
													<path stroke-linecap="round" stroke-linejoin="round" d="M4 8h16M4 16h16" />
												</svg>
											</div>
											<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
											<label
												class="relative inline-flex items-center cursor-pointer shrink-0"
												onpointerdown={(e) => e.stopPropagation()}
												onmousedown={(e) => e.stopPropagation()}
												ontouchstart={(e) => e.stopPropagation()}
											>
												<input
													type="checkbox"
													class="sr-only peer"
													checked={isNavItemEnabled(item.viewId)}
													onchange={() => toggleNavItem(item.viewId)}
												/>
												<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
											</label>
											<div class="flex-1 min-w-0">
												<span class="font-medium text-gray-900 dark:text-white truncate block">{item.view.name}</span>
												<span class="text-xs text-gray-500 dark:text-gray-400">/{item.view.slug}</span>
											</div>
										</div>
										<!-- svelte-ignore a11y_no_static_element_interactions -->
										<div
											class="sm:w-48"
											onpointerdown={(e) => e.stopPropagation()}
											onmousedown={(e) => e.stopPropagation()}
											ontouchstart={(e) => e.stopPropagation()}
										>
											<input
												type="text"
												class="input input-sm w-full"
												placeholder={item.view.name}
												value={getNavItemLabel(item.viewId)}
												oninput={(e) => updateNavItemLabel(item.viewId, e.currentTarget.value)}
											/>
										</div>
									</div>
								{/each}
							</div>
							{/key}
							<p class="text-xs text-gray-500 mt-3">
								Navigation appears on all public pages. The CTA button will move to the right side of the navigation bar.
							</p>
						{/if}
					</div>
				{/if}
			</div>

		</form>
	{/if}

	<!-- Section Order -->
	<div class="card p-6 mt-6">
		{#if settingsLoading || customContentLoading}
			<div class="animate-pulse text-center py-4">Loading sections...</div>
		{:else}
			<HomepageSectionManager
				bind:sections={sections}
				bind:sectionOrder={sectionOrder}
				bind:sectionItems={sectionItems}
				{customContentItems}
				loading={settingsLoading || customContentLoading}
			/>
		{/if}
	</div>

	<!-- Single Save Button -->
	<div class="flex justify-end gap-3 mt-6">
		<a href="/" target="_blank" class="btn btn-secondary">
			View Homepage
		</a>
		<button type="button" class="btn btn-primary" disabled={saving} onclick={handleSubmit}>
			{#if saving}
				<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			{/if}
			Save Homepage
		</button>
	</div>
</div>

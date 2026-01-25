<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount, onDestroy } from 'svelte';
	import { pb, type View } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';

	// Homepage visibility settings
	let settingsLoading = $state(true);
	let settingsSaving = $state(false);
	let homepageEnabled = $state(true);
	let landingPageMessage = $state('This profile is being set up.');
	let hideLoginButton = $state(false);

	// Site navigation settings
	interface NavItem {
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

	// Homepage sections settings
	interface SectionConfig {
		section: string;
		enabled: boolean;
		layout?: string;
		width?: string;
	}
	const DEFAULT_SECTIONS = [
		'experience', 'projects', 'education', 'certifications', 
		'awards', 'skills', 'posts', 'talks', 'testimonials', 'contacts'
	];
	const SECTION_LABELS: Record<string, string> = {
		experience: 'Experience',
		projects: 'Projects',
		education: 'Education',
		certifications: 'Certifications',
		awards: 'Awards',
		skills: 'Skills',
		posts: 'Posts',
		talks: 'Talks',
		testimonials: 'Testimonials',
		contacts: 'Contact Methods'
	};
	let homepageSections = $state<SectionConfig[]>([]);

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
		await Promise.all([loadSettings(), loadProfile()]);
	});

	function handleNavToggle(event: Event) {
		const checked = (event.target as HTMLInputElement).checked;
		navEnabled = checked;
		console.log('[NAV] Toggle changed to:', checked);
		if (checked && publicViews.length > 0 && navItems.length === 0) {
			syncNavItemsWithViews();
		}
	}

	async function loadSettings() {
		try {
			const [settingsResponse, viewsResponse] = await Promise.all([
				fetch('/api/site-settings'),
				pb.collection('views').getFullList({ filter: "visibility = 'public' && is_active = true", sort: 'name' })
			]);

			if (settingsResponse.ok) {
				const data = await settingsResponse.json();
				console.log('[LOAD] Settings from server:', JSON.stringify(data, null, 2));

				homepageEnabled = data.homepage_enabled !== false;
				landingPageMessage = data.landing_page_message || '';
				hideLoginButton = data.hide_login_button === true;

				if (data.site_nav) {
					console.log('[LOAD] site_nav.enabled from server:', data.site_nav.enabled, typeof data.site_nav.enabled);
					navEnabled = data.site_nav.enabled === true;
					navHomeLabel = data.site_nav.home_label || 'Home';
					navItems = data.site_nav.items || [];
					console.log('[LOAD] Parsed navEnabled:', navEnabled, 'navItems:', navItems.length);
				} else {
					console.log('[LOAD] No site_nav in response');
				}

				// Load homepage sections
				if (data.homepage_sections && Array.isArray(data.homepage_sections)) {
					homepageSections = data.homepage_sections;
				}
				syncHomepageSections();
			}

			publicViews = viewsResponse.map((v) => ({
				id: v.id,
				name: v.name as string,
				slug: v.slug as string
			}));
			console.log('[LOAD] Public views:', publicViews.length);

			syncNavItemsWithViews();
			console.log('[LOAD] After sync, navItems:', navItems.length);
		} catch (err) {
			console.error('Failed to load settings:', err);
		} finally {
			settingsLoading = false;
		}
	}

	function syncNavItemsWithViews() {
		const existingIds = new Set(navItems.map(item => item.view_id));
		const viewIds = new Set(publicViews.map(v => v.id));

		navItems = navItems.filter(item => viewIds.has(item.view_id));

		for (const view of publicViews) {
			if (!existingIds.has(view.id)) {
				navItems = [...navItems, { view_id: view.id, label: '', visible: true }];
			}
		}
	}

	function syncHomepageSections() {
		const existingSections = new Set(homepageSections.map(s => s.section));
		
		// Add missing sections with defaults
		for (const section of DEFAULT_SECTIONS) {
			if (!existingSections.has(section)) {
				homepageSections = [...homepageSections, { section, enabled: true }];
			}
		}
		
		// Remove sections that are no longer valid
		homepageSections = homepageSections.filter(s => DEFAULT_SECTIONS.includes(s.section));
	}

	function moveSectionItem(index: number, direction: 'up' | 'down') {
		const newIndex = direction === 'up' ? index - 1 : index + 1;
		if (newIndex < 0 || newIndex >= homepageSections.length) return;

		const items = [...homepageSections];
		[items[index], items[newIndex]] = [items[newIndex], items[index]];
		homepageSections = items;
	}

	function getViewName(viewId: string): string {
		const view = publicViews.find(v => v.id === viewId);
		return view?.name || 'Unknown';
	}

	function moveNavItem(index: number, direction: 'up' | 'down') {
		const newIndex = direction === 'up' ? index - 1 : index + 1;
		if (newIndex < 0 || newIndex >= navItems.length) return;

		const items = [...navItems];
		[items[index], items[newIndex]] = [items[newIndex], items[index]];
		navItems = items;
	}

	async function saveSettings() {
		settingsSaving = true;
		try {
			const payload = {
				homepage_enabled: homepageEnabled,
				landing_page_message: landingPageMessage,
				hide_login_button: hideLoginButton,
				site_nav: {
					enabled: navEnabled,
					home_label: navHomeLabel,
					items: navItems
				},
				homepage_sections: homepageSections
			};
			console.log('[SAVE] navEnabled before send:', navEnabled, typeof navEnabled);
			console.log('[SAVE] Sending payload:', JSON.stringify(payload, null, 2));

			const response = await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify(payload)
			});

			const result = await response.json();
			console.log('[SAVE] Response:', JSON.stringify(result, null, 2));

			if (!response.ok) {
				toasts.add('error', result.error || 'Failed to save settings');
				return;
			}

			homepageEnabled = result.homepage_enabled !== false;
			landingPageMessage = result.landing_page_message || '';
			hideLoginButton = result.hide_login_button === true;
			if (result.site_nav) {
				console.log('[SAVE] Response site_nav.enabled:', result.site_nav.enabled, typeof result.site_nav.enabled);
				navEnabled = result.site_nav.enabled === true;
				navHomeLabel = result.site_nav.home_label || 'Home';
				navItems = result.site_nav.items || [];
				console.log('[SAVE] After update, navEnabled:', navEnabled);
			}
			if (result.homepage_sections && Array.isArray(result.homepage_sections)) {
				homepageSections = result.homepage_sections;
			}
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
			const views = await collection('views').getList(1, 100);
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
				await collection('profile').create(formData);
			}

			toasts.add('success', 'Profile saved successfully');

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
			console.error('Failed to save profile:', err);
			toasts.add('error', 'Failed to save profile');
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
</script>

<svelte:head>
	<title>Homepage | Facet</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<PageHelp pageKey="homepage">
		<p><strong>Homepage</strong> controls what visitors see at your root URL.</p>
		<p>Enable or disable public access, customize the landing page message for when your site is hidden, and edit your core profile information that appears across all facets.</p>
		<p><strong>Site Navigation</strong> adds a navigation bar letting visitors browse between your homepage and public facets. Great when you have multiple facets for different audiences (e.g., "Portfolio", "Speaking", "Consulting").</p>
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
				{settingsSaving ? 'Saving...' : 'Save Settings'}
			</button>
		</div>
	</div>

	<div class="card p-6 mb-6">
		<div class="flex items-start justify-between gap-4 mb-4">
			<div class="flex-1">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
					Site Navigation
				</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					{#if navEnabled}
						Navigation bar is <span class="font-medium text-green-600 dark:text-green-400">enabled</span>.
						Visitors can navigate between your homepage and public facets.
					{:else}
						Navigation bar is <span class="font-medium text-gray-500">disabled</span>.
						Enable it to let visitors navigate between your facets.
					{/if}
				</p>
			</div>
			<label class="relative inline-flex items-center cursor-pointer">
				<input
					type="checkbox"
					class="sr-only peer"
					checked={navEnabled}
					onchange={handleNavToggle}
					disabled={settingsSaving || settingsLoading}
				/>
				<div class="w-14 h-7 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
			</label>
		</div>

		{#if navEnabled}
			<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 space-y-4">
				<div>
					<label for="nav-home-label" class="label">Home Button Label</label>
					<input
						type="text"
						id="nav-home-label"
						bind:value={navHomeLabel}
						placeholder="Home"
						class="input max-w-xs"
						disabled={settingsSaving}
						maxlength="50"
					/>
				</div>

				{#if publicViews.length > 0}
					<div>
						<span class="label">Facets in Navigation</span>
						<p class="text-xs text-gray-500 mb-2">Drag to reorder. Only public facets appear here.</p>
						<div class="space-y-2">
							{#each navItems as item, i (item.view_id)}
								<div class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-800 rounded-lg">
									<div class="flex flex-col gap-0.5">
										<button
											type="button"
											class="p-0.5 text-gray-400 hover:text-gray-600 disabled:opacity-30 text-xs leading-none"
											onclick={() => moveNavItem(i, 'up')}
											disabled={i === 0 || settingsSaving}
											title="Move up"
										>
											▲
										</button>
										<button
											type="button"
											class="p-0.5 text-gray-400 hover:text-gray-600 disabled:opacity-30 text-xs leading-none"
											onclick={() => moveNavItem(i, 'down')}
											disabled={i === navItems.length - 1 || settingsSaving}
											title="Move down"
										>
											▼
										</button>
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
										disabled={settingsSaving}
										maxlength="50"
										title="Custom label (leave empty to use facet name)"
									/>
									<label class="relative inline-flex items-center cursor-pointer">
										<input
											type="checkbox"
											class="sr-only peer"
											bind:checked={item.visible}
											disabled={settingsSaving}
										/>
										<div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
									</label>
								</div>
							{/each}
						</div>
					</div>
				{:else}
					<p class="text-sm text-gray-500 dark:text-gray-400 italic">
						No public facets available. Create a public facet in the Views section to add it to navigation.
					</p>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Homepage Sections Editor -->
	<div class="card p-6 mb-6">
		<div class="mb-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
				Homepage Sections
			</h2>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Choose which sections appear on your homepage and arrange their order.
			</p>
		</div>

		<div class="space-y-2">
			{#each homepageSections as section, i (section.section)}
				<div class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-800 rounded-lg">
					<div class="flex flex-col gap-0.5">
						<button
							type="button"
							class="p-0.5 text-gray-400 hover:text-gray-600 disabled:opacity-30 text-xs leading-none"
							onclick={() => moveSectionItem(i, 'up')}
							disabled={i === 0 || settingsSaving}
							title="Move up"
						>
							▲
						</button>
						<button
							type="button"
							class="p-0.5 text-gray-400 hover:text-gray-600 disabled:opacity-30 text-xs leading-none"
							onclick={() => moveSectionItem(i, 'down')}
							disabled={i === homepageSections.length - 1 || settingsSaving}
							title="Move down"
						>
							▼
						</button>
					</div>
					<div class="flex-1 min-w-0">
						<div class="text-sm font-medium text-gray-900 dark:text-white">
							{SECTION_LABELS[section.section] || section.section}
						</div>
					</div>
					<label class="relative inline-flex items-center cursor-pointer">
						<input
							type="checkbox"
							class="sr-only peer"
							bind:checked={section.enabled}
							disabled={settingsSaving}
						/>
						<div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
					</label>
				</div>
			{/each}
		</div>

		{#if homepageSections.length === 0}
			<p class="text-sm text-gray-500 dark:text-gray-400 italic">
				Loading sections...
			</p>
		{/if}
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
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Call to Action</h2>
				<p class="text-sm text-gray-500 dark:text-gray-400">Add a prominent banner to your homepage hero section.</p>

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

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Profile Visibility</h2>

				<div>
					<label for="visibility" class="label">Who can see your profile</label>
					<select id="visibility" bind:value={visibility} class="input">
						<option value="public">Public - Anyone can view</option>
						<option value="unlisted">Unlisted - Only accessible via direct link or views</option>
						<option value="private">Private - Only you can view</option>
					</select>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<a href="/" target="_blank" class="btn btn-secondary">
					View Homepage
				</a>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					Save Profile
				</button>
			</div>
		</form>
	{/if}
</div>

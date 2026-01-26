<script lang="ts">
	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import { adminSidebarOpen, sidebarSectionStates, sidebarFacetsVersion } from '$lib/stores';
	import { collection } from '$lib/stores/demo';
	import { testimonialsStore, refreshTestimonialsPendingCount } from '$lib/stores/testimonials';
	import { browser } from '$app/environment';

	const appVersion = __APP_VERSION__;

	// Version check state
	let newVersionAvailable = $state(false);
	let latestVersion = $state('');

	// Check for new version (cached daily)
	async function checkForNewVersion() {
		if (!browser) return;

		const CACHE_KEY = 'facet_version_check';
		const CACHE_DURATION = 24 * 60 * 60 * 1000; // 24 hours

		try {
			// Check cache first
			const cached = localStorage.getItem(CACHE_KEY);
			if (cached) {
				const { timestamp, latest, hasUpdate } = JSON.parse(cached);
				if (Date.now() - timestamp < CACHE_DURATION) {
					latestVersion = latest;
					newVersionAvailable = hasUpdate;
					return;
				}
			}

			// Fetch latest tag from GitHub (sorted by creation date, most recent first)
			const response = await fetch('https://api.github.com/repos/jesposito/Facet/tags?per_page=1', {
				headers: { 'Accept': 'application/vnd.github.v3+json' }
			});

			if (!response.ok) return;

			const data = await response.json();
			const latest = (data[0]?.name) || '';

			// Compare versions (strip 'v' prefix for comparison)
			const currentClean = appVersion.replace(/^v/, '').split('-')[0];
			const latestClean = latest.replace(/^v/, '');

			const hasUpdate = latestClean && currentClean !== latestClean &&
				compareVersions(latestClean, currentClean) > 0;

			// Cache result
			localStorage.setItem(CACHE_KEY, JSON.stringify({
				timestamp: Date.now(),
				latest,
				hasUpdate
			}));

			latestVersion = latest;
			newVersionAvailable = hasUpdate;
		} catch {
			// Silently fail - version check is non-critical
		}
	}

	// Compare semver versions: returns 1 if a > b, -1 if a < b, 0 if equal
	function compareVersions(a: string, b: string): number {
		const partsA = a.split('.').map(Number);
		const partsB = b.split('.').map(Number);

		for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
			const numA = partsA[i] || 0;
			const numB = partsB[i] || 0;
			if (numA > numB) return 1;
			if (numA < numB) return -1;
		}
		return 0;
	}

	interface Props {
		isMobile?: boolean;
	}
	
	let { isMobile = false }: Props = $props();

	// State for dynamically loaded facets
	let facets: Array<Record<string, unknown>> = $state([]);
	let facetsLoading = $state(true);
	let facetsError = $state(false);
	let facetsTotalCount = $state(0);



	// Debounce timer to prevent rapid successive loadFacets calls
	let loadFacetsTimer: ReturnType<typeof setTimeout> | null = null;

	// Section IDs for collapsible sections (only these are in the accordion)
	const SECTION_IDS = {
		information: 'sidebar-information',
		voice: 'sidebar-voice',
		testimonials: 'sidebar-testimonials',
		settings: 'sidebar-settings'
	};

	// All section IDs as array for accordion behavior
	const ALL_SECTION_IDS = Object.values(SECTION_IDS);

	// Map section titles to section IDs (for collapsible sections)
	const sectionTitleToId: Record<string, string> = {
		'Your Information': SECTION_IDS.information,
		'Your Voice': SECTION_IDS.voice,
		'Testimonials': SECTION_IDS.testimonials,
		'Settings': SECTION_IDS.settings
	};

	// Helper to check if a section is expanded
	function isSectionExpanded(sectionId: string): boolean {
		return sidebarSectionStates.isExpanded($sidebarSectionStates, sectionId, false);
	}

	onMount(() => {
		sidebarSectionStates.initialize(ALL_SECTION_IDS, SECTION_IDS.information);
		scheduleFacetsLoad();
		refreshTestimonialsPendingCount();
		checkForNewVersion();
	});

	// Reload facets when sidebarFacetsVersion changes (e.g., after demo mode toggle)
	let lastFacetsVersion = $sidebarFacetsVersion;
	$effect(() => {
		if ($sidebarFacetsVersion !== lastFacetsVersion) {
			lastFacetsVersion = $sidebarFacetsVersion;
			scheduleFacetsLoad();
		}
	});



	// Refresh facets after navigation (e.g., after creating/editing/deleting a view)
	afterNavigate(({ from }) => {
		const fromPath = from?.url.pathname || '';
		if (fromPath.includes('/admin/views') || facetsError) {
			scheduleFacetsLoad();
		}
	});



	// Debounced load to prevent rapid successive calls
	function scheduleFacetsLoad() {
		if (loadFacetsTimer) {
			clearTimeout(loadFacetsTimer);
		}
		loadFacetsTimer = setTimeout(() => {
			loadFacetsTimer = null;
			loadFacets();
		}, 100);
	}

	async function loadFacets() {
		facetsLoading = true;
		facetsError = false;
		try {
			const recentResult = await collection('views').getList(1, 4, {
				sort: '-id',
				$cancelKey: 'sidebar-facets-load'
			});
			facets = recentResult?.items ?? [];
			facetsTotalCount = recentResult?.totalItems ?? 0;
		} catch (err) {
			if (err instanceof Error && (err.message.includes('autocancelled') || err.name === 'AbortError')) {
				return;
			}
			console.error('[Sidebar] Failed to load facets:', err);
			facetsError = true;
			facets = [];
			facetsTotalCount = 0;
		} finally {
			facetsLoading = false;
		}
	}

const navSections = [
	{
		id: 'dashboard',
		title: 'Dashboard',
		items: [{ href: '/admin', label: 'Dashboard', icon: 'home' }]
	},
	{
		id: 'information',
		title: 'Your Information',
		items: [
			{ href: '/admin/contacts', label: 'Contact Methods', icon: 'mail' },
			{ href: '/admin/experience', label: 'Experience', icon: 'briefcase' },
			{ href: '/admin/projects', label: 'Projects', icon: 'folder' },
			{ href: '/admin/education', label: 'Education', icon: 'academic' },
			{ href: '/admin/certifications', label: 'Certifications', icon: 'badge' },
			{ href: '/admin/awards', label: 'Awards', icon: 'star' },
			{ href: '/admin/skills', label: 'Skills', icon: 'chip' },
			{ href: '/admin/custom', label: 'Custom Content', icon: 'puzzle' },
			{ href: '/admin/import', label: 'Import & AI', icon: 'sparkle' }
		]
	},
	{
		id: 'voice',
		title: 'Your Voice',
		items: [
			{ href: '/admin/posts', label: 'Posts', icon: 'document' },
			{ href: '/admin/talks', label: 'Talks', icon: 'presentation' }
		]
	},
	{
		id: 'settings',
		title: 'Settings',
		items: [
			{ href: '/admin/settings#account', label: 'Account & Security', icon: 'badge' },
			{ href: '/admin/settings#appearance', label: 'Appearance', icon: 'star' },
			{ href: '/admin/settings#general', label: 'General', icon: 'cog' },
			{ href: '/admin/settings#analytics', label: 'Analytics', icon: 'eye' },
			{ href: '/admin/settings#integrations', label: 'Integrations', icon: 'sparkle' },
			{ href: '/admin/settings/tags', label: 'Admin Tags', icon: 'chip' },
			{ href: '/admin/media', label: 'Media Library', icon: 'image' },
			{ href: '/admin/tokens', label: 'Share Tokens', icon: 'link' },
{ href: '/admin/settings/about', label: 'About Facet', icon: 'info' }
		]
	}
];

const primarySections = navSections.filter(s => s.id === 'information' || s.id === 'voice');
const settingsSections = navSections.filter(s => s.id === 'settings');

// Reactive function that updates when $page changes
let isActive = $derived((href: string): boolean => {
	const currentUrl = $page.url;
	const currentPath = currentUrl.pathname;
	const [targetPath, targetHash] = href.split('#');

	// Handle hash links (specifically for Settings sections)
	if (targetHash) {
		// Must match the path part exactly
		if (currentPath !== targetPath) {
			return false;
		}

		// If current URL has a hash, it must match the target hash
		if (currentUrl.hash) {
			return currentUrl.hash === '#' + targetHash;
		}

		// If current URL has NO hash, default to the 'security' tab (first item)
		// This ensures 'Account & Security' appears active when visiting /admin/settings directly
		return targetHash === 'security';
	}

	// Exact match for dashboard
	if (targetPath === '/admin') {
		return currentPath === '/admin' || currentPath === '/admin/';
	}

	// For other paths, check if current path starts with the href
	// and is followed by either nothing, a slash, or end of string
	return currentPath === targetPath || currentPath.startsWith(targetPath + '/');
});
</script>

<aside
	id="admin-sidebar"
	class="fixed top-16 h-[calc(100vh-4rem)] bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 transition-all duration-200 z-30 flex flex-col
		{isMobile 
			? ($adminSidebarOpen ? 'left-0 w-64' : '-left-64 w-64')
			: ($adminSidebarOpen ? 'left-0 w-64' : 'left-0 w-16')
		}"
	aria-label="Admin navigation"
>
	<nav class="p-3 space-y-4 flex-1 min-h-0 overflow-y-auto overscroll-contain" aria-label="Main menu">
		<!-- Dashboard and Profile - always visible -->
		<div class="space-y-1">
			<a
				href="/admin"
				class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin')
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
				title={!$adminSidebarOpen ? 'Dashboard' : undefined}
				aria-current={isActive('/admin') ? 'page' : undefined}
			>
				<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
				</svg>
				<span class={$adminSidebarOpen ? '' : 'sr-only'}>Dashboard</span>
			</a>
			<a
				href="/admin/homepage"
				class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/homepage')
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
				title={!$adminSidebarOpen ? 'Homepage' : undefined}
				aria-current={isActive('/admin/homepage') ? 'page' : undefined}
			>
				<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
				</svg>
				<span class={$adminSidebarOpen ? '' : 'sr-only'}>Homepage</span>
			</a>
		</div>

		<!-- Facets Section - always visible -->
		<div class="space-y-2">
			<span class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 {$adminSidebarOpen ? '' : 'sr-only'}">Facets</span>
			<div class="space-y-1">
					{#if facetsLoading}
						<div class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400 {$adminSidebarOpen ? '' : 'sr-only'}">
							Loading...
						</div>
					{:else if facetsError}
						<!-- Error state with retry -->
						<div class="px-3 py-2 {$adminSidebarOpen ? '' : 'sr-only'}">
							<p class="text-sm text-red-500 dark:text-red-400 mb-2">Unable to load facets.</p>
							<button
								type="button"
								onclick={() => loadFacets()}
								class="inline-flex items-center gap-1 text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
								</svg>
								Retry
							</button>
						</div>
					{:else}
						{#each facets as facet}
							<a
								href="/admin/views/{facet.id}"
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(`/admin/views/${facet.id}`)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? `Facets: ${facet.name}` : undefined}
								aria-current={isActive(`/admin/views/${facet.id}`) ? 'page' : undefined}
							>
								<!-- Diamond icon -->
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3l9 9-9 9-9-9 9-9z" />
								</svg>
								<span class="flex items-center gap-1.5 min-w-0 overflow-hidden {$adminSidebarOpen ? '' : 'sr-only'}">
									<span class="truncate" title={facet.name as string}>{facet.name}</span>
									<!-- Visibility indicator -->
									{#if facet.visibility === 'public'}
										<span class="w-2 h-2 rounded-full bg-green-500 shrink-0" title="Public"></span>
									{:else if facet.visibility === 'unlisted'}
										<span class="w-2 h-2 rounded-full bg-yellow-500 shrink-0" title="Unlisted"></span>
									{:else if facet.visibility === 'private' || facet.visibility === 'password'}
										<span class="w-2 h-2 rounded-full bg-gray-400 shrink-0" title="Private"></span>
									{/if}
								</span>
							</a>
						{/each}
					{/if}
					<!-- View more link - only show if there are more than 4 facets -->
					{#if facetsTotalCount > 4}
						<a
							href="/admin/views"
							class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
							title={!$adminSidebarOpen ? 'View more facets' : undefined}
						>
							<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
							</svg>
							<span class={$adminSidebarOpen ? '' : 'sr-only'}>View more...</span>
						</a>
					{/if}
					<!-- New Facet button - always visible -->
					<a
						href="/admin/views/new"
						class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/views/new')
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
							: 'text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20'}"
						title={!$adminSidebarOpen ? 'Create new facet' : undefined}
						aria-current={isActive('/admin/views/new') ? 'page' : undefined}
					>
						<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						<span class={$adminSidebarOpen ? '' : 'sr-only'}>New Facet</span>
					</a>
			</div>
		</div>

		<!-- Your Information and Your Voice Sections -->
		{#each primarySections as section (section.id)}
			{@const sectionId = sectionTitleToId[section.title] || `sidebar-${section.title.toLowerCase().replace(/\s+/g, '-')}`}
			<div class="space-y-2">
				<button
					type="button"
					onclick={() => sidebarSectionStates.toggle(sectionId, ALL_SECTION_IDS)}
					class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
					aria-expanded={isSectionExpanded(sectionId)}
					aria-controls="{sectionId}-items"
				>
					<span>{section.title}</span>
					<svg
						class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(sectionId) ? 'rotate-0' : '-rotate-90'}"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
				</button>
				{#if isSectionExpanded(sectionId)}
					<div id="{sectionId}-items" class="space-y-1">
						{#each section.items as item (item.href)}
							<a
								href={item.href}
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(item.href)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? `${section.title}: ${item.label}` : undefined}
								aria-current={isActive(item.href) ? 'page' : undefined}
							>
							{#if item.icon === 'home'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
								</svg>
							{:else if item.icon === 'user'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
								</svg>
							{:else if item.icon === 'mail'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
							{:else if item.icon === 'briefcase'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
							{:else if item.icon === 'folder'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
								</svg>
							{:else if item.icon === 'academic'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path d="M12 14l9-5-9-5-9 5 9 5z" />
									<path d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14zm-4 6v-7.5l4-2.222" />
								</svg>
							{:else if item.icon === 'badge'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
								</svg>
							{:else if item.icon === 'chip'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
								</svg>
							{:else if item.icon === 'document'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
								</svg>
							{:else if item.icon === 'star'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.286 3.967a1 1 0 00.95.69h4.184c.969 0 1.371 1.24.588 1.81l-3.39 2.463a1 1 0 00-.364 1.118l1.287 3.966c.3.922-.755 1.688-1.54 1.118l-3.39-2.462a1 1 0 00-1.176 0l-3.39 2.462c-.784.57-1.838-.196-1.539-1.118l1.287-3.966a1 1 0 00-.364-1.118L2.04 9.394c-.783-.57-.38-1.81.588-1.81h4.184a1 1 0 00.95-.69l1.287-3.967z" />
								</svg>
							{:else if item.icon === 'presentation'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
								</svg>
							{:else if item.icon === 'eye'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							{:else if item.icon === 'link'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
								</svg>
							{:else if item.icon === 'download'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
								</svg>
							{:else if item.icon === 'image'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
									<circle cx="9" cy="9" r="2" />
									<path d="m21 15-4-4a3 3 0 0 0-4.24 0L3 21" />
								</svg>
							{:else if item.icon === 'cog'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
								</svg>
							{:else if item.icon === 'sparkle'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8l2 2-2 2-2-2 2-2zm12-5l1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3zm-4 9l1.5 4.5L19 18l-4.5 1.5L13 24l-1.5-4.5L7 18l4.5-1.5L13 12z" />
								</svg>
							{:else if item.icon === 'puzzle'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 4a2 2 0 114 0v1a1 1 0 001 1h3a1 1 0 011 1v3a1 1 0 01-1 1h-1a2 2 0 100 4h1a1 1 0 011 1v3a1 1 0 01-1 1h-3a1 1 0 01-1-1v-1a2 2 0 10-4 0v1a1 1 0 01-1 1H6a1 1 0 01-1-1v-3a1 1 0 00-1-1H3a2 2 0 110-4h1a1 1 0 001-1V7a1 1 0 011-1h3a1 1 0 001-1V4z" />
								</svg>
							{:else if item.icon === 'info'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							{/if}

							<!-- Always render label for screen readers, visually hide when sidebar collapsed -->
							<span class={$adminSidebarOpen ? '' : 'sr-only'}>{item.label}</span>
						</a>
						{/each}
					</div>
				{/if}
			</div>
		{/each}

		<!-- Testimonials Section -->
		<div class="space-y-2">
			<button
				type="button"
				onclick={() => sidebarSectionStates.toggle(SECTION_IDS.testimonials, ALL_SECTION_IDS)}
				class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
				aria-expanded={isSectionExpanded(SECTION_IDS.testimonials)}
				aria-controls="sidebar-testimonials-items"
			>
				<span class="flex items-center gap-2">
					Testimonials
					{#if $testimonialsStore.pendingCount > 0}
						<span class="inline-flex items-center justify-center px-1.5 py-0.5 text-xs font-medium rounded-full bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200">
							{$testimonialsStore.pendingCount}
						</span>
					{/if}
				</span>
				<svg
					class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(SECTION_IDS.testimonials) ? 'rotate-0' : '-rotate-90'}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
			{#if isSectionExpanded(SECTION_IDS.testimonials)}
				<div id="sidebar-testimonials-items" class="space-y-1">
					<a
						href="/admin/testimonials"
						class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/testimonials')
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
							: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
						title={!$adminSidebarOpen ? 'Testimonials' : undefined}
						aria-current={isActive('/admin/testimonials') ? 'page' : undefined}
					>
						<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
						</svg>
						<span class={$adminSidebarOpen ? '' : 'sr-only'}>Manage</span>
					</a>
					<a
						href="/admin/testimonials/requests"
						class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/testimonials/requests')
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
							: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
						title={!$adminSidebarOpen ? 'Request Links' : undefined}
						aria-current={isActive('/admin/testimonials/requests') ? 'page' : undefined}
					>
						<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
						</svg>
						<span class={$adminSidebarOpen ? '' : 'sr-only'}>Request Links</span>
					</a>
				</div>
			{/if}
		</div>

		<!-- Settings Section -->
		{#each settingsSections as section (section.id)}
			{@const sectionId = sectionTitleToId[section.title] || `sidebar-${section.title.toLowerCase().replace(/\s+/g, '-')}`}
			<div class="space-y-2">
				<button
					type="button"
					onclick={() => sidebarSectionStates.toggle(sectionId, ALL_SECTION_IDS)}
					class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
					aria-expanded={isSectionExpanded(sectionId)}
					aria-controls="{sectionId}-items"
				>
					<span>{section.title}</span>
					<svg
						class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(sectionId) ? 'rotate-0' : '-rotate-90'}"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
				</button>
				{#if isSectionExpanded(sectionId)}
					<div id="{sectionId}-items" class="space-y-1">
						{#each section.items as item (item.href)}
							<a
								href={item.href}
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(item.href)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? `${section.title}: ${item.label}` : undefined}
								aria-current={isActive(item.href) ? 'page' : undefined}
							>
							{#if item.icon === 'cog'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
								</svg>
							{:else if item.icon === 'image'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
									<circle cx="9" cy="9" r="2" />
									<path d="m21 15-4-4a3 3 0 0 0-4.24 0L3 21" />
								</svg>
							{:else if item.icon === 'link'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
								</svg>
							{:else if item.icon === 'badge'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
								</svg>
							{:else if item.icon === 'eye'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							{:else if item.icon === 'star'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.286 3.967a1 1 0 00.95.69h4.184c.969 0 1.371 1.24.588 1.81l-3.39 2.463a1 1 0 00-.364 1.118l1.287 3.966c.3.922-.755 1.688-1.54 1.118l-3.39-2.462a1 1 0 00-1.176 0l-3.39 2.462c-.784.57-1.838-.196-1.539-1.118l1.287-3.966a1 1 0 00-.364-1.118L2.04 9.394c-.783-.57-.38-1.81.588-1.81h4.184a1 1 0 00.95-.69l1.287-3.967z" />
								</svg>
							{:else if item.icon === 'sparkle'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8l2 2-2 2-2-2 2-2zm12-5l1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3zm-4 9l1.5 4.5L19 18l-4.5 1.5L13 24l-1.5-4.5L7 18l4.5-1.5L13 12z" />
								</svg>
							{:else if item.icon === 'chip'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
								</svg>
							{:else if item.icon === 'info'}
								<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							{/if}

							<span class={$adminSidebarOpen ? '' : 'sr-only'}>{item.label}</span>
						</a>
						{/each}
					</div>
				{/if}
			</div>
		{/each}
	</nav>

	<!-- Version -->
	{#if $adminSidebarOpen}
		<div class="shrink-0 px-3 pb-4">
			<div class="text-center text-xs text-gray-400 dark:text-gray-500">
				{appVersion}
			</div>
			{#if newVersionAvailable}
				<div class="mt-1 text-center">
					<span class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300">
						<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
						</svg>
						Update available
					</span>
				</div>
			{/if}
		</div>
	{/if}
</aside>

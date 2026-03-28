<script lang="ts">
	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import { t } from 'svelte-i18n';
	import { adminSidebarOpen, sidebarSectionStates, sidebarFacetsVersion } from '$lib/stores';
	import { collection } from '$lib/stores/demo';
	import { testimonialsStore, refreshTestimonialsPendingCount } from '$lib/stores/testimonials';
	import { hasFeature } from '$lib/stores/plan';
	import {
		newVersionAvailable,
		isCheckingVersion,
		checkForNewVersion,
		forceVersionCheck
	} from '$lib/stores/version';

	const appVersion = __APP_VERSION__;

	const hasAnalytics = hasFeature('basic_analytics');
	const hasCourses = hasFeature('courses');
	const hasPricing = hasFeature('pricing');
	const hasNewsletter = hasFeature('newsletter');
	const hasDiscussions = hasFeature('discussions');

	interface Props {
		isMobile?: boolean;
	}

	let { isMobile = false }: Props = $props();

	// State for dynamically loaded facets
	let facets: Array<Record<string, unknown>> = $state([]);
	let facetsLoading = $state(true);
	let facetsError = $state(false);
	let facetsTotalCount = $state(0);

	// Badge counts
	let draftCount = $state(0);
	let newSubscriberCount = $state(0);

	// Quick-create dropdown state
	let contentMenuOpen = $state(false);
	let audienceMenuOpen = $state(false);
	let contentMenuPos = $state({ top: 0, left: 0 });
	let audienceMenuPos = $state({ top: 0, left: 0 });

	// Debounce timer to prevent rapid successive loadFacets calls
	let loadFacetsTimer: ReturnType<typeof setTimeout> | null = null;

	// Section IDs for collapsible sections (independent collapse)
	const SECTION_IDS = {
		facets: 'sidebar-facets',
		content: 'sidebar-content',
		audience: 'sidebar-audience',
		settings: 'sidebar-settings',
	};

	// All section IDs as array
	const ALL_SECTION_IDS = Object.values(SECTION_IDS);

	// Default first-visit: Facets + Content expanded
	const DEFAULT_EXPANDED = [SECTION_IDS.facets, SECTION_IDS.content];

	// Helper to check if a section is expanded
	function isSectionExpanded(sectionId: string): boolean {
		return sidebarSectionStates.isExpanded($sidebarSectionStates, sectionId, false);
	}

	onMount(() => {
		sidebarSectionStates.initialize(ALL_SECTION_IDS, DEFAULT_EXPANDED);
		scheduleFacetsLoad();
		refreshTestimonialsPendingCount();
		loadBadgeCounts();
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

	// Close menus on click outside
	$effect(() => {
		if (!contentMenuOpen && !audienceMenuOpen) return;
		function handleClick(e: MouseEvent) {
			const target = e.target as HTMLElement;
			if (!target.closest('[data-quick-create-menu]') && !target.closest('[data-quick-create-trigger]')) {
				contentMenuOpen = false;
				audienceMenuOpen = false;
			}
		}
		function handleKeydown(e: KeyboardEvent) {
			if (e.key === 'Escape') {
				contentMenuOpen = false;
				audienceMenuOpen = false;
			}
		}
		document.addEventListener('click', handleClick, true);
		document.addEventListener('keydown', handleKeydown);
		return () => {
			document.removeEventListener('click', handleClick, true);
			document.removeEventListener('keydown', handleKeydown);
		};
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

	async function loadBadgeCounts() {
		try {
			// Draft count: unpublished posts, talks, custom content
			const cols = ['posts', 'talks', 'custom_content'];
			let drafts = 0;
			for (const col of cols) {
				try {
					const result = await collection(col).getList(1, 1, {
						filter: 'is_draft = true',
						$cancelKey: `sidebar-draft-${col}`
					});
					drafts += result.totalItems;
				} catch {
					// Collection may not exist or have is_draft field
				}
			}
			draftCount = drafts;

			// New subscriber count (last 7 days)
			try {
				const sevenDaysAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().replace('T', ' ');
				const subResult = await collection('subscribers').getList(1, 1, {
					filter: `created >= "${sevenDaysAgo}"`,
					$cancelKey: 'sidebar-new-subs'
				});
				newSubscriberCount = subResult.totalItems;
			} catch {
				newSubscriberCount = 0;
			}
		} catch {
			// Non-critical - badges just won't show
		}
	}

	// Nav item type with optional subgroup headers
	interface NavItem {
		href: string;
		labelKey: string;
		icon: string;
	}
	interface SubGroupHeader {
		type: 'subgroup';
		labelKey: string;
	}
	type SidebarEntry = NavItem | SubGroupHeader;

	function isSubGroup(entry: SidebarEntry): entry is SubGroupHeader {
		return 'type' in entry && entry.type === 'subgroup';
	}

	// Reactive content items (merged Portfolio + Content)
	const contentItems = $derived.by((): SidebarEntry[] => {
		const items: SidebarEntry[] = [];

		// Portfolio items
		items.push({ href: '/admin/experience', labelKey: 'admin.sidebar.experience', icon: 'briefcase' });
		items.push({ href: '/admin/projects', labelKey: 'admin.sidebar.projects', icon: 'folder' });
		items.push({ href: '/admin/education', labelKey: 'admin.sidebar.education', icon: 'academic' });
		items.push({ href: '/admin/certifications', labelKey: 'admin.sidebar.certifications', icon: 'badge' });
		items.push({ href: '/admin/awards', labelKey: 'admin.sidebar.awards', icon: 'star' });
		items.push({ href: '/admin/skills', labelKey: 'admin.sidebar.skills', icon: 'chip' });

		// Publish subgroup
		items.push({ type: 'subgroup', labelKey: 'admin.sidebar.content' });
		items.push({ href: '/admin/posts', labelKey: 'admin.sidebar.posts', icon: 'document' });
		items.push({ href: '/admin/talks', labelKey: 'admin.sidebar.talks', icon: 'presentation' });
		if ($hasCourses) {
			items.push({ href: '/admin/courses', labelKey: 'admin.sidebar.courses', icon: 'academic' });
		}
		items.push({ href: '/admin/custom', labelKey: 'admin.sidebar.custom_content', icon: 'puzzle' });
		items.push({ href: '/admin/import', labelKey: 'admin.sidebar.import_ai', icon: 'sparkle' });
		items.push({ href: '/admin/settings/tags', labelKey: 'admin.sidebar.admin_tags', icon: 'chip' });

		return items;
	});

	// Reactive audience items (Analytics, Contacts, Testimonials, Comments, Newsletter)
	const audienceItems = $derived.by((): SidebarEntry[] => {
		const items: SidebarEntry[] = [];

		if ($hasAnalytics) {
			items.push({ href: '/admin/analytics', labelKey: 'admin.sidebar.analytics', icon: 'chart' });
		}
		items.push({ href: '/admin/contacts', labelKey: 'admin.sidebar.contact_methods', icon: 'mail' });

		// Testimonials subgroup
		items.push({ type: 'subgroup', labelKey: 'admin.sidebar.testimonials' });
		items.push({ href: '/admin/testimonials', labelKey: 'admin.sidebar.manage', icon: 'chat' });
		items.push({ href: '/admin/testimonials/requests', labelKey: 'admin.sidebar.request_links', icon: 'link' });
		if ($hasDiscussions) {
			items.push({ href: '/admin/comments', labelKey: 'admin.sidebar.comments', icon: 'comment' });
		}

		// Newsletter subgroup
		if ($hasNewsletter) {
			items.push({ type: 'subgroup', labelKey: 'admin.sidebar.subscribers' });
			items.push({ href: '/admin/subscribers', labelKey: 'admin.sidebar.subscribers', icon: 'users' });
		}

		return items;
	});

	// Reactive settings items
	const settingsItems = $derived.by((): SidebarEntry[] => {
		const items: SidebarEntry[] = [];

		items.push({ href: '/admin/settings/account', labelKey: 'admin.sidebar.account_security', icon: 'badge' });
		items.push({ href: '/admin/settings/site', labelKey: 'admin.sidebar.site_settings', icon: 'cog' });
		items.push({ href: '/admin/settings/features', labelKey: 'admin.sidebar.features', icon: 'puzzle' });
		items.push({ href: '/admin/settings/integrations', labelKey: 'admin.sidebar.integrations', icon: 'sparkle' });

		// Commerce subgroup
		if ($hasPricing) {
			items.push({ type: 'subgroup', labelKey: 'admin.sidebar.commerce' });
			items.push({ href: '/admin/pricing', labelKey: 'admin.sidebar.pricing', icon: 'currency' });
			items.push({ href: '/admin/coupons', labelKey: 'admin.sidebar.coupons', icon: 'tag' });
		}

		items.push({ href: '/admin/settings/audit', labelKey: 'admin.sidebar.audit_log', icon: 'shield' });
		items.push({ href: '/admin/settings/about', labelKey: 'admin.sidebar.about_facet', icon: 'info' });

		return items;
	});

	// Collect all sidebar hrefs for longest-match comparison
	const allSidebarHrefs = $derived.by(() => {
		const hrefs = new Set<string>();
		hrefs.add('/admin');
		hrefs.add('/admin/homepage');
		hrefs.add('/admin/media');
		hrefs.add('/admin/views/new');
		for (const entry of contentItems) {
			if (!isSubGroup(entry)) hrefs.add(entry.href);
		}
		for (const entry of audienceItems) {
			if (!isSubGroup(entry)) hrefs.add(entry.href);
		}
		for (const entry of settingsItems) {
			if (!isSubGroup(entry)) hrefs.add(entry.href);
		}
		for (const facet of facets) hrefs.add(`/admin/views/${facet.id}`);
		return hrefs;
	});

	// Reactive function that updates when $page changes
	// Uses longest-match to avoid highlighting parent items when a child item matches
	let isActive = $derived((href: string): boolean => {
		const currentPath = $page.url.pathname;

		// Exact match for dashboard
		if (href === '/admin') {
			return currentPath === '/admin' || currentPath === '/admin/';
		}

		// Exact match always wins
		if (currentPath === href) return true;

		// Prefix match: only if no other sidebar item is a longer match
		if (currentPath.startsWith(href + '/')) {
			for (const other of allSidebarHrefs) {
				if (other.length > href.length && other !== href &&
					(currentPath === other || currentPath.startsWith(other + '/'))) {
					return false;
				}
			}
			return true;
		}

		return false;
	});

	function openQuickCreate(e: MouseEvent, section: 'content' | 'audience') {
		e.stopPropagation();
		const trigger = e.currentTarget as HTMLElement;
		const rect = trigger.getBoundingClientRect();
		const pos = { top: rect.bottom + 4, left: rect.left };
		if (pos.left + 192 > window.innerWidth) {
			pos.left = window.innerWidth - 200;
		}
		if (section === 'content') {
			audienceMenuOpen = false;
			contentMenuPos = pos;
			contentMenuOpen = !contentMenuOpen;
		} else {
			contentMenuOpen = false;
			audienceMenuPos = pos;
			audienceMenuOpen = !audienceMenuOpen;
		}
	}

	function handleMenuKeydown(e: KeyboardEvent) {
		const menu = (e.currentTarget as HTMLElement);
		const menuItems = menu.querySelectorAll<HTMLAnchorElement>('[role="menuitem"]');
		const current = Array.from(menuItems).indexOf(document.activeElement as HTMLAnchorElement);
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			menuItems[(current + 1) % menuItems.length]?.focus();
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			menuItems[(current - 1 + menuItems.length) % menuItems.length]?.focus();
		} else if (e.key === 'Tab') {
			contentMenuOpen = false;
			audienceMenuOpen = false;
		}
	}

	// Focus first menu item after dropdown opens
	$effect(() => {
		if (contentMenuOpen || audienceMenuOpen) {
			requestAnimationFrame(() => {
				const menu = document.querySelector('[data-quick-create-menu]');
				const firstItem = menu?.querySelector<HTMLAnchorElement>('[role="menuitem"]');
				firstItem?.focus();
			});
		}
	});

	// Badge values
	const contentBadge = $derived(draftCount);
	const audienceBadge = $derived($testimonialsStore.pendingCount > 0 ? $testimonialsStore.pendingCount : newSubscriberCount);
	const audienceBadgeAmber = $derived($testimonialsStore.pendingCount > 0);
</script>

<aside
	id="admin-sidebar"
	class="fixed top-16 h-[calc(100vh-4rem)] bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 transition-all duration-200 z-30 flex flex-col
		{isMobile
			? ($adminSidebarOpen ? 'left-0 w-64' : '-left-64 w-64')
			: ($adminSidebarOpen ? 'left-0 w-64' : 'left-0 w-16')
		}"
	aria-label={$t('shared.aria.admin_navigation')}
>
	<nav class="p-3 space-y-4 flex-1 min-h-0 overflow-y-auto overscroll-contain" aria-label={$t('shared.aria.main_menu')}>
		<!-- Dashboard and Profile - always visible -->
		<div class="space-y-1">
			<a
				href="/admin"
				class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin')
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
				title={!$adminSidebarOpen ? $t('admin.sidebar.dashboard') : undefined}
				aria-current={isActive('/admin') ? 'page' : undefined}
			>
				<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
				</svg>
				<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t('admin.sidebar.dashboard')}</span>
			</a>
			<a
				href="/admin/homepage"
				class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/homepage')
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
				title={!$adminSidebarOpen ? $t('admin.sidebar.homepage') : undefined}
				aria-current={isActive('/admin/homepage') ? 'page' : undefined}
			>
				<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
				</svg>
				<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t('admin.sidebar.homepage')}</span>
			</a>
			<a
				href="/admin/media"
				class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/media')
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
				title={!$adminSidebarOpen ? $t('admin.sidebar.media_library') : undefined}
				aria-current={isActive('/admin/media') ? 'page' : undefined}
			>
				<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<rect x="3" y="3" width="18" height="18" rx="2" ry="2" stroke-width="2" />
					<circle cx="9" cy="9" r="2" stroke-width="2" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m21 15-4-4a3 3 0 0 0-4.24 0L3 21" />
				</svg>
				<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t('admin.sidebar.media_library')}</span>
			</a>
		</div>

		<!-- Facets Section - collapsible -->
		<div class="space-y-2">
			<button
				type="button"
				onclick={() => sidebarSectionStates.toggle(SECTION_IDS.facets)}
				class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
				aria-expanded={isSectionExpanded(SECTION_IDS.facets)}
				aria-controls="sidebar-facets-items"
			>
				<span class="flex items-center gap-1.5">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3l9 9-9 9-9-9 9-9z" />
					</svg>
					{$t('admin.sidebar.facets')}
				</span>
				<svg
					class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(SECTION_IDS.facets) ? 'rotate-0' : '-rotate-90'}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
			{#if isSectionExpanded(SECTION_IDS.facets)}
			<div id="sidebar-facets-items" class="space-y-1 animate-slide-in-down">
					{#if facetsLoading}
						<div class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400 {$adminSidebarOpen ? '' : 'sr-only'}">
							{$t('admin.sidebar.loading')}
						</div>
					{:else if facetsError}
						<!-- Error state with retry -->
						<div class="px-3 py-2 {$adminSidebarOpen ? '' : 'sr-only'}">
							<p class="text-sm text-red-500 dark:text-red-400 mb-2">{$t('admin.sidebar.unable_to_load_facets')}</p>
							<button
								type="button"
								onclick={() => loadFacets()}
								class="inline-flex items-center gap-1 text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
								</svg>
								{$t('admin.sidebar.retry')}
							</button>
						</div>
					{:else}
						{#each facets as facet}
							<a
								href="/admin/views/{facet.id}"
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(`/admin/views/${facet.id}`)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? `${$t('admin.sidebar.facets')}: ${facet.name}` : undefined}
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
										<span class="w-2 h-2 rounded-full bg-green-500 shrink-0" title={$t('admin.sidebar.public')}></span>
									{:else if facet.visibility === 'unlisted'}
										<span class="w-2 h-2 rounded-full bg-yellow-500 shrink-0" title={$t('admin.sidebar.unlisted')}></span>
									{:else if facet.visibility === 'private' || facet.visibility === 'password'}
										<span class="w-2 h-2 rounded-full bg-gray-400 shrink-0" title={$t('admin.sidebar.private')}></span>
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
							title={!$adminSidebarOpen ? $t('admin.sidebar.view_more') : undefined}
						>
							<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
							</svg>
							<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t('admin.sidebar.view_more')}</span>
						</a>
					{/if}
					<!-- New Facet button - always visible -->
					<a
						href="/admin/views/new"
						class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive('/admin/views/new')
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
							: 'text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20'}"
						title={!$adminSidebarOpen ? $t('admin.sidebar.new_facet') : undefined}
						aria-current={isActive('/admin/views/new') ? 'page' : undefined}
					>
						<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t('admin.sidebar.new_facet')}</span>
					</a>
			</div>
			{:else}
				<div id="sidebar-facets-items" hidden></div>
			{/if}
		</div>

		<!-- Content Section (merged Portfolio + Content) -->
		<div class="space-y-2">
			<button
				type="button"
				onclick={() => sidebarSectionStates.toggle(SECTION_IDS.content)}
				class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
				aria-expanded={isSectionExpanded(SECTION_IDS.content)}
				aria-controls="sidebar-content-items"
			>
				<span class="flex items-center gap-1.5">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
					</svg>
					{$t('admin.sidebar.content')}
					{#if contentBadge > 0}
						<span class="inline-flex items-center justify-center bg-gray-700 text-gray-300 text-[10px] font-semibold rounded-full h-5 min-w-5 px-1.5" aria-label="{contentBadge} drafts">
							{contentBadge}
						</span>
					{/if}
					{#if $adminSidebarOpen}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<span
							role="button"
							tabindex="-1"
							data-quick-create-trigger
							onclick={(e) => { e.stopPropagation(); openQuickCreate(e, 'content'); }}
							class="inline-flex items-center justify-center w-5 h-5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors rounded"
							aria-label={$t('admin.sidebar.new_content')}
							aria-expanded={contentMenuOpen}
							aria-haspopup="menu"
						>
							<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
							</svg>
						</span>
					{/if}
				</span>
				<svg
					class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(SECTION_IDS.content) ? 'rotate-0' : '-rotate-90'}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
			{#if isSectionExpanded(SECTION_IDS.content)}
				<div id="sidebar-content-items" class="space-y-0 animate-slide-in-down">
					{#each contentItems as entry, i}
						{#if isSubGroup(entry)}
							<div class="text-[10px] font-semibold tracking-wider text-gray-500 dark:text-gray-400 uppercase {i === 0 ? '' : 'mt-5'} mb-1 px-3 {$adminSidebarOpen ? '' : 'sr-only'}">
								{$t(entry.labelKey)}
							</div>
						{:else}
							{@const item = entry}
							<a
								href={item.href}
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(item.href)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? $t(item.labelKey) : undefined}
								aria-current={isActive(item.href) ? 'page' : undefined}
							>
								{@render navIcon(item.icon)}
								<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t(item.labelKey)}</span>
							</a>
						{/if}
					{/each}
				</div>
			{:else}
				<div id="sidebar-content-items" hidden></div>
			{/if}
		</div>

		<!-- Audience Section -->
		<div class="space-y-2">
			<button
				type="button"
				onclick={() => sidebarSectionStates.toggle(SECTION_IDS.audience)}
				class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
				aria-expanded={isSectionExpanded(SECTION_IDS.audience)}
				aria-controls="sidebar-audience-items"
			>
				<span class="flex items-center gap-1.5">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
					</svg>
					{$t('admin.sidebar.audience')}
					{#if audienceBadge > 0}
						<span
							class="inline-flex items-center justify-center text-[10px] font-semibold rounded-full h-5 min-w-5 px-1.5 {audienceBadgeAmber ? 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200' : 'bg-gray-700 text-gray-300'}"
							aria-label={audienceBadgeAmber ? $t('admin.sidebar.testimonials') : $t('admin.sidebar.subscribers')}
						>
							{audienceBadge}
						</span>
					{/if}
					{#if $adminSidebarOpen}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<span
							role="button"
							tabindex="-1"
							data-quick-create-trigger
							onclick={(e) => { e.stopPropagation(); openQuickCreate(e, 'audience'); }}
							class="inline-flex items-center justify-center w-5 h-5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors rounded"
							aria-label={$t('admin.sidebar.quick_create')}
							aria-expanded={audienceMenuOpen}
							aria-haspopup="menu"
						>
							<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
							</svg>
						</span>
					{/if}
				</span>
				<svg
					class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(SECTION_IDS.audience) ? 'rotate-0' : '-rotate-90'}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
			{#if isSectionExpanded(SECTION_IDS.audience)}
				<div id="sidebar-audience-items" class="space-y-0 animate-slide-in-down">
					{#each audienceItems as entry, i}
						{#if isSubGroup(entry)}
							<div class="text-[10px] font-semibold tracking-wider text-gray-500 dark:text-gray-400 uppercase {i === 0 ? '' : 'mt-5'} mb-1 px-3 {$adminSidebarOpen ? '' : 'sr-only'}">
								{$t(entry.labelKey)}
							</div>
						{:else}
							{@const item = entry}
							<a
								href={item.href}
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(item.href)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? $t(item.labelKey) : undefined}
								aria-current={isActive(item.href) ? 'page' : undefined}
							>
								{@render navIcon(item.icon)}
								<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t(item.labelKey)}</span>
							</a>
						{/if}
					{/each}
				</div>
			{:else}
				<div id="sidebar-audience-items" hidden></div>
			{/if}
		</div>

		<!-- Settings Section -->
		<div class="space-y-2">
			<button
				type="button"
				onclick={() => sidebarSectionStates.toggle(SECTION_IDS.settings)}
				class="flex items-center justify-between w-full text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors {$adminSidebarOpen ? '' : 'sr-only'}"
				aria-expanded={isSectionExpanded(SECTION_IDS.settings)}
				aria-controls="sidebar-settings-items"
			>
				<span class="flex items-center gap-1.5">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
					</svg>
					{$t('admin.sidebar.settings')}
				</span>
				<svg
					class="w-4 h-4 transition-transform duration-200 {isSectionExpanded(SECTION_IDS.settings) ? 'rotate-0' : '-rotate-90'}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>
			{#if isSectionExpanded(SECTION_IDS.settings)}
				<div id="sidebar-settings-items" class="space-y-1 animate-slide-in-down">
					{#each settingsItems as entry}
						{#if isSubGroup(entry)}
							<div class="text-[10px] font-semibold tracking-wider text-gray-500 dark:text-gray-400 uppercase mt-3 mb-1 px-3 {$adminSidebarOpen ? '' : 'sr-only'}">
								{$t(entry.labelKey)}
							</div>
						{:else}
							{@const item = entry}
							<a
								href={item.href}
								class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(item.href)
									? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
									: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'}"
								title={!$adminSidebarOpen ? `${$t('admin.sidebar.settings')}: ${$t(item.labelKey)}` : undefined}
								aria-current={isActive(item.href) ? 'page' : undefined}
							>
								{@render navIcon(item.icon)}
								<span class={$adminSidebarOpen ? '' : 'sr-only'}>{$t(item.labelKey)}</span>
							</a>
						{/if}
					{/each}
				</div>
			{:else}
				<div id="sidebar-settings-items" hidden></div>
			{/if}
		</div>
	</nav>

	<!-- Version -->
	{#if $adminSidebarOpen}
		<div class="shrink-0 px-3 pb-3 pt-2 border-t border-gray-200 dark:border-gray-700">
			<div class="flex items-center justify-center gap-1.5">
				<span class="text-xs text-gray-400 dark:text-gray-500">{appVersion}</span>
				<button
					type="button"
					class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded transition-colors disabled:opacity-50"
					onclick={() => forceVersionCheck()}
					disabled={$isCheckingVersion}
					title={$t('admin.sidebar.check_for_updates')}
				>
					<svg
						class="w-3.5 h-3.5 {$isCheckingVersion ? 'animate-spin' : ''}"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
				</button>
			</div>
			{#if $newVersionAvailable}
				<div class="mt-1.5 text-center">
					<span class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300">
						<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
						</svg>
						{$t('admin.sidebar.update_available')}
					</span>
				</div>
			{/if}
		</div>
	{/if}
</aside>

<!-- Quick-create dropdown: Content -->
{#if contentMenuOpen}
	<div
		data-quick-create-menu
		role="menu"
		tabindex="-1"
		class="fixed bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-lg rounded-lg w-48 py-1 z-50"
		style="top: {contentMenuPos.top}px; left: {contentMenuPos.left}px;"
		onkeydown={(e) => handleMenuKeydown(e)}
	>
		<a href="/admin/posts" role="menuitem" class="flex items-center gap-2 px-3 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm" onclick={() => { contentMenuOpen = false; }}>
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
			{$t('admin.sidebar.new_post')}
		</a>
		<a href="/admin/talks" role="menuitem" class="flex items-center gap-2 px-3 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm" onclick={() => { contentMenuOpen = false; }}>
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" /></svg>
			{$t('admin.sidebar.new_talk')}
		</a>
		{#if $hasCourses}
			<a href="/admin/courses" role="menuitem" class="flex items-center gap-2 px-3 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm" onclick={() => { contentMenuOpen = false; }}>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path d="M12 14l9-5-9-5-9 5 9 5z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14zm-4 6v-7.5l4-2.222" /></svg>
				{$t('admin.sidebar.new_course')}
			</a>
		{/if}
	</div>
{/if}

<!-- Quick-create dropdown: Audience -->
{#if audienceMenuOpen}
	<div
		data-quick-create-menu
		role="menu"
		tabindex="-1"
		class="fixed bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-lg rounded-lg w-48 py-1 z-50"
		style="top: {audienceMenuPos.top}px; left: {audienceMenuPos.left}px;"
		onkeydown={(e) => handleMenuKeydown(e)}
	>
		<a href="/admin/testimonials/requests" role="menuitem" class="flex items-center gap-2 px-3 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm" onclick={() => { audienceMenuOpen = false; }}>
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" /></svg>
			{$t('admin.sidebar.request_links')}
		</a>
	</div>
{/if}

<!-- Reusable icon snippet -->
{#snippet navIcon(icon: string)}
	{#if icon === 'home'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
		</svg>
	{:else if icon === 'user'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
		</svg>
	{:else if icon === 'mail'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
		</svg>
	{:else if icon === 'briefcase'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
		</svg>
	{:else if icon === 'folder'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
		</svg>
	{:else if icon === 'academic'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path d="M12 14l9-5-9-5-9 5 9 5z" />
			<path d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14zm-4 6v-7.5l4-2.222" />
		</svg>
	{:else if icon === 'badge'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
		</svg>
	{:else if icon === 'chip'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
		</svg>
	{:else if icon === 'document'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
		</svg>
	{:else if icon === 'star'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.286 3.967a1 1 0 00.95.69h4.184c.969 0 1.371 1.24.588 1.81l-3.39 2.463a1 1 0 00-.364 1.118l1.287 3.966c.3.922-.755 1.688-1.54 1.118l-3.39-2.462a1 1 0 00-1.176 0l-3.39 2.462c-.784.57-1.838-.196-1.539-1.118l1.287-3.966a1 1 0 00-.364-1.118L2.04 9.394c-.783-.57-.38-1.81.588-1.81h4.184a1 1 0 00.95-.69l1.287-3.967z" />
		</svg>
	{:else if icon === 'presentation'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
		</svg>
	{:else if icon === 'puzzle'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 4a2 2 0 114 0v1a1 1 0 001 1h3a1 1 0 011 1v3a1 1 0 01-1 1h-1a2 2 0 100 4h1a1 1 0 011 1v3a1 1 0 01-1 1h-3a1 1 0 01-1-1v-1a2 2 0 10-4 0v1a1 1 0 01-1 1H6a1 1 0 01-1-1v-3a1 1 0 00-1-1H3a2 2 0 110-4h1a1 1 0 001-1V7a1 1 0 011-1h3a1 1 0 001-1V4z" />
		</svg>
	{:else if icon === 'sparkle'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8l2 2-2 2-2-2 2-2zm12-5l1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3zm-4 9l1.5 4.5L19 18l-4.5 1.5L13 24l-1.5-4.5L7 18l4.5-1.5L13 12z" />
		</svg>
	{:else if icon === 'info'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
		</svg>
	{:else if icon === 'shield'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
		</svg>
	{:else if icon === 'cog'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
		</svg>
	{:else if icon === 'chat'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
		</svg>
	{:else if icon === 'link'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
		</svg>
	{:else if icon === 'comment'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
		</svg>
	{:else if icon === 'users'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
		</svg>
	{:else if icon === 'chart'}
		<svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
		</svg>
	{/if}
{/snippet}

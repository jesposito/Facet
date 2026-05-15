<script lang="ts">
	import { self } from 'svelte/legacy';

	import type { PageData } from './$types';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { navigating, page } from '$app/stores';
	import { t } from 'svelte-i18n';
	import { brandName, isDemoMode } from '$lib/stores/plan';
	import ProfileHero from '$components/public/ProfileHero.svelte';
	import ProfileNav from '$components/public/ProfileNav.svelte';
	import ExperienceSection from '$components/public/ExperienceSection.svelte';
	import ProjectsSection from '$components/public/ProjectsSection.svelte';
	import EducationSection from '$components/public/EducationSection.svelte';
	import CertificationsSection from '$components/public/CertificationsSection.svelte';
	import AwardsSection from '$components/public/AwardsSection.svelte';
	import SkillsSection from '$components/public/SkillsSection.svelte';
	import PostsSection from '$components/public/PostsSection.svelte';
	import TalksSection from '$components/public/TalksSection.svelte';
	import CoursesSection from '$components/public/CoursesSection.svelte';
	import TestimonialsSection from '$components/public/TestimonialsSection.svelte';
	import ContactMethodsList from '$components/public/ContactMethodsList.svelte';
	import CustomContentSection from '$components/public/CustomContentSection.svelte';
	import SiteNav from '$components/public/SiteNav.svelte';
	import ATSContent from '$components/public/ATSContent.svelte';
	import Footer from '$components/public/Footer.svelte';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import WelcomePage from '$components/public/WelcomePage.svelte';
	import { ACCENT_COLORS, type AccentColor } from '$lib/colors';
	import { getFontPack, DEFAULT_FONT_PACK } from '$lib/fonts';
	import { pb, currentUser, performLogout, getSectionLayout as getDefaultSectionLayout } from '$lib/pocketbase';
	import { generatePersonJsonLd, generateWebSiteJsonLd, serializeJsonLd, getCanonicalUrl, generateOpenGraphTags, type OpenGraphData } from '$lib/seo';
	import { goto } from '$app/navigation';

	// Track navigation to prevent showing WelcomePage during transitions
	let isNavigating = $derived($navigating !== null);

	// Track whether data has been loaded at least once to prevent WelcomePage flash.
	// During SPA navigation, $navigating becomes null BEFORE new data propagates,
	// causing a brief frame where !data.profile && !isNavigating is true.
	let hasHydratedData = $state(false);

	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'awards', 'skills', 'posts', 'talks', 'courses', 'testimonials', 'contacts'];

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	// Mark data as hydrated once we have real data (profile or content)
	$effect(() => {
		if (data.profile || data.experience?.length || data.projects?.length) {
			hasHydratedData = true;
		}
	});

	// Get headline, summary, and location - use view overrides if this is a default view
	let headline = $derived(data.view?.hero_headline || data.profile?.headline);
	let summary = $derived(data.view?.hero_summary || data.profile?.summary);
	let location = $derived(data.view?.hero_location || data.profile?.location);

	// Generate JSON-LD and Open Graph for SEO
	// Use APP_URL from server for correct https:// canonical URLs behind reverse proxy
	let baseUrl = $derived($page.data.appUrl || $page.url.origin);
	let personJsonLd = $derived(data.profile ? serializeJsonLd(generatePersonJsonLd(data.profile, baseUrl)) : null);
	let websiteJsonLd = $derived(data.profile ? serializeJsonLd(generateWebSiteJsonLd(data.profile, baseUrl)) : null);
	let canonicalUrl = $derived(getCanonicalUrl(baseUrl, ''));
	let ogTags = $derived(data.profile ? generateOpenGraphTags({
		title: data.profile.name || 'Profile',
		description: headline || 'Personal profile and portfolio',
		image: data.profile.avatar_url || undefined,
		url: canonicalUrl,
		type: 'profile',
		siteName: 'Facet'
	}) : {});

	// Section ordering - use view's custom order if available, then homepage order
	// Ensure any sections with data that aren't in the saved order get appended
	let effectiveSectionOrder = $derived.by(() => {
		const baseOrder = (data.sectionOrder && data.sectionOrder.length > 0)
			? data.sectionOrder
			: (data.homepageSectionOrder && data.homepageSectionOrder.length > 0)
				? data.homepageSectionOrder
				: DEFAULT_SECTION_ORDER;

		// If using a saved order, append any default sections with data that are missing
		if (baseOrder !== DEFAULT_SECTION_ORDER) {
			const missingWithData = DEFAULT_SECTION_ORDER.filter(section => {
				if (baseOrder.includes(section)) return false;
				// Check if this section has data
				switch (section) {
					case 'experience': return data.experience?.length > 0;
					case 'projects': return data.projects?.length > 0;
					case 'education': return data.education?.length > 0;
					case 'certifications': return data.certifications?.length > 0;
					case 'awards': return data.awards?.length > 0;
					case 'skills': return data.skills?.length > 0;
					case 'posts': return data.posts?.length > 0;
					case 'talks': return data.talks?.length > 0;
					case 'courses': return data.courses?.length > 0;
					case 'testimonials': return data.testimonials?.length > 0;
					case 'contacts': return data.contacts?.length > 0;
					default: return false;
				}
			});
			if (missingWithData.length > 0) {
				return [...baseOrder, ...missingWithData];
			}
		}
		return baseOrder;
	});

	// Helper to check if section key is a custom content section
	function isCustomSection(key: string): boolean {
		return key.startsWith('custom:');
	}

	// Get custom content ID from section key
	function getCustomContentId(key: string): string {
		return key.replace('custom:', '');
	}

	// Create a map of custom content by ID for easy lookup
	// The items have CustomContent-like structure from the API
	type CustomContentItem = {
		id: string;
		title: string;
		content: string;
		visibility: string;
		is_draft: boolean;
		sort_order: number;
		cover_image_url?: string;
		media_urls?: string[];
		media_refs?: string[];
		media_refs_expand?: Array<{
			id: string;
			url: string;
			title?: string;
			mime?: string;
			thumbnail_url?: string;
			description?: string;
			alt_text?: string;
		}>;
	};
	let customContentMap = $derived(new Map<string, CustomContentItem>(
		(data.customContent || []).map((item: Record<string, unknown>) => [item.id as string, item as unknown as CustomContentItem])
	));

	function getSectionLayout(sectionKey: string): string {
		// First check view layouts, then homepage sections
		if (data.sectionLayouts?.[sectionKey]) {
			return data.sectionLayouts[sectionKey];
		}
		if (data.homepageSections?.[sectionKey]?.layout) {
			return data.homepageSections[sectionKey].layout;
		}
		// Use the section-specific default (e.g. 'wall' for testimonials, 'grouped' for certifications)
		return getDefaultSectionLayout(sectionKey);
	}

	type ContactLayoutType = 'vertical' | 'horizontal' | 'grid';
	function getContactLayout(): ContactLayoutType {
		const layout = getSectionLayout('contacts');
		if (layout === 'vertical' || layout === 'horizontal' || layout === 'grid') return layout;
		return 'vertical';
	}

	function getSectionWidth(sectionKey: string): string {
		// First check view widths, then homepage sections
		if (data.sectionWidths?.[sectionKey]) {
			return data.sectionWidths[sectionKey];
		}
		if (data.homepageSections?.[sectionKey]?.width) {
			return data.homepageSections[sectionKey].width;
		}
		return 'full';
	}

	function getCategoryOrder(sectionKey: string): string[] | undefined {
		// First check view category orders (default view), then homepage settings (legacy)
		if (data.sectionCategoryOrders?.[sectionKey]) {
			return data.sectionCategoryOrders[sectionKey];
		}
		// For skills, check the top-level skillsCategoryOrder (legacy homepage)
		if (sectionKey === 'skills' && data.skillsCategoryOrder?.length) {
			return data.skillsCategoryOrder;
		}
		return undefined;
	}

	function getDisabledCategories(sectionKey: string): string[] | undefined {
		// First check view disabled categories (default view), then homepage settings (legacy)
		if (data.sectionDisabledCategories?.[sectionKey]) {
			return data.sectionDisabledCategories[sectionKey];
		}
		// For legacy homepage, check the section config
		if (data.homepageSections?.[sectionKey]?.disabledCategories) {
			return data.homepageSections[sectionKey].disabledCategories;
		}
		return undefined;
	}

	function getCategoryDisplayModes(sectionKey: string): Record<string, string> | undefined {
		// First check view display modes (default view), then homepage settings (legacy)
		if (data.sectionCategoryDisplayModes?.[sectionKey]) {
			return data.sectionCategoryDisplayModes[sectionKey];
		}
		// For legacy homepage, check the section config
		if (data.homepageSections?.[sectionKey]?.categoryDisplayModes) {
			return data.homepageSections[sectionKey].categoryDisplayModes;
		}
		return undefined;
	}

	function getFeaturedId(sectionKey: string): string | undefined {
		const val = data.sectionFeaturedIds?.[sectionKey];
		return typeof val === 'string' ? val : undefined;
	}

	function getWidthClass(width: string): string {
		switch (width) {
			case 'half': return 'section-half';
			case 'third': return 'section-third';
			default: return 'section-full';
		}
	}

	// Print menu state
	let showPrintMenu = $state(false);
	let showGenerateModal = $state(false);
	let generating = $state(false);
	let modalEl: HTMLDivElement | undefined = $state();
	let printMenuTriggerEl: HTMLButtonElement | undefined = $state();
	let previousActiveElement: HTMLElement | null = $state(null);
	let aiPrintStatus = $state({
		available: false,
		ai_configured: false,
		pandoc_installed: false
	});
	let generationConfig = $state({
		format: 'pdf' as 'pdf' | 'docx',
		target_role: '',
		style: 'chronological' as 'chronological' | 'functional' | 'hybrid',
		length: 'two-page' as 'one-page' | 'two-page' | 'full'
	});
	let generatedUrl: string | null = $state(null);
	let landingMessage = $derived(data.landingPageMessage || 'This profile is being set up.');

	// Floating buttons visibility - hide when nav is pinned (sticky)
	let navPinned = $state(false);
	let sentinelEl: HTMLDivElement | null = $state(null);

	// Apply view-specific accent color if default view has one
	function applyAccentColor(colorName: AccentColor) {
		if (!browser) return;

		const color = ACCENT_COLORS[colorName];
		if (!color) return;

		const root = document.documentElement;
		root.style.setProperty('--color-primary-50', color.scale[50]);
		root.style.setProperty('--color-primary-100', color.scale[100]);
		root.style.setProperty('--color-primary-200', color.scale[200]);
		root.style.setProperty('--color-primary-300', color.scale[300]);
		root.style.setProperty('--color-primary-400', color.scale[400]);
		root.style.setProperty('--color-primary-500', color.scale[500]);
		root.style.setProperty('--color-primary-600', color.scale[600]);
		root.style.setProperty('--color-primary-700', color.scale[700]);
		root.style.setProperty('--color-primary-800', color.scale[800]);
		root.style.setProperty('--color-primary-900', color.scale[900]);
		root.style.setProperty('--color-primary-950', color.scale[950]);
	}

	onMount(() => {
		if (data.homepageDisabled) {
			console.log('[ROOT PAGE CLIENT] Homepage disabled');
			return;
		}

		console.log('[ROOT PAGE CLIENT] Page mounted', {
			hasProfile: !!data.profile,
			profileName: data.profile?.name,
			viewSlug: data.view?.slug,
			isDefaultView: data.isDefaultView,
			postsCount: data.posts?.length || 0,
			talksCount: data.talks?.length || 0
		});

		// View accent color takes priority over profile accent color
		// (default view may have its own accent color override)
		const accentColor = data.view?.accent_color || data.profile?.accent_color;
		if (accentColor) {
			applyAccentColor(accentColor as AccentColor);
		}

		// Apply font pack (view override > profile default)
		const fontPackName = data.view?.font_pack || data.profile?.font_pack;
		if (fontPackName && fontPackName !== DEFAULT_FONT_PACK) {
			const pack = getFontPack(fontPackName);
			const root = document.documentElement;
			root.style.setProperty('--font-heading', `'${pack.heading}', ${pack.headingFallback}`);
			root.style.setProperty('--font-body', `'${pack.body}', ${pack.bodyFallback}`);
			root.style.setProperty('--font-code', `'${pack.code}', ${pack.codeFallback}`);
			const existing = document.getElementById('view-google-fonts');
			if (existing) existing.remove();
			const link = document.createElement('link');
			link.id = 'view-google-fonts';
			link.rel = 'stylesheet';
			link.href = pack.googleFontsUrl;
			document.head.appendChild(link);
		}

		// Check AI Print availability
		checkAIPrintStatus();

		// Track when sentinel scrolls past top (nav becomes sticky)
		const checkSentinel = () => {
			if (!sentinelEl) return;
			const rect = sentinelEl.getBoundingClientRect();
			navPinned = rect.bottom <= 0;
		};

		// Defer initial check to next frame to allow async content (SiteNav) to render
		// SiteNav fetches data on mount, so we need to wait for it to potentially change layout
		requestAnimationFrame(checkSentinel);

		// Also check after a short delay in case SiteNav is still loading
		const delayedCheck = setTimeout(checkSentinel, 300);

		window.addEventListener('scroll', checkSentinel, { passive: true });
		window.addEventListener('resize', checkSentinel, { passive: true });

		return () => {
			clearTimeout(delayedCheck);
			window.removeEventListener('scroll', checkSentinel);
			window.removeEventListener('resize', checkSentinel);
		};
	});

	async function checkAIPrintStatus() {
		if (data.homepageDisabled) return;

		try {
			const response = await fetch('/api/ai-print/status', {
				headers: { Authorization: pb.authStore.token || '' }
			});
			if (response.ok) {
				const result = await response.json();
				aiPrintStatus = {
					available: result.available,
					ai_configured: result.ai_configured,
					pandoc_installed: result.pandoc_installed
				};
			}
		} catch (err) {
			console.error('[AI-PRINT] Failed to check status:', err);
		}
	}

	async function generateResume() {
		if (data.homepageDisabled) return;

		const slug = data.view?.slug;
		if (!slug) return;
		generating = true;
		generatedUrl = null;

		try {
			// Use the view's hero_headline as target role (configured by profile owner)
			const config = {
				...generationConfig,
				target_role: data.view?.hero_headline || data.profile?.headline || ''
			};
			const response = await fetch(`/api/view/${slug}/generate`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify(config)
			});

			const result = await response.json();

			if (!response.ok) {
				throw new Error(result.error || 'Generation failed');
			}

			generatedUrl = result.download_url;

			// Auto-download the file
			if (generatedUrl) {
				const link = document.createElement('a');
				link.href = generatedUrl;
				link.download = `resume.${generationConfig.format}`;
				document.body.appendChild(link);
				link.click();
				document.body.removeChild(link);
			}
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to generate resume';
			alert(message);
		} finally {
			generating = false;
		}
	}

	function closePrintMenu() {
		showPrintMenu = false;
	}

	function closeGenerateModal() {
		showGenerateModal = false;
		generatedUrl = null;
		// Restore focus to the element that triggered the modal
		if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
			previousActiveElement.focus();
		}
	}

	// Modal focus trap, Escape key, and body scroll lock
	function handleModalKeydown(event: KeyboardEvent) {
		if (!showGenerateModal) return;

		if (event.key === 'Escape') {
			event.preventDefault();
			closeGenerateModal();
			return;
		}

		if (event.key === 'Tab') {
			const focusableElements = modalEl?.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			);
			if (!focusableElements?.length) return;

			const firstElement = focusableElements[0];
			const lastElement = focusableElements[focusableElements.length - 1];

			if (event.shiftKey && document.activeElement === firstElement) {
				event.preventDefault();
				lastElement.focus();
			} else if (!event.shiftKey && document.activeElement === lastElement) {
				event.preventDefault();
				firstElement.focus();
			}
		}
	}

	// Lock body scroll and manage focus when modal opens/closes
	$effect(() => {
		if (typeof document === 'undefined') return;
		if (showGenerateModal) {
			previousActiveElement = document.activeElement as HTMLElement;
			document.body.style.overflow = 'hidden';
			// Focus the first focusable element in the modal after render
			requestAnimationFrame(() => {
				const firstFocusable = modalEl?.querySelector<HTMLElement>(
					'select, button:not([disabled]), [href], input:not([disabled])'
				);
				firstFocusable?.focus();
			});
		} else {
			document.body.style.overflow = '';
		}
	});

	// Print menu keyboard handler
	function handlePrintMenuKeydown(event: KeyboardEvent) {
		if (!showPrintMenu) return;
		if (event.key === 'Escape') {
			event.preventDefault();
			closePrintMenu();
			printMenuTriggerEl?.focus();
		}
	}

	async function handleLogout() {
		await performLogout();
		window.location.href = '/admin/login';
	}
</script>

<svelte:head>
	<title>{data.profile?.name || 'Profile'} | {$brandName}</title>
	<meta name="description" content={headline || 'Personal profile and portfolio'} />

	<!-- Canonical URL -->
	<link rel="canonical" href={canonicalUrl} />

	<!-- Open Graph and Twitter Card meta tags -->
	{#each Object.entries(ogTags) as [property, content]}
		<meta property={property} content={content} />
	{/each}

	<!-- JSON-LD Structured Data -->
	{#if personJsonLd}
		{@html `<script type="application/ld+json">${personJsonLd}</script>`}
	{/if}
	{#if websiteJsonLd}
		{@html `<script type="application/ld+json">${websiteJsonLd}</script>`}
	{/if}
</svelte:head>

{#if data.homepageDisabled}
	<!-- Profile exists but homepage is disabled - show landing message -->
	<div class="min-h-screen bg-stone-50 dark:bg-stone-900 flex items-center justify-center px-4">
		<div class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden">
			<ThemeToggle />
		</div>
		<div class="max-w-2xl w-full bg-white dark:bg-stone-800 rounded-2xl shadow-lg border border-stone-200 dark:border-stone-700 p-8 text-center space-y-4">
			<h1 class="text-2xl font-semibold text-stone-900 dark:text-white">{$t('public.homepage.profile_private')}</h1>
			<p class="text-stone-600 dark:text-stone-300 leading-relaxed whitespace-pre-wrap">{landingMessage}</p>
			<p class="text-sm text-stone-500 dark:text-stone-400">
				{$t('public.homepage.views_accessible')}
			</p>
		</div>
	</div>
{:else if !data.profile && !data.experience?.length && !data.projects?.length && !isNavigating && !hasHydratedData}
	<!-- First-time visitor: no profile exists yet (don't show during navigation transitions or after data was already loaded) -->
	<WelcomePage />
{:else}
<div class="min-h-screen">
	<div
		class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden transition-opacity duration-200"
		class:opacity-0={navPinned}
		class:pointer-events-none={navPinned}
		inert={navPinned}
	>
		<!-- Print Menu -->
		<div class="relative">
			<button
				bind:this={printMenuTriggerEl}
				onclick={() => showPrintMenu = !showPrintMenu}
				class="p-2 rounded-lg bg-white/80 dark:bg-stone-800/80 backdrop-blur-sm shadow-sm border border-stone-200 dark:border-stone-700 hover:bg-stone-100 dark:hover:bg-stone-700 transition-colors"
				title={$t('public.homepage.print_options')}
				aria-label={$t('public.homepage.print_options')}
				aria-expanded={showPrintMenu}
				aria-haspopup="menu"
			>
				<svg class="w-5 h-5 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
				</svg>
			</button>

			{#if showPrintMenu}
				<div class="fixed inset-0" onclick={closePrintMenu} onkeydown={handlePrintMenuKeydown} role="presentation" tabindex="-1"></div>
				<div class="absolute right-0 mt-2 w-48 bg-white dark:bg-stone-800 rounded-lg shadow-lg border border-stone-200 dark:border-stone-700 py-1 z-50" role="menu" onkeydown={handlePrintMenuKeydown}>
					<button
						onclick={() => { window.print(); closePrintMenu(); }}
						class="w-full px-4 py-2 text-left text-sm text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-700 flex items-center gap-2"
						role="menuitem"
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
						</svg>
						{$t('public.homepage.simple_print')}
					</button>
					{#if aiPrintStatus.ai_configured && data.view?.slug}
						<button
							onclick={() => { showGenerateModal = true; closePrintMenu(); }}
							class="w-full px-4 py-2 text-left text-sm text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-700 flex items-center gap-2"
							role="menuitem"
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
							</svg>
							{$t('public.homepage.ai_resume')}
						</button>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Login button - for demo users, auto-logout and go to login page -->
		<!-- Always show on first-time screen, even if hide setting is enabled -->
		{#if !$isDemoMode && (!data.hideLoginButton || (!data.profile?.name && data.experience.length === 0 && data.projects.length === 0))}
			{#if $currentUser}
				<button
					onclick={handleLogout}
					class="px-3 py-2 rounded-lg bg-white/80 dark:bg-stone-800/80 backdrop-blur-sm shadow-sm border border-stone-200 dark:border-stone-700 hover:bg-stone-100 dark:hover:bg-stone-700 transition-colors text-sm font-medium text-stone-700 dark:text-stone-300"
					title={$t('public.homepage.log_in_account')}
					aria-label={$t('public.homepage.log_in_account')}
				>
					{$t('public.homepage.log_in')}
				</button>
			{:else}
				<a
					href="/admin/login"
					class="px-3 py-2 rounded-lg bg-white/80 dark:bg-stone-800/80 backdrop-blur-sm shadow-sm border border-stone-200 dark:border-stone-700 hover:bg-stone-100 dark:hover:bg-stone-700 transition-colors text-sm font-medium text-stone-700 dark:text-stone-300"
					title={$t('public.homepage.log_in')}
					aria-label={$t('public.homepage.log_in')}
				>
					{$t('public.homepage.log_in')}
				</a>
			{/if}
		{/if}

		<ThemeToggle />
	</div>

	<!-- Site Navigation (above hero) - only renders when position='above' -->
	<SiteNav
		slot="above"
		ctaUrl={data.profile?.cta_url || data.view?.cta_url || ''}
		ctaButtonText={data.profile?.cta_button_text || data.view?.cta_button_text || 'Learn More'}
		ctaText={data.profile?.cta_text || data.view?.cta_text || ''}
		ctaEnabled={data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
		ssrNavEnabled={data.siteNav?.enabled}
		ssrNavMode={data.siteNav?.mode}
		ssrNavPosition={data.siteNav?.position}
		ssrNavItems={data.siteNav?.items}
	/>

	<!-- Hero section with possible view overrides -->
	<ProfileHero
		profile={{
			...data.profile,
			headline,
			summary,
			location
		}}
		layout={(data.view?.hero_layout || data.profile?.hero_layout || 'standard') as 'standard' | 'centered' | 'split' | 'minimal' | 'stacked'}
		spacing={(data.view?.hero_spacing || data.profile?.hero_spacing || '') as '' | 'compact' | 'default' | 'spacious'}
	/>

	<!-- Site Navigation (below hero) / CTA Banner - only renders when position='below' (default) -->
	<SiteNav
		slot="below"
		ctaUrl={data.profile?.cta_url || data.view?.cta_url || ''}
		ctaButtonText={data.profile?.cta_button_text || data.view?.cta_button_text || 'Learn More'}
		ctaText={data.profile?.cta_text || data.view?.cta_text || ''}
		ctaEnabled={data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
		ssrNavEnabled={data.siteNav?.enabled}
		ssrNavMode={data.siteNav?.mode}
		ssrNavPosition={data.siteNav?.position}
		ssrNavItems={data.siteNav?.items}
	/>

	<div
		aria-hidden="true"
		class="h-px"
		bind:this={sentinelEl}
	></div>

	<ProfileNav
		hasExperience={data.experience.length > 0}
		hasProjects={data.projects.length > 0}
		hasEducation={data.education.length > 0}
		hasCertifications={data.certifications && data.certifications.length > 0}
		hasAwards={data.awards && data.awards.length > 0}
		hasSkills={data.skills.length > 0}
		hasPosts={data.posts && data.posts.length > 0}
		hasTalks={data.talks && data.talks.length > 0}
		hasCourses={data.courses && data.courses.length > 0}
		hasTestimonials={data.testimonials && data.testimonials.length > 0}
		hasContacts={data.contacts && data.contacts.length > 0}
		viewSlug=""
		sectionOrder={effectiveSectionOrder}
		customContent={data.customContent || []}
	/>

	<!-- Main content -->
	<main id="main-content" class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		{#if !data.profile?.name && data.experience.length === 0 && data.projects.length === 0}
			<!-- Empty profile state -->
			<div class="text-center py-16">
				<p class="text-stone-500 dark:text-stone-400 text-lg">
					{$t('public.homepage.profile_setup')}
				</p>
			</div>
		{:else}
			<!-- Dynamic section rendering based on homepage_section_order -->
			<div class="sections-grid">
				{#each effectiveSectionOrder as sectionKey}
					{@const widthClass = getWidthClass(getSectionWidth(sectionKey))}
					{#if isCustomSection(sectionKey)}
						{@const customItem = data.sections?.[sectionKey]?.[0] || customContentMap.get(getCustomContentId(sectionKey))}
						{#if customItem}
							<div class={widthClass}>
								<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
								<CustomContentSection item={customItem as any} layout={getSectionLayout(sectionKey)} />
							</div>
						{/if}
					{:else if sectionKey === 'experience' && data.experience.length > 0}
						<div class={widthClass}>
							<ExperienceSection items={data.experience} layout={getSectionLayout('experience')} />
						</div>
					{:else if sectionKey === 'projects' && data.projects.length > 0}
						<div class={widthClass}>
							<ProjectsSection items={data.projects} layout={getSectionLayout('projects')} viewSlug={data.isDefaultView ? '' : (data.view?.slug || '')} />
						</div>
					{:else if sectionKey === 'education' && data.education.length > 0}
						<div class={widthClass}>
							<EducationSection items={data.education} layout={getSectionLayout('education')} />
						</div>
					{:else if sectionKey === 'certifications' && data.certifications && data.certifications.length > 0}
						<div class={widthClass}>
							<CertificationsSection items={data.certifications} layout={getSectionLayout('certifications')} />
						</div>
					{:else if sectionKey === 'awards' && data.awards && data.awards.length > 0}
						<div class={widthClass}>
							<AwardsSection items={data.awards} layout={getSectionLayout('awards')} />
						</div>
					{:else if sectionKey === 'skills' && data.skills.length > 0}
						<div class={widthClass}>
							<SkillsSection items={data.skills} layout={getSectionLayout('skills')} categoryOrder={getCategoryOrder('skills')} disabledCategories={getDisabledCategories('skills')} categoryDisplayModes={getCategoryDisplayModes('skills')} />
						</div>
					{:else if sectionKey === 'posts' && data.posts && data.posts.length > 0}
						<div class={widthClass}>
							{#if (data.postsTotalCount ?? 0) > data.posts.length}
								<div class="flex items-center justify-between gap-3 mb-4">
									<h2 class="section-title mb-0">{$t('public.sections.posts')}</h2>
									<a
										href="/posts"
										class="inline-flex items-center gap-2 text-sm font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
									>
										{$t('public.homepage.browse_all')}
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
										</svg>
									</a>
								</div>
								<!-- Note: Don't pass viewSlug - we're on root page, back navigation should go to "/" -->
								<PostsSection items={data.posts} layout={getSectionLayout('posts')} viewSlug="" showHeader={false} />
							{:else}
								<!-- Note: Don't pass viewSlug - we're on root page, back navigation should go to "/" -->
								<PostsSection items={data.posts} layout={getSectionLayout('posts')} viewSlug="" />
							{/if}
						</div>
					{:else if sectionKey === 'talks' && data.talks && data.talks.length > 0}
						<div class={widthClass}>
							{#if (data.talksTotalCount ?? 0) > data.talks.length}
								<div class="flex items-center justify-between gap-3 mb-4">
									<h2 class="section-title mb-0">{$t('public.sections.talks')}</h2>
									<a
										href="/talks"
										class="inline-flex items-center gap-2 text-sm font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
									>
										{$t('public.homepage.browse_all')}
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
										</svg>
									</a>
								</div>
								<!-- Note: Don't pass viewSlug - we're on root page, back navigation should go to "/" -->
								<TalksSection items={data.talks} layout={getSectionLayout('talks')} viewSlug="" showHeader={false} />
							{:else}
								<!-- Note: Don't pass viewSlug - we're on root page, back navigation should go to "/" -->
								<TalksSection items={data.talks} layout={getSectionLayout('talks')} viewSlug="" />
							{/if}
						</div>
					{:else if sectionKey === 'courses' && data.courses && data.courses.length > 0}
						<div class={widthClass}>
							<CoursesSection items={data.courses} layout={getSectionLayout('courses') as 'grid-3' | 'grid-2' | 'list' | 'featured'} viewSlug="" />
						</div>
					{:else if sectionKey === 'testimonials' && data.testimonials && data.testimonials.length > 0}
						<div class={widthClass}>
							<TestimonialsSection items={data.testimonials} layout={getSectionLayout('testimonials') as 'wall' | 'carousel' | 'featured'} featuredId={getFeaturedId('testimonials')} />
						</div>
					{:else if sectionKey === 'contacts' && data.contacts && data.contacts.length > 0}
						<div class={widthClass}>
							<ContactMethodsList contacts={data.contacts} layout={getContactLayout()} />
						</div>
					{/if}
				{/each}
			</div>

			<!-- Render any custom content not in the section order (for backwards compatibility) -->
			{#if data.customContent && data.customContent.length > 0 && (!data.homepageSectionOrder || data.homepageSectionOrder.length === 0)}
				{#each data.customContent as item}
					<CustomContentSection {item} layout="default" />
				{/each}
			{/if}
		{/if}
	</main>

	<Footer profile={data.profile} />

	<!-- ATS-optimized hidden content for resume parsing (placed at end to avoid blank first page) -->
	<ATSContent
		profile={data.profile}
		experience={data.experience}
		education={data.education}
		skills={data.skills}
		contacts={data.contacts}
		certifications={data.certifications}
	/>
</div>

<!-- AI Resume Generation Modal -->
{#if showGenerateModal}
	<div
		class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 print:hidden"
		onclick={self(closeGenerateModal)}
		onkeydown={handleModalKeydown}
		role="presentation"
	>
		<div
			bind:this={modalEl}
			role="dialog"
			aria-modal="true"
			aria-labelledby="ai-resume-modal-title"
			class="bg-white dark:bg-stone-800 rounded-lg shadow-xl max-w-md w-full mx-4 overflow-hidden"
		>
			<div class="p-4 border-b border-stone-200 dark:border-stone-700">
				<h2 id="ai-resume-modal-title" class="text-lg font-semibold text-stone-900 dark:text-white">{$t('public.ai_resume.title')}</h2>
				<p class="text-sm text-stone-500 dark:text-stone-400 mt-1">
					{$t('public.ai_resume.description')}
				</p>
			</div>

			<div class="p-4 space-y-4">
				{#if generatedUrl}
					<div class="bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded-lg p-4 text-center">
						<p class="text-green-700 dark:text-green-300 mb-3">{$t('public.ai_resume.success')}</p>
						<a
							href={generatedUrl}
							download
							class="btn btn-primary"
						>
							{$t('public.ai_resume.download')}
						</a>
					</div>
				{:else}
					<div>
						<label for="format" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">{$t('public.ai_resume.format')}</label>
						<select id="format" bind:value={generationConfig.format} class="input">
							<option value="pdf">{$t('public.ai_resume.format_pdf')}</option>
							<option value="docx">{$t('public.ai_resume.format_word')}</option>
						</select>
					</div>

					<div>
						<label for="style" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">{$t('public.ai_resume.style')}</label>
						<select id="style" bind:value={generationConfig.style} class="input">
							<option value="chronological">{$t('public.ai_resume.style_chronological')}</option>
							<option value="functional">{$t('public.ai_resume.style_functional')}</option>
							<option value="hybrid">{$t('public.ai_resume.style_hybrid')}</option>
						</select>
					</div>

					<div>
						<label for="length" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">{$t('public.ai_resume.length')}</label>
						<select id="length" bind:value={generationConfig.length} class="input">
							<option value="one-page">{$t('public.ai_resume.length_one')}</option>
							<option value="two-page">{$t('public.ai_resume.length_two')}</option>
							<option value="full">{$t('public.ai_resume.length_full')}</option>
						</select>
					</div>
				{/if}
			</div>

			<div class="p-4 border-t border-stone-200 dark:border-stone-700 flex justify-end gap-2">
				<button
					type="button"
					class="btn btn-ghost"
					onclick={closeGenerateModal}
				>
					{generatedUrl ? $t('shared.close') : $t('shared.cancel')}
				</button>
				{#if !generatedUrl}
					<button
						type="button"
						class="btn btn-primary"
						onclick={generateResume}
						disabled={generating}
					>
						{#if generating}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" aria-hidden="true">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							{$t('public.ai_resume.generating')}
						{:else}
							{$t('public.ai_resume.generate')}
						{/if}
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}
{/if}


<style>
	/* Section grid layout */
	.sections-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 1.5rem;
	}

	/* Full width: spans all 6 columns */
	:global(.section-full) {
		grid-column: span 6;
	}

	/* Half width: spans 3 columns (50%) */
	:global(.section-half) {
		grid-column: span 3;
	}

	/* Third width: spans 2 columns (33%) */
	:global(.section-third) {
		grid-column: span 2;
	}

	/* Responsive: collapse to full width on mobile */
	@media (max-width: 768px) {
		:global(.section-half),
		:global(.section-third) {
			grid-column: span 6;
		}
	}

	/* Print: allow side-by-side on wider paper */
	@media print {
		.sections-grid {
			gap: 1rem;
		}
	}
</style>

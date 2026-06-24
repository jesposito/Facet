<script lang="ts">
	import { t } from 'svelte-i18n';

	import type { PageData } from './$types';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { navigating, page } from '$app/stores';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
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
	import NewsletterSection from '$components/public/NewsletterSection.svelte';
	import { checkFeature } from '$lib/stores/plan';
	import SiteNav from '$components/public/SiteNav.svelte';
	import ATSContent from '$components/public/ATSContent.svelte';
	import AIResumeModal from '$components/public/AIResumeModal.svelte';
	import { createAIResumeController } from '$lib/ai-resume/controller.svelte';

	// Helper to check if a section key is for custom content
	function isCustomSection(sectionKey: string): boolean {
		return sectionKey.startsWith('custom:');
	}
	import Footer from '$components/public/Footer.svelte';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import ShareButton from '$components/shared/ShareButton.svelte';
	import PasswordPrompt from '$components/public/PasswordPrompt.svelte';
	import { ACCENT_COLORS, type AccentColor, applyPaletteToRoot } from '$lib/colors';
	import { getFontPack, DEFAULT_FONT_PACK } from '$lib/fonts';
	import { pb, getSectionLayout as getDefaultSectionLayout } from '$lib/pocketbase';
	import { generatePersonJsonLd, serializeJsonLd } from '$lib/seo';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	// Generate JSON-LD for SEO (Person schema)
	let personJsonLd = $derived(data.profile ? serializeJsonLd(generatePersonJsonLd(data.profile, browser ? window.location.origin : '')) : null);

	// Track navigation to prevent showing "Not Found" during transitions
	let isNavigating = $derived($navigating !== null);

	// Print menu state
	let showPrintMenu = $state(false);
	let printMenuTriggerEl: HTMLButtonElement | undefined = $state();
	// AI-resume controller owns the modal state, fetch, and download flow.
	const aiResume = createAIResumeController({
		getSlug: () => data.view?.slug,
		getTargetRole: () => data.view?.hero_headline || data.profile?.headline || ''
	});

	// Floating buttons visibility - hide when nav is pinned (sticky)
	let navPinned = $state(false);
	let sentinelEl: HTMLDivElement | null = $state(null);

	// Rail layout: 2-column desktop with a sticky left sidebar. The hero
	// renders as col 1 (the <aside>), page content flows in col 2. Non-rail
	// layouts fall back to display:contents so they render byte-identically.
	let effectiveHeroLayout = $derived(
		(data.view?.hero_layout || data.profile?.hero_layout || 'standard') as
			'standard' | 'centered' | 'split' | 'minimal' | 'stacked' | 'rail'
	);
	let useRailLayout = $derived(effectiveHeroLayout === 'rail');

	// Apply view-specific accent color (or profile default)
	function applyAccentColor(colorName: AccentColor) {
		if (!browser) return;

		const color = ACCENT_COLORS[colorName];
		if (!color) return;

		applyPaletteToRoot(color.scale);
	}

	onMount(() => {
		// In site-mode, use profile accent for visual coherence across Facet tabs.
		// View accent only applies when site nav is off or view has an explicit override.
		const accentColor = (data.siteNavEnabled && !data.view?.accent_color)
			? data.profile?.accent_color
			: (data.view?.accent_color || data.profile?.accent_color);
		if (accentColor) {
			applyAccentColor(accentColor as AccentColor);
		}

		// Apply font pack with same resolution: view override > profile default
		const fontPackName = (data.siteNavEnabled && !data.view?.font_pack)
			? data.profile?.font_pack
			: (data.view?.font_pack || data.profile?.font_pack);
		if (fontPackName && fontPackName !== DEFAULT_FONT_PACK) {
			const pack = getFontPack(fontPackName);
			const root = document.documentElement;
			root.style.setProperty('--font-heading', `'${pack.heading}', ${pack.headingFallback}`);
			root.style.setProperty('--font-body', `'${pack.body}', ${pack.bodyFallback}`);
			root.style.setProperty('--font-code', `'${pack.code}', ${pack.codeFallback}`);
			// Load Google Fonts
			const existing = document.getElementById('view-google-fonts');
			if (existing) existing.remove();
			const link = document.createElement('link');
			link.id = 'view-google-fonts';
			link.rel = 'stylesheet';
			link.href = pack.googleFontsUrl;
			document.head.appendChild(link);
		}

		// Check AI Print availability (always check - API handles auth)
		aiResume.checkStatus();

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

	function closePrintMenu() {
		showPrintMenu = false;
	}

	// Print menu keyboard handler
	function handlePrintMenuKeydown(event: KeyboardEvent) {
		if (!showPrintMenu) return;
		if (event.key === 'Escape') {
			event.preventDefault();
			closePrintMenu();
			printMenuTriggerEl?.focus();
		}
	}

	// Default section order (fallback when no custom order is specified)
	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'awards', 'skills', 'posts', 'talks', 'testimonials', 'contacts'];

	// Compute effective section order: use custom order if provided, otherwise use default
	let effectiveSectionOrder = $derived((data.sectionOrder && data.sectionOrder.length > 0)
		? data.sectionOrder
		: DEFAULT_SECTION_ORDER);

	// Extract custom content items from sections for ProfileNav
	let customContentForNav = $derived.by(() => {
		if (!data.sections) return [];
		return Object.entries(data.sections)
			.filter(([key]) => key.startsWith('custom:'))
			.map(([key, items]) => {
				const item = (items as Array<{ id: string; title: string }>)?.[0];
				if (item) {
					return { id: item.id, title: item.title };
				}
				return null;
			})
			.filter((item): item is { id: string; title: string } => item !== null);
	});

	// Get layout for a section (from API response or default)
	function getSectionLayout(sectionKey: string): string {
		if (data.sectionLayouts?.[sectionKey]) {
			return data.sectionLayouts[sectionKey];
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

	// Get width for a section (from API response or default)
	function getSectionWidth(sectionKey: string): string {
		return data.sectionWidths?.[sectionKey] || 'full';
	}

	function getCategoryOrder(sectionKey: string): string[] | undefined {
		return data.sectionCategoryOrders?.[sectionKey];
	}

	function getDisabledCategories(sectionKey: string): string[] | undefined {
		return data.sectionDisabledCategories?.[sectionKey];
	}

	function getCategoryDisplayModes(sectionKey: string): Record<string, string> | undefined {
		return data.sectionCategoryDisplayModes?.[sectionKey];
	}

	function getFeaturedId(sectionKey: string): string | undefined {
		return data.sectionFeaturedIds?.[sectionKey];
	}

	// Get CSS class for section width (using 6-column grid)
	function getWidthClass(width: string): string {
		switch (width) {
			case 'half': return 'section-half';
			case 'third': return 'section-third';
			default: return 'section-full';
		}
	}

	// Hidden form ref for setting password token cookie
	let passwordForm: HTMLFormElement | undefined = $state();
	let tokenInput: HTMLInputElement | undefined = $state();
	let maxAgeInput: HTMLInputElement | undefined = $state();

	async function handlePasswordVerified(event: CustomEvent<{ token: string; expiresIn: number }>) {
		const { token, expiresIn } = event.detail;

		// Set form values and submit to set cookie via server action
		if (tokenInput) tokenInput.value = token;
		if (maxAgeInput) maxAgeInput.value = String(expiresIn);
		if (passwordForm) passwordForm.requestSubmit();
	}
</script>

<svelte:head>
	<title>{data.view?.name || 'View'} | {data.profile?.name || 'Profile'}</title>
	<meta name="description" content={data.view?.hero_summary || data.profile?.summary || data.view?.hero_headline || data.profile?.headline || ''} />
	<link rel="canonical" href="{$page.data.appUrl || $page.url.origin}/{data.view?.slug}" />
	<!-- JSON-LD structured data for SEO -->
	{#if personJsonLd}
		{@html `<script type="application/ld+json">${personJsonLd}</script>`}
	{/if}
</svelte:head>

<!-- Hidden form for setting password token cookie (must be outside conditional for password flow) -->
<form
	bind:this={passwordForm}
	method="POST"
	action="?/setPasswordToken"
	class="hidden"
	use:enhance={() => {
		return async ({ result }) => {
			if (result.type === 'success') {
				await invalidateAll();
			}
		};
	}}
>
	<input bind:this={tokenInput} type="hidden" name="token" value="" />
	<input bind:this={maxAgeInput} type="hidden" name="maxAge" value="3600" />
</form>

{#if data.requiresPassword}
	<PasswordPrompt
		viewId={data.view?.id || ''}
		on:verified={handlePasswordVerified}
	/>
{:else if !data.view && !isNavigating}
	<!-- Only show "Not Found" when not navigating (prevents flash during transitions) -->
	<div class="min-h-screen flex items-center justify-center">
		<div class="text-center">
			<h1 class="text-4xl font-bold text-stone-900 dark:text-white mb-4">Not Found</h1>
			<p class="text-stone-600 dark:text-stone-400">This page doesn't exist.</p>
			<a href="/" class="mt-4 inline-block btn btn-primary">Go Home</a>
		</div>
	</div>
{:else if data.view}
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
					title={$t('public.aria.print_options')}
					aria-label={$t('public.aria.print_options')}
					aria-expanded={showPrintMenu}
					aria-haspopup="true"
				>
					<svg class="w-5 h-5 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
					</svg>
				</button>

				{#if showPrintMenu}
					<div class="fixed inset-0" onclick={closePrintMenu} onkeydown={handlePrintMenuKeydown} role="presentation" tabindex="-1"></div>
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div class="absolute right-0 mt-2 w-48 bg-white dark:bg-stone-800 rounded-lg shadow-lg border border-stone-200 dark:border-stone-700 py-1 z-50" onkeydown={handlePrintMenuKeydown}>
						<button
							onclick={() => { window.print(); closePrintMenu(); }}
							class="w-full px-4 py-2 text-left text-sm text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-700 flex items-center gap-2"
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
							</svg>
							{$t('public.homepage.simple_print')}
						</button>
						{#if aiResume.status.ai_configured}
							<button
								onclick={() => {
									// Capture the menu trigger BEFORE closing the popover, so the
									// AI-resume dialog can return focus to it on close (SC 2.4.3).
									aiResume.show(printMenuTriggerEl ?? null);
									closePrintMenu();
								}}
								class="w-full px-4 py-2 text-left text-sm text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-700 flex items-center gap-2"
								aria-haspopup="dialog"
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
			<ShareButton 
				url={browser 
					? (data.shareToken ? `${window.location.origin}/s/${data.shareToken}` : window.location.href)
					: (data.shareToken ? `/s/${data.shareToken}` : `/${data.view?.slug || ''}`)}
				title={`${data.profile?.name || 'Profile'} - ${data.view?.name || 'View'}`}
				text={data.view?.hero_headline || data.profile?.headline || ''}
				isUnlisted={!!data.shareToken}
			/>
			{#if pb.authStore.isValid}
				<a
					href="/admin"
					class="p-2 rounded-lg bg-white/80 dark:bg-stone-800/80 backdrop-blur-sm shadow-sm border border-stone-200 dark:border-stone-700 hover:bg-stone-100 dark:hover:bg-stone-700 transition-colors"
					title="Go to Admin"
					aria-label={$t('public.aria.go_to_admin')}
				>
					<svg class="w-5 h-5 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
						<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
					</svg>
				</a>
			{/if}
			<ThemeToggle />
		</div>

		<!-- Site Navigation (above hero) - only renders when position='above' -->
		{#if data.isPublicView}
			<SiteNav
				slot="above"
				ctaUrl={data.view?.cta_url || ''}
				ctaButtonText={data.view?.cta_button_text || 'Learn More'}
				ctaText={data.view?.cta_text || ''}
				ctaEnabled={data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
				ssrNavEnabled={data.siteNav?.enabled}
				ssrNavMode={data.siteNav?.mode}
				ssrNavPosition={data.siteNav?.position}
				ssrNavItems={data.siteNav?.items}
			/>
		{/if}

		<!-- For rail layout, the below-hero SiteNav / CTA banner is lifted ABOVE
		     the rail grid so it spans full width instead of being trapped in
		     the content column. -->
		{#if useRailLayout}
			{#if data.isPublicView}
				<SiteNav
					slot="below"
					ctaUrl={data.view?.cta_url || ''}
					ctaButtonText={data.view?.cta_button_text || 'Learn More'}
					ctaText={data.view?.cta_text || ''}
					ctaEnabled={data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
					ssrNavEnabled={data.siteNav?.enabled}
					ssrNavMode={data.siteNav?.mode}
					ssrNavPosition={data.siteNav?.position}
					ssrNavItems={data.siteNav?.items}
				/>
			{:else if data.view?.cta_text && data.view?.cta_url && data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
				<!-- Fallback CTA for non-public views (no site nav) -->
				<div class="bg-primary-600 text-white py-4">
					<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
						<span class="font-medium">{data.view.cta_text}</span>
						<a
							href={data.view.cta_url}
							target="_blank"
							rel="noopener noreferrer"
							class="btn bg-white text-primary-600 hover:bg-stone-100"
						>
							{data.view.cta_button_text || 'Learn More'}
						</a>
					</div>
				</div>
			{/if}
		{/if}

		<!-- Rail layout: 2-col grid wraps the hero (aside, col 1) + main flow
		     (col 2). Other layouts fall back to display:contents (no structural
		     change, so they render byte-identically to before). -->
		<div class={useRailLayout ? 'md:grid md:grid-cols-[320px_minmax(0,1fr)] md:items-start' : 'contents'}>
		<!-- Modified hero with view overrides -->
		<ProfileHero
			profile={{
				...data.profile,
				headline: data.view?.hero_headline || data.profile?.headline,
				summary: data.view?.hero_summary || data.profile?.summary,
				location: data.view?.hero_location || data.profile?.location,
				hero_image_url: data.view?.hero_image_url || data.profile?.hero_image_url
			}}
			layout={effectiveHeroLayout}
			spacing={(data.view?.hero_spacing || data.profile?.hero_spacing || '') as '' | 'compact' | 'default' | 'spacious'}
			heroBgColor={data.view?.hero_bg_color || data.profile?.hero_bg_color || ''}
			showAvatar={((data as unknown as { showAvatar?: boolean }).showAvatar) !== false}
		/>
		<div class={useRailLayout ? 'min-w-0' : 'contents'}>

		<!-- Site Navigation (below hero) / CTA banner - only renders here for
		     non-rail layouts (rail lifts it above the grid). -->
		{#if !useRailLayout}
			{#if data.isPublicView}
				<SiteNav
					slot="below"
					ctaUrl={data.view?.cta_url || ''}
					ctaButtonText={data.view?.cta_button_text || 'Learn More'}
					ctaText={data.view?.cta_text || ''}
					ctaEnabled={data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
					ssrNavEnabled={data.siteNav?.enabled}
					ssrNavMode={data.siteNav?.mode}
					ssrNavPosition={data.siteNav?.position}
					ssrNavItems={data.siteNav?.items}
				/>
			{:else if data.view?.cta_text && data.view?.cta_url && data.siteCtaEnabled !== false && data.view?.cta_enabled !== false}
				<!-- Fallback CTA for non-public views (no site nav) -->
				<div class="bg-primary-600 text-white py-4">
					<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
						<span class="font-medium">{data.view.cta_text}</span>
						<a
							href={data.view.cta_url}
							target="_blank"
							rel="noopener noreferrer"
							class="btn bg-white text-primary-600 hover:bg-stone-100"
						>
							{data.view.cta_button_text || 'Learn More'}
						</a>
					</div>
				</div>
			{/if}
		{/if}

		<!-- Sentinel to detect when ProfileNav becomes sticky -->
		<div
			aria-hidden="true"
			class="h-px"
			bind:this={sentinelEl}
		></div>

		<ProfileNav
			hasExperience={data.sections?.experience?.length > 0}
			hasProjects={data.sections?.projects?.length > 0}
			hasEducation={data.sections?.education?.length > 0}
			hasCertifications={data.sections?.certifications?.length > 0}
			hasAwards={data.sections?.awards?.length > 0}
			hasSkills={data.sections?.skills?.length > 0}
			hasPosts={data.sections?.posts?.length > 0}
			hasTalks={data.sections?.talks?.length > 0}
			hasCourses={data.sections?.courses?.length > 0}
			hasTestimonials={data.sections?.testimonials?.length > 0}
			hasContacts={data.sections?.contacts?.length > 0}
			hasNewsletter={effectiveSectionOrder.includes('newsletter') && checkFeature('newsletter')}
			viewSlug={data.view?.slug || ''}
			sectionOrder={effectiveSectionOrder}
			customContent={customContentForNav}
		/>

		<main id="main-content" class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
			<div class="sections-grid">
				{#each effectiveSectionOrder as sectionKey}
					{#if sectionKey === 'experience' && data.sections?.experience?.length > 0}
						<div class={getWidthClass(getSectionWidth('experience'))}>
							<ExperienceSection items={data.sections.experience} layout={getSectionLayout('experience')} />
						</div>
					{:else if sectionKey === 'projects' && data.sections?.projects?.length > 0}
						<div class={getWidthClass(getSectionWidth('projects'))}>
							<ProjectsSection
								items={data.sections.projects}
								layout={getSectionLayout('projects')}
								viewSlug={data.view?.slug || ''}
							/>
						</div>
					{:else if sectionKey === 'education' && data.sections?.education?.length > 0}
						<div class={getWidthClass(getSectionWidth('education'))}>
							<EducationSection items={data.sections.education} layout={getSectionLayout('education')} />
						</div>
					{:else if sectionKey === 'certifications' && data.sections?.certifications?.length > 0}
						<div class={getWidthClass(getSectionWidth('certifications'))}>
							<CertificationsSection items={data.sections.certifications} layout={getSectionLayout('certifications')} />
						</div>
					{:else if sectionKey === 'awards' && data.sections?.awards?.length > 0}
						<div class={getWidthClass(getSectionWidth('awards'))}>
							<AwardsSection items={data.sections.awards} layout={getSectionLayout('awards')} />
						</div>
					{:else if sectionKey === 'skills' && data.sections?.skills?.length > 0}
						<div class={getWidthClass(getSectionWidth('skills'))}>
							<SkillsSection items={data.sections.skills} layout={getSectionLayout('skills')} categoryOrder={getCategoryOrder('skills')} disabledCategories={getDisabledCategories('skills')} categoryDisplayModes={getCategoryDisplayModes('skills')} />
						</div>
					{:else if sectionKey === 'posts' && data.sections?.posts?.length > 0}
						<div class={getWidthClass(getSectionWidth('posts'))}>
							{#if (data.postsTotalCount ?? 0) > data.sections.posts.length}
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
								<PostsSection items={data.sections.posts} layout={getSectionLayout('posts')} viewSlug={data.view?.slug || ''} showHeader={false} />
							{:else}
								<PostsSection items={data.sections.posts} layout={getSectionLayout('posts')} viewSlug={data.view?.slug || ''} />
							{/if}
						</div>
					{:else if sectionKey === 'talks' && data.sections?.talks?.length > 0}
						<div class={getWidthClass(getSectionWidth('talks'))}>
							{#if (data.talksTotalCount ?? 0) > data.sections.talks.length}
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
								<TalksSection items={data.sections.talks} layout={getSectionLayout('talks')} viewSlug={data.view?.slug || ''} showHeader={false} />
							{:else}
								<TalksSection items={data.sections.talks} layout={getSectionLayout('talks')} viewSlug={data.view?.slug || ''} />
							{/if}
						</div>
					{:else if sectionKey === 'courses' && data.sections?.courses?.length > 0}
						<div class={getWidthClass(getSectionWidth('courses'))}>
							<CoursesSection items={data.sections.courses} layout={getSectionLayout('courses') as 'grid-3' | 'grid-2' | 'list' | 'featured'} viewSlug={data.view?.slug || ''} />
						</div>
					{:else if sectionKey === 'testimonials' && data.sections?.testimonials?.length > 0}
						<div class={getWidthClass(getSectionWidth('testimonials'))}>
							<TestimonialsSection items={data.sections.testimonials} layout={getSectionLayout('testimonials') as 'wall' | 'carousel' | 'featured'} featuredId={getFeaturedId('testimonials')} />
						</div>
					{:else if sectionKey === 'contacts' && data.sections?.contacts?.length > 0}
						<div class={getWidthClass(getSectionWidth('contacts'))}>
							<ContactMethodsList contacts={data.sections.contacts} viewId={data.view?.id || ''} layout={getContactLayout()} />
						</div>
					{:else if sectionKey === 'newsletter' && checkFeature('newsletter')}
						<div class={getWidthClass(getSectionWidth('newsletter'))}>
							<NewsletterSection viewSlug={data.view?.slug || ''} />
						</div>
					{:else if isCustomSection(sectionKey) && data.sections?.[sectionKey]?.[0]}
						<div class={getWidthClass(getSectionWidth(sectionKey))}>
							<CustomContentSection item={data.sections[sectionKey][0]} layout={getSectionLayout(sectionKey)} />
						</div>
					{/if}
				{/each}
			</div>
		</main>
		</div><!-- /content column (rail col 2) -->
		</div><!-- /rail grid wrapper -->

		<Footer profile={data.profile} footerCtaEnabled={data.footerCtaEnabled} />

		<!-- ATS-optimized hidden content for resume parsing (placed at end to avoid blank first page) -->
		<ATSContent
			profile={data.profile}
			experience={data.sections?.experience}
			education={data.sections?.education}
			skills={data.sections?.skills}
			contacts={data.sections?.contacts}
			certifications={data.sections?.certifications}
		/>
	</div>
{/if}

<!-- AI Resume Generation Modal -->
<AIResumeModal controller={aiResume} />

<style>
	/* Section grid layout (Phase 6.3) */
	.sections-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 1.5rem;
	}

	/* Full width: spans all 6 columns */
	.sections-grid :global(.section-full) {
		grid-column: span 6;
	}

	/* Half width: spans 3 columns (50%) */
	.sections-grid :global(.section-half) {
		grid-column: span 3;
	}

	/* Third width: spans 2 columns (33%) */
	.sections-grid :global(.section-third) {
		grid-column: span 2;
	}

	/* Responsive: collapse to full width on mobile */
	@media (max-width: 768px) {
		.sections-grid :global(.section-half),
		.sections-grid :global(.section-third) {
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

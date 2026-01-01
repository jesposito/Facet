<script lang="ts">
	import type { PageData } from './$types';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import ProfileHero from '$components/public/ProfileHero.svelte';
	import ProfileNav from '$components/public/ProfileNav.svelte';
	import ExperienceSection from '$components/public/ExperienceSection.svelte';
	import ProjectsSection from '$components/public/ProjectsSection.svelte';
	import EducationSection from '$components/public/EducationSection.svelte';
	import CertificationsSection from '$components/public/CertificationsSection.svelte';
	import SkillsSection from '$components/public/SkillsSection.svelte';
	import PostsSection from '$components/public/PostsSection.svelte';
	import TalksSection from '$components/public/TalksSection.svelte';
	import Footer from '$components/public/Footer.svelte';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import { ACCENT_COLORS, type AccentColor } from '$lib/colors';

	export let data: PageData;

	// Get headline and summary - use view overrides if this is a default view
	$: headline = data.view?.hero_headline || data.profile?.headline;
	$: summary = data.view?.hero_summary || data.profile?.summary;

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
	});
</script>

<svelte:head>
	<title>{data.profile?.name || 'Profile'} | Facet</title>
	<meta name="description" content={headline || 'Personal profile and portfolio'} />
	{#if headline}
		<meta property="og:title" content={data.profile?.name} />
		<meta property="og:description" content={headline} />
	{/if}
	<!-- Canonical URL is always / for the homepage -->
	<link rel="canonical" href="/" />
</svelte:head>

<div class="min-h-screen">
	<!-- Theme toggle and print button -->
	<div class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden">
		<!-- Print Button -->
		<button
			on:click={() => window.print()}
			class="p-2 rounded-lg bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm shadow-sm border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
			title="Print / Save as PDF"
			aria-label="Print or save as PDF"
		>
			<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
			</svg>
		</button>
		<ThemeToggle />
	</div>

	<!-- Hero section with possible view overrides -->
	<ProfileHero
		profile={{
			...data.profile,
			headline,
			summary
		}}
	/>

	<!-- CTA banner if this is a view with CTA configured -->
	{#if data.view?.cta_text && data.view?.cta_url}
		<div class="bg-primary-600 text-white py-4">
			<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
				<span class="font-medium">{data.view.cta_text}</span>
				<a
					href={data.view.cta_url}
					target="_blank"
					rel="noopener noreferrer"
					class="btn bg-white text-primary-600 hover:bg-gray-100"
				>
					Learn More
				</a>
			</div>
		</div>
	{/if}

	<!-- Profile navigation tabs -->
	<!-- Note: Don't pass viewSlug here - we're on the root page, so back navigation should go to "/" -->
	<ProfileNav
		hasExperience={data.experience.length > 0}
		hasProjects={data.projects.length > 0}
		hasEducation={data.education.length > 0}
		hasCertifications={data.certifications && data.certifications.length > 0}
		hasSkills={data.skills.length > 0}
		hasPosts={data.posts && data.posts.length > 0}
		hasTalks={data.talks && data.talks.length > 0}
		viewSlug=""
	/>

	<!-- Main content -->
	<main id="main-content" class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		{#if !data.profile?.name && data.experience.length === 0 && data.projects.length === 0}
			<!-- Empty profile state -->
			<div class="text-center py-16">
				<p class="text-gray-500 dark:text-gray-400 text-lg">
					This profile is being set up.
				</p>
			</div>
		{:else}
			{#if data.experience.length > 0}
				<ExperienceSection items={data.experience} />
			{/if}

			{#if data.projects.length > 0}
				<ProjectsSection items={data.projects} />
			{/if}

			{#if data.education.length > 0}
				<EducationSection items={data.education} />
			{/if}

			{#if data.certifications && data.certifications.length > 0}
				<CertificationsSection items={data.certifications} />
			{/if}

			{#if data.skills.length > 0}
				<SkillsSection items={data.skills} />
			{/if}

			{#if data.posts && data.posts.length > 0}
				<!-- Note: Don't pass viewSlug - we're on root page, back navigation should go to "/" -->
				<PostsSection items={data.posts} viewSlug="" />
			{/if}

			{#if data.talks && data.talks.length > 0}
				<!-- Note: Don't pass viewSlug - we're on root page, back navigation should go to "/" -->
				<TalksSection items={data.talks} viewSlug="" />
			{/if}
		{/if}
	</main>

	<Footer profile={data.profile} />
</div>

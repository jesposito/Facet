<script lang="ts">
	import type { PageData } from './$types';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import ProfileHero from '$components/public/ProfileHero.svelte';
	import ExperienceSection from '$components/public/ExperienceSection.svelte';
	import ProjectsSection from '$components/public/ProjectsSection.svelte';
	import EducationSection from '$components/public/EducationSection.svelte';
	import CertificationsSection from '$components/public/CertificationsSection.svelte';
	import SkillsSection from '$components/public/SkillsSection.svelte';
	import PostsSection from '$components/public/PostsSection.svelte';
	import TalksSection from '$components/public/TalksSection.svelte';
	import Footer from '$components/public/Footer.svelte';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import PasswordPrompt from '$components/public/PasswordPrompt.svelte';
	import { ACCENT_COLORS, type AccentColor } from '$lib/colors';

	export let data: PageData;

	// Apply view-specific accent color (or profile default)
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
		// View accent color takes priority over profile accent color
		const accentColor = data.view?.accent_color || data.profile?.accent_color;
		if (accentColor) {
			applyAccentColor(accentColor as AccentColor);
		}
	});

	// Default section order (fallback when no custom order is specified)
	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'skills', 'posts', 'talks'];

	// Compute effective section order: use custom order if provided, otherwise use default
	$: effectiveSectionOrder = (data.sectionOrder && data.sectionOrder.length > 0)
		? data.sectionOrder
		: DEFAULT_SECTION_ORDER;

	// Get layout for a section (from API response or default)
	function getSectionLayout(sectionKey: string): string {
		return data.sectionLayouts?.[sectionKey] || 'default';
	}

	// Get width for a section (from API response or default)
	function getSectionWidth(sectionKey: string): string {
		return data.sectionWidths?.[sectionKey] || 'full';
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
	let passwordForm: HTMLFormElement;
	let tokenInput: HTMLInputElement;
	let maxAgeInput: HTMLInputElement;

	async function handlePasswordVerified(event: CustomEvent<{ token: string; expiresIn: number }>) {
		const { token, expiresIn } = event.detail;

		// Set form values and submit to set cookie via server action
		tokenInput.value = token;
		maxAgeInput.value = String(expiresIn);
		passwordForm.requestSubmit();
	}
</script>

<svelte:head>
	<title>{data.view?.name || 'View'} | {data.profile?.name || 'Profile'}</title>
	<meta name="description" content={data.view?.hero_headline || data.profile?.headline || ''} />
	<!-- Canonical URL is /<slug> -->
	<link rel="canonical" href="/{data.view?.slug}" />
</svelte:head>

<!-- Hidden form for setting password token cookie -->
<form
	bind:this={passwordForm}
	method="POST"
	action="?/setPasswordToken"
	class="hidden"
	use:enhance={() => {
		return async ({ result }) => {
			if (result.type === 'success') {
				// Reload page to fetch data with the new token
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
{:else if !data.view}
	<div class="min-h-screen flex items-center justify-center">
		<div class="text-center">
			<h1 class="text-4xl font-bold text-gray-900 dark:text-white mb-4">Not Found</h1>
			<p class="text-gray-600 dark:text-gray-400">This page doesn't exist.</p>
			<a href="/" class="mt-4 inline-block btn btn-primary">Go Home</a>
		</div>
	</div>
{:else}
	<div class="min-h-screen">
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

		<!-- Modified hero with view overrides -->
		<ProfileHero
			profile={{
				...data.profile,
				headline: data.view?.hero_headline || data.profile?.headline,
				summary: data.view?.hero_summary || data.profile?.summary
			}}
		/>

		<!-- CTA banner if configured -->
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

		<main id="main-content" class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
			<div class="sections-grid">
				{#each effectiveSectionOrder as sectionKey}
					{#if sectionKey === 'experience' && data.sections?.experience?.length > 0}
						<div class={getWidthClass(getSectionWidth('experience'))}>
							<ExperienceSection items={data.sections.experience} layout={getSectionLayout('experience')} />
						</div>
					{:else if sectionKey === 'projects' && data.sections?.projects?.length > 0}
						<div class={getWidthClass(getSectionWidth('projects'))}>
							<ProjectsSection items={data.sections.projects} layout={getSectionLayout('projects')} />
						</div>
					{:else if sectionKey === 'education' && data.sections?.education?.length > 0}
						<div class={getWidthClass(getSectionWidth('education'))}>
							<EducationSection items={data.sections.education} layout={getSectionLayout('education')} />
						</div>
					{:else if sectionKey === 'certifications' && data.sections?.certifications?.length > 0}
						<div class={getWidthClass(getSectionWidth('certifications'))}>
							<CertificationsSection items={data.sections.certifications} layout={getSectionLayout('certifications')} />
						</div>
					{:else if sectionKey === 'skills' && data.sections?.skills?.length > 0}
						<div class={getWidthClass(getSectionWidth('skills'))}>
							<SkillsSection items={data.sections.skills} layout={getSectionLayout('skills')} />
						</div>
					{:else if sectionKey === 'posts' && data.sections?.posts?.length > 0}
						<div class={getWidthClass(getSectionWidth('posts'))}>
							<PostsSection items={data.sections.posts} layout={getSectionLayout('posts')} viewSlug={data.view?.slug || ''} />
						</div>
					{:else if sectionKey === 'talks' && data.sections?.talks?.length > 0}
						<div class={getWidthClass(getSectionWidth('talks'))}>
							<TalksSection items={data.sections.talks} layout={getSectionLayout('talks')} viewSlug={data.view?.slug || ''} />
						</div>
					{/if}
				{/each}
			</div>
		</main>

		<Footer profile={data.profile} />
	</div>
{/if}

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

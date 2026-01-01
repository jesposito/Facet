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
	import { pb } from '$lib/pocketbase';

	export let data: PageData;

	// Print menu state
	let showPrintMenu = false;
	let showGenerateModal = false;
	let generating = false;
	let aiPrintStatus = {
		available: false,
		ai_configured: false,
		pandoc_installed: false
	};
	let generationConfig = {
		format: 'pdf' as 'pdf' | 'docx',
		target_role: '',
		style: 'chronological' as 'chronological' | 'functional' | 'hybrid',
		length: 'two-page' as 'one-page' | 'two-page' | 'full'
	};
	let generatedUrl: string | null = null;

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

		// Check AI Print availability (always check - API handles auth)
		checkAIPrintStatus();
	});

	async function checkAIPrintStatus() {
		try {
			console.log('[AI-PRINT] Checking status, auth valid:', pb.authStore.isValid);
			const response = await fetch('/api/ai-print/status', {
				headers: { Authorization: pb.authStore.token || '' }
			});
			console.log('[AI-PRINT] Status response:', response.status);
			if (response.ok) {
				const result = await response.json();
				console.log('[AI-PRINT] Status result:', result);
				aiPrintStatus = {
					available: result.available,
					ai_configured: result.ai_configured,
					pandoc_installed: result.pandoc_installed
				};
			} else if (response.status === 401) {
				console.log('[AI-PRINT] Not authenticated - AI Resume requires login');
			}
		} catch (err) {
			console.error('[AI-PRINT] Failed to check status:', err);
		}
	}

	async function generateResume() {
		if (!data.view?.slug) return;
		generating = true;
		generatedUrl = null;

		try {
			console.log('[AI-PRINT] Starting generation for:', data.view.slug);
			const response = await fetch(`/api/view/${data.view.slug}/generate`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify(generationConfig)
			});

			const result = await response.json();
			console.log('[AI-PRINT] Generation result:', result);

			if (!response.ok) {
				throw new Error(result.error || 'Generation failed');
			}

			generatedUrl = result.download_url;

			// Auto-download the file
			if (generatedUrl) {
				console.log('[AI-PRINT] Auto-downloading from:', generatedUrl);
				const link = document.createElement('a');
				link.href = generatedUrl;
				link.download = `resume.${generationConfig.format}`;
				document.body.appendChild(link);
				link.click();
				document.body.removeChild(link);
			}
		} catch (err) {
			console.error('[AI-PRINT] Generation error:', err);
			const message = err instanceof Error ? err.message : 'Failed to generate resume';
			alert(message); // Simple alert for public page
		} finally {
			generating = false;
		}
	}

	function closePrintMenu() {
		showPrintMenu = false;
	}

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
			<!-- Print Menu -->
			<div class="relative">
				<button
					on:click={() => showPrintMenu = !showPrintMenu}
					class="p-2 rounded-lg bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm shadow-sm border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
					title="Print options"
					aria-label="Print options"
					aria-expanded={showPrintMenu}
				>
					<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
					</svg>
				</button>

				{#if showPrintMenu}
					<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
					<div class="fixed inset-0" on:click={closePrintMenu}></div>
					<div class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50">
						<button
							on:click={() => { window.print(); closePrintMenu(); }}
							class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
							</svg>
							Simple Print
						</button>
						{#if aiPrintStatus.ai_configured}
							<button
								on:click={() => { showGenerateModal = true; closePrintMenu(); }}
								class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
								</svg>
								AI Resume
							</button>
						{/if}
					</div>
				{/if}
			</div>
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
							<PostsSection items={data.sections.posts} layout={getSectionLayout('posts')} />
						</div>
					{:else if sectionKey === 'talks' && data.sections?.talks?.length > 0}
						<div class={getWidthClass(getSectionWidth('talks'))}>
							<TalksSection items={data.sections.talks} layout={getSectionLayout('talks')} />
						</div>
					{/if}
				{/each}
			</div>
		</main>

		<Footer profile={data.profile} />
	</div>
{/if}

<!-- AI Resume Generation Modal -->
{#if showGenerateModal}
	<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 print:hidden" on:click|self={() => showGenerateModal = false}>
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4 overflow-hidden">
			<div class="p-4 border-b border-gray-200 dark:border-gray-700">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Generate AI Resume</h2>
				<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
					Create a professionally formatted resume from this view.
				</p>
			</div>

			<div class="p-4 space-y-4">
				{#if generatedUrl}
					<div class="bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded-lg p-4 text-center">
						<p class="text-green-700 dark:text-green-300 mb-3">Resume generated successfully!</p>
						<a
							href={generatedUrl}
							download
							class="btn btn-primary"
						>
							Download Resume
						</a>
					</div>
				{:else}
					<div>
						<label for="format" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Format</label>
						<select id="format" bind:value={generationConfig.format} class="input">
							<option value="pdf">PDF</option>
							<option value="docx">Word Document</option>
						</select>
					</div>

					<div>
						<label for="target_role" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Target Role (optional)</label>
						<input
							type="text"
							id="target_role"
							bind:value={generationConfig.target_role}
							class="input"
							placeholder="e.g., Senior Software Engineer"
						/>
					</div>

					<div>
						<label for="style" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Style</label>
						<select id="style" bind:value={generationConfig.style} class="input">
							<option value="chronological">Chronological</option>
							<option value="functional">Functional</option>
							<option value="hybrid">Hybrid</option>
						</select>
					</div>

					<div>
						<label for="length" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Length</label>
						<select id="length" bind:value={generationConfig.length} class="input">
							<option value="one-page">One Page</option>
							<option value="two-page">Two Pages</option>
							<option value="full">Full</option>
						</select>
					</div>
				{/if}
			</div>

			<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
				<button
					type="button"
					class="btn btn-ghost"
					on:click={() => { showGenerateModal = false; generatedUrl = null; }}
				>
					{generatedUrl ? 'Close' : 'Cancel'}
				</button>
				{#if !generatedUrl}
					<button
						type="button"
						class="btn btn-primary"
						on:click={generateResume}
						disabled={generating}
					>
						{#if generating}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Generating...
						{:else}
							Generate
						{/if}
					</button>
				{/if}
			</div>
		</div>
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

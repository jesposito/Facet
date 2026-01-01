<script lang="ts">
	/**
	 * ViewPreview.svelte
	 *
	 * Live preview component for the view editor. Renders a scaled-down version
	 * of the public view using the same section components.
	 *
	 * Features:
	 * - Reuses public section components for consistency
	 * - Applies item overrides from editor state
	 * - Filters items based on selection
	 * - Respects section order and layouts
	 * - Scaled-down for side-by-side editing
	 */
	import type {
		Profile,
		Experience,
		Project,
		Education,
		Certification,
		Skill,
		Post,
		Talk,
		ItemConfig
	} from '$lib/pocketbase';
	import ProfileHero from '$components/public/ProfileHero.svelte';
	import ExperienceSection from '$components/public/ExperienceSection.svelte';
	import ProjectsSection from '$components/public/ProjectsSection.svelte';
	import EducationSection from '$components/public/EducationSection.svelte';
	import CertificationsSection from '$components/public/CertificationsSection.svelte';
	import SkillsSection from '$components/public/SkillsSection.svelte';
	import PostsSection from '$components/public/PostsSection.svelte';
	import TalksSection from '$components/public/TalksSection.svelte';

	// Props from parent editor
	export let profile: Profile | null = null;
	export let heroHeadline: string = '';
	export let heroSummary: string = '';
	export let ctaText: string = '';
	export let ctaUrl: string = '';

	// Section configuration from editor
	export let sections: Record<
		string,
		{
			enabled: boolean;
			items: string[];
			layout: string;
			itemConfig: Record<string, ItemConfig>;
		}
	> = {};

	// Section order from drag-drop
	export let sectionOrder: Array<{ id: string; key: string }> = [];

	// Raw section data (all items)
	export let sectionItems: Record<
		string,
		Array<{
			id: string;
			label: string;
			visibility: string;
			is_draft?: boolean;
			data: Record<string, unknown>;
		}>
	> = {};

	// Compute effective profile with hero overrides
	$: effectiveProfile = profile
		? {
				...profile,
				headline: heroHeadline || profile.headline,
				summary: heroSummary || profile.summary
			}
		: null;

	// Helper to get filtered and transformed items for a section
	function getSectionData<T extends { id: string }>(
		sectionKey: string
	): T[] {
		const config = sections[sectionKey];
		const items = sectionItems[sectionKey] || [];

		if (!config?.enabled || items.length === 0) {
			return [];
		}

		// Filter to selected items (or all public/non-draft if no selection)
		const filteredItems =
			config.items.length > 0
				? items.filter((item) => config.items.includes(item.id))
				: items.filter((item) => item.visibility !== 'private' && !item.is_draft);

		// Preserve order from config.items if specified
		const orderedItems =
			config.items.length > 0
				? config.items
						.map((id) => filteredItems.find((item) => item.id === id))
						.filter(Boolean) as typeof filteredItems
				: filteredItems;

		// Apply overrides to items
		return orderedItems.map((item) => {
			const itemConfig = config.itemConfig?.[item.id];
			const overrides = itemConfig?.overrides || {};

			// Deep clone and apply overrides
			const transformedData = { ...item.data };
			for (const [field, value] of Object.entries(overrides)) {
				if (value !== undefined && value !== null && value !== '') {
					transformedData[field] = value;
				}
			}

			return transformedData as T;
		});
	}

	// Get layout for a section
	function getSectionLayout(sectionKey: string): string {
		return sections[sectionKey]?.layout || 'default';
	}

	// Check if a section should be shown
	function shouldShowSection(sectionKey: string): boolean {
		const config = sections[sectionKey];
		if (!config?.enabled) return false;

		const data = getSectionData(sectionKey);
		return data.length > 0;
	}
</script>

<div class="preview-container">
	<!-- Mini Hero -->
	{#if effectiveProfile}
		<div class="preview-hero">
			<ProfileHero profile={effectiveProfile} />
		</div>
	{:else}
		<div class="preview-hero-placeholder">
			<div class="flex flex-col items-center justify-center py-8 text-gray-400">
				<svg class="w-12 h-12 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
				</svg>
				<span class="text-sm">No profile data</span>
			</div>
		</div>
	{/if}

	<!-- CTA Banner Preview -->
	{#if ctaText && ctaUrl}
		<div class="bg-primary-600 text-white py-2 px-4">
			<div class="flex items-center justify-between text-sm">
				<span class="font-medium truncate">{ctaText}</span>
				<span class="px-2 py-0.5 bg-white/20 rounded text-xs">CTA</span>
			</div>
		</div>
	{/if}

	<!-- Sections Preview -->
	<div class="preview-content">
		{#each sectionOrder as { key: sectionKey } (sectionKey)}
			{#if sectionKey === 'experience' && shouldShowSection('experience')}
				<div class="preview-section">
					<ExperienceSection
						items={getSectionData('experience')}
						layout={getSectionLayout('experience')}
					/>
				</div>
			{:else if sectionKey === 'projects' && shouldShowSection('projects')}
				<div class="preview-section">
					<ProjectsSection
						items={getSectionData('projects')}
						layout={getSectionLayout('projects')}
					/>
				</div>
			{:else if sectionKey === 'education' && shouldShowSection('education')}
				<div class="preview-section">
					<EducationSection
						items={getSectionData('education')}
						layout={getSectionLayout('education')}
					/>
				</div>
			{:else if sectionKey === 'certifications' && shouldShowSection('certifications')}
				<div class="preview-section">
					<CertificationsSection
						items={getSectionData('certifications')}
						layout={getSectionLayout('certifications')}
					/>
				</div>
			{:else if sectionKey === 'skills' && shouldShowSection('skills')}
				<div class="preview-section">
					<SkillsSection
						items={getSectionData('skills')}
						layout={getSectionLayout('skills')}
					/>
				</div>
			{:else if sectionKey === 'posts' && shouldShowSection('posts')}
				<div class="preview-section">
					<PostsSection
						items={getSectionData('posts')}
						layout={getSectionLayout('posts')}
					/>
				</div>
			{:else if sectionKey === 'talks' && shouldShowSection('talks')}
				<div class="preview-section">
					<TalksSection
						items={getSectionData('talks')}
						layout={getSectionLayout('talks')}
					/>
				</div>
			{/if}
		{/each}

		{#if sectionOrder.filter(s => shouldShowSection(s.key)).length === 0}
			<div class="flex flex-col items-center justify-center py-12 text-gray-400">
				<svg class="w-12 h-12 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
				</svg>
				<span class="text-sm">No sections enabled</span>
				<span class="text-xs mt-1">Enable sections in the editor</span>
			</div>
		{/if}
	</div>
</div>

<style>
	.preview-container {
		background: var(--color-bg-primary, white);
		border-radius: 0.5rem;
		overflow: hidden;
		box-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1);
		max-height: calc(100vh - 200px);
		overflow-y: auto;
	}

	:global(.dark) .preview-container {
		background: rgb(17 24 39);
	}

	/* Scale down the hero for preview */
	.preview-hero {
		transform-origin: top left;
	}

	.preview-hero :global(header) {
		padding-top: 2rem;
		padding-bottom: 2rem;
	}

	.preview-hero :global(.py-16) {
		padding-top: 2rem;
		padding-bottom: 2rem;
	}

	.preview-hero :global(.sm\:py-24) {
		padding-top: 2rem;
		padding-bottom: 2rem;
	}

	.preview-hero :global(h1) {
		font-size: 1.5rem;
		line-height: 1.2;
	}

	.preview-hero :global(.sm\:text-4xl),
	.preview-hero :global(.lg\:text-5xl) {
		font-size: 1.5rem;
	}

	.preview-hero :global(.text-xl),
	.preview-hero :global(.sm\:text-2xl) {
		font-size: 1rem;
	}

	.preview-hero :global(.w-32),
	.preview-hero :global(.sm\:w-40) {
		width: 4rem;
		height: 4rem;
	}

	.preview-hero :global(.h-32),
	.preview-hero :global(.sm\:h-40) {
		width: 4rem;
		height: 4rem;
	}

	.preview-hero :global(.prose) {
		font-size: 0.875rem;
	}

	.preview-hero-placeholder {
		background: linear-gradient(to bottom right, rgb(17 24 39), rgb(31 41 55));
		color: rgb(156 163 175);
	}

	.preview-content {
		padding: 1rem;
	}

	/* Scale down sections for preview */
	.preview-section {
		margin-bottom: 1rem;
	}

	.preview-section :global(section) {
		margin-bottom: 0;
	}

	.preview-section :global(.section-title) {
		font-size: 1rem;
		margin-bottom: 0.75rem;
	}

	.preview-section :global(.card) {
		padding: 0.75rem;
	}

	.preview-section :global(h3) {
		font-size: 0.875rem;
	}

	.preview-section :global(p),
	.preview-section :global(span),
	.preview-section :global(li) {
		font-size: 0.75rem;
	}

	.preview-section :global(.space-y-8) {
		gap: 0.75rem;
	}

	.preview-section :global(.gap-4) {
		gap: 0.5rem;
	}

	.preview-section :global(.gap-6) {
		gap: 0.5rem;
	}

	/* Grid layouts */
	.preview-section :global(.grid-cols-3) {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.preview-section :global(.grid-cols-2) {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
</style>

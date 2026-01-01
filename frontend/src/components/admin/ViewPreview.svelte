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

	// Section configuration from editor (with width support for Phase 6.3)
	export let sections: Record<
		string,
		{
			enabled: boolean;
			items: string[];
			layout: string;
			width?: 'full' | 'half' | 'third';
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

	// Reactive computation of all section data - ensures updates when props change
	$: computedSections = computeAllSections(sections, sectionItems);

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	function computeAllSections(
		sectionsConfig: typeof sections,
		items: typeof sectionItems
	): Record<string, {
		data: any[];
		layout: string;
		width: string;
		widthClass: string;
		visible: boolean;
	}> {
		const result: Record<string, {
			data: any[];
			layout: string;
			width: string;
			widthClass: string;
			visible: boolean;
		}> = {};

		for (const key of Object.keys(sectionsConfig)) {
			const config = sectionsConfig[key];
			const sectionItems = items[key] || [];
			const layout = config?.layout || 'default';
			const width = config?.width || 'full';

			let widthClass = 'preview-section section-full';
			if (width === 'half') widthClass = 'preview-section section-half';
			else if (width === 'third') widthClass = 'preview-section section-third';

			let data: unknown[] = [];
			if (config?.enabled && sectionItems.length > 0) {
				// Filter to selected items (or all public/non-draft if no selection)
				const filteredItems =
					config.items.length > 0
						? sectionItems.filter((item) => config.items.includes(item.id))
						: sectionItems.filter((item) => item.visibility !== 'private' && !item.is_draft);

				// Preserve order from config.items if specified
				const orderedItems =
					config.items.length > 0
						? config.items
								.map((id) => filteredItems.find((item) => item.id === id))
								.filter(Boolean) as typeof filteredItems
						: filteredItems;

				// Apply overrides to items
				data = orderedItems.map((item) => {
					const itemConfig = config.itemConfig?.[item.id];
					const overrides = itemConfig?.overrides || {};

					// Deep clone and apply overrides
					const transformedData = { ...item.data };
					for (const [field, value] of Object.entries(overrides)) {
						if (value !== undefined && value !== null && value !== '') {
							transformedData[field] = value;
						}
					}

					return transformedData;
				});
			}

			result[key] = {
				data,
				layout,
				width,
				widthClass,
				visible: config?.enabled && data.length > 0
			};
		}

		return result;
	}

	// Reactive count of visible sections for empty state check
	$: visibleSectionCount = sectionOrder.filter(s => computedSections[s.key]?.visible).length;
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

	<!-- Sections Preview (with grid layout for Phase 6.3) -->
	<div class="preview-content">
		<div class="preview-sections-grid">
			{#each sectionOrder as { key: sectionKey } (sectionKey)}
				{@const computed = computedSections[sectionKey]}
				{#if sectionKey === 'experience' && computed?.visible}
					<div class={computed.widthClass}>
						<ExperienceSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{:else if sectionKey === 'projects' && computed?.visible}
					<div class={computed.widthClass}>
						<ProjectsSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{:else if sectionKey === 'education' && computed?.visible}
					<div class={computed.widthClass}>
						<EducationSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{:else if sectionKey === 'certifications' && computed?.visible}
					<div class={computed.widthClass}>
						<CertificationsSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{:else if sectionKey === 'skills' && computed?.visible}
					<div class={computed.widthClass}>
						<SkillsSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{:else if sectionKey === 'posts' && computed?.visible}
					<div class={computed.widthClass}>
						<PostsSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{:else if sectionKey === 'talks' && computed?.visible}
					<div class={computed.widthClass}>
						<TalksSection
							items={computed.data}
							layout={computed.layout}
						/>
					</div>
				{/if}
			{/each}
		</div>

		{#if visibleSectionCount === 0}
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

	/* Grid layout for section widths (Phase 6.3) */
	.preview-sections-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 0.75rem;
	}

	/* Full width: spans all 6 columns */
	.section-full {
		grid-column: span 6;
	}

	/* Half width: spans 3 columns (50%) */
	.section-half {
		grid-column: span 3;
	}

	/* Third width: spans 2 columns (33%) */
	.section-third {
		grid-column: span 2;
	}

	/* Scale down sections for preview */
	.preview-section {
		margin-bottom: 0;
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

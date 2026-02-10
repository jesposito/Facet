<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';

	interface CustomContentItem {
		id: string;
		title: string;
	}

	interface Props {
		hasExperience?: boolean;
		hasProjects?: boolean;
		hasEducation?: boolean;
		hasCertifications?: boolean;
		hasAwards?: boolean;
		hasSkills?: boolean;
		hasPosts?: boolean;
		hasTalks?: boolean;
		hasTestimonials?: boolean;
		hasContacts?: boolean;
		viewSlug?: string;
		sectionOrder?: string[];
		customContent?: CustomContentItem[];
	}

	let {
		hasExperience = false,
		hasProjects = false,
		hasEducation = false,
		hasCertifications = false,
		hasAwards = false,
		hasSkills = false,
		hasPosts = false,
		hasTalks = false,
		hasTestimonials = false,
		hasContacts = false,
		viewSlug = '',
		sectionOrder = [],
		customContent = []
	}: Props = $props();

	// Build URL with optional from parameter for back navigation
	function buildUrl(path: string): string {
		if (viewSlug) {
			return `${path}?from=${encodeURIComponent(viewSlug)}`;
		}
		return path;
	}

	// Track active section for highlighting
	let activeSection = $state('');

	onMount(() => {
		if (typeof IntersectionObserver !== 'undefined') {
			const sections = document.querySelectorAll('section[id]');

			if (sections.length > 0) {
				activeSection = sections[0].id;
			}

			const observer = new IntersectionObserver(
				(entries) => {
					for (const entry of entries) {
						if (entry.isIntersecting) {
							activeSection = entry.target.id;
							break;
						}
					}
				},
				{ rootMargin: '-5% 0px -70% 0px' }
			);

			sections.forEach((section) => observer.observe(section));

			return () => observer.disconnect();
		}
	});

	interface NavItem {
		id: string;
		label: string;
		href?: string;
		show: boolean;
	}

	// Get translated label for section
	function getSectionLabel(sectionId: string): string {
		return $t(`public.sections.${sectionId}`) || sectionId;
	}

	// Map of section IDs to their display info
	const sectionConfig: Record<string, { href?: string; getShow: () => boolean }> = {
		experience: { getShow: () => hasExperience },
		projects: { getShow: () => hasProjects },
		education: { getShow: () => hasEducation },
		certifications: { getShow: () => hasCertifications },
		awards: { getShow: () => hasAwards },
		skills: { getShow: () => hasSkills },
		posts: { href: buildUrl('/posts'), getShow: () => hasPosts },
		talks: { href: buildUrl('/talks'), getShow: () => hasTalks },
		testimonials: { getShow: () => hasTestimonials },
		contacts: { getShow: () => hasContacts }
	};

	// Default order (used when no sectionOrder provided)
	const DEFAULT_NAV_ORDER = ['experience', 'projects', 'education', 'certifications', 'awards', 'skills', 'posts', 'talks', 'testimonials', 'contacts'];

	// Build nav items respecting the provided section order
	let navItems = $derived.by(() => {
		const order = sectionOrder.length > 0 ? sectionOrder : DEFAULT_NAV_ORDER;
		const items: NavItem[] = [];

		for (const sectionId of order) {
			// Handle custom content sections
			if (sectionId.startsWith('custom:')) {
				const customId = sectionId.replace('custom:', '');
				const customItem = customContent.find(c => c.id === customId);
				if (customItem) {
					items.push({
						id: `custom-${customId}`,
						label: customItem.title,
						show: true
					});
				}
				continue;
			}

			const config = sectionConfig[sectionId];
			if (config && config.getShow()) {
				items.push({
					id: sectionId,
					label: getSectionLabel(sectionId),
					href: config.href,
					show: true
				});
			}
		}

		return items;
	});

	function scrollToSection(id: string) {
		const element = document.getElementById(id);
		if (element) {
			element.scrollIntoView({ behavior: 'smooth', block: 'start' });
		}
	}
</script>

{#if navItems.length > 0}
	<nav class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-30 print:hidden">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex items-center gap-1 py-2 overflow-x-auto scrollbar-hide -mx-4 px-4 sm:mx-0 sm:px-0">
				{#each navItems as item (item.id)}
					{#if item.href}
						<a
							href={item.href}
							class="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-full transition-colors
								{activeSection === item.id
									? 'bg-primary-100 dark:bg-primary-700 text-primary-700 dark:text-white'
									: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700'}"
						>
							{item.label}
							<svg class="inline-block w-3 h-3 ml-0.5 -mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
							</svg>
						</a>
					{:else}
						<button
							type="button"
							onclick={() => scrollToSection(item.id)}
							class="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-full transition-colors
								{activeSection === item.id
									? 'bg-primary-100 dark:bg-primary-700 text-primary-700 dark:text-white'
									: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700'}"
						>
							{item.label}
						</button>
					{/if}
				{/each}
			</div>
		</div>
	</nav>
{/if}

<style>
	.scrollbar-hide {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none;
	}
</style>

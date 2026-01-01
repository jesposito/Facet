<script lang="ts">
	import { onMount } from 'svelte';

	export let hasExperience = false;
	export let hasProjects = false;
	export let hasEducation = false;
	export let hasCertifications = false;
	export let hasSkills = false;
	export let hasPosts = false;
	export let hasTalks = false;

	// Track active section for highlighting
	let activeSection = '';

	onMount(() => {
		// Set up intersection observer for section highlighting
		if (typeof IntersectionObserver !== 'undefined') {
			const sections = document.querySelectorAll('section[id]');
			const observer = new IntersectionObserver(
				(entries) => {
					for (const entry of entries) {
						if (entry.isIntersecting) {
							activeSection = entry.target.id;
							break;
						}
					}
				},
				{ rootMargin: '-20% 0px -60% 0px' }
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

	$: navItems = [
		{ id: 'experience', label: 'Experience', show: hasExperience },
		{ id: 'projects', label: 'Projects', show: hasProjects },
		{ id: 'education', label: 'Education', show: hasEducation },
		{ id: 'certifications', label: 'Certifications', show: hasCertifications },
		{ id: 'skills', label: 'Skills', show: hasSkills },
		{ id: 'posts', label: 'Posts', href: '/posts', show: hasPosts },
		{ id: 'talks', label: 'Talks', href: '/talks', show: hasTalks }
	].filter((item) => item.show) as NavItem[];

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
									? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
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
							on:click={() => scrollToSection(item.id)}
							class="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-full transition-colors
								{activeSection === item.id
									? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
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

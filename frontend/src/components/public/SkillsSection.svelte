<script lang="ts">
	import type { Skill } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';

	interface Props {
		items: Skill[];
		layout?: string;
		categoryOrder?: string[];
		disabledCategories?: string[];
		categoryDisplayModes?: Record<string, string>;
	}

	let { items, layout = 'grouped', categoryOrder, disabledCategories, categoryDisplayModes }: Props = $props();

	// Get the effective layout for a category (per-category override or section default)
	function getCategoryLayout(category: string): string {
		return categoryDisplayModes?.[category] || layout || 'grouped';
	}

	// Filter out skills from disabled categories
	let filteredItems = $derived.by(() => {
		if (!disabledCategories?.length) return items;
		return items.filter(skill => {
			const category = skill.category || 'Other';
			return !disabledCategories.includes(category);
		});
	});

	let groupedSkills = $derived(filteredItems.reduce((acc, skill) => {
		const category = skill.category || 'Other';
		if (!acc[category]) {
			acc[category] = [];
		}
		acc[category].push(skill);
		return acc;
	}, {} as Record<string, Skill[]>));

	let orderedCategories = $derived.by(() => {
		const allCategories = Object.keys(groupedSkills);
		if (!categoryOrder?.length) {
			return allCategories.sort();
		}
		const ordered: string[] = [];
		for (const cat of categoryOrder) {
			if (allCategories.includes(cat)) {
				ordered.push(cat);
			}
		}
		for (const cat of allCategories) {
			if (!ordered.includes(cat)) {
				ordered.push(cat);
			}
		}
		return ordered;
	});

	// Order items by category order for flat/cloud layouts
	let orderedItems = $derived.by(() => {
		const result: Skill[] = [];
		for (const category of orderedCategories) {
			const categorySkills = groupedSkills[category] || [];
			result.push(...categorySkills);
		}
		return result;
	});

	// Editorial chip palette: primary highlight for expert, stone for proficient/familiar
	const proficiencyColors: Record<string, string> = {
		expert: 'bg-primary-50 dark:bg-primary-500/15 text-primary-800 dark:text-primary-300 border border-primary-200/60 dark:border-primary-500/30 font-semibold',
		proficient: 'bg-stone-100 dark:bg-stone-600/30 text-stone-700 dark:text-stone-300 border border-stone-200/60 dark:border-stone-500/30 font-medium',
		familiar: 'bg-stone-50 dark:bg-stone-700/30 text-stone-500 dark:text-stone-400 border border-stone-200/40 dark:border-stone-600/25'
	};

	const proficiencyBarColors: Record<string, string> = {
		expert: 'bg-primary-500 dark:bg-primary-400',
		proficient: 'bg-stone-400 dark:bg-stone-400',
		familiar: 'bg-stone-300 dark:bg-stone-500'
	};

	const proficiencyWidth: Record<string, string> = {
		expert: 'w-full',
		proficient: 'w-3/4',
		familiar: 'w-1/2'
	};

	// Visible proficiency indicator dots (non-color-only signal — WCAG 1.4.1)
	const proficiencyDots: Record<string, number> = {
		expert: 3,
		proficient: 2,
		familiar: 1
	};

	// Tag cloud sizing based on proficiency
	const cloudSizes: Record<string, string> = {
		expert: 'text-xl font-semibold',
		proficient: 'text-base font-medium',
		familiar: 'text-sm'
	};
</script>

<section id="skills" class="mb-16" itemscope itemtype="https://schema.org/Person">
	<h2 class="section-title">{$t('public.sections.skills')}</h2>

	{#if layout === 'cloud'}
		<!-- Tag Cloud Layout -->
		<div class="card p-8 animate-fade-in">
			<div class="flex flex-wrap items-center justify-center gap-x-5 gap-y-3">
				{#each orderedItems as skill, index (skill.id)}
					<span
						class="{cloudSizes[skill.proficiency || 'familiar']} text-stone-700 dark:text-stone-300 hover:text-primary-600 dark:hover:text-primary-400 transition-colors cursor-default animate-fade-in opacity-0"
						style="animation-delay: {index * 20}ms; animation-fill-mode: forwards;"
						title={skill.proficiency ? `${skill.category || 'Skill'} - ${skill.proficiency}` : skill.category}
						itemprop="knowsAbout"
					>
						{skill.name}
					</span>
				{/each}
			</div>
		</div>

	{:else if layout === 'flat'}
		<!-- Flat List Layout -->
		<div class="card p-6 animate-fade-in">
			<div class="flex flex-wrap gap-2">
				{#each orderedItems as skill, index (skill.id)}
					<span
						class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full transition-colors duration-150 {proficiencyColors[skill.proficiency || 'familiar']} animate-fade-in opacity-0"
						style="animation-delay: {index * 20}ms; animation-fill-mode: forwards;"
						title={skill.proficiency ? `${skill.category || 'Skill'} - ${skill.proficiency}` : skill.category}
						itemprop="knowsAbout"
					>
						{skill.name}
						{#if skill.proficiency}
							<span class="inline-flex gap-0.5 ms-0.5" aria-hidden="true">
								{#each Array(proficiencyDots[skill.proficiency] || 1) as _}
									<span class="w-1 h-1 rounded-full bg-current opacity-60"></span>
								{/each}
							</span>
						{/if}
					</span>
				{/each}
			</div>
		</div>

	{:else}
		<!-- Per-category rendering with support for mixed layouts (grouped/bars) -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			{#each orderedCategories as category (category)}
				{@const skills = groupedSkills[category]}
				{@const categoryLayout = getCategoryLayout(category)}

				{#if categoryLayout === 'bars'}
					<!-- Bars layout for this category -->
					<div class="card p-6 animate-fade-in relative overflow-hidden">
						<div class="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-primary-500/60 via-primary-400/40 to-transparent" aria-hidden="true"></div>
						<h3 class="text-lg font-semibold text-stone-900 dark:text-white mb-4">
							{category}
						</h3>
						<div class="space-y-4">
							{#each skills as skill, skillIndex (skill.id)}
								<div class="animate-fade-in opacity-0" style="animation-delay: {skillIndex * 40}ms; animation-fill-mode: forwards;">
									<div class="flex items-center justify-between mb-1">
										<span class="text-sm font-medium text-stone-700 dark:text-stone-300" itemprop="knowsAbout">
											{skill.name}
										</span>
										{#if skill.proficiency}
											<span class="text-xs text-stone-500 dark:text-stone-400 capitalize">
												{skill.proficiency}
											</span>
										{/if}
									</div>
									<div class="w-full h-2 bg-stone-200 dark:bg-stone-700 rounded-full overflow-hidden">
										<div
											class="h-full rounded-full transition-all duration-500 {proficiencyBarColors[skill.proficiency || 'familiar']} {proficiencyWidth[skill.proficiency || 'familiar']}"
										></div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{:else}
					<!-- Grouped (badges) layout for this category (default) -->
					<div class="card p-6 animate-fade-in relative overflow-hidden">
						<div class="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-primary-500/60 via-primary-400/40 to-transparent" aria-hidden="true"></div>
						<h3 class="text-lg font-semibold text-stone-900 dark:text-white mb-4">
							{category}
						</h3>
						<div class="flex flex-wrap gap-2">
							{#each skills as skill, skillIndex (skill.id)}
								<span
									class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full transition-colors duration-150 {proficiencyColors[skill.proficiency || 'familiar']} animate-fade-in opacity-0"
									style="animation-delay: {skillIndex * 20}ms; animation-fill-mode: forwards;"
									title={skill.proficiency ? `Proficiency: ${skill.proficiency}` : undefined}
									itemprop="knowsAbout"
								>
									{skill.name}
									{#if skill.proficiency}
										<span class="inline-flex gap-0.5 ms-0.5" aria-hidden="true">
											{#each Array(proficiencyDots[skill.proficiency] || 1) as _}
												<span class="w-1 h-1 rounded-full bg-current opacity-60"></span>
											{/each}
										</span>
									{/if}
								</span>
							{/each}
						</div>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</section>

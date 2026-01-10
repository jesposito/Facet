<script lang="ts">
	import type { Skill } from '$lib/pocketbase';

	interface Props {
		items: Skill[];
		layout?: string;
	}

	let { items, layout = 'grouped' }: Props = $props();

	// Group skills by category
	let groupedSkills = $derived(items.reduce((acc, skill) => {
		const category = skill.category || 'Other';
		if (!acc[category]) {
			acc[category] = [];
		}
		acc[category].push(skill);
		return acc;
	}, {} as Record<string, Skill[]>));

	const proficiencyColors = {
		expert: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
		proficient: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
		familiar: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
	};

	const proficiencyBarColors = {
		expert: 'bg-green-500 dark:bg-green-400',
		proficient: 'bg-blue-500 dark:bg-blue-400',
		familiar: 'bg-gray-400 dark:bg-gray-500'
	};

	const proficiencyWidth = {
		expert: 'w-full',
		proficient: 'w-3/4',
		familiar: 'w-1/2'
	};

	// Tag cloud sizing based on proficiency
	const cloudSizes = {
		expert: 'text-xl font-semibold',
		proficient: 'text-base font-medium',
		familiar: 'text-sm'
	};
</script>

<section id="skills" class="mb-16">
	<h2 class="section-title">Skills</h2>

	{#if layout === 'cloud'}
		<!-- Tag Cloud Layout -->
		<div class="card p-8 animate-fade-in">
			<div class="flex flex-wrap items-center justify-center gap-x-4 gap-y-3">
				{#each items as skill (skill.id)}
					<span
						class="{cloudSizes[skill.proficiency || 'familiar']} text-gray-700 dark:text-gray-300 hover:text-primary-600 dark:hover:text-primary-400 transition-colors cursor-default"
						title={skill.proficiency ? `${skill.category || 'Skill'} - ${skill.proficiency}` : skill.category}
					>
						{skill.name}
					</span>
				{/each}
			</div>
		</div>

	{:else if layout === 'bars'}
		<!-- Skill Bars Layout -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			{#each Object.entries(groupedSkills) as [category, skills] (category)}
				<div class="card p-6 animate-fade-in">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						{category}
					</h3>
					<div class="space-y-4">
						{#each skills as skill (skill.id)}
							<div>
								<div class="flex items-center justify-between mb-1">
									<span class="text-sm font-medium text-gray-700 dark:text-gray-300">
										{skill.name}
									</span>
									{#if skill.proficiency}
										<span class="text-xs text-gray-500 dark:text-gray-400 capitalize">
											{skill.proficiency}
										</span>
									{/if}
								</div>
								<div class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
									<div
										class="h-full rounded-full transition-all duration-500 {proficiencyBarColors[skill.proficiency || 'familiar']} {proficiencyWidth[skill.proficiency || 'familiar']}"
									></div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>

	{:else if layout === 'flat'}
		<!-- Flat List Layout -->
		<div class="card p-6 animate-fade-in">
			<div class="flex flex-wrap gap-2">
				{#each items as skill (skill.id)}
					<span
						class="px-3 py-1.5 text-sm rounded-full {proficiencyColors[skill.proficiency || 'familiar']}"
						title={skill.proficiency ? `${skill.category || 'Skill'} - ${skill.proficiency}` : skill.category}
					>
						{skill.name}
					</span>
				{/each}
			</div>
		</div>

	{:else}
		<!-- Default: Grouped Layout -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			{#each Object.entries(groupedSkills) as [category, skills] (category)}
				<div class="card p-6 animate-fade-in">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						{category}
					</h3>
					<div class="flex flex-wrap gap-2">
						{#each skills as skill (skill.id)}
							<span
								class="px-3 py-1.5 text-sm rounded-full {proficiencyColors[skill.proficiency || 'familiar']}"
								title={skill.proficiency ? `Proficiency: ${skill.proficiency}` : undefined}
							>
								{skill.name}
							</span>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>

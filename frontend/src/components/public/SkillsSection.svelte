<script lang="ts">
	import type { Skill } from '$lib/pocketbase';

	export let items: Skill[];

	// Group skills by category
	$: groupedSkills = items.reduce((acc, skill) => {
		const category = skill.category || 'Other';
		if (!acc[category]) {
			acc[category] = [];
		}
		acc[category].push(skill);
		return acc;
	}, {} as Record<string, Skill[]>);

	const proficiencyColors = {
		expert: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
		proficient: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
		familiar: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
	};
</script>

<section id="skills" class="mb-16">
	<h2 class="section-title">Skills</h2>

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
</section>

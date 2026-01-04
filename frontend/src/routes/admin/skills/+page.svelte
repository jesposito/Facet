<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Skill } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';

	let skills: Skill[] = [];
	let loading = true;
	let showForm = false;
	let editingSkill: Skill | null = null;

	// Form fields
	let name = '';
	let category = '';
	let proficiency: 'expert' | 'proficient' | 'familiar' = 'proficient';
	let visibility = 'public';
	let sortOrder = 0;
	let saving = false;

	// Available categories (derived from existing skills)
	let availableCategories: string[] = [];

	onMount(loadSkills);

	async function loadSkills() {
		loading = true;
		try {
			const records = await await collection('skills').getList(1, 200, {
				sort: 'category,sort_order,name'
			});
			skills = records.items as unknown as Skill[];

			// Extract unique categories
			availableCategories = [...new Set(skills.map(s => s.category).filter(Boolean))] as string[];
		} catch (err) {
			console.error('Failed to load skills:', err);
			toasts.add('error', 'Failed to load skills');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		name = '';
		category = '';
		proficiency = 'proficient';
		visibility = 'public';
		sortOrder = 0;
		editingSkill = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(skill: Skill) {
		editingSkill = skill;
		name = skill.name;
		category = skill.category || '';
		proficiency = skill.proficiency || 'proficient';
		visibility = skill.visibility;
		sortOrder = skill.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	async function handleSubmit() {
		if (!name.trim()) {
			toasts.add('error', 'Skill name is required');
			return;
		}

		saving = true;
		try {
			const data = {
				name: name.trim(),
				category: category.trim(),
				proficiency,
				visibility,
				sort_order: sortOrder
			};

			if (editingSkill) {
				await await collection('skills').update(editingSkill.id, data);
				toasts.add('success', 'Skill updated successfully');
			} else {
				await await collection('skills').create(data);
				toasts.add('success', 'Skill created successfully');
			}

			closeForm();
			await loadSkills();
		} catch (err) {
			console.error('Failed to save skill:', err);
			toasts.add('error', 'Failed to save skill');
		} finally {
			saving = false;
		}
	}

	async function deleteSkill(skill: Skill) {
		if (!confirm(`Are you sure you want to delete "${skill.name}"?`)) {
			return;
		}

		try {
			await await collection('skills').delete(skill.id);
			toasts.add('success', 'Skill deleted');
			await loadSkills();
		} catch (err) {
			console.error('Failed to delete skill:', err);
			toasts.add('error', 'Failed to delete skill');
		}
	}

	// Group skills by category
	function groupByCategory(skillList: Skill[]): Map<string, Skill[]> {
		const groups = new Map<string, Skill[]>();
		for (const skill of skillList) {
			const categoryKey = skill.category || 'Other';
			if (!groups.has(categoryKey)) {
				groups.set(categoryKey, []);
			}
			groups.get(categoryKey)!.push(skill);
		}
		return groups;
	}

	function getProficiencyColor(level: string | undefined): string {
		switch (level) {
			case 'expert':
				return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200';
			case 'proficient':
				return 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200';
			case 'familiar':
				return 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400';
			default:
				return 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400';
		}
	}

	function getProficiencyLabel(level: string | undefined): string {
		switch (level) {
			case 'expert':
				return 'Expert';
			case 'proficient':
				return 'Proficient';
			case 'familiar':
				return 'Familiar';
			default:
				return '';
		}
	}

	$: groupedSkills = groupByCategory(skills);

	// Default categories to suggest
	const suggestedCategories = [
		'Languages',
		'Frameworks',
		'Databases',
		'DevOps',
		'Cloud',
		'Tools',
		'Soft Skills'
	];

	$: categoryOptions = [...new Set([...suggestedCategories, ...availableCategories])].sort();
</script>

<svelte:head>
	<title>Skills | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Skills</h1>
		<button class="btn btn-primary" on:click={openNewForm}>
			+ New Skill
		</button>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading skills...</div>
		</div>
	{:else if showForm}
		<!-- Skill Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingSkill ? 'Edit Skill' : 'New Skill'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" on:click={closeForm}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="name" class="label">Skill Name *</label>
					<input
						type="text"
						id="name"
						bind:value={name}
						class="input"
						placeholder="Python"
						required
					/>
				</div>

				<div>
					<label for="category" class="label">Category</label>
					<input
						type="text"
						id="category"
						bind:value={category}
						list="category-options"
						class="input"
						placeholder="Languages"
					/>
					<datalist id="category-options">
						{#each categoryOptions as cat}
							<option value={cat} />
						{/each}
					</datalist>
					<p class="text-xs text-gray-500 mt-1">Group related skills together</p>
				</div>

				<div>
					<label for="proficiency" class="label">Proficiency Level</label>
					<select id="proficiency" bind:value={proficiency} class="input">
						<option value="expert">Expert - Deep expertise, can teach others</option>
						<option value="proficient">Proficient - Strong working knowledge</option>
						<option value="familiar">Familiar - Basic understanding</option>
					</select>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="visibility" class="label">Visibility</label>
						<select id="visibility" bind:value={visibility} class="input">
							<option value="public">Public</option>
							<option value="unlisted">Unlisted</option>
							<option value="private">Private</option>
						</select>
					</div>

					<div>
						<label for="sort_order" class="label">Sort Order</label>
						<input
							type="number"
							id="sort_order"
							bind:value={sortOrder}
							class="input"
							min="0"
						/>
						<p class="text-xs text-gray-500 mt-1">Higher numbers appear first in category</p>
					</div>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<button type="button" class="btn btn-secondary" on:click={closeForm}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingSkill ? 'Update Skill' : 'Create Skill'}
				</button>
			</div>
		</form>
	{:else if skills.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No skills added yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your technical and professional skills, organized by category.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>
				+ Add Your First Skill
			</button>
		</div>
	{:else}
		<!-- Skills List - Grouped by Category -->
		<div class="space-y-6">
			{#each [...groupedSkills] as [categoryName, categorySkills] (categoryName)}
				<div class="card p-4">
					<h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-3">
						{categoryName}
					</h2>
					<div class="flex flex-wrap gap-2">
						{#each categorySkills as skill (skill.id)}
							<div class="group relative inline-flex items-center gap-1 px-3 py-1.5 bg-gray-100 dark:bg-gray-700 rounded-lg text-sm">
								<span class="font-medium text-gray-800 dark:text-gray-200">{skill.name}</span>
								{#if skill.proficiency}
									<span class="px-1.5 py-0.5 text-xs rounded {getProficiencyColor(skill.proficiency)}">
										{getProficiencyLabel(skill.proficiency)}
									</span>
								{/if}
								{#if skill.visibility !== 'public'}
									<span class="px-1.5 py-0.5 text-xs rounded bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200">
										{skill.visibility}
									</span>
								{/if}

								<!-- Action buttons on hover -->
								<div class="hidden group-hover:flex items-center gap-1 ml-1 pl-1 border-l border-gray-300 dark:border-gray-600">
									<button
										class="p-1 text-gray-500 hover:text-blue-600 rounded"
										on:click={() => openEditForm(skill)}
										title="Edit"
									>
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
										</svg>
									</button>
									<button
										class="p-1 text-gray-500 hover:text-red-600 rounded"
										on:click={() => deleteSkill(skill)}
										title="Delete"
									>
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
										</svg>
									</button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>

		<!-- Legend -->
		<div class="mt-6 card p-4">
			<h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Proficiency Levels</h3>
			<div class="flex flex-wrap gap-3 text-xs">
				<div class="flex items-center gap-1">
					<span class="px-2 py-0.5 rounded {getProficiencyColor('expert')}">Expert</span>
					<span class="text-gray-500">Deep expertise</span>
				</div>
				<div class="flex items-center gap-1">
					<span class="px-2 py-0.5 rounded {getProficiencyColor('proficient')}">Proficient</span>
					<span class="text-gray-500">Strong knowledge</span>
				</div>
				<div class="flex items-center gap-1">
					<span class="px-2 py-0.5 rounded {getProficiencyColor('familiar')}">Familiar</span>
					<span class="text-gray-500">Basic understanding</span>
				</div>
			</div>
		</div>
	{/if}
</div>

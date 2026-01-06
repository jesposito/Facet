<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Experience } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import { formatDate } from '$lib/utils';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';

	let experiences: Experience[] = [];
	let loading = true;
	let showForm = false;
	let editingExp: Experience | null = null;

	let selectMode = false;
	let selectedIds: Set<string> = new Set();

	// Form fields
	let company = '';
	let title = '';
	let location = '';
	let startDate = '';
	let endDate = '';
	let description = '';
	let bullets: string[] = [];
	let bulletsText = '';
	let skills: string[] = [];
	let skillsText = '';
	let visibility = 'public';
	let isDraft = false;
	let sortOrder = 0;
	let saving = false;

	onMount(loadExperiences);

	async function loadExperiences() {
		loading = true;
		try {
			const records = await await collection('experience').getList(1, 100, {
				sort: '-sort_order,-start_date'
			});
			experiences = records.items as unknown as Experience[];
		} catch (err) {
			console.error('Failed to load experiences:', err);
			toasts.add('error', 'Failed to load experiences');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		company = '';
		title = '';
		location = '';
		startDate = '';
		endDate = '';
		description = '';
		bullets = [];
		bulletsText = '';
		skills = [];
		skillsText = '';
		visibility = 'public';
		isDraft = false;
		sortOrder = 0;
		editingExp = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(exp: Experience) {
		editingExp = exp;
		company = exp.company;
		title = exp.title;
		location = exp.location || '';
		startDate = exp.start_date ? exp.start_date.split('T')[0] : '';
		endDate = exp.end_date ? exp.end_date.split('T')[0] : '';
		description = exp.description || '';
		bullets = exp.bullets || [];
		bulletsText = bullets.join('\n');
		skills = exp.skills || [];
		skillsText = skills.join(', ');
		visibility = exp.visibility;
		isDraft = exp.is_draft;
		sortOrder = exp.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	async function handleSubmit() {
		if (!company.trim() || !title.trim()) {
			toasts.add('error', 'Company and title are required');
			return;
		}

		saving = true;
		try {
			// Parse bullets from text (one per line)
			const parsedBullets = bulletsText
				.split('\n')
				.map((b) => b.trim())
				.filter((b) => b);

			// Parse skills from comma-separated
			const parsedSkills = skillsText
				.split(',')
				.map((s) => s.trim())
				.filter((s) => s);

			const data = {
				company: company.trim(),
				title: title.trim(),
				location: location.trim(),
				start_date: startDate ? new Date(startDate).toISOString() : null,
				end_date: endDate ? new Date(endDate).toISOString() : null,
				description: description.trim(),
				bullets: parsedBullets,
				skills: parsedSkills,
				visibility,
				is_draft: isDraft,
				sort_order: sortOrder
			};

			if (editingExp) {
				await await collection('experience').update(editingExp.id, data);
				toasts.add('success', 'Experience updated successfully');
			} else {
				await await collection('experience').create(data);
				toasts.add('success', 'Experience created successfully');
			}

			closeForm();
			await loadExperiences();
		} catch (err) {
			console.error('Failed to save experience:', err);
			toasts.add('error', 'Failed to save experience');
		} finally {
			saving = false;
		}
	}

	async function deleteExperience(exp: Experience) {
		if (!confirm(`Are you sure you want to delete "${exp.title} at ${exp.company}"?`)) {
			return;
		}

		try {
			await await collection('experience').delete(exp.id);
			toasts.add('success', 'Experience deleted');
			await loadExperiences();
		} catch (err) {
			console.error('Failed to delete experience:', err);
			toasts.add('error', 'Failed to delete experience');
		}
	}

	async function togglePublish(exp: Experience) {
		try {
			await await collection('experience').update(exp.id, {
				is_draft: !exp.is_draft
			});
			toasts.add('success', exp.is_draft ? 'Experience published' : 'Experience unpublished');
			await loadExperiences();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update experience');
		}
	}

	function formatDateRange(start: string | undefined, end: string | undefined): string {
		if (!start) return '';
		const startStr = new Date(start).toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
		if (!end) return `${startStr} - Present`;
		const endStr = new Date(end).toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
		return `${startStr} - ${endStr}`;
	}

	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		if (selectedIds.has(id)) {
			selectedIds.delete(id);
		} else {
			selectedIds.add(id);
		}
		selectedIds = selectedIds;
	}

	function selectAll() {
		selectedIds = new Set(experiences.map(e => e.id));
	}

	function clearSelection() {
		selectedIds = new Set();
	}

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) {
				await collection('experience').update(id, { visibility });
			}
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadExperiences();
		} catch (err) {
			console.error('Failed to update visibility:', err);
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		if (!confirm(`Are you sure you want to delete ${ids.length} experience(s)?`)) return;
		
		try {
			for (const id of ids) {
				await collection('experience').delete(id);
			}
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadExperiences();
		} catch (err) {
			console.error('Failed to delete:', err);
			toasts.add('error', 'Failed to delete items');
		}
	}
</script>

<svelte:head>
	<title>Experience | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={experiences.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Experience</h1>
		<div class="flex items-center gap-2">
			{#if experiences.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					on:click={toggleSelectMode}
				>
					{selectMode ? 'Cancel' : 'Select'}
				</button>
			{/if}
			<button class="btn btn-primary" on:click={openNewForm}>
				+ New Experience
			</button>
		</div>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading experiences...</div>
		</div>
	{:else if showForm}
		<!-- Experience Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingExp ? 'Edit Experience' : 'New Experience'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" on:click={closeForm}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="company" class="label">Company *</label>
						<input
							type="text"
							id="company"
							bind:value={company}
							class="input"
							placeholder="Acme Inc."
							required
						/>
					</div>

					<div>
						<label for="title" class="label">Job Title *</label>
						<input
							type="text"
							id="title"
							bind:value={title}
							class="input"
							placeholder="Senior Software Engineer"
							required
						/>
					</div>
				</div>

				<div>
					<label for="location" class="label">Location</label>
					<input
						type="text"
						id="location"
						bind:value={location}
						class="input"
						placeholder="San Francisco, CA"
					/>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="start_date" class="label">Start Date</label>
						<input
							type="date"
							id="start_date"
							bind:value={startDate}
							class="input"
						/>
					</div>

					<div>
						<label for="end_date" class="label">End Date</label>
						<input
							type="date"
							id="end_date"
							bind:value={endDate}
							class="input"
						/>
						<p class="text-xs text-gray-500 mt-1">Leave blank for current position</p>
					</div>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Details</h2>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="description" class="label mb-0">Description</label>
						<AIContentHelper
							fieldType="description"
							content={description}
							context={{ role: title, company, location }}
							on:apply={(e) => (description = e.detail.content)}
						/>
					</div>
					<textarea
						id="description"
						bind:value={description}
						class="input min-h-[100px]"
						placeholder="Brief overview of your role and responsibilities..."
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown supported</p>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="bullets" class="label mb-0">Key Achievements</label>
						<AIContentHelper
							fieldType="bullets"
							content={bulletsText}
							context={{ role: title, company, description }}
							on:apply={(e) => (bulletsText = e.detail.content)}
						/>
					</div>
					<textarea
						id="bullets"
						bind:value={bulletsText}
						class="input min-h-[120px]"
						placeholder="Led migration to microservices architecture&#10;Reduced API response times by 40%&#10;Mentored junior developers"
					></textarea>
					<p class="text-xs text-gray-500 mt-1">One achievement per line. Use AI Assistant for rewriting or feedback!</p>
				</div>

				<div>
					<label for="skills" class="label">Technologies & Skills</label>
					<input
						type="text"
						id="skills"
						bind:value={skillsText}
						class="input"
						placeholder="Go, Docker, Kubernetes, PostgreSQL"
					/>
					<p class="text-xs text-gray-500 mt-1">Comma-separated list</p>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Settings</h2>

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
						<p class="text-xs text-gray-500 mt-1">Higher numbers appear first</p>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<input
						type="checkbox"
						id="is_draft"
						bind:checked={isDraft}
						class="w-4 h-4 text-primary-600 rounded border-gray-300"
					/>
					<label for="is_draft" class="text-sm text-gray-700 dark:text-gray-300">
						Save as draft (won't be visible publicly)
					</label>
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
					{editingExp ? 'Update Experience' : 'Create Experience'}
				</button>
			</div>
		</form>
	{:else if experiences.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No experience added yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your work history, including positions, achievements, and skills used.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>
				+ Add Your First Experience
			</button>
		</div>
	{:else}
		<!-- Experience List -->
		<div class="space-y-4">
			{#each experiences as exp (exp.id)}
				<div class="card p-4 {selectMode && selectedIds.has(exp.id) ? 'ring-2 ring-primary-500' : ''}">
					<div class="flex items-start justify-between gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(exp.id)}
								on:change={() => toggleSelect(exp.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-gray-900 dark:text-white">
									{exp.title}
								</h3>
								<span class="text-gray-500 dark:text-gray-400">at</span>
								<span class="font-medium text-gray-800 dark:text-gray-200">{exp.company}</span>
								{#if exp.is_draft}
									<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
										Draft
									</span>
								{/if}
								{#if exp.visibility !== 'public'}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{exp.visibility}
									</span>
								{/if}
								{#if !exp.end_date}
									<span class="px-2 py-0.5 text-xs bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded">
										Current
									</span>
								{/if}
							</div>

							<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
								{#if exp.location}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
										</svg>
										{exp.location}
									</span>
								{/if}
								{#if exp.start_date}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
										</svg>
										{formatDateRange(exp.start_date, exp.end_date)}
									</span>
								{/if}
							</div>

							{#if exp.skills && exp.skills.length > 0}
								<div class="flex flex-wrap gap-1 mt-2">
									{#each exp.skills.slice(0, 5) as skill}
										<span class="px-2 py-0.5 text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded">
											{skill}
										</span>
									{/each}
									{#if exp.skills.length > 5}
										<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
											+{exp.skills.length - 5} more
										</span>
									{/if}
								</div>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<button
								class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
								on:click={() => togglePublish(exp)}
								title={exp.is_draft ? 'Publish' : 'Unpublish'}
							>
								{#if exp.is_draft}
									<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
									</svg>
								{:else}
									<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.542 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
									</svg>
								{/if}
							</button>
							<button
								class="p-2 text-gray-500 hover:text-blue-600"
								on:click={() => openEditForm(exp)}
								title="Edit"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="p-2 text-gray-500 hover:text-red-600"
								on:click={() => deleteExperience(exp)}
								title="Delete"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
								</svg>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

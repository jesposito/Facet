<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Education } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';

	let educations: Education[] = [];
	let loading = true;
	let showForm = false;
	let editingEdu: Education | null = null;

	// Form fields
	let institution = '';
	let degree = '';
	let field = '';
	let startDate = '';
	let endDate = '';
	let description = '';
	let visibility = 'public';
	let isDraft = false;
	let sortOrder = 0;
	let saving = false;

	onMount(loadEducations);

	async function loadEducations() {
		loading = true;
		try {
			const records = await pb.collection('education').getList(1, 100, {
				sort: '-sort_order,-end_date'
			});
			educations = records.items as unknown as Education[];
		} catch (err) {
			console.error('Failed to load education:', err);
			toasts.add('error', 'Failed to load education');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		institution = '';
		degree = '';
		field = '';
		startDate = '';
		endDate = '';
		description = '';
		visibility = 'public';
		isDraft = false;
		sortOrder = 0;
		editingEdu = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(edu: Education) {
		editingEdu = edu;
		institution = edu.institution;
		degree = edu.degree || '';
		field = edu.field || '';
		startDate = edu.start_date ? edu.start_date.split('T')[0] : '';
		endDate = edu.end_date ? edu.end_date.split('T')[0] : '';
		description = edu.description || '';
		visibility = edu.visibility;
		isDraft = edu.is_draft;
		sortOrder = edu.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	async function handleSubmit() {
		if (!institution.trim()) {
			toasts.add('error', 'Institution name is required');
			return;
		}

		saving = true;
		try {
			const data = {
				institution: institution.trim(),
				degree: degree.trim(),
				field: field.trim(),
				start_date: startDate ? new Date(startDate).toISOString() : null,
				end_date: endDate ? new Date(endDate).toISOString() : null,
				description: description.trim(),
				visibility,
				is_draft: isDraft,
				sort_order: sortOrder
			};

			if (editingEdu) {
				await pb.collection('education').update(editingEdu.id, data);
				toasts.add('success', 'Education updated successfully');
			} else {
				await pb.collection('education').create(data);
				toasts.add('success', 'Education created successfully');
			}

			closeForm();
			await loadEducations();
		} catch (err) {
			console.error('Failed to save education:', err);
			toasts.add('error', 'Failed to save education');
		} finally {
			saving = false;
		}
	}

	async function deleteEducation(edu: Education) {
		if (!confirm(`Are you sure you want to delete "${edu.degree} at ${edu.institution}"?`)) {
			return;
		}

		try {
			await pb.collection('education').delete(edu.id);
			toasts.add('success', 'Education deleted');
			await loadEducations();
		} catch (err) {
			console.error('Failed to delete education:', err);
			toasts.add('error', 'Failed to delete education');
		}
	}

	async function togglePublish(edu: Education) {
		try {
			await pb.collection('education').update(edu.id, {
				is_draft: !edu.is_draft
			});
			toasts.add('success', edu.is_draft ? 'Education published' : 'Education unpublished');
			await loadEducations();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update education');
		}
	}

	function formatDateRange(start: string | undefined, end: string | undefined): string {
		if (!start && !end) return '';
		const startStr = start ? new Date(start).getFullYear().toString() : '';
		const endStr = end ? new Date(end).getFullYear().toString() : 'Present';
		if (!startStr) return endStr;
		return `${startStr} - ${endStr}`;
	}
</script>

<svelte:head>
	<title>Education | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Education</h1>
		<button class="btn btn-primary" on:click={openNewForm}>
			+ New Education
		</button>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading education...</div>
		</div>
	{:else if showForm}
		<!-- Education Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingEdu ? 'Edit Education' : 'New Education'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" on:click={closeForm}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="institution" class="label">Institution *</label>
					<input
						type="text"
						id="institution"
						bind:value={institution}
						class="input"
						placeholder="Stanford University"
						required
					/>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="degree" class="label">Degree</label>
						<input
							type="text"
							id="degree"
							bind:value={degree}
							class="input"
							placeholder="Bachelor of Science"
						/>
					</div>

					<div>
						<label for="field" class="label">Field of Study</label>
						<input
							type="text"
							id="field"
							bind:value={field}
							class="input"
							placeholder="Computer Science"
						/>
					</div>
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
						<p class="text-xs text-gray-500 mt-1">Leave blank if still attending</p>
					</div>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="description" class="label mb-0">Description</label>
						<AIContentHelper
							fieldType="description"
							content={description}
							context={{ degree, field, institution }}
							on:apply={(e) => (description = e.detail.content)}
						/>
					</div>
					<textarea
						id="description"
						bind:value={description}
						class="input min-h-[100px]"
						placeholder="Activities, honors, relevant coursework..."
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown supported</p>
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
					{editingEdu ? 'Update Education' : 'Create Education'}
				</button>
			</div>
		</form>
	{:else if educations.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path d="M12 14l9-5-9-5-9 5 9 5z" />
				<path d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14zm-4 6v-7.5l4-2.222" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No education added yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your educational background, degrees, and certifications.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>
				+ Add Your First Education
			</button>
		</div>
	{:else}
		<!-- Education List -->
		<div class="space-y-4">
			{#each educations as edu (edu.id)}
				<div class="card p-4">
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-gray-900 dark:text-white">
									{edu.degree || 'Education'}
									{#if edu.field}
										<span class="text-gray-500 dark:text-gray-400 font-normal">in {edu.field}</span>
									{/if}
								</h3>
								{#if edu.is_draft}
									<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
										Draft
									</span>
								{/if}
								{#if edu.visibility !== 'public'}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{edu.visibility}
									</span>
								{/if}
								{#if !edu.end_date}
									<span class="px-2 py-0.5 text-xs bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded">
										Current
									</span>
								{/if}
							</div>

							<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
								<span class="flex items-center gap-1">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path d="M12 14l9-5-9-5-9 5 9 5z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0v7" />
									</svg>
									{edu.institution}
								</span>
								{#if edu.start_date || edu.end_date}
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
										</svg>
										{formatDateRange(edu.start_date, edu.end_date)}
									</span>
								{/if}
							</div>

							{#if edu.description}
								<p class="text-sm text-gray-600 dark:text-gray-400 mt-2 line-clamp-2">
									{edu.description}
								</p>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<button
								class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
								on:click={() => togglePublish(edu)}
								title={edu.is_draft ? 'Publish' : 'Unpublish'}
							>
								{#if edu.is_draft}
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
								on:click={() => openEditForm(edu)}
								title="Edit"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="p-2 text-gray-500 hover:text-red-600"
								on:click={() => deleteEducation(edu)}
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

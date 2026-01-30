<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { pb, type Experience } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { formatDate, toDateInputValue } from '$lib/utils';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import AdminTagSelector from '$components/admin/AdminTagSelector.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import SingleMediaPicker from '$lib/components/admin/SingleMediaPicker.svelte';

	let experiences: Experience[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingExp: Experience | null = $state(null);
	let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});

	let selectMode = $state(false);
	let selectedIds: Set<string> = $state(new Set());

	// Filtering
	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: true,
		enableDraftFilter: true,
		enableTagFilter: true,
		searchPlaceholder: 'Search experience...'
	});
	let availableTags: Array<{ id: string; name: string; color: string }> = $state([]);
	let showAdvancedFilters = $state(false);

	// Form fields
	let company = $state('');
	let title = $state('');
	let location = $state('');
	let startDate = $state('');
	let endDate = $state('');
	let description = $state('');
	let bullets: string[] = [];
	let bulletsText = $state('');
	let skills: string[] = [];
	let skillsText = $state('');
	let visibility = $state('public');
	let isDraft = $state(false);
	let sortOrder = $state(0);
	let companyLogoFile: FileList | null = $state(null);
	let companyLogoLibraryUrl = $state('');
	let clearCompanyLogo = $state(false);
	let logoBackground: 'none' | 'white' | 'light-gray' | 'dark-gray' | 'black' | 'accent' = $state('none');
	let saving = $state(false);
	let adminTagIds: string[] = $state([]);

	const autosave = createAutosave('admin-experience', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	function getFormData() {
		return { 
			company, title, location, startDate, endDate, description, bulletsText, skillsText, 
			visibility, isDraft, sortOrder, hasLogoFile: !!(companyLogoFile && companyLogoFile.length > 0) 
		};
	}

	function restoreFromDraft(data: Record<string, any>) {
		company = data.company || '';
		title = data.title || '';
		location = data.location || '';
		startDate = data.startDate || '';
		endDate = data.endDate || '';
		description = data.description || '';
		bulletsText = data.bulletsText || '';
		skillsText = data.skillsText || '';
		visibility = data.visibility || 'public';
		isDraft = data.isDraft || false;
		sortOrder = data.sortOrder || 0;
	}

	function handleFormChange() {
		if (showForm) {
			autosave.save(getFormData(), !!editingExp, editingExp?.id);
		}
	}

	afterNavigate(() => {
		showForm = false;
		editingExp = null;
		selectMode = false;
		selectedIds = new Set();
		
		const draft = autosave.loadDraft();
		if (draft?.data && Object.values(draft.data).some(v => v !== '' && v !== false && v !== 0)) {
			recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
			showRecoveryBanner = true;
		} else {
			showRecoveryBanner = false;
			recoveryData = null;
		}
	});



	onMount(() => {
		loadExperiences();
		loadAvailableTags();
	});

	async function loadAvailableTags() {
		try {
			const records = await collection('admin_tags').getList(1, 100, {
				sort: 'sort_order,name'
			});
			availableTags = records.items.map(tag => ({
				id: tag.id,
				name: tag.name,
				color: tag.color
			}));
		} catch (err) {
			console.error('Failed to load available tags:', err);
		}
	}

	// Computed filtered experiences
	let filteredExperiences = $derived(
		filterStore.filterItems(
			experiences,
			(exp) => `${exp.company} ${exp.title} ${exp.location || ''} ${exp.description || ''}`,
			(exp) => exp.visibility,
			(exp) => exp.is_draft,
			(exp) => (exp as any).admin_tags || []
		)
	);

	async function loadExperiences() {
		loading = true;
		try {
			const [records, membershipResp] = await Promise.all([
				await collection('experience').getList(1, 100, {
					sort: '-sort_order,-start_date',
					expand: 'admin_tags'
				}),
				fetch('/api/admin/view-memberships?collection=experience', {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
				}).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed memberships'))))
			]);
			experiences = records.items as unknown as Experience[];
			memberships = (membershipResp.memberships as typeof memberships) || {};
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
		companyLogoFile = null;
		companyLogoLibraryUrl = '';
		clearCompanyLogo = false;
		logoBackground = 'none';
		adminTagIds = [];
		editingExp = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function handleRestoreDraft() {
		const draft = autosave.loadDraft();
		if (draft?.data) {
			restoreFromDraft(draft.data);
			if (draft.isEditing && draft.editingId) {
				const exp = experiences.find(e => e.id === draft.editingId);
				if (exp) editingExp = exp;
			}
			showForm = true;
		}
		showRecoveryBanner = false;
	}

	function handleDismissDraft() {
		autosave.clearDraft();
		showRecoveryBanner = false;
		recoveryData = null;
	}

	function openEditForm(exp: Experience) {
		editingExp = exp;
		company = exp.company;
		title = exp.title;
		location = exp.location || '';
		startDate = toDateInputValue(exp.start_date);
		endDate = toDateInputValue(exp.end_date);
		description = exp.description || '';
		bullets = exp.bullets || [];
		bulletsText = bullets.join('\n');
		skills = exp.skills || [];
		skillsText = skills.join(', ');
		visibility = exp.visibility;
		isDraft = exp.is_draft;
		sortOrder = exp.sort_order;
		companyLogoLibraryUrl = exp.company_logo_library_url || '';
		clearCompanyLogo = false;
		logoBackground = exp.logo_background || 'none';
		adminTagIds = (exp as any).admin_tags || [];
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
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

			// Use FormData for file uploads or clearing files, plain object otherwise
			const hasFile = companyLogoFile && companyLogoFile.length > 0;
			const needsClearFile = clearCompanyLogo && editingExp?.company_logo;
			const needsFormData = hasFile || needsClearFile;

			if (needsFormData) {
				const formData = new FormData();
				formData.append('company', company.trim());
				formData.append('title', title.trim());
				formData.append('location', location.trim());
				formData.append('start_date', startDate ? new Date(startDate).toISOString() : '');
				formData.append('end_date', endDate ? new Date(endDate).toISOString() : '');
				formData.append('description', description.trim());
				formData.append('bullets', JSON.stringify(parsedBullets));
				formData.append('skills', JSON.stringify(parsedSkills));
				formData.append('visibility', visibility);
				formData.append('is_draft', String(isDraft));
				formData.append('sort_order', String(sortOrder));
				formData.append('logo_background', logoBackground);
				formData.append('admin_tags', JSON.stringify(adminTagIds));

				if (hasFile) {
					formData.append('company_logo', companyLogoFile![0]);
					// Clear library URL when uploading a new file
					formData.append('company_logo_library_url', '');
				} else if (needsClearFile) {
					// Clear existing file - PocketBase accepts empty string to remove file
					formData.append('company_logo', '');
					formData.append('company_logo_library_url', companyLogoLibraryUrl || '');
				}

				if (editingExp) {
					await collection('experience').update(editingExp.id, formData);
					toasts.add('success', 'Experience updated successfully');
				} else {
					await collection('experience').create(formData);
					toasts.add('success', 'Experience created successfully');
				}
			} else {
				const data: Record<string, unknown> = {
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
					sort_order: sortOrder,
					logo_background: logoBackground,
					admin_tags: adminTagIds,
					company_logo_library_url: companyLogoLibraryUrl || null
				};

				if (editingExp) {
					await collection('experience').update(editingExp.id, data);
					toasts.add('success', 'Experience updated successfully');
				} else {
					await collection('experience').create(data);
					toasts.add('success', 'Experience created successfully');
				}
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

	async function duplicateExperience(exp: Experience) {
		try {
			const response = await fetch(`/api/admin/duplicate/experience/${exp.id}`, {
				method: 'POST',
				headers: {
					'Authorization': `Bearer ${pb.authStore.token}`,
					'Content-Type': 'application/json'
				}
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				throw new Error(errorData.message || `HTTP ${response.status}`);
			}

			const data = await response.json();
			await loadExperiences();
			toasts.add('success', `Experience at ${exp.company} duplicated successfully`);
		} catch (err) {
			console.error('Failed to duplicate experience:', err);
			toasts.add('error', `Failed to duplicate experience: ${err instanceof Error ? err.message : 'Unknown error'}`);
		}
	}

	async function deleteExperience(exp: Experience) {
		const confirmed = await confirm({
			title: 'Delete Experience',
			message: `Are you sure you want to delete your experience at "${exp.company}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;
		try {
			await collection('experience').delete(exp.id);
			await loadExperiences();
			toasts.add('success', `Experience at ${exp.company} deleted`);
		} catch (err) {
			console.error('Failed to delete experience:', err);
			toasts.add('error', 'Failed to delete experience');
		}
	}

	async function togglePublish(exp: Experience) {
		try {
			await collection('experience').update(exp.id, {
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
		const confirmed = await confirm({
			title: 'Delete Experiences',
			message: `Are you sure you want to delete ${ids.length} experience(s)? This action cannot be undone.`,
			confirmText: 'Delete All',
			danger: true
		});
		if (!confirmed) return;
		
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
	<title>{$t('admin.content.experience.title')} {$t('admin.content.common.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="experience">
		<p>{@html $t('admin.content.experience.help_text')}</p>
		<p>{$t('admin.content.experience.help_tip_1')}</p>
		<p>{@html $t('admin.content.experience.help_tip_2')}</p>
	</PageHelp>

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
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.content.experience.title')}</h1>
		<div class="flex items-center gap-2">
			{#if experiences.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={toggleSelectMode}
				>
					{selectMode ? $t('admin.content.common.cancel') : $t('admin.content.common.select')}
				</button>
			{/if}
			<button class="btn btn-primary" onclick={openNewForm}>
				{$t('admin.content.common.new_button', { values: { type: $t('admin.content.experience.title') } })}
			</button>
		</div>
	</div>

	{#if showRecoveryBanner && recoveryData}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

	<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} {availableTags} />

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.content.common.loading', { values: { type: $t('admin.content.experience.title').toLowerCase() } })}</div>
		</div>
	{:else if showForm}
		<!-- Experience Form -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingExp ? $t('admin.content.common.form_title_edit', { values: { type: $t('admin.content.experience.title') } }) : $t('admin.content.common.form_title_new', { values: { type: $t('admin.content.experience.title') } })}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" onclick={closeForm} aria-label={$t('admin.content.common.close_form')}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="company" class="label">{$t('admin.content.experience.company_label')} {$t('admin.content.common.required_field')}</label>
						<input
							type="text"
							id="company"
							bind:value={company}
							class="input"
							placeholder={$t('admin.content.experience.company_placeholder')}
							required
						/>
					</div>

					<div>
						<label for="title" class="label">{$t('admin.content.experience.job_title_label')} {$t('admin.content.common.required_field')}</label>
						<input
							type="text"
							id="title"
							bind:value={title}
							class="input"
							placeholder={$t('admin.content.experience.job_title_placeholder')}
							required
						/>
					</div>
				</div>

				<div>
					<label for="location" class="label">{$t('admin.content.experience.location_label')}</label>
					<input
						type="text"
						id="location"
						bind:value={location}
						class="input"
						placeholder={$t('admin.content.experience.location_placeholder')}
					/>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<SingleMediaPicker
							label={$t('admin.content.experience.company_logo_label')}
							helpText={$t('admin.content.experience.company_logo_help')}
							bind:value={companyLogoLibraryUrl}
							bind:fileInput={companyLogoFile}
							bind:clearExisting={clearCompanyLogo}
							currentFileUrl={editingExp?.company_logo ? pb.files.getUrl(editingExp, editingExp.company_logo, { thumb: '64x64' }) : ''}
							currentFileName={editingExp?.company_logo || ''}
							imagesOnly={true}
						/>
					</div>
					<div>
						<label for="logo_background" class="label">{$t('admin.content.experience.logo_background_label')}</label>
						<select
							id="logo_background"
							bind:value={logoBackground}
							class="input"
						>
							<option value="none">{$t('admin.content.common.logo_background.none')}</option>
							<option value="white">{$t('admin.content.common.logo_background.white')}</option>
							<option value="light-gray">{$t('admin.content.common.logo_background.light_gray')}</option>
							<option value="dark-gray">{$t('admin.content.common.logo_background.dark_gray')}</option>
							<option value="black">{$t('admin.content.common.logo_background.black')}</option>
							<option value="accent">{$t('admin.content.common.logo_background.accent')}</option>
						</select>
						<p class="text-xs text-gray-500 mt-1">{$t('admin.content.experience.logo_background_help')}</p>
					</div>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="start_date" class="label">{$t('admin.content.experience.start_date_label')}</label>
						<input
							type="text"
							id="start_date"
							bind:value={startDate}
							placeholder={$t('admin.content.experience.start_date_placeholder')}
							class="input"
						/>
					</div>

					<div>
						<label for="end_date" class="label">{$t('admin.content.experience.end_date_label')}</label>
						<input
							type="text"
							id="end_date"
							bind:value={endDate}
							placeholder={$t('admin.content.experience.end_date_placeholder')}
							class="input"
						/>
						<p class="text-xs text-gray-500 mt-1">{$t('admin.content.experience.end_date_help')}</p>
					</div>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.common.details_section')}</h2>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="description" class="label mb-0">{$t('admin.content.common.description_label')}</label>
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
						placeholder={$t('admin.content.experience.description_placeholder')}
					></textarea>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.common.markdown_supported')}</p>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="bullets" class="label mb-0">{$t('admin.content.experience.key_achievements_label')}</label>
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
						placeholder={$t('admin.content.experience.key_achievements_placeholder')}
					></textarea>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.experience.key_achievements_help')}</p>
				</div>

				<div>
					<label for="skills" class="label">{$t('admin.content.experience.skills_label')}</label>
					<input
						type="text"
						id="skills"
						bind:value={skillsText}
						class="input"
						placeholder={$t('admin.content.experience.skills_placeholder')}
					/>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.experience.skills_help')}</p>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.content.common.settings_section')}</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="visibility" class="label">{$t('admin.content.common.visibility_label')}</label>
						<select id="visibility" bind:value={visibility} class="input">
							<option value="public">{$t('admin.content.common.visibility_public')}</option>
							<option value="unlisted">{$t('admin.content.common.visibility_unlisted')}</option>
							<option value="private">{$t('admin.content.common.visibility_private')}</option>
						</select>
					</div>

					<div>
						<label for="sort_order" class="label">{$t('admin.content.common.sort_order_label')}</label>
						<input
							type="number"
							id="sort_order"
							bind:value={sortOrder}
							class="input"
							min="0"
						/>
						<p class="text-xs text-gray-500 mt-1">{$t('admin.content.common.sort_order_help')}</p>
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
						{$t('admin.content.common.is_draft_label')}
					</label>
				</div>

				<div>
					<span id="admin-tags-label" class="label">{$t('admin.content.common.admin_tags_label')}</span>
					<AdminTagSelector bind:selectedIds={adminTagIds} labelledBy="admin-tags-label" />
					<p class="text-xs text-gray-500 mt-1">{$t('admin.content.common.admin_tags_help')}</p>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<button type="button" class="btn btn-secondary" onclick={closeForm}>{$t('admin.content.common.cancel')}</button>
				{#if editingExp}
					<button
						type="button"
						class="btn btn-outline"
						onclick={() => editingExp && duplicateExperience(editingExp)}
						disabled={saving}
					>
						<svg class="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
						{$t('admin.content.common.duplicate_button')}
					</button>
				{/if}
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingExp ? $t('admin.content.common.update_button', { values: { type: $t('admin.content.experience.title') } }) : $t('admin.content.common.create_button', { values: { type: $t('admin.content.experience.title') } })}
				</button>
			</div>
		</form>
	{:else if experiences.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.content.common.empty_state_no_items', { values: { type: $t('admin.content.experience.title').toLowerCase() } })}</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				{$t('admin.content.experience.empty_state_message')}
			</p>
			<button class="btn btn-primary" onclick={openNewForm}>
				{$t('admin.content.common.add_first_button', { values: { type: $t('admin.content.experience.title') } })}
			</button>
		</div>
	{:else}
		{#if filteredExperiences.length === 0}
			<div class="card p-8 text-center">
				<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.content.common.empty_state_no_matches', { values: { type: $t('admin.content.experience.title').toLowerCase() } })}</h3>
				<p class="text-gray-500 dark:text-gray-400 mb-4">
					{$t('admin.content.common.empty_state_adjust_filters')}
				</p>
				{#if filterStore.hasActiveFilters()}
					<button class="btn btn-secondary mr-2" onclick={() => filterStore.clearAllFilters()}>
						{$t('admin.content.common.clear_filters')}
					</button>
				{/if}
				<button class="btn btn-primary" onclick={openNewForm}>
					{$t('admin.content.common.add_button', { values: { type: $t('admin.content.experience.title') } })}
				</button>
			</div>
		{:else}
			<div class="space-y-4">
				{#each filteredExperiences as exp (exp.id)}
				<div class="card p-4 {selectMode && selectedIds.has(exp.id) ? 'ring-2 ring-primary-500' : ''}">
					<div class="flex items-start justify-between gap-4">
						{#if selectMode}
							<input
								type="checkbox"
								checked={selectedIds.has(exp.id)}
								onchange={() => toggleSelect(exp.id)}
								class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
							/>
						{/if}
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-gray-900 dark:text-white">
									{exp.title}
								</h3>
								<span class="text-gray-500 dark:text-gray-400">{$t('admin.content.experience.at')}</span>
								<span class="font-medium text-gray-800 dark:text-gray-200">{exp.company}</span>
								{#if exp.is_draft}
									<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
										{$t('admin.content.common.badge_draft')}
									</span>
								{/if}
								{#if exp.visibility !== 'public'}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{exp.visibility}
									</span>
								{/if}
								{#if !exp.end_date}
									<span class="px-2 py-0.5 text-xs bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded">
										{$t('admin.content.common.badge_current')}
									</span>
								{/if}
								{#if (exp as any).expand?.admin_tags && (exp as any).expand.admin_tags.length > 0}
									{#each (exp as any).expand.admin_tags as tag (tag.id)}
										<AdminTagBadge name={tag.name} color={tag.color} />
									{/each}
								{/if}
							</div>

							{#if memberships[exp.id]?.length}
								<div class="flex flex-wrap gap-1 mt-2">
									{#each memberships[exp.id].slice(0, 3) as viewRef}
										<a
											href={`/admin/views/${viewRef.id}`}
											target="_blank"
											class="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
											title={`Open view: ${viewRef.name}`}
										>
											{viewRef.slug || viewRef.name}
										</a>
									{/each}
									{#if memberships[exp.id].length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-400">
											+{memberships[exp.id].length - 3}
										</span>
									{/if}
								</div>
							{:else}
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.content.common.not_in_view')}</p>
							{/if}

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
											{$t('admin.content.common.more_items', { values: { count: exp.skills.length - 5 } })}
										</span>
									{/if}
								</div>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<button
								class="p-3 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
								onclick={() => togglePublish(exp)}
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
								class="p-3 text-gray-500 hover:text-blue-600"
								onclick={() => openEditForm(exp)}
								title="Edit"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>
							<button
								class="p-3 text-gray-500 hover:text-green-600"
								onclick={() => duplicateExperience(exp)}
								title="Duplicate"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
								</svg>
							</button>
							<button
								class="p-3 text-gray-500 hover:text-red-600"
								onclick={() => deleteExperience(exp)}
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
	{/if}
</div>

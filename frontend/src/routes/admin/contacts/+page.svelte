<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onDestroy, onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import {
		type ContactMethod,
		type ContactMethodType,
		type ProtectionLevel,
		type View
	} from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { collection, getCollectionName } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import { createListLoader } from '$lib/admin/list-load';
	import { readAllRecords } from '$lib/admin/read-all-records';
	import {
		DraftReadError,
		findDraftRecord,
		isRecordId,
		mayHaveLanded,
		newRecordId
	} from '$lib/admin/create-record';
	import ListLoadStatus from '$components/admin/ListLoadStatus.svelte';
	import SettingsLoadError from '$components/admin/SettingsLoadError.svelte';

	// State
	let contacts: ContactMethod[] = $state([]);
	let views: View[] = $state([]);
	const listLoad = createListLoader('contacts');
	let loading = $derived($listLoad.loading && !$listLoad.loaded);
	let viewsLoading = $state(true);
	let viewsReady = $state(false);
	let viewsFailed = $state(false);
	let canWrite = $derived(
		$listLoad.loaded &&
			!$listLoad.loading &&
			!$listLoad.failed &&
			!$listLoad.mutating &&
			viewsReady &&
			!viewsLoading &&
			!viewsFailed
	);
	let formRun = $state(0);
	let savingFormRun: number | null = $state(null);
	let pendingContactId: string | null = $state(null);
	let draftId: string | null = null;
	let createSent = false;
	let showForm = $state(false);
	let editingContact: ContactMethod | null = $state(null);
	let saving = $state(false);
	let formSaving = $derived(saving && savingFormRun === formRun);
	let memberships: Record<string, { id: string; name: string; slug: string }[]> = $state({});

	// Form fields
	let type: ContactMethodType = $state('email');
	let value = $state('');
	let label = $state('');
	let icon = $state('');
	let protectionLevel: ProtectionLevel = $state('none');
	let viewVisibility: Record<string, boolean> = $state({});
	let isPrimary = $state(false);
	let sortOrder = $state(0);

	const autosave = createAutosave('admin-contacts', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: false,
		enableDraftFilter: false,
		enableTagFilter: false,
		searchPlaceholder: 'Search contact methods...'
	});
	let showAdvancedFilters = $state(false);

	let filteredContacts = $derived(
		filterStore.filterItems(
			contacts,
			(c) => `${c.value} ${c.label || ''} ${c.type}`,
			() => 'public',
			() => false
		)
	);

	function getFormData() {
		return {
			draftId,
			type,
			value,
			label,
			icon,
			protectionLevel,
			viewVisibility,
			isPrimary,
			sortOrder
		};
	}

	function restoreFromDraft(data: Record<string, any>) {
		type = data.type || 'email';
		value = data.value || '';
		label = data.label || '';
		icon = data.icon || '';
		protectionLevel = data.protectionLevel || 'none';
		viewVisibility = { ...(data.viewVisibility || {}) };
		isPrimary = data.isPrimary || false;
		sortOrder = data.sortOrder || 0;
	}

	function handleFormChange() {
		if (showForm) {
			autosave.save(getFormData(), !!editingContact, editingContact?.id);
		}
	}

	function canChangeContacts() {
		if (!listLoad.canMutate()) return false;
		if (!viewsReady || viewsLoading || viewsFailed) {
			toasts.add('error', $t('admin.list_status.changes_paused'));
			return false;
		}
		return true;
	}

	function handleRestoreDraft() {
		if (!canChangeContacts()) return;
		const draft = autosave.loadDraft();
		if (!draft?.data || saving) return;
		const existing =
			draft.isEditing && draft.editingId
				? contacts.find((contact) => contact.id === draft.editingId)
				: null;
		if (draft.isEditing && !existing) {
			toasts.add(
				'error',
				'This draft belongs to a contact that is no longer available. Your draft has been kept.'
			);
			return;
		}
		resetForm();
		restoreFromDraft(draft.data);
		editingContact = existing || null;
		draftId = existing ? null : isRecordId(draft.data.draftId) ? draft.data.draftId : newRecordId();
		createSent = !existing && isRecordId(draft.data.draftId);
		showForm = true;
		showRecoveryBanner = false;
	}

	function handleDismissDraft() {
		autosave.clearDraft();
		showRecoveryBanner = false;
		recoveryData = null;
	}

	function checkRecoveryDraft() {
		const draft = autosave.loadDraft();
		if (
			draft?.data &&
			Object.values(draft.data).some(
				(v) =>
					v !== '' &&
					v !== false &&
					v !== 0 &&
					v !== null &&
					!(typeof v === 'object' && Object.keys(v).length === 0)
			)
		) {
			recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
			showRecoveryBanner = true;
		} else {
			showRecoveryBanner = false;
			recoveryData = null;
		}
	}

	onMount(checkRecoveryDraft);

	afterNavigate(({ shallow, type }) => {
		if (shallow && type === 'goto') return;

		showForm = false;
		editingContact = null;

		checkRecoveryDraft();
	});

	// Contact method type options with icons
	const contactTypes: {
		value: ContactMethodType;
		label: string;
		icon: string;
		placeholder: string;
	}[] = [
		{ value: 'email', label: 'Email', icon: '📧', placeholder: 'your@email.com' },
		{ value: 'phone', label: 'Phone', icon: '📱', placeholder: '+1 (555) 123-4567' },
		{
			value: 'linkedin',
			label: 'LinkedIn',
			icon: '💼',
			placeholder: 'https://linkedin.com/in/username'
		},
		{ value: 'github', label: 'GitHub', icon: '🐙', placeholder: 'https://github.com/username' },
		{ value: 'twitter', label: 'Twitter', icon: '🐦', placeholder: 'https://twitter.com/username' },
		{
			value: 'facebook',
			label: 'Facebook',
			icon: '👥',
			placeholder: 'https://facebook.com/username'
		},
		{
			value: 'instagram',
			label: 'Instagram',
			icon: '📷',
			placeholder: 'https://instagram.com/username'
		},
		{ value: 'youtube', label: 'YouTube', icon: '🎬', placeholder: 'https://youtube.com/@channel' },
		{
			value: 'mastodon',
			label: 'Mastodon',
			icon: '🐘',
			placeholder: 'https://mastodon.social/@username'
		},
		{ value: 'bluesky', label: 'Bluesky', icon: '🦋', placeholder: 'username.bsky.social' },
		{ value: 'website', label: 'Website', icon: '🌐', placeholder: 'https://yourwebsite.com' },
		{ value: 'portfolio', label: 'Portfolio', icon: '💼', placeholder: 'https://portfolio.com' },
		{ value: 'blog', label: 'Blog', icon: '✍️', placeholder: 'https://blog.com' },
		{ value: 'whatsapp', label: 'WhatsApp', icon: '💬', placeholder: '+1 (555) 123-4567' },
		{ value: 'telegram', label: 'Telegram', icon: '✈️', placeholder: '@username' },
		{ value: 'discord', label: 'Discord', icon: '🎮', placeholder: 'username' },
		{ value: 'slack', label: 'Slack', icon: '💬', placeholder: '@username' },
		{ value: 'custom', label: 'Custom', icon: '⚙️', placeholder: 'Custom contact info' },
		{ value: 'other', label: 'Other', icon: '🔗', placeholder: 'Contact information' }
	];

	// Protection levels
	const protectionLevels: { value: ProtectionLevel; label: string; description: string }[] = [
		{ value: 'none', label: 'None', description: 'Visible to everyone (no protection)' },
		{
			value: 'obfuscation',
			label: 'CSS Obfuscation',
			description: 'Hidden from bots via CSS tricks'
		},
		{
			value: 'click_to_reveal',
			label: 'Click to Reveal',
			description: 'User must click to see contact'
		},
		{
			value: 'captcha',
			label: 'CAPTCHA (not available)',
			description:
				'Verification is not built yet. Visitors see the label only and the value is withheld from public pages.'
		}
	];

	// CAPTCHA has no working verification, so it can't be chosen for a new
	// setting. A contact already saved with it keeps the option checked so the
	// form shows the truth, with an explanation and a one-click replacement.
	function protectionUnavailable(level: ProtectionLevel): boolean {
		return level === 'captcha' && protectionLevel !== 'captcha';
	}

	// Lifecycle
	onMount(() => {
		loadContacts();
		loadViews();
	});

	// A failed required read never stands in for an empty list.
	function loadContacts() {
		listLoad.loadMemberships('contact_methods', (value) => (memberships = value));
		return listLoad.load(
			(_requestKey, signal) =>
				readAllRecords<ContactMethod>(
					getCollectionName('contact_methods'),
					{ sort: '-sort_order' },
					signal
				),
			(records) => (contacts = records)
		);
	}

	async function loadViews() {
		viewsLoading = true;
		await listLoad.read(
			'views',
			(_requestKey, signal) =>
				readAllRecords<View>(getCollectionName('views'), { sort: '-id' }, signal),
			(records) => {
				views = [
					...records.filter((view) => view.is_default),
					...records.filter((view) => !view.is_default)
				];
				viewsReady = true;
				viewsFailed = false;
			},
			() => {
				viewsFailed = true;
			}
		);
		viewsLoading = false;
	}

	onDestroy(() => {
		formRun++;
	});

	// Form management
	function resetForm() {
		formRun++;
		draftId = null;
		createSent = false;
		type = 'email';
		value = '';
		label = '';
		icon = '';
		protectionLevel = 'none';
		viewVisibility = {};
		isPrimary = false;
		sortOrder = 0;
		editingContact = null;
	}

	function openNewForm() {
		if (!canChangeContacts()) return;
		resetForm();
		draftId = newRecordId();
		showForm = true;
		showRecoveryBanner = false;
	}

	function openEditForm(contact: ContactMethod) {
		if (!canChangeContacts() || pendingContactId === contact.id) return;
		resetForm();
		editingContact = contact;
		type = contact.type;
		value = contact.value;
		label = contact.label || '';
		icon = contact.icon || '';
		protectionLevel = contact.protection_level;
		viewVisibility = { ...(contact.view_visibility || {}) };
		isPrimary = contact.is_primary;
		sortOrder = contact.sort_order;
		showForm = true;
		showRecoveryBanner = false;
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
	}

	// CRUD operations
	async function handleSubmit() {
		if (saving || !canChangeContacts()) return;
		// Validation
		if (!value.trim()) {
			toasts.add('error', 'Contact value is required');
			return;
		}

		// Email validation
		if (type === 'email' && !value.includes('@')) {
			toasts.add('error', 'Please enter a valid email address');
			return;
		}

		// URL validation for social links
		const urlTypes = ['linkedin', 'github', 'twitter', 'facebook', 'instagram', 'website'];
		if (urlTypes.includes(type) && !value.startsWith('http')) {
			toasts.add('error', 'Please enter a valid URL starting with http:// or https://');
			return;
		}

		const run = formRun;
		const editingId = editingContact?.id;
		const id = editingId || draftId || newRecordId();
		if (!editingId) {
			draftId = id;
			autosave.saveNow(getFormData(), false);
		}
		saving = true;
		savingFormRun = run;
		pendingContactId = id;
		try {
			const data = {
				type,
				value: value.trim(),
				label: label.trim() || null,
				icon: icon.trim() || null,
				protection_level: protectionLevel,
				view_visibility: { ...viewVisibility },
				is_primary: isPrimary,
				sort_order: sortOrder
			};

			const records = collection('contact_methods');
			const adoptStored = (stored: ContactMethod) => {
				if (formRun === run) editingContact = stored;
				const same = Object.entries(data).every(([key, expected]) => {
					const actual = (stored as unknown as Record<string, unknown>)[key];
					if (key === 'view_visibility') {
						const left = Object.entries((actual || {}) as Record<string, boolean>).sort();
						const right = Object.entries(expected as Record<string, boolean>).sort();
						return JSON.stringify(left) === JSON.stringify(right);
					}
					return (actual ?? '') === (expected ?? '');
				});
				if (!same)
					throw new Error(
						'The contact was created, but its saved fields differ from this draft. Review your fields and press Update Contact to save them.'
					);
			};
			if (editingId) {
				await records.update(editingId, data, {
					requestKey: null,
					signal: AbortSignal.timeout(30_000)
				});
			} else {
				const stored = createSent ? await findDraftRecord<ContactMethod>(records, id) : null;
				if (stored) {
					adoptStored(stored);
				} else {
					if (formRun === run) createSent = true;
					try {
						await records.create(
							{ ...data, id },
							{ requestKey: null, signal: AbortSignal.timeout(30_000) }
						);
					} catch (error) {
						if (!mayHaveLanded(error)) throw error;
						const found = await findDraftRecord<ContactMethod>(records, id);
						if (!found) throw error;
						adoptStored(found);
					}
				}
			}
			toasts.add(
				'success',
				editingId ? 'Contact method updated successfully' : 'Contact method added successfully'
			);
			if (formRun === run) closeForm();
			await loadContacts();
		} catch (err) {
			console.error('Failed to save contact:', err);
			toasts.add(
				'error',
				err instanceof DraftReadError
					? "Couldn't confirm whether this contact was saved. Your draft has been kept; retry before creating another."
					: err instanceof Error && err.message.startsWith('The contact was created')
						? err.message
						: 'Failed to save contact method'
			);
		} finally {
			saving = false;
			savingFormRun = null;
			pendingContactId = null;
		}
	}

	const deleteContact = listLoad.exclusive(async (contact: ContactMethod) => {
		if (saving) return;
		const displayValue = contact.label || contact.value;
		const confirmed = await confirm({
			title: 'Delete Contact Method',
			message: `Are you sure you want to delete "${displayValue}"? This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed || saving || !listLoad.canMutate()) {
			return;
		}

		listLoad.setMutating(true);
		try {
			await collection('contact_methods').delete(contact.id, {
				requestKey: null,
				signal: AbortSignal.timeout(30_000)
			});
			toasts.add('success', 'Contact method deleted');
			await loadContacts();
		} catch (err) {
			console.error('Failed to delete contact:', err);
			toasts.add('error', 'Failed to delete contact method');
		} finally {
			listLoad.setMutating(false);
		}
	});

	const togglePrimary = listLoad.exclusive(async (contact: ContactMethod) => {
		if (saving) return;
		listLoad.setMutating(true);
		try {
			await collection('contact_methods').update(
				contact.id,
				{ is_primary: !contact.is_primary },
				{ requestKey: null, signal: AbortSignal.timeout(30_000) }
			);
			await loadContacts();
		} catch (err) {
			console.error('Failed to update primary status:', err);
			toasts.add('error', 'Failed to update primary status');
		} finally {
			listLoad.setMutating(false);
		}
	});

	// View visibility helpers
	function toggleViewVisibility(viewId: string) {
		viewVisibility[viewId] = !viewVisibility[viewId];
		viewVisibility = { ...viewVisibility }; // Trigger reactivity
	}

	function selectAllViews() {
		views.forEach((view) => {
			viewVisibility[view.id] = true;
		});
		viewVisibility = { ...viewVisibility };
	}

	function deselectAllViews() {
		const offered = new Set(views.map((view) => view.id));
		viewVisibility = Object.fromEntries(
			Object.entries(viewVisibility).filter(([id]) => !offered.has(id))
		);
	}

	// Get display info for contact type
	function getContactTypeInfo(contactType: ContactMethodType) {
		return contactTypes.find((ct) => ct.value === contactType) || contactTypes[0];
	}

	// Get protection level label
	function getProtectionLabel(level: ProtectionLevel) {
		return protectionLevels.find((pl) => pl.value === level)?.label || level;
	}

	// Type change handler - update icon automatically
	function handleTypeChange() {
		const typeInfo = getContactTypeInfo(type);
		if (!icon || icon === '') {
			icon = typeInfo.icon;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.content.contacts.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="contacts">
		{@html $t('admin.content.contacts.help_text')}
		<p>{$t('admin.content.contacts.help_tip_1')}</p>
		{@html $t('admin.content.contacts.help_tip_2')}
	</PageHelp>

	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">
		{$t('admin.content.contacts.title')}
	</h1>

	<div class="flex items-center justify-between mb-6">
		<p class="text-sm text-gray-600 dark:text-gray-400">
			Add contact methods with per-view visibility and anti-scraping protection.
		</p>
		<button class="btn btn-primary" onclick={openNewForm} disabled={!canWrite}
			>{$t('admin.content.common.new_button', { values: { type: 'Contact' } })}</button
		>
	</div>

	{#if showRecoveryBanner && recoveryData && canWrite}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

	<ListLoadStatus
		status={$listLoad}
		label={$t('admin.content.contacts.title').toLowerCase()}
		onretry={loadContacts}
	/>
	{#if viewsFailed}
		<SettingsLoadError
			message={$t('admin.list_status.load_failed', { values: { type: 'Facets' } })}
			pending={viewsLoading}
			onretry={loadViews}
		/>
	{/if}
	<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} />

	{#if loading}
		<div class="card p-8 text-center">
			<div
				class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"
			></div>
			<p class="mt-4 text-gray-600 dark:text-gray-400">
				{$t('admin.content.common.loading', {
					values: { type: $t('admin.content.contacts.title').toLowerCase() }
				})}
			</p>
		</div>
	{:else if showForm}
		<!-- Form View -->
		<div class="card p-6 mb-6">
			<div class="flex items-center justify-between mb-6">
				<h2 class="text-xl font-semibold">
					{editingContact ? 'Edit Contact Method' : 'New Contact Method'}
				</h2>
				<button class="btn btn-secondary text-sm" onclick={closeForm}>Cancel</button>
			</div>

			<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-6">
				<fieldset class="contents" disabled={formSaving || !canWrite}>
					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label for="contact-type" class="label">
								Contact Type <span class="text-red-500">*</span>
							</label>
							<select id="contact-type" bind:value={type} onchange={handleTypeChange} class="input">
								{#each contactTypes as contactType}
									<option value={contactType.value}>
										{contactType.icon}
										{contactType.label}
									</option>
								{/each}
							</select>
						</div>

						<div>
							<label for="contact-value" class="label">
								Value <span class="text-red-500">*</span>
							</label>
							<input
								id="contact-value"
								type="text"
								bind:value
								placeholder={getContactTypeInfo(type).placeholder}
								class="input"
								required
							/>
						</div>
					</div>

					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label for="contact-label" class="label">Label (optional)</label>
							<input
								id="contact-label"
								type="text"
								bind:value={label}
								placeholder="e.g., 'Work Email' or 'Personal Phone'"
								class="input"
							/>
							<p class="text-xs text-gray-500 mt-1">Custom display label for this contact</p>
						</div>

						<div>
							<label for="contact-icon" class="label">Icon (optional)</label>
							<input
								id="contact-icon"
								type="text"
								bind:value={icon}
								placeholder="Emoji or icon"
								class="input"
								maxlength="4"
							/>
							<p class="text-xs text-gray-500 mt-1">
								Emoji or icon to display (defaults to type icon)
							</p>
						</div>
					</div>

					<fieldset>
						<legend class="label">
							Protection Level <span class="text-red-500">*</span>
						</legend>
						{#if protectionLevel === 'captcha'}
							<div
								class="mb-3 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 text-sm text-amber-900 dark:text-amber-200"
								role="note"
								data-testid="captcha-unavailable-note"
							>
								<p>
									This contact uses CAPTCHA protection, which isn't available yet. Visitors can't
									reveal it, so public pages show the label and keep the value private.
								</p>
								<button
									type="button"
									class="mt-2 underline font-medium"
									onclick={() => (protectionLevel = 'click_to_reveal')}
								>
									Switch to Click to Reveal
								</button>
							</div>
						{/if}
						<div class="space-y-2">
							{#each protectionLevels as level}
								<label
									class="flex items-start space-x-3 p-3 border rounded-lg transition-colors {protectionUnavailable(
										level.value
									)
										? 'cursor-not-allowed opacity-60'
										: 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800'} {protectionLevel ===
									level.value
										? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
										: 'border-gray-300 dark:border-gray-700'}"
								>
									<input
										type="radio"
										bind:group={protectionLevel}
										value={level.value}
										disabled={protectionUnavailable(level.value)}
										class="mt-1"
									/>
									<div class="flex-1">
										<div class="font-medium">{level.label}</div>
										<div class="text-sm text-gray-600 dark:text-gray-400">{level.description}</div>
									</div>
								</label>
							{/each}
						</div>
					</fieldset>

					<fieldset>
						<legend class="label">View Visibility</legend>
						<p class="text-sm text-gray-600 dark:text-gray-400 mb-3">
							{#if Object.keys(viewVisibility).some((id) => !views.some((view) => view.id === id))}
								Saved visibility choices for unavailable Facets are retained. These choices still
								control where this contact appears, even when no visible box is checked.
							{:else}
								Select which views should display this contact method. Leave all unchecked to show
								on all views.
							{/if}
						</p>

						<div class="flex gap-2 mb-3">
							<button type="button" class="btn btn-secondary text-sm" onclick={selectAllViews}>
								Select All
							</button>
							<button type="button" class="btn btn-secondary text-sm" onclick={deselectAllViews}>
								Deselect All
							</button>
						</div>

						{#if views.length === 0}
							<p class="text-sm text-gray-500 italic">No views available. Create a view first.</p>
						{:else}
							<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
								{#each views as view}
									<label
										class="flex items-center space-x-2 p-2 border rounded cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors {viewVisibility[
											view.id
										]
											? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
											: 'border-gray-300 dark:border-gray-700'}"
									>
										<input
											type="checkbox"
											checked={viewVisibility[view.id] || false}
											onchange={() => toggleViewVisibility(view.id)}
										/>
										<span class="text-sm">{view.name}</span>
										{#if view.is_default}
											<span
												class="text-xs px-1.5 py-0.5 bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded"
											>
												Default
											</span>
										{/if}
									</label>
								{/each}
							</div>
						{/if}
					</fieldset>

					<!-- Primary & Sort Order -->
					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label class="flex items-center space-x-2">
								<input type="checkbox" bind:checked={isPrimary} />
								<span class="label mb-0">Mark as primary contact</span>
							</label>
							<p class="text-xs text-gray-500 mt-1 ml-6">
								Primary contacts are highlighted in views
							</p>
						</div>

						<div>
							<label for="contact-sort-order" class="label">Sort Order</label>
							<input
								id="contact-sort-order"
								type="number"
								bind:value={sortOrder}
								placeholder="0"
								class="input"
							/>
							<p class="text-xs text-gray-500 mt-1">Higher numbers appear first</p>
						</div>
					</div>
				</fieldset>
				<!-- Submit -->
				<div class="flex gap-2 pt-4 border-t">
					<button type="submit" class="btn btn-primary" disabled={saving || !canWrite}>
						{#if saving}
							Saving...
						{:else}
							{editingContact ? 'Update Contact' : 'Add Contact'}
						{/if}
					</button>
					<button type="button" class="btn btn-secondary" onclick={closeForm}>Cancel</button>
				</div>
			</form>
		</div>
	{:else if !$listLoad.loaded}
		<!-- The failed required read is shown above; it is not an empty list. -->
	{:else if contacts.length === 0}
		<!-- Empty State -->
		<div class="card p-8 text-center">
			<svg
				class="w-12 h-12 mx-auto text-gray-400 mb-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
				></path>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No contact methods yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your first contact method with privacy protection settings.
			</p>
			<button class="btn btn-primary" onclick={openNewForm} disabled={!canWrite}
				>+ Add Your First Contact</button
			>
		</div>
	{:else if filteredContacts.length === 0}
		<div class="card p-8 text-center">
			<svg
				class="w-12 h-12 mx-auto text-gray-400 mb-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				></path>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
				No contact methods match your filters
			</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">Try adjusting your search query.</p>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				Clear Search
			</button>
		</div>
	{:else}
		<!-- List View -->
		<div class="space-y-3">
			{#each filteredContacts as contact}
				<div class="card p-4 flex items-center justify-between hover:shadow-md transition-shadow">
					<div class="flex items-center space-x-4 flex-1">
						<!-- Icon & Type -->
						<div class="text-2xl">{contact.icon || getContactTypeInfo(contact.type).icon}</div>

						<!-- Details -->
						<div class="flex-1">
							<div class="flex items-center gap-2">
								<span class="font-medium">
									{contact.label || getContactTypeInfo(contact.type).label}
								</span>
								{#if contact.is_primary}
									<span
										class="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded"
									>
										Primary
									</span>
								{/if}
							</div>
							<div class="text-sm text-gray-600 dark:text-gray-400">
								{contact.value}
							</div>
							<div class="flex items-center gap-3 mt-1 text-xs text-gray-500">
								<span>🔒 {getProtectionLabel(contact.protection_level)}</span>
								<span>📊 Sort: {contact.sort_order}</span>
								{#if contact.view_visibility && Object.keys(contact.view_visibility).length > 0}
									<span>
										👁️ {Object.values(contact.view_visibility).filter(Boolean).length} view(s)
									</span>
								{:else}
									<span>👁️ All views</span>
								{/if}
							</div>

							{#if memberships[contact.id]?.length}
								<div class="flex flex-wrap gap-1 mt-2">
									{#each memberships[contact.id].slice(0, 3) as viewRef}
										<a
											href={`/admin/views/${viewRef.id}`}
											target="_blank"
											class="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
											title={`Open view: ${viewRef.name}`}
										>
											{viewRef.slug || viewRef.name}
										</a>
									{/each}
									{#if memberships[contact.id].length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-400">
											+{memberships[contact.id].length - 3}
										</span>
									{/if}
								</div>
							{:else}
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">Not in any view</p>
							{/if}
						</div>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2">
						<button
							class="btn btn-secondary text-sm"
							onclick={() => togglePrimary(contact)}
							title="Toggle primary"
							disabled={saving ||
								!$listLoad.loaded ||
								$listLoad.failed ||
								$listLoad.loading ||
								$listLoad.mutating}
						>
							{contact.is_primary ? '⭐' : '☆'}
						</button>
						<button
							class="btn btn-secondary text-sm"
							onclick={() => openEditForm(contact)}
							disabled={!canWrite || pendingContactId === contact.id}
						>
							Edit
						</button>
						<button
							class="btn btn-danger-ghost btn-sm"
							onclick={() => deleteContact(contact)}
							disabled={saving ||
								!$listLoad.loaded ||
								$listLoad.failed ||
								$listLoad.loading ||
								$listLoad.mutating}
						>
							Delete
						</button>
					</div>
				</div>
			{/each}
		</div>

		<!-- Info Footer -->
		<div class="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
			<h4 class="font-medium text-blue-900 dark:text-blue-300 mb-2">About Contact Protection</h4>
			<ul class="text-sm text-blue-800 dark:text-blue-400 space-y-1">
				<li><strong>None:</strong> Contact visible to everyone including bots</li>
				<li>
					<strong>CSS Obfuscation:</strong> Hides contact from simple scrapers using CSS tricks
				</li>
				<li><strong>Click to Reveal:</strong> Requires user interaction to display contact</li>
				<li>
					<strong>CAPTCHA:</strong> Not available yet. Existing CAPTCHA contacts show only their label
					publicly; switch them to Click to Reveal so visitors can reach you.
				</li>
			</ul>
		</div>
	{/if}
</div>

<style>
	/* Custom styles if needed */
</style>

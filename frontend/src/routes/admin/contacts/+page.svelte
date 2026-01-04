<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type ContactMethod, type ContactMethodType, type ProtectionLevel, type View } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import AdminHeader from '$components/admin/AdminHeader.svelte';

	// State
	let contacts: ContactMethod[] = [];
	let views: View[] = [];
	let loading = true;
	let showForm = false;
	let editingContact: ContactMethod | null = null;
	let saving = false;

	// Form fields
	let type: ContactMethodType = 'email';
	let value = '';
	let label = '';
	let icon = '';
	let protectionLevel: ProtectionLevel = 'none';
	let viewVisibility: Record<string, boolean> = {};
	let isPrimary = false;
	let sortOrder = 0;

	// Contact method type options with icons
	const contactTypes: { value: ContactMethodType; label: string; icon: string; placeholder: string }[] = [
		{ value: 'email', label: 'Email', icon: '📧', placeholder: 'your@email.com' },
		{ value: 'phone', label: 'Phone', icon: '📱', placeholder: '+1 (555) 123-4567' },
		{ value: 'linkedin', label: 'LinkedIn', icon: '💼', placeholder: 'https://linkedin.com/in/username' },
		{ value: 'github', label: 'GitHub', icon: '🐙', placeholder: 'https://github.com/username' },
		{ value: 'twitter', label: 'Twitter', icon: '🐦', placeholder: 'https://twitter.com/username' },
		{ value: 'facebook', label: 'Facebook', icon: '👥', placeholder: 'https://facebook.com/username' },
		{ value: 'instagram', label: 'Instagram', icon: '📷', placeholder: 'https://instagram.com/username' },
		{ value: 'website', label: 'Website', icon: '🌐', placeholder: 'https://yourwebsite.com' },
		{ value: 'whatsapp', label: 'WhatsApp', icon: '💬', placeholder: '+1 (555) 123-4567' },
		{ value: 'telegram', label: 'Telegram', icon: '✈️', placeholder: '@username' },
		{ value: 'discord', label: 'Discord', icon: '🎮', placeholder: 'username#1234' },
		{ value: 'slack', label: 'Slack', icon: '💼', placeholder: '@username' },
		{ value: 'other', label: 'Other', icon: '🔗', placeholder: 'Contact information' }
	];

	// Protection levels
	const protectionLevels: { value: ProtectionLevel; label: string; description: string }[] = [
		{ value: 'none', label: 'None', description: 'Visible to everyone (no protection)' },
		{ value: 'obfuscation', label: 'CSS Obfuscation', description: 'Hidden from bots via CSS tricks' },
		{ value: 'click_to_reveal', label: 'Click to Reveal', description: 'User must click to see contact' },
		{ value: 'captcha', label: 'CAPTCHA', description: 'Cloudflare Turnstile required' }
	];

	// Lifecycle
	onMount(() => {
		loadContacts();
		loadViews();
	});

	// Data loading
	async function loadContacts() {
		loading = true;
		try {
			const records = await await collection('contact_methods').getList(1, 100, {
				sort: '-sort_order'
			});
			contacts = records.items as unknown as ContactMethod[];
		} catch (err) {
			console.error('Failed to load contacts:', err);
			toasts.add('error', 'Failed to load contact methods');
		} finally {
			loading = false;
		}
	}

	async function loadViews() {
		try {
			const records = await await collection('views').getList(1, 100, {
				sort: '-is_default,-created'
			});
			views = records.items as unknown as View[];
		} catch (err) {
			console.error('Failed to load views:', err);
		}
	}

	// Form management
	function resetForm() {
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
		resetForm();
		showForm = true;
	}

	function openEditForm(contact: ContactMethod) {
		editingContact = contact;
		type = contact.type;
		value = contact.value;
		label = contact.label || '';
		icon = contact.icon || '';
		protectionLevel = contact.protection_level;
		viewVisibility = contact.view_visibility || {};
		isPrimary = contact.is_primary;
		sortOrder = contact.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	// CRUD operations
	async function handleSubmit() {
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

		saving = true;
		try {
			const data = {
				type,
				value: value.trim(),
				label: label.trim() || null,
				icon: icon.trim() || null,
				protection_level: protectionLevel,
				view_visibility: viewVisibility,
				is_primary: isPrimary,
				sort_order: sortOrder
			};

			if (editingContact) {
				await await collection('contact_methods').update(editingContact.id, data);
				toasts.add('success', 'Contact method updated successfully');
			} else {
				await await collection('contact_methods').create(data);
				toasts.add('success', 'Contact method added successfully');
			}

			closeForm();
			await loadContacts();
		} catch (err) {
			console.error('Failed to save contact:', err);
			toasts.add('error', 'Failed to save contact method');
		} finally {
			saving = false;
		}
	}

	async function deleteContact(contact: ContactMethod) {
		const displayValue = contact.label || contact.value;
		if (!confirm(`Are you sure you want to delete "${displayValue}"?`)) {
			return;
		}

		try {
			await await collection('contact_methods').delete(contact.id);
			toasts.add('success', 'Contact method deleted');
			await loadContacts();
		} catch (err) {
			console.error('Failed to delete contact:', err);
			toasts.add('error', 'Failed to delete contact method');
		}
	}

	async function togglePrimary(contact: ContactMethod) {
		try {
			await await collection('contact_methods').update(contact.id, {
				is_primary: !contact.is_primary
			});
			await loadContacts();
		} catch (err) {
			console.error('Failed to update primary status:', err);
			toasts.add('error', 'Failed to update primary status');
		}
	}

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
		viewVisibility = {};
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
	<title>Contact Methods | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<AdminHeader title="Contact Methods" subtitle="Manage contact information with privacy protection" />

	<div class="flex items-center justify-between mb-6">
		<p class="text-sm text-gray-600 dark:text-gray-400">
			Add contact methods with per-view visibility and anti-scraping protection.
		</p>
		<button class="btn btn-primary" on:click={openNewForm}>+ New Contact Method</button>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"></div>
			<p class="mt-4 text-gray-600 dark:text-gray-400">Loading contact methods...</p>
		</div>
	{:else if showForm}
		<!-- Form View -->
		<div class="card p-6 mb-6">
			<div class="flex items-center justify-between mb-6">
				<h2 class="text-xl font-semibold">
					{editingContact ? 'Edit Contact Method' : 'New Contact Method'}
				</h2>
				<button class="btn btn-secondary text-sm" on:click={closeForm}>Cancel</button>
			</div>

			<form on:submit|preventDefault={handleSubmit} class="space-y-6">
				<!-- Contact Type & Value -->
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="label">
							Contact Type <span class="text-red-500">*</span>
						</label>
						<select bind:value={type} on:change={handleTypeChange} class="input">
							{#each contactTypes as contactType}
								<option value={contactType.value}>
									{contactType.icon} {contactType.label}
								</option>
							{/each}
						</select>
					</div>

					<div>
						<label class="label">
							Value <span class="text-red-500">*</span>
						</label>
						<input
							type="text"
							bind:value
							placeholder={getContactTypeInfo(type).placeholder}
							class="input"
							required
						/>
					</div>
				</div>

				<!-- Label & Icon -->
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="label">Label (optional)</label>
						<input
							type="text"
							bind:value={label}
							placeholder="e.g., 'Work Email' or 'Personal Phone'"
							class="input"
						/>
						<p class="text-xs text-gray-500 mt-1">Custom display label for this contact</p>
					</div>

					<div>
						<label class="label">Icon (optional)</label>
						<input
							type="text"
							bind:value={icon}
							placeholder="Emoji or icon"
							class="input"
							maxlength="4"
						/>
						<p class="text-xs text-gray-500 mt-1">Emoji or icon to display (defaults to type icon)</p>
					</div>
				</div>

				<!-- Protection Level -->
				<div>
					<label class="label">
						Protection Level <span class="text-red-500">*</span>
					</label>
					<div class="space-y-2">
						{#each protectionLevels as level}
							<label class="flex items-start space-x-3 p-3 border rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors {protectionLevel === level.value ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-gray-300 dark:border-gray-700'}">
								<input
									type="radio"
									bind:group={protectionLevel}
									value={level.value}
									class="mt-1"
								/>
								<div class="flex-1">
									<div class="font-medium">{level.label}</div>
									<div class="text-sm text-gray-600 dark:text-gray-400">{level.description}</div>
								</div>
							</label>
						{/each}
					</div>
				</div>

				<!-- View Visibility -->
				<div>
					<label class="label">View Visibility</label>
					<p class="text-sm text-gray-600 dark:text-gray-400 mb-3">
						Select which views should display this contact method. Leave all unchecked to show on all views.
					</p>

					<div class="flex gap-2 mb-3">
						<button type="button" class="btn btn-secondary text-sm" on:click={selectAllViews}>
							Select All
						</button>
						<button type="button" class="btn btn-secondary text-sm" on:click={deselectAllViews}>
							Deselect All
						</button>
					</div>

					{#if views.length === 0}
						<p class="text-sm text-gray-500 italic">No views available. Create a view first.</p>
					{:else}
						<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
							{#each views as view}
								<label class="flex items-center space-x-2 p-2 border rounded cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors {viewVisibility[view.id] ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-gray-300 dark:border-gray-700'}">
									<input
										type="checkbox"
										checked={viewVisibility[view.id] || false}
										on:change={() => toggleViewVisibility(view.id)}
									/>
									<span class="text-sm">{view.name}</span>
									{#if view.is_default}
										<span class="text-xs px-1.5 py-0.5 bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded">
											Default
										</span>
									{/if}
								</label>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Primary & Sort Order -->
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="flex items-center space-x-2">
							<input type="checkbox" bind:checked={isPrimary} />
							<span class="label mb-0">Mark as primary contact</span>
						</label>
						<p class="text-xs text-gray-500 mt-1 ml-6">Primary contacts are highlighted in views</p>
					</div>

					<div>
						<label class="label">Sort Order</label>
						<input
							type="number"
							bind:value={sortOrder}
							placeholder="0"
							class="input"
						/>
						<p class="text-xs text-gray-500 mt-1">Higher numbers appear first</p>
					</div>
				</div>

				<!-- Submit -->
				<div class="flex gap-2 pt-4 border-t">
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{#if saving}
							Saving...
						{:else}
							{editingContact ? 'Update Contact' : 'Add Contact'}
						{/if}
					</button>
					<button type="button" class="btn btn-secondary" on:click={closeForm}>Cancel</button>
				</div>
			</form>
		</div>
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
				/>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No contact methods yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your first contact method with privacy protection settings.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>+ Add Your First Contact</button>
		</div>
	{:else}
		<!-- List View -->
		<div class="space-y-3">
			{#each contacts as contact}
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
									<span class="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded">
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
						</div>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2">
						<button
							class="btn btn-secondary text-sm"
							on:click={() => togglePrimary(contact)}
							title="Toggle primary"
						>
							{contact.is_primary ? '⭐' : '☆'}
						</button>
						<button
							class="btn btn-secondary text-sm"
							on:click={() => openEditForm(contact)}
						>
							Edit
						</button>
						<button
							class="btn btn-secondary text-sm text-red-600 hover:text-red-700"
							on:click={() => deleteContact(contact)}
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
				<li><strong>CSS Obfuscation:</strong> Hides contact from simple scrapers using CSS tricks</li>
				<li><strong>Click to Reveal:</strong> Requires user interaction to display contact</li>
				<li><strong>CAPTCHA:</strong> Cloudflare Turnstile verification required (coming soon)</li>
			</ul>
		</div>
	{/if}
</div>

<style>
	/* Custom styles if needed */
</style>

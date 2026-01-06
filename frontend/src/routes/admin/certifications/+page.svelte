<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type Certification } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import { formatDate } from '$lib/utils';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';

	let certifications: Certification[] = [];
	let loading = true;
	let showForm = false;
	let editingCert: Certification | null = null;

	// Form fields
	let name = '';
	let issuer = '';
	let issueDate = '';
	let expiryDate = '';
	let credentialId = '';
	let credentialUrl = '';
	let visibility = 'public';
	let isDraft = false;
	let sortOrder = 0;
	let saving = false;

	let selectMode = false;
	let selectedIds: Set<string> = new Set();

	onMount(loadCertifications);

	async function loadCertifications() {
		loading = true;
		try {
			const records = await await collection('certifications').getList(1, 100, {
				sort: 'issuer,sort_order,-issue_date'
			});
			certifications = records.items as unknown as Certification[];
		} catch (err) {
			console.error('Failed to load certifications:', err);
			toasts.add('error', 'Failed to load certifications');
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		name = '';
		issuer = '';
		issueDate = '';
		expiryDate = '';
		credentialId = '';
		credentialUrl = '';
		visibility = 'public';
		isDraft = false;
		sortOrder = 0;
		editingCert = null;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(cert: Certification) {
		editingCert = cert;
		name = cert.name;
		issuer = cert.issuer || '';
		issueDate = cert.issue_date ? cert.issue_date.split('T')[0] : '';
		expiryDate = cert.expiry_date ? cert.expiry_date.split('T')[0] : '';
		credentialId = cert.credential_id || '';
		credentialUrl = cert.credential_url || '';
		visibility = cert.visibility;
		isDraft = cert.is_draft;
		sortOrder = cert.sort_order;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	async function handleSubmit() {
		if (!name.trim()) {
			toasts.add('error', 'Certification name is required');
			return;
		}

		saving = true;
		try {
			const data = {
				name: name.trim(),
				issuer: issuer.trim(),
				issue_date: issueDate ? new Date(issueDate).toISOString() : null,
				expiry_date: expiryDate ? new Date(expiryDate).toISOString() : null,
				credential_id: credentialId.trim(),
				credential_url: credentialUrl.trim(),
				visibility,
				is_draft: isDraft,
				sort_order: sortOrder
			};

			if (editingCert) {
				await await collection('certifications').update(editingCert.id, data);
				toasts.add('success', 'Certification updated successfully');
			} else {
				await await collection('certifications').create(data);
				toasts.add('success', 'Certification created successfully');
			}

			closeForm();
			await loadCertifications();
		} catch (err) {
			console.error('Failed to save certification:', err);
			toasts.add('error', 'Failed to save certification');
		} finally {
			saving = false;
		}
	}

	async function deleteCertification(cert: Certification) {
		if (!confirm(`Are you sure you want to delete "${cert.name}"?`)) {
			return;
		}

		try {
			await await collection('certifications').delete(cert.id);
			toasts.add('success', 'Certification deleted');
			await loadCertifications();
		} catch (err) {
			console.error('Failed to delete certification:', err);
			toasts.add('error', 'Failed to delete certification');
		}
	}

	async function togglePublish(cert: Certification) {
		try {
			await await collection('certifications').update(cert.id, {
				is_draft: !cert.is_draft
			});
			toasts.add('success', cert.is_draft ? 'Certification published' : 'Certification unpublished');
			await loadCertifications();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', 'Failed to update certification');
		}
	}

	// Group certifications by issuer
	function groupByIssuer(certs: Certification[]): Map<string, Certification[]> {
		const groups = new Map<string, Certification[]>();
		for (const cert of certs) {
			const issuerKey = cert.issuer || 'Other';
			if (!groups.has(issuerKey)) {
				groups.set(issuerKey, []);
			}
			groups.get(issuerKey)!.push(cert);
		}
		return groups;
	}

	// Check if a certification is expired
	function isExpired(cert: Certification): boolean {
		if (!cert.expiry_date) return false;
		return new Date(cert.expiry_date) < new Date();
	}

	// Check if a certification expires soon (within 30 days)
	function expiresSoon(cert: Certification): boolean {
		if (!cert.expiry_date) return false;
		const expiry = new Date(cert.expiry_date);
		const now = new Date();
		const thirtyDaysFromNow = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
		return expiry > now && expiry <= thirtyDaysFromNow;
	}

	$: groupedCertifications = groupByIssuer(certifications);

	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		if (selectedIds.has(id)) selectedIds.delete(id);
		else selectedIds.add(id);
		selectedIds = selectedIds;
	}

	function selectAll() { selectedIds = new Set(certifications.map(e => e.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		try {
			for (const id of ids) await collection('certifications').update(id, { visibility });
			toasts.add('success', `Updated ${ids.length} items to ${visibility}`);
			selectedIds = new Set();
			selectMode = false;
			await loadCertifications();
		} catch (err) {
			toasts.add('error', 'Failed to update visibility');
		}
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		if (!confirm(`Delete ${ids.length} item(s)?`)) return;
		try {
			for (const id of ids) await collection('certifications').delete(id);
			toasts.add('success', `Deleted ${ids.length} items`);
			selectedIds = new Set();
			selectMode = false;
			await loadCertifications();
		} catch (err) {
			toasts.add('error', 'Failed to delete items');
		}
	}
</script>

<svelte:head>
	<title>Certifications | Facet Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={certifications.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Certifications</h1>
		<div class="flex items-center gap-2">
			{#if certifications.length > 0}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					on:click={toggleSelectMode}
				>
					{selectMode ? 'Cancel' : 'Select'}
				</button>
			{/if}
			<button class="btn btn-primary" on:click={openNewForm}>
				+ New Certification
			</button>
		</div>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading certifications...</div>
		</div>
	{:else if showForm}
		<!-- Certification Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
						{editingCert ? 'Edit Certification' : 'New Certification'}
					</h2>
					<button type="button" class="text-gray-500 hover:text-gray-700" on:click={closeForm}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div>
					<label for="name" class="label">Certification Name *</label>
					<input
						type="text"
						id="name"
						bind:value={name}
						class="input"
						placeholder="AWS Solutions Architect Professional"
						required
					/>
				</div>

				<div>
					<label for="issuer" class="label">Issuing Organization</label>
					<input
						type="text"
						id="issuer"
						bind:value={issuer}
						class="input"
						placeholder="Amazon Web Services"
					/>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label for="issue_date" class="label">Issue Date</label>
						<input
							type="date"
							id="issue_date"
							bind:value={issueDate}
							class="input"
						/>
					</div>

					<div>
						<label for="expiry_date" class="label">Expiry Date</label>
						<input
							type="date"
							id="expiry_date"
							bind:value={expiryDate}
							class="input"
						/>
						<p class="text-xs text-gray-500 mt-1">Leave blank if it doesn't expire</p>
					</div>
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Verification</h2>

				<div>
					<label for="credential_id" class="label">Credential ID</label>
					<input
						type="text"
						id="credential_id"
						bind:value={credentialId}
						class="input"
						placeholder="ABC123XYZ"
					/>
				</div>

				<div>
					<label for="credential_url" class="label">Verification URL</label>
					<input
						type="url"
						id="credential_url"
						bind:value={credentialUrl}
						class="input"
						placeholder="https://www.credly.com/badges/..."
					/>
					<p class="text-xs text-gray-500 mt-1">Link where others can verify this certification</p>
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
						<p class="text-xs text-gray-500 mt-1">Higher numbers appear first within issuer group</p>
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
					{editingCert ? 'Update Certification' : 'Create Certification'}
				</button>
			</div>
		</form>
	{:else if certifications.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No certifications yet</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				Add your professional certifications, licenses, and credentials.
			</p>
			<button class="btn btn-primary" on:click={openNewForm}>
				+ Add Your First Certification
			</button>
		</div>
	{:else}
		<!-- Certifications List - Grouped by Issuer -->
		<div class="space-y-6">
			{#each [...groupedCertifications] as [issuerName, certs] (issuerName)}
				<div>
					<h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-3">
						{issuerName}
					</h2>
					<div class="space-y-3">
						{#each certs as cert (cert.id)}
							<div class="card p-4 {selectMode && selectedIds.has(cert.id) ? 'ring-2 ring-primary-500' : ''}">
								<div class="flex items-start justify-between gap-4">
									{#if selectMode}
										<input
											type="checkbox"
											checked={selectedIds.has(cert.id)}
											on:change={() => toggleSelect(cert.id)}
											class="mt-1 w-5 h-5 text-primary-600 rounded border-gray-300"
										/>
									{/if}
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2 flex-wrap">
											<h3 class="font-medium text-gray-900 dark:text-white">
												{cert.name}
											</h3>
											{#if cert.is_draft}
												<span class="px-2 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
													Draft
												</span>
											{/if}
											{#if cert.visibility !== 'public'}
												<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
													{cert.visibility}
												</span>
											{/if}
											{#if isExpired(cert)}
												<span class="px-2 py-0.5 text-xs bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200 rounded">
													Expired
												</span>
											{:else if expiresSoon(cert)}
												<span class="px-2 py-0.5 text-xs bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 rounded">
													Expires Soon
												</span>
											{/if}
										</div>

										<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
											{#if cert.issue_date}
												<span class="flex items-center gap-1">
													<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
													</svg>
													Issued {formatDate(cert.issue_date)}
												</span>
											{/if}
											{#if cert.expiry_date}
												<span class="flex items-center gap-1">
													<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
													</svg>
													{isExpired(cert) ? 'Expired' : 'Expires'} {formatDate(cert.expiry_date)}
												</span>
											{/if}
											{#if cert.credential_id}
												<span class="flex items-center gap-1">
													<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />
													</svg>
													{cert.credential_id}
												</span>
											{/if}
										</div>

										{#if cert.credential_url}
											<div class="mt-2">
												<a href={cert.credential_url} target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 dark:text-primary-400 hover:underline flex items-center gap-1">
													<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
													</svg>
													Verify Credential
												</a>
											</div>
										{/if}
									</div>

									<div class="flex items-center gap-2">
										<button
											class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
											on:click={() => togglePublish(cert)}
											title={cert.is_draft ? 'Publish' : 'Unpublish'}
										>
											{#if cert.is_draft}
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
											on:click={() => openEditForm(cert)}
											title="Edit"
										>
											<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
											</svg>
										</button>
										<button
											class="p-2 text-gray-500 hover:text-red-600"
											on:click={() => deleteCertification(cert)}
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
				</div>
			{/each}
		</div>
	{/if}
</div>

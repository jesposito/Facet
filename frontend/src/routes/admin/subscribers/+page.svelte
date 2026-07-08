<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';
	import { focusAfterRemove } from '$lib/a11y/focusAfterRemove';

	type Subscriber = {
		id: string;
		email: string;
		name: string;
		source: string;
		status: 'active' | 'unsubscribed' | 'bounced';
		created: string;
	};

	type SourceBreakdown = {
		source: string;
		count: number;
	};

	type SubscriberStats = {
		sources: SourceBreakdown[];
		total_active: number;
		total_unsubscribed: number;
		total_bounced: number;
		growth: { date: string; count: number }[];
	};

	type ImportResult = {
		created: number;
		updated: number;
		skipped_cap: number;
		failed: number;
		errors: { row: number; email: string; error: string }[];
		created_tags: string[];
		unknown_tags: string[];
	};

	let subscribers: Subscriber[] = $state([]);
	let loading = $state(true);
	let stats: SubscriberStats | null = $state(null);
	let search = $state('');
	let statusFilter = $state('');
	let currentPage = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);
	let selectedIds: Set<string> = $state(new Set());
	let selectAll = $state(false);
	let showImport = $state(false);
	let importFile = $state<File | null>(null);
	let importFileInput: HTMLInputElement | undefined = $state();
	let importing = $state(false);
	let importFileError = $state('');
	let importResult: ImportResult | null = $state(null);

	const perPage = 50;

	// Detect if filters are active
	let hasFilters = $derived(search !== '' || statusFilter !== '');

	// Debounce + AbortController for search
	let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
	let abortController: AbortController | null = null;

	onMount(async () => {
		await Promise.all([loadSubscribers(), loadStats()]);
	});

	async function loadStats() {
		try {
			const response = await fetch('/api/admin/subscribers/stats', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (response.ok) {
				stats = await response.json();
			}
		} catch (err) {
			console.debug('Failed to load subscriber stats:', err);
		}
	}

	onDestroy(() => {
		if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
		if (abortController) abortController.abort();
	});

	async function loadSubscribers() {
		// Cancel any in-flight request
		if (abortController) abortController.abort();
		abortController = new AbortController();

		loading = true;
		try {
			const params = new URLSearchParams({
				page: String(currentPage),
				perPage: String(perPage)
			});
			if (search) params.set('search', search);
			if (statusFilter) params.set('status', statusFilter);

			const response = await fetch(`/api/admin/subscribers?${params}`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
				signal: abortController.signal
			});
			if (!response.ok) throw new Error('Failed to load subscribers');
			const data = await response.json();
			subscribers = data.data || [];
			totalPages = data.totalPages || 1;
			totalItems = data.totalItems || 0;
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') return;
			console.error('Failed to load subscribers:', err);
			toasts.add('error', $t('admin.subscribers.error_load'));
		} finally {
			loading = false;
		}
	}

	function handleSearch() {
		// 300ms debounce on search input
		if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
		searchDebounceTimer = setTimeout(() => {
			currentPage = 1;
			selectedIds = new Set();
			selectAll = false;
			loadSubscribers();
		}, 300);
	}

	function setStatusFilter(status: string) {
		statusFilter = status;
		currentPage = 1;
		selectedIds = new Set();
		selectAll = false;
		loadSubscribers();
	}

	function clearFilters() {
		search = '';
		statusFilter = '';
		currentPage = 1;
		selectedIds = new Set();
		selectAll = false;
		loadSubscribers();
	}

	function goToPage(page: number) {
		currentPage = page;
		selectedIds = new Set();
		selectAll = false;
		loadSubscribers();
	}

	function toggleSelectAll() {
		if (selectAll) {
			selectedIds = new Set();
			selectAll = false;
		} else {
			selectedIds = new Set(subscribers.map((s) => s.id));
			selectAll = true;
		}
	}

	function toggleSelect(id: string) {
		const next = new Set(selectedIds);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		selectedIds = next;
		selectAll = next.size === subscribers.length && subscribers.length > 0;
	}

	async function deleteSubscriber(sub: Subscriber) {
		const confirmed = await confirm({
			title: $t('admin.subscribers.delete_title'),
			message: $t('admin.subscribers.delete_message', { values: { email: sub.email } }),
			confirmText: $t('admin.subscribers.delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		try {
			const deletedIndex = subscribers.findIndex((s) => s.id === sub.id);
			const response = await fetch(`/api/admin/subscribers/${sub.id}`, {
				method: 'DELETE',
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!response.ok) throw new Error('Failed to delete subscriber');
			toasts.add('success', $t('admin.subscribers.deleted'));
			await loadSubscribers();
			focusAfterRemove({
				deletedIndex,
				remainingIds: subscribers.map((s) => s.id),
				selectRow: (id) => document.querySelector<HTMLElement>(`[data-row-id="${id}"]`)
			});
		} catch (err) {
			console.error('Failed to delete subscriber:', err);
			toasts.add('error', $t('admin.subscribers.error_delete'));
		}
	}

	async function bulkDelete() {
		if (selectedIds.size === 0) return;
		const confirmed = await confirm({
			title: $t('admin.subscribers.bulk_delete_title'),
			message: $t('admin.subscribers.bulk_delete_message', { values: { count: selectedIds.size } }),
			confirmText: $t('admin.subscribers.delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		try {
			const response = await fetch('/api/admin/subscribers/bulk', {
				method: 'DELETE',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({ ids: [...selectedIds] })
			});
			if (!response.ok) throw new Error('Failed to bulk delete');
			const data = await response.json();
			toasts.add('success', data.message || $t('admin.subscribers.deleted'));
			selectedIds = new Set();
			selectAll = false;
			await loadSubscribers();
		} catch (err) {
			console.error('Failed to bulk delete:', err);
			toasts.add('error', $t('admin.subscribers.error_delete'));
		}
	}

	// CSV export passes current filters, detects empty results
	async function exportCSV() {
		try {
			const params = new URLSearchParams();
			if (search) params.set('search', search);
			if (statusFilter) params.set('status', statusFilter);

			const response = await fetch(`/api/admin/subscribers/export?${params}`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!response.ok) throw new Error('Failed to export');

			// Check Content-Type — JSON means empty results
			const contentType = response.headers.get('Content-Type') || '';
			if (contentType.includes('application/json')) {
				toasts.add('info', $t('admin.subscribers.export_empty'));
				return;
			}

			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'subscribers.csv';
			a.click();
			URL.revokeObjectURL(url);
		} catch (err) {
			console.error('Failed to export:', err);
			toasts.add('error', $t('admin.subscribers.error_export'));
		}
	}

	function onImportFileChange(event: Event) {
		importFile = (event.currentTarget as HTMLInputElement).files?.[0] ?? null;
		importFileError = '';
		importResult = null;
	}

	async function submitImport() {
		if (!importFile) {
			importFileError = $t('admin.subscribers.import_error_no_file');
			importFileInput?.focus();
			return;
		}
		if (!/\.csv$/i.test(importFile.name)) {
			importFileError = $t('admin.subscribers.import_error_wrong_type');
			importFileInput?.focus();
			return;
		}

		importing = true;
		importFileError = '';
		importResult = null;
		try {
			const form = new FormData();
			form.append('file', importFile);
			const response = await fetch('/api/admin/subscribers/import', {
				method: 'POST',
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
				body: form
			});
			const data = await response.json().catch(() => null);
			if (!response.ok || !data) {
				throw new Error(data?.error || $t('admin.subscribers.import_failed_all'));
			}

			importResult = {
				created: data.created ?? 0,
				updated: data.updated ?? 0,
				skipped_cap: data.skipped_cap ?? 0,
				failed: data.failed ?? 0,
				errors: data.errors ?? [],
				created_tags: data.created_tags ?? [],
				unknown_tags: data.unknown_tags ?? []
			};

			toasts.add(
				importResult.created + importResult.updated > 0 ? 'success' : 'info',
				$t('admin.subscribers.import_result_summary', {
					values: {
						created: importResult.created,
						updated: importResult.updated,
						skipped: importResult.skipped_cap,
						failed: importResult.failed
					}
				})
			);
			await Promise.all([loadSubscribers(), loadStats()]);
		} catch (err) {
			const message = err instanceof Error ? err.message : $t('admin.subscribers.import_failed_all');
			importFileError = message;
			toasts.add('error', message);
		} finally {
			importing = false;
		}
	}

	// Status badge colors including bounced
	function getStatusClass(status: string): string {
		if (status === 'active')
			return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300';
		if (status === 'bounced')
			return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300';
		return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400';
	}

	function getStatusLabel(status: string): string {
		if (status === 'active') return $t('admin.subscribers.status_active');
		if (status === 'bounced') return $t('admin.subscribers.status_bounced');
		return $t('admin.subscribers.status_unsubscribed');
	}
</script>

<svelte:head>
	<title>{$t('admin.subscribers.page_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
				{$t('admin.subscribers.page_title')}
			</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
				{$t('admin.subscribers.description')}
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2 w-full sm:w-auto">
			<a href="/admin/subscribers/compose" class="btn btn-primary w-full sm:w-auto">
				{$t('admin.subscribers.compose')}
			</a>
			<button type="button" class="btn btn-secondary w-full sm:w-auto" onclick={() => (showImport = !showImport)}>
				{$t('admin.subscribers.import')}
			</button>
			<button type="button" class="btn btn-secondary w-full sm:w-auto" onclick={exportCSV}>
				{$t('admin.subscribers.export')}
			</button>
		</div>
	</div>

	{#if showImport}
		<div class="card p-4 space-y-4">
			<div class="flex flex-col sm:flex-row gap-3 sm:items-end">
				<div class="flex-1">
					<label for="csv-import-file" class="label">
						{$t('admin.subscribers.import_file_label')}
					</label>
					<input
						bind:this={importFileInput}
						id="csv-import-file"
						type="file"
						accept=".csv,text/csv"
						onchange={onImportFileChange}
						class="input text-sm w-full"
						aria-invalid={importFileError ? 'true' : undefined}
					/>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
						{$t('admin.subscribers.import_file_hint')}
					</p>
				</div>
				<button
					type="button"
					class="btn btn-primary"
					onclick={submitImport}
					disabled={importing}
					aria-busy={importing ? 'true' : undefined}
				>
					{importing
						? $t('admin.subscribers.import_button_importing')
						: $t('admin.subscribers.import_submit')}
				</button>
			</div>

			{#if importFileError}
				<p class="text-sm text-red-600 dark:text-red-400">{importFileError}</p>
			{/if}

			{#if importResult}
				<div class="rounded-lg border border-gray-200 dark:border-gray-700 p-3 text-sm">
					<p class="font-medium text-gray-900 dark:text-white">
						{$t('admin.subscribers.import_result_summary', {
							values: {
								created: importResult.created,
								updated: importResult.updated,
								skipped: importResult.skipped_cap,
								failed: importResult.failed
							}
						})}
					</p>
					{#if importResult.created_tags.length > 0}
						<p class="mt-2 text-gray-600 dark:text-gray-300">
							{$t('admin.subscribers.import_created_tags', {
								values: { tags: importResult.created_tags.join(', ') }
							})}
						</p>
					{/if}
					{#if importResult.unknown_tags.length > 0}
						<p class="mt-2 text-gray-600 dark:text-gray-300">
							{$t('admin.subscribers.import_unknown_tags', {
								values: { tags: importResult.unknown_tags.join(', ') }
							})}
						</p>
					{/if}
					{#if importResult.errors.length > 0}
						<p class="mt-3 text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
							{$t('admin.subscribers.import_errors_heading', {
								values: { count: importResult.errors.length }
							})}
						</p>
						<ul class="mt-1 max-h-32 overflow-y-auto text-xs text-gray-600 dark:text-gray-300">
							{#each importResult.errors as rowError (rowError.row)}
								<li>
									{$t('admin.subscribers.import_error_row', {
										values: {
											row: rowError.row,
											error: rowError.error,
											email: rowError.email
										}
									})}
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Stats Overview -->
	{#if stats}
		<div class="grid grid-cols-3 gap-4">
			<div class="card p-4 text-center">
				<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_active}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.subscribers.status_active')}</p>
			</div>
			<div class="card p-4 text-center">
				<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_unsubscribed}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.subscribers.status_unsubscribed')}</p>
			</div>
			<div class="card p-4 text-center">
				<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_bounced}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.subscribers.status_bounced')}</p>
			</div>
		</div>

		<!-- Source Breakdown -->
		{#if stats.sources && stats.sources.length > 0}
			{@const maxCount = Math.max(...stats.sources.map((s) => s.count))}
			<div class="card p-4">
				<h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">
					{$t('admin.subscribers.source_breakdown')}
				</h3>
				<div class="space-y-2">
					{#each stats.sources as src}
						<div class="flex items-center gap-3">
							<span class="text-xs text-gray-600 dark:text-gray-400 w-24 truncate" title={src.source}
								>{src.source}</span
							>
							<div class="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden">
								<div
									class="bg-primary-500 h-full rounded-full transition-all"
									style="width: {(src.count / maxCount) * 100}%"
								></div>
							</div>
							<span class="text-xs font-medium text-gray-700 dark:text-gray-300 w-8 text-right"
								>{src.count}</span
							>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}

	<!-- Search + Filters -->
	<div class="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
		<div class="flex-1 w-full sm:w-auto">
			<label for="subscriber-search" class="sr-only"
				>{$t('admin.subscribers.search_placeholder')}</label
			>
			<input
				type="search"
				id="subscriber-search"
				bind:value={search}
				oninput={handleSearch}
				class="input text-sm w-full"
				placeholder={$t('admin.subscribers.search_placeholder')}
			/>
		</div>
		<div
			class="flex items-center gap-1"
			role="group"
			aria-label={$t('admin.subscribers.filter_label')}
		>
			<button
				type="button"
				class="px-3 py-1.5 text-xs rounded-full transition-colors {statusFilter === ''
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
				aria-pressed={statusFilter === ''}
				onclick={() => setStatusFilter('')}
			>
				{$t('admin.subscribers.filter_all')}
			</button>
			<button
				type="button"
				class="px-3 py-1.5 text-xs rounded-full transition-colors {statusFilter === 'active'
					? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
					: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
				aria-pressed={statusFilter === 'active'}
				onclick={() => setStatusFilter('active')}
			>
				{$t('admin.subscribers.filter_active')}
			</button>
			<button
				type="button"
				class="px-3 py-1.5 text-xs rounded-full transition-colors {statusFilter === 'unsubscribed'
					? 'bg-gray-200 text-gray-700 dark:bg-gray-600 dark:text-gray-300'
					: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
				aria-pressed={statusFilter === 'unsubscribed'}
				onclick={() => setStatusFilter('unsubscribed')}
			>
				{$t('admin.subscribers.filter_unsubscribed')}
			</button>
			<button
				type="button"
				class="px-3 py-1.5 text-xs rounded-full transition-colors {statusFilter === 'bounced'
					? 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300'
					: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
				aria-pressed={statusFilter === 'bounced'}
				onclick={() => setStatusFilter('bounced')}
			>
				{$t('admin.subscribers.filter_bounced')}
			</button>
		</div>
	</div>

	<!-- Bulk Actions -->
	{#if selectedIds.size > 0}
		<div
			class="flex items-center gap-3 p-3 bg-primary-50 dark:bg-primary-900/20 rounded-lg border border-primary-200 dark:border-primary-800"
			role="status"
		>
			<span class="text-sm text-primary-700 dark:text-primary-300">
				{$t('admin.subscribers.selected_count', { values: { count: selectedIds.size } })}
			</span>
			<button type="button" class="btn btn-danger btn-sm" onclick={bulkDelete}>
				{$t('admin.subscribers.bulk_delete')}
			</button>
		</div>
	{/if}

	<!-- Subscriber List -->
	<div class="card">
		{#if loading}
			<div
				class="flex items-center justify-center py-12"
				role="status"
				aria-label={$t('admin.subscribers.loading')}
			>
				<svg
					class="animate-spin h-6 w-6 text-primary-600"
					fill="none"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
			</div>
		{:else if subscribers.length === 0}
			<div class="text-center py-12 px-6">
				<svg
					class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="1.5"
						d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
					/>
				</svg>
				{#if hasFilters}
					<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
						{$t('admin.subscribers.empty_filtered_title')}
					</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
						{$t('admin.subscribers.empty_filtered_description')}
					</p>
					<button type="button" class="btn btn-secondary btn-sm" onclick={clearFilters}>
						{$t('admin.subscribers.clear_filters')}
					</button>
				{:else}
					<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
						{$t('admin.subscribers.empty_title')}
					</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
						{$t('admin.subscribers.empty_description')}
					</p>
					<a href="/admin/homepage" class="btn btn-primary btn-sm">
						{$t('admin.subscribers.empty_cta')}
					</a>
				{/if}
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm" aria-label={$t('admin.subscribers.page_title')}>
					<thead>
						<tr class="border-b border-gray-200 dark:border-gray-700">
							<th class="py-3 px-4 w-8" scope="col">
								<input
									type="checkbox"
									checked={selectAll}
									onchange={toggleSelectAll}
									aria-label={$t('admin.subscribers.select_all')}
									class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
								/>
							</th>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.subscribers.col_email')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.subscribers.col_name')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.subscribers.col_source')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.subscribers.col_status')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.subscribers.col_date')}</th
							>
							<th
								class="text-right py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.subscribers.col_actions')}</th
							>
						</tr>
					</thead>
					<tbody>
						{#each subscribers as sub (sub.id)}
							<tr
								class="border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50"
								data-row-id={sub.id}
								tabindex="-1"
							>
								<td class="py-3 px-4">
									<input
										type="checkbox"
										checked={selectedIds.has(sub.id)}
										onchange={() => toggleSelect(sub.id)}
										aria-label={$t('admin.subscribers.select_subscriber', {
											values: { email: sub.email }
										})}
										class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
									/>
								</td>
								<td class="py-3 px-4 text-gray-900 dark:text-white font-medium">{sub.email}</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">{sub.name || '\u2014'}</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400 capitalize">{sub.source}</td>
								<td class="py-3 px-4">
									<span
										class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {getStatusClass(sub.status)}"
									>
										{getStatusLabel(sub.status)}
									</span>
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400"
									>{new Date(sub.created).toLocaleDateString()}</td
								>
								<td class="py-3 px-4 text-right">
									<button
										type="button"
										class="p-1.5 text-gray-400 hover:text-red-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
										onclick={() => deleteSubscriber(sub)}
										aria-label={$t('admin.subscribers.delete_aria', { values: { email: sub.email } })}
									>
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
											/>
										</svg>
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Pagination -->
			{#if totalPages > 1}
				<nav
					class="flex items-center justify-between px-4 py-3 border-t border-gray-200 dark:border-gray-700"
					aria-label={$t('admin.subscribers.pagination')}
				>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						{$t('admin.subscribers.showing', {
							values: {
								from: (currentPage - 1) * perPage + 1,
								to: Math.min(currentPage * perPage, totalItems),
								total: totalItems
							}
						})}
					</p>
					<div class="flex gap-1">
						<button
							type="button"
							class="btn btn-ghost btn-sm"
							disabled={currentPage <= 1}
							onclick={() => goToPage(currentPage - 1)}
							aria-label={$t('admin.subscribers.prev_page')}
						>
							&laquo;
						</button>
						{#each Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
							const start = Math.max(1, Math.min(currentPage - 2, totalPages - 4));
							return start + i;
						}) as p (p)}
							<button
								type="button"
								class="btn btn-sm {p === currentPage ? 'btn-primary' : 'btn-ghost'}"
								onclick={() => goToPage(p)}
								aria-current={p === currentPage ? 'page' : undefined}
							>
								{p}
							</button>
						{/each}
						<button
							type="button"
							class="btn btn-ghost btn-sm"
							disabled={currentPage >= totalPages}
							onclick={() => goToPage(currentPage + 1)}
							aria-label={$t('admin.subscribers.next_page')}
						>
							&raquo;
						</button>
					</div>
				</nav>
			{/if}
		{/if}
	</div>
</div>

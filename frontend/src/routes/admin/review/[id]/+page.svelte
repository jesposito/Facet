<script lang="ts">
	import { page } from '$app/state';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { goto } from '$app/navigation';
	import { onMount, tick } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';

	let loading = $state(true);
	let applying = $state(false);
	let proposal: Record<string, unknown> | null = $state(null);
	let proposedData: Record<string, unknown> = $state({});
	let diff: Record<string, { type: string; old?: unknown; new?: unknown }> = $state({});

	// Field decisions
	let fieldDecisions: Record<string, 'apply' | 'ignore' | 'lock'> = $state({});
	let fieldEdits: Record<string, unknown> = $state({});
	let fieldEditText: Record<string, string> = $state({});
	let fieldEditErrors: Record<string, string> = $state({});
	// Shown beside the Apply button. A toast here would sit over the button
	// the admin needs to press again once the edit is fixed.
	let applyError = $state('');

	const fieldLabels: Record<string, string> = {
		title: 'Title',
		summary: 'Summary',
		description: 'Description',
		tech_stack: 'Tech Stack',
		categories: 'Categories',
		links: 'Links'
	};

	onMount(async () => {
		try {
			const proposalId = page.params.id;
			if (!proposalId) {
				toasts.add('error', 'Missing proposal ID');
				goto('/admin/import', { replace: true });
				return;
			}

			proposal = await pb.collection('import_proposals').getOne(proposalId);

			if (!proposal || proposal.status !== 'pending') {
				toasts.add('info', 'This proposal has already been processed');
				goto('/admin/import', { replace: true });
				return;
			}

			// PocketBase returns JSON fields as parsed objects, not strings
			// Handle both cases for safety
			if (typeof proposal.proposed_data === 'string') {
				proposedData = JSON.parse(proposal.proposed_data || '{}');
			} else {
				proposedData = (proposal.proposed_data as Record<string, unknown>) || {};
			}

			if (proposal.diff) {
				if (typeof proposal.diff === 'string') {
					diff = JSON.parse(proposal.diff);
				} else {
					diff = proposal.diff as Record<string, { type: string; old?: unknown; new?: unknown }>;
				}
			}

			// Initialize decisions - default to apply for all fields
			for (const field of Object.keys(proposedData)) {
				fieldDecisions[field] = 'apply';
				fieldEdits[field] = proposedData[field];
				const value = proposedData[field];
				if (Array.isArray(value)) {
					fieldEditText[field] = isObjectArrayField(field, value)
						? JSON.stringify(value, null, 2)
						: value.join(', ');
				}
			}
		} catch (err) {
			console.error('Failed to load proposal:', err);
			toasts.add('error', 'Failed to load proposal');
		} finally {
			loading = false;
		}
	});

	function setAllDecisions(decision: 'apply' | 'ignore') {
		for (const field of Object.keys(fieldDecisions)) {
			fieldDecisions[field] = decision;
		}
		fieldDecisions = { ...fieldDecisions };
	}

	function isObjectArrayField(field: string, value: unknown[]): boolean {
		return (
			field === 'links' ||
			(value.length > 0 &&
				value[0] !== null &&
				typeof value[0] === 'object' &&
				!Array.isArray(value[0]))
		);
	}

	function parseArrayEdit(field: string, original: unknown[]): unknown[] {
		const raw = fieldEditText[field] || '';
		if (!isObjectArrayField(field, original)) {
			return raw.split(',').map((item) => item.trim());
		}
		try {
			const parsed: unknown = JSON.parse(raw);
			if (
				!Array.isArray(parsed) ||
				parsed.some((item) => item === null || typeof item !== 'object' || Array.isArray(item))
			) {
				throw new Error('invalid shape');
			}
			return parsed;
		} catch {
			throw new Error(`${fieldLabels[field] || field} must be a valid JSON array of objects.`);
		}
	}

	async function handleApply() {
		applying = true;
		try {
			const appliedFields: Record<string, boolean> = {};
			const lockedFields: string[] = [];
			const edits: Record<string, unknown> = {};
			let hasInvalidEdits = false;

			for (const [field, decision] of Object.entries(fieldDecisions)) {
				if (decision === 'apply' || decision === 'lock') {
					appliedFields[field] = true;
					const original = proposedData[field];
					if (Array.isArray(original)) {
						try {
							edits[field] = parseArrayEdit(field, original);
						} catch (err) {
							fieldEditErrors[field] = err instanceof Error ? err.message : 'Invalid field edit';
							hasInvalidEdits = true;
						}
					} else {
						edits[field] = fieldEdits[field];
					}
				}
				if (decision === 'lock') {
					lockedFields.push(field);
				}
			}

			if (hasInvalidEdits) {
				applyError = 'Correct the highlighted field edits before applying.';
				await tick();
				document.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
				return;
			}
			applyError = '';

			const response = await fetch(`/api/proposals/${page.params.id}/apply`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					applied_fields: appliedFields,
					locked_fields: lockedFields,
					edits
				})
			});

			if (!response.ok) {
				throw new Error('Failed to apply proposal');
			}

			toasts.add('success', 'Changes applied successfully!');
			goto(`/admin/projects`, { replace: true });
		} catch (err) {
			console.error('Failed to apply proposal:', err);
			toasts.add('error', 'Failed to apply changes');
		} finally {
			applying = false;
		}
	}

	async function handleReject() {
		try {
			const response = await fetch(`/api/proposals/${page.params.id}/reject`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${pb.authStore.token}`
				}
			});

			if (!response.ok) {
				throw new Error('Failed to reject proposal');
			}

			toasts.add('info', 'Proposal rejected');
			goto('/admin/import', { replace: true });
		} catch {
			toasts.add('error', 'Failed to reject proposal');
		}
	}

	function formatValue(value: unknown): string {
		if (Array.isArray(value)) {
			if (value.length > 0 && typeof value[0] === 'object') {
				return JSON.stringify(value, null, 2);
			}
			return value.join(', ');
		}
		if (typeof value === 'object' && value !== null) {
			return JSON.stringify(value, null, 2);
		}
		return String(value || '-');
	}

	function updateArrayEdit(field: string, value: string) {
		fieldEditText[field] = value;
		delete fieldEditErrors[field];
		if (Object.keys(fieldEditErrors).length === 0) applyError = '';
	}
</script>

<svelte:head>
	<title>{$t('admin.review.page_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Review Import</h1>
		{#if proposal?.ai_enriched}
			<span
				class="px-3 py-1 text-sm bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300 rounded-full"
			>
				AI Enhanced
			</span>
		{/if}
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading proposal...</div>
		</div>
	{:else if proposal}
		<!-- Bulk actions -->
		<div class="card p-4 mb-6 flex items-center justify-between">
			<span class="text-sm text-gray-600 dark:text-gray-400">
				Review each field and choose what to apply
			</span>
			<div class="flex gap-2">
				<button class="btn btn-sm btn-secondary" onclick={() => setAllDecisions('apply')}>
					Apply All
				</button>
				<button class="btn btn-sm btn-secondary" onclick={() => setAllDecisions('ignore')}>
					Ignore All
				</button>
			</div>
		</div>

		<!-- Field-by-field review -->
		<div class="space-y-4">
			{#each Object.entries(proposedData) as [field, value]}
				<div class="card p-4">
					<div class="flex items-start justify-between mb-3">
						<div>
							<h3 class="font-medium text-gray-900 dark:text-white">
								{fieldLabels[field] || field}
							</h3>
							{#if diff[field]}
								<span
									class="text-xs px-2 py-0.5 rounded bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
								>
									{diff[field].type === 'added' ? 'New' : 'Changed'}
								</span>
							{/if}
						</div>

						<div class="flex gap-1">
							<button
								class="px-3 py-1 text-sm rounded {fieldDecisions[field] === 'apply'
									? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}"
								onclick={() => (fieldDecisions[field] = 'apply')}
							>
								Apply
							</button>
							<button
								class="px-3 py-1 text-sm rounded {fieldDecisions[field] === 'ignore'
									? 'bg-gray-300 text-gray-700 dark:bg-gray-600 dark:text-gray-200'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}"
								onclick={() => (fieldDecisions[field] = 'ignore')}
							>
								Ignore
							</button>
							<button
								class="px-3 py-1 text-sm rounded inline-flex items-center gap-1 {fieldDecisions[
									field
								] === 'lock'
									? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}"
								onclick={() => (fieldDecisions[field] = 'lock')}
								title="Apply and lock (won't be overwritten on refresh)"
							>
								{@html icon('lock')} Lock
							</button>
						</div>
					</div>

					<!-- Show diff if exists -->
					{#if diff[field] && diff[field].type === 'changed'}
						<div class="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
							<div class="p-3 bg-red-50 dark:bg-red-900/20 rounded">
								<span class="text-xs text-red-700 dark:text-red-400 font-medium">Current</span>
								<pre class="mt-1 text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{formatValue(
										diff[field].old
									)}</pre>
							</div>
							<div class="p-3 bg-green-50 dark:bg-green-900/20 rounded">
								<span class="text-xs text-green-700 dark:text-green-400 font-medium">Proposed</span>
								<pre class="mt-1 text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{formatValue(
										value
									)}</pre>
							</div>
						</div>
					{:else}
						<div class="p-3 bg-gray-50 dark:bg-gray-800 rounded">
							<pre
								class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{formatValue(
									value
								)}</pre>
						</div>
					{/if}

					<!-- Editable if applying -->
					{#if fieldDecisions[field] === 'apply' || fieldDecisions[field] === 'lock'}
						<div class="mt-3">
							<span class="text-xs text-gray-500 dark:text-gray-400">Edit before applying:</span>
							{#if Array.isArray(value)}
								{#if isObjectArrayField(field, value)}
									<textarea
										class="input mt-1 min-h-[120px] font-mono text-sm"
										value={fieldEditText[field]}
										aria-label={`Edit ${field} as JSON`}
										aria-invalid={fieldEditErrors[field] ? 'true' : undefined}
										aria-describedby={fieldEditErrors[field] ? `${field}-edit-error` : undefined}
										oninput={(e) => updateArrayEdit(field, e.currentTarget.value)}></textarea>
								{:else}
									<input
										type="text"
										class="input mt-1"
										value={fieldEditText[field]}
										aria-label={`Edit ${field}`}
										oninput={(e) => updateArrayEdit(field, e.currentTarget.value)}
									/>
								{/if}
								{#if fieldEditErrors[field]}
									<p
										id={`${field}-edit-error`}
										class="mt-1 text-sm text-red-600 dark:text-red-400"
										role="alert"
									>
										{fieldEditErrors[field]}
									</p>
								{/if}
							{:else if typeof value === 'string' && value.length > 100}
								<textarea class="input mt-1 min-h-[100px]" bind:value={fieldEdits[field]}
								></textarea>
							{:else}
								<input type="text" class="input mt-1" bind:value={fieldEdits[field]} />
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Actions -->
		{#if applyError}
			<p class="mt-8 text-sm text-red-600 dark:text-red-400 text-right" role="alert">
				{applyError}
			</p>
		{/if}
		<div class="flex justify-between {applyError ? 'mt-3' : 'mt-8'}">
			<button class="btn btn-danger-ghost" onclick={handleReject}> Reject All </button>
			<div class="flex gap-3">
				<a href="/admin/import" class="btn btn-secondary">Cancel</a>
				<button class="btn btn-primary" onclick={handleApply} disabled={applying}>
					{#if applying}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					{/if}
					Apply Selected Changes
				</button>
			</div>
		</div>
	{/if}
</div>

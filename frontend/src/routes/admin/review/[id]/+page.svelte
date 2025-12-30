<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';

	let loading = true;
	let applying = false;
	let proposal: Record<string, unknown> | null = null;
	let proposedData: Record<string, unknown> = {};
	let diff: Record<string, { type: string; old?: unknown; new?: unknown }> = {};

	// Field decisions
	let fieldDecisions: Record<string, 'apply' | 'ignore' | 'lock'> = {};
	let fieldEdits: Record<string, unknown> = {};

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
			proposal = await pb.collection('import_proposals').getOne($page.params.id);

			if (proposal.status !== 'pending') {
				toasts.add('info', 'This proposal has already been processed');
				goto('/admin/proposals');
				return;
			}

			proposedData = JSON.parse((proposal.proposed_data as string) || '{}');
			if (proposal.diff) {
				diff = JSON.parse(proposal.diff as string);
			}

			// Initialize decisions - default to apply for all fields
			for (const field of Object.keys(proposedData)) {
				fieldDecisions[field] = 'apply';
				fieldEdits[field] = proposedData[field];
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

	async function handleApply() {
		applying = true;
		try {
			const appliedFields: Record<string, boolean> = {};
			const lockedFields: string[] = [];
			const edits: Record<string, unknown> = {};

			for (const [field, decision] of Object.entries(fieldDecisions)) {
				if (decision === 'apply' || decision === 'lock') {
					appliedFields[field] = true;
					edits[field] = fieldEdits[field];
				}
				if (decision === 'lock') {
					lockedFields.push(field);
				}
			}

			const response = await fetch(`/api/proposals/${$page.params.id}/apply`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
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

			const result = await response.json();
			toasts.add('success', 'Changes applied successfully!');
			goto(`/admin/projects`);
		} catch (err) {
			console.error('Failed to apply proposal:', err);
			toasts.add('error', 'Failed to apply changes');
		} finally {
			applying = false;
		}
	}

	async function handleReject() {
		try {
			const response = await fetch(`/api/proposals/${$page.params.id}/reject`, {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token
				}
			});

			if (!response.ok) {
				throw new Error('Failed to reject proposal');
			}

			toasts.add('info', 'Proposal rejected');
			goto('/admin/proposals');
		} catch (err) {
			toasts.add('error', 'Failed to reject proposal');
		}
	}

	function formatValue(value: unknown): string {
		if (Array.isArray(value)) {
			return value.join(', ');
		}
		if (typeof value === 'object' && value !== null) {
			return JSON.stringify(value, null, 2);
		}
		return String(value || '-');
	}
</script>

<svelte:head>
	<title>Review Import | Admin</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Review Import</h1>
		{#if proposal?.ai_enriched}
			<span class="px-3 py-1 text-sm bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300 rounded-full">
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
				<button class="btn btn-sm btn-secondary" on:click={() => setAllDecisions('apply')}>
					Apply All
				</button>
				<button class="btn btn-sm btn-secondary" on:click={() => setAllDecisions('ignore')}>
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
								<span class="text-xs px-2 py-0.5 rounded bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300">
									{diff[field].type === 'added' ? 'New' : 'Changed'}
								</span>
							{/if}
						</div>

						<div class="flex gap-1">
							<button
								class="px-3 py-1 text-sm rounded {fieldDecisions[field] === 'apply'
									? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}"
								on:click={() => (fieldDecisions[field] = 'apply')}
							>
								Apply
							</button>
							<button
								class="px-3 py-1 text-sm rounded {fieldDecisions[field] === 'ignore'
									? 'bg-gray-300 text-gray-700 dark:bg-gray-600 dark:text-gray-200'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}"
								on:click={() => (fieldDecisions[field] = 'ignore')}
							>
								Ignore
							</button>
							<button
								class="px-3 py-1 text-sm rounded {fieldDecisions[field] === 'lock'
									? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'}"
								on:click={() => (fieldDecisions[field] = 'lock')}
								title="Apply and lock (won't be overwritten on refresh)"
							>
								🔒 Lock
							</button>
						</div>
					</div>

					<!-- Show diff if exists -->
					{#if diff[field] && diff[field].type === 'changed'}
						<div class="grid grid-cols-2 gap-4 text-sm">
							<div class="p-3 bg-red-50 dark:bg-red-900/20 rounded">
								<span class="text-xs text-red-600 dark:text-red-400 font-medium">Current</span>
								<pre class="mt-1 text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{formatValue(diff[field].old)}</pre>
							</div>
							<div class="p-3 bg-green-50 dark:bg-green-900/20 rounded">
								<span class="text-xs text-green-600 dark:text-green-400 font-medium">Proposed</span>
								<pre class="mt-1 text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{formatValue(value)}</pre>
							</div>
						</div>
					{:else}
						<div class="p-3 bg-gray-50 dark:bg-gray-800 rounded">
							<pre class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{formatValue(value)}</pre>
						</div>
					{/if}

					<!-- Editable if applying -->
					{#if fieldDecisions[field] === 'apply' || fieldDecisions[field] === 'lock'}
						<div class="mt-3">
							<label class="text-xs text-gray-500 dark:text-gray-400">Edit before applying:</label>
							{#if Array.isArray(value)}
								<input
									type="text"
									class="input mt-1"
									value={fieldEdits[field] ? (fieldEdits[field] as string[]).join(', ') : ''}
									on:input={(e) => (fieldEdits[field] = e.currentTarget.value.split(',').map((s) => s.trim()))}
								/>
							{:else if typeof value === 'string' && value.length > 100}
								<textarea
									class="input mt-1 min-h-[100px]"
									bind:value={fieldEdits[field]}
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
		<div class="flex justify-between mt-8">
			<button class="btn btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20" on:click={handleReject}>
				Reject All
			</button>
			<div class="flex gap-3">
				<a href="/admin/import" class="btn btn-secondary">Cancel</a>
				<button class="btn btn-primary" on:click={handleApply} disabled={applying}>
					{#if applying}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					Apply Selected Changes
				</button>
			</div>
		</div>
	{/if}
</div>

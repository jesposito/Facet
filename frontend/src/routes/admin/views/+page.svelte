<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import { formatDate } from '$lib/utils';

	let loading = $state(true);
	let views: Array<Record<string, unknown>> = $state([]);

	// Simple pattern - admin layout handles auth
	onMount(loadViews);

	async function loadViews() {
		loading = true;
		try {
			// Sort by id only - is_default field may not exist in older schemas
			const result = await collection('views').getList(1, 50, {
				sort: '-id'
			});
			// Reorder to put the default view first
			const items = result.items;
			const defaultIndex = items.findIndex((v) => v.is_default);
			if (defaultIndex > 0) {
				const [defaultView] = items.splice(defaultIndex, 1);
				items.unshift(defaultView);
			}
			views = items;
		} catch (err) {
			console.error('Failed to load facets:', err);
			toasts.add('error', 'Failed to load facets');
		} finally {
			loading = false;
		}
	}

	async function toggleActive(view: Record<string, unknown>) {
		try {
			await collection('views').update(view.id as string, {
				is_active: !view.is_active
			});
			await loadViews();
		} catch (err) {
			toasts.add('error', 'Failed to update facet');
		}
	}

	async function deleteView(id: string) {
		const confirmed = await confirm({
			title: 'Delete Facet',
			message: 'Are you sure you want to delete this facet? This action cannot be undone.',
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;
		try {
			await collection('views').delete(id);
			toasts.add('success', 'Facet deleted');
			await loadViews();
		} catch (err) {
			toasts.add('error', 'Failed to delete facet');
		}
	}

	function copyViewUrl(slug: string) {
		const url = `${window.location.origin}/${slug}`;
		navigator.clipboard.writeText(url);
		toasts.add('success', 'URL copied to clipboard');
	}
</script>

<svelte:head>
	<title>Facets | Facet</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Facets</h1>
		<a href="/admin/views/new" class="btn btn-primary">+ New Facet</a>
	</div>

	<p class="text-gray-600 dark:text-gray-400 mb-6">
		Facets are curated versions of your profile for different audiences. Each facet can have its own URL, visibility settings, and content selection.
	</p>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading facets...</div>
		</div>
	{:else if views.length === 0}
		<div class="card p-8 text-center">
			<p class="text-gray-600 dark:text-gray-400 mb-2">You haven't created any facets yet.</p>
			<p class="text-gray-500 dark:text-gray-500 text-sm mb-4">Facets let you show different versions of your profile to different audiences.</p>
			<a href="/admin/views/new" class="btn btn-primary">Create a Facet</a>
		</div>
	{:else}
		<div class="space-y-4">
			{#each views as view}
				<div class="card p-4">
					<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-gray-900 dark:text-white">{view.name}</h3>
								{#if view.is_default}
									<span class="px-2 py-0.5 text-xs bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300 rounded font-medium">
										Default
									</span>
								{/if}
								{#if !view.is_active}
									<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
										Inactive
									</span>
								{/if}
								<span class="px-2 py-0.5 text-xs rounded
									{view.visibility === 'public'
										? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
										: view.visibility === 'unlisted'
											? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
											: view.visibility === 'password'
												? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
												: 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300'}">
									{view.visibility}
								</span>
							</div>
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-1 break-all">
								<code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">/{view.slug}</code>
								{#if view.description}
									<span class="break-normal"> - {view.description}</span>
								{/if}
							</p>
						</div>

						<div class="flex items-center gap-1 shrink-0">
							<button
								class="btn btn-sm btn-ghost p-2"
								onclick={() => copyViewUrl(String(view.slug))}
								title="Copy URL"
							>
								{@html icon('copy')}
							</button>
							<a
								href="/{view.slug}"
								target="_blank"
								class="btn btn-sm btn-ghost p-2"
								title="Preview"
							>
								{@html icon('eye')}
							</a>
							<a href="/admin/views/{view.id}" class="btn btn-sm btn-secondary">
								Edit
							</a>
							<button
								class="btn btn-sm btn-ghost p-2"
								onclick={() => toggleActive(view)}
								title={view.is_active ? 'Deactivate' : 'Activate'}
							>
								{@html view.is_active ? icon('toggleOn') : icon('toggleOff')}
							</button>
							<button
								class="btn btn-sm btn-ghost p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
								onclick={() => deleteView(String(view.id))}
								title="Delete"
							>
								{@html icon('trash')}
							</button>
						</div>
					</div>

					{#if view.hero_headline}
						<div class="mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded text-sm">
							<span class="text-gray-500">Custom headline:</span>
							<span class="text-gray-900 dark:text-white ml-1">{view.hero_headline}</span>
						</div>
					{/if}

					<div class="mt-3 flex flex-wrap gap-4 text-sm text-gray-600 dark:text-gray-300">
						<div class="flex items-center gap-1">
							{@html icon('eye')}
							<span>{(view.view_count ?? 0).toLocaleString()} views</span>
						</div>
						<div class="flex items-center gap-1 text-gray-500 dark:text-gray-400">
							{@html icon('clock')}
							<span>
								{view.last_viewed_at
									? `Last viewed ${formatDate(String(view.last_viewed_at), { month: 'short', day: 'numeric', year: 'numeric' })}`
									: 'Never viewed'}
							</span>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Share Tokens Section -->
	<div class="mt-12">
		<h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Share Tokens</h2>
		<p class="text-gray-600 dark:text-gray-400 mb-4">
			Generate private links to share unlisted views with specific people.
		</p>
		<a href="/admin/tokens" class="btn btn-secondary">Manage Share Tokens</a>
	</div>
</div>

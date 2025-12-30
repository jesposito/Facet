<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';

	let loading = true;
	let views: Array<Record<string, unknown>> = [];

	onMount(async () => {
		await loadViews();
	});

	async function loadViews() {
		try {
			const result = await pb.collection('views').getList(1, 50, {
				sort: '-created'
			});
			views = result.items;
		} catch (err) {
			console.error('Failed to load views:', err);
		} finally {
			loading = false;
		}
	}

	async function toggleActive(view: Record<string, unknown>) {
		try {
			await pb.collection('views').update(view.id as string, {
				is_active: !view.is_active
			});
			await loadViews();
		} catch (err) {
			toasts.add('error', 'Failed to update view');
		}
	}

	async function deleteView(id: string) {
		if (!confirm('Are you sure you want to delete this view?')) return;
		try {
			await pb.collection('views').delete(id);
			toasts.add('success', 'View deleted');
			await loadViews();
		} catch (err) {
			toasts.add('error', 'Failed to delete view');
		}
	}

	function copyViewUrl(slug: string) {
		const url = `${window.location.origin}/v/${slug}`;
		navigator.clipboard.writeText(url);
		toasts.add('success', 'URL copied to clipboard');
	}
</script>

<svelte:head>
	<title>Views | Admin</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Views</h1>
		<a href="/admin/views/new" class="btn btn-primary">+ Create View</a>
	</div>

	<p class="text-gray-600 dark:text-gray-400 mb-6">
		Views are curated versions of your profile for different audiences. Each view can have its own URL, visibility settings, and content selection.
	</p>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading views...</div>
		</div>
	{:else if views.length === 0}
		<div class="card p-8 text-center">
			<p class="text-gray-500 dark:text-gray-400 mb-4">No views created yet</p>
			<a href="/admin/views/new" class="btn btn-primary">Create Your First View</a>
		</div>
	{:else}
		<div class="space-y-4">
			{#each views as view}
				<div class="card p-4">
					<div class="flex items-start justify-between">
						<div class="flex-1">
							<div class="flex items-center gap-2">
								<h3 class="font-medium text-gray-900 dark:text-white">{view.name}</h3>
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
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
								<code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">/v/{view.slug}</code>
								{#if view.description}
									— {view.description}
								{/if}
							</p>
						</div>

						<div class="flex items-center gap-2">
							<button
								class="btn btn-sm btn-ghost"
								on:click={() => copyViewUrl(view.slug as string)}
								title="Copy URL"
							>
								📋
							</button>
							<a
								href="/v/{view.slug}"
								target="_blank"
								class="btn btn-sm btn-ghost"
								title="Preview"
							>
								👁
							</a>
							<a href="/admin/views/{view.id}" class="btn btn-sm btn-secondary">
								Edit
							</a>
							<button
								class="btn btn-sm btn-ghost"
								on:click={() => toggleActive(view)}
								title={view.is_active ? 'Deactivate' : 'Activate'}
							>
								{view.is_active ? '🔴' : '🟢'}
							</button>
							<button
								class="btn btn-sm btn-ghost text-red-600"
								on:click={() => deleteView(view.id as string)}
							>
								🗑
							</button>
						</div>
					</div>

					{#if view.hero_headline}
						<div class="mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded text-sm">
							<span class="text-gray-500">Custom headline:</span>
							<span class="text-gray-900 dark:text-white ml-1">{view.hero_headline}</span>
						</div>
					{/if}
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

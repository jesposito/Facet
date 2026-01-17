<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { pb, type ShareToken, type View } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import PageHelp from '$components/admin/PageHelp.svelte';

	let loading = $state(true);
	let tokens: ShareToken[] = $state([]);
	let views: View[] = $state([]);
	let showCreateModal = $state(false);
	let creating = $state(false);

	// Form state for creating tokens
	let newToken = $state({
		view_id: '',
		name: '',
		expires_at: '',
		max_uses: 0
	});

	// Store newly created token (shown once)
	let createdToken: { raw: string; url: string } | null = $state(null);

	onMount(async () => {
		await Promise.all([loadTokens(), loadViews()]);
	});

	async function loadTokens() {
		try {
			const result = await pb.collection('share_tokens').getList<ShareToken>(1, 100, {
				sort: '-id',
				expand: 'view_id'
			});
			tokens = result.items;
		} catch (err) {
			console.error('Failed to load tokens:', err);
			toasts.add('error', 'Failed to load share tokens');
		} finally {
			loading = false;
		}
	}

	async function loadViews() {
		try {
			const result = await collection('views').getList<View>(1, 100, {
				filter: "visibility = 'unlisted'",
				sort: 'name'
			});
			views = result.items;
		} catch (err) {
			console.error('Failed to load views:', err);
		}
	}

	async function createToken() {
		if (!newToken.view_id) {
			toasts.add('error', 'Please select a view');
			return;
		}

		creating = true;
		try {
			const response = await fetch('/api/share/generate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					view_id: newToken.view_id,
					name: newToken.name || undefined,
					expires_at: newToken.expires_at || undefined,
					max_uses: newToken.max_uses || 0
				})
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to create token');
			}

			const data = await response.json();

			// Store the raw token to show once
			createdToken = {
				raw: data.token,
				url: `${window.location.origin}/s/${data.token}`
			};

			toasts.add('success', 'Token created successfully');
			await loadTokens();
			resetForm();
		} catch (err) {
			toasts.add('error', err instanceof Error ? err.message : 'Failed to create token');
		} finally {
			creating = false;
		}
	}

	async function revokeToken(tokenId: string) {
		const confirmed = await confirm({
			title: 'Revoke Token',
			message: 'Are you sure you want to revoke this token? It will no longer be usable.',
			confirmText: 'Revoke',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			const response = await fetch(`/api/share/revoke/${tokenId}`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${pb.authStore.token}`
				}
			});

			if (!response.ok) {
				throw new Error('Failed to revoke token');
			}

			toasts.add('success', 'Token revoked');
			await loadTokens();
		} catch (err) {
			toasts.add('error', 'Failed to revoke token');
		}
	}

	async function deleteToken(tokenId: string) {
		const confirmed = await confirm({
			title: 'Delete Token',
			message: 'Are you sure you want to permanently delete this token? This action cannot be undone.',
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await pb.collection('share_tokens').delete(tokenId);
			toasts.add('success', 'Token deleted');
			await loadTokens();
		} catch (err) {
			toasts.add('error', 'Failed to delete token');
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		toasts.add('success', 'Copied to clipboard');
	}

	function resetForm() {
		newToken = {
			view_id: '',
			name: '',
			expires_at: '',
			max_uses: 0
		};
		showCreateModal = false;
	}

	function dismissCreatedToken() {
		createdToken = null;
	}

	function getViewName(token: ShareToken): string {
		return token.expand?.view_id?.name || 'Unknown View';
	}

	function getViewSlug(token: ShareToken): string {
		return token.expand?.view_id?.slug || '';
	}

	function isExpired(token: ShareToken): boolean {
		if (!token.expires_at) return false;
		return new Date(token.expires_at) < new Date();
	}

	function isMaxUsesReached(token: ShareToken): boolean {
		if (!token.max_uses || token.max_uses === 0) return false;
		return token.use_count >= token.max_uses;
	}

	function getTokenStatus(token: ShareToken): { label: string; class: string } {
		if (!token.is_active) {
			return { label: 'Revoked', class: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300' };
		}
		if (isExpired(token)) {
			return { label: 'Expired', class: 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400' };
		}
		if (isMaxUsesReached(token)) {
			return { label: 'Max Uses Reached', class: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300' };
		}
		return { label: 'Active', class: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' };
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatRelativeDate(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Yesterday';
		if (diffDays < 7) return `${diffDays} days ago`;
		if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
		return formatDate(dateStr);
	}

	// Group tokens by view
	let tokensByView = $derived(tokens.reduce(
		(acc, token) => {
			const viewId = token.view_id;
			if (!acc[viewId]) {
				acc[viewId] = {
					viewName: getViewName(token),
					viewSlug: getViewSlug(token),
					tokens: []
				};
			}
			acc[viewId].tokens.push(token);
			return acc;
		},
		{} as Record<string, { viewName: string; viewSlug: string; tokens: ShareToken[] }>
	));
</script>

<svelte:head>
	<title>Share Tokens | Facet</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<PageHelp pageKey="tokens">
		<p><strong>Share Tokens</strong> provide secure, time-limited access to your unlisted facets.</p>
		<p>Perfect for job applications: generate a token that expires after 30 days, send it to recruiters, and revoke it anytime. The URL looks clean (like <code>/recruiter</code>) while the token handles access control.</p>
		<p><strong>Tip:</strong> Set a max use count to limit how many times a link can be viewed, or leave it unlimited for ongoing access.</p>
	</PageHelp>

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Share Tokens</h1>
		<button class="btn btn-primary" onclick={() => (showCreateModal = true)}>
			{@html icon('plus')} Generate Token
		</button>
	</div>

	<!-- Newly Created Token Banner -->
	{#if createdToken}
		<div class="card p-4 mb-6 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
			<div class="flex items-start justify-between">
				<div class="flex-1">
					<h3 class="font-medium text-green-800 dark:text-green-200 mb-2">
						{@html icon('check')} Token Created Successfully
					</h3>
					<p class="text-sm text-green-700 dark:text-green-300 mb-3">
						Copy this token now. For security, it will not be shown again.
					</p>
					<div class="flex flex-col gap-2">
						<div class="flex items-center gap-2">
							<code class="flex-1 bg-white dark:bg-gray-800 px-3 py-2 rounded border text-sm font-mono break-all">
								{createdToken.url}
							</code>
							<button
								class="btn btn-secondary shrink-0"
								onclick={() => copyToClipboard(createdToken?.url || '')}
							>
								{@html icon('copy')} Copy URL
							</button>
						</div>
					</div>
				</div>
				<button
					class="btn btn-ghost text-green-700 dark:text-green-300 ml-4"
					onclick={dismissCreatedToken}
				>
					{@html icon('x')}
				</button>
			</div>
		</div>
	{/if}

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading tokens...</div>
		</div>
	{:else if tokens.length === 0}
		<div class="card p-8 text-center">
			<p class="text-gray-600 dark:text-gray-400 mb-2">No share tokens yet.</p>
			<p class="text-gray-500 dark:text-gray-500 text-sm mb-4">
				Generate a token to share an unlisted view with specific people.
			</p>
			{#if views.length === 0}
				<p class="text-gray-500 dark:text-gray-500 text-sm mb-4">
					First, <a href="/admin/views/new" class="text-primary-600 hover:underline">create an unlisted view</a> to generate tokens for.
				</p>
			{:else}
				<button class="btn btn-primary" onclick={() => (showCreateModal = true)}>
					Generate Your First Token
				</button>
			{/if}
		</div>
	{:else}
		<!-- Token List Grouped by View -->
		<div class="space-y-6">
			{#each Object.entries(tokensByView) as [viewId, group]}
				<div class="card">
					<div class="p-4 border-b border-gray-200 dark:border-gray-700">
						<div class="flex items-center justify-between">
							<div>
								<h3 class="font-medium text-gray-900 dark:text-white">{group.viewName}</h3>
								{#if group.viewSlug}
									<code class="text-sm text-gray-500 dark:text-gray-400">/{group.viewSlug}</code>
								{/if}
							</div>
							<a href="/admin/views/{viewId}" class="btn btn-sm btn-ghost">
								Edit View
							</a>
						</div>
					</div>

					<div class="divide-y divide-gray-100 dark:divide-gray-800">
						{#each group.tokens as token}
							{@const status = getTokenStatus(token)}
							<div class="p-4">
								<div class="flex items-start justify-between">
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2 flex-wrap mb-1">
											<span class="font-medium text-gray-900 dark:text-white">
												{token.name || 'Unnamed Token'}
											</span>
											<span class="px-2 py-0.5 text-xs rounded {status.class}">
												{status.label}
											</span>
										</div>

										<div class="text-sm text-gray-500 dark:text-gray-400 space-y-1">
											<div class="flex items-center gap-4 flex-wrap">
												<span>
													{@html icon('clock')}
													Created {formatRelativeDate(token.created)}
												</span>
												{#if token.expires_at}
													<span>
														Expires: {formatDate(token.expires_at)}
													</span>
												{/if}
											</div>

											<div class="flex items-center gap-4 flex-wrap">
												<span>
													Uses: {token.use_count}{token.max_uses ? ` / ${token.max_uses}` : ' (unlimited)'}
												</span>
												{#if token.last_used_at}
													<span>
														Last used: {formatDate(token.last_used_at)}
													</span>
												{:else}
													<span>Never used</span>
												{/if}
											</div>

											{#if token.token_prefix}
												<div>
													<span class="text-gray-400">Prefix:</span>
													<code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">{token.token_prefix}...</code>
												</div>
											{/if}
										</div>
									</div>

									<div class="flex items-center gap-1 ml-4">
										{#if token.is_active && !isExpired(token) && !isMaxUsesReached(token)}
											<button
												class="btn btn-sm btn-ghost text-yellow-600 hover:bg-yellow-50 dark:hover:bg-yellow-900/20"
												onclick={() => revokeToken(token.id)}
												title="Revoke token"
											>
												{@html icon('lock')}
											</button>
										{/if}
										<button
											class="btn btn-sm btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
											onclick={() => deleteToken(token.id)}
											title="Delete token"
										>
											{@html icon('trash')}
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

<!-- Create Token Modal -->
{#if showCreateModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="card w-full max-w-md">
			<div class="p-4 border-b border-gray-200 dark:border-gray-700">
				<h2 class="text-lg font-bold text-gray-900 dark:text-white">Generate Share Token</h2>
			</div>

			<form onsubmit={preventDefault(createToken)} class="p-4 space-y-4">
				<div>
					<label for="view_id" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						View <span class="text-red-500">*</span>
					</label>
					{#if views.length === 0}
						<p class="text-sm text-gray-500 dark:text-gray-400">
							No unlisted views available. <a href="/admin/views/new" class="text-primary-600 hover:underline">Create one first</a>.
						</p>
					{:else}
						<select
							id="view_id"
							bind:value={newToken.view_id}
							class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
							required
						>
							<option value="">Select a view...</option>
							{#each views as view}
								<option value={view.id}>{view.name} (/{view.slug})</option>
							{/each}
						</select>
					{/if}
				</div>

				<div>
					<label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						Name (optional)
					</label>
					<input
						type="text"
						id="name"
						bind:value={newToken.name}
						placeholder="e.g., Sent to Company X"
						class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
					/>
					<p class="text-xs text-gray-500 mt-1">A label to help you remember who this token was shared with.</p>
				</div>

				<div>
					<label for="expires_at" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						Expiration (optional)
					</label>
					<input
						type="datetime-local"
						id="expires_at"
						bind:value={newToken.expires_at}
						class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
					/>
					<p class="text-xs text-gray-500 mt-1">Leave empty for no expiration.</p>
				</div>

				<div>
					<label for="max_uses" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						Max Uses (optional)
					</label>
					<input
						type="number"
						id="max_uses"
						bind:value={newToken.max_uses}
						min="0"
						placeholder="0"
						class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
					/>
					<p class="text-xs text-gray-500 mt-1">0 = unlimited uses.</p>
				</div>

				<div class="flex justify-end gap-2 pt-4">
					<button type="button" class="btn btn-ghost" onclick={resetForm}>
						Cancel
					</button>
					<button
						type="submit"
						class="btn btn-primary"
						disabled={creating || views.length === 0}
					>
						{#if creating}
							Generating...
						{:else}
							Generate Token
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { pb, type ShareToken, type View } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import { t } from 'svelte-i18n';

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
			toasts.add('error', $t('admin.tokens.toast_load_failed'));
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
			toasts.add('error', $t('admin.tokens.toast_select_view'));
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

			toasts.add('success', $t('admin.tokens.toast_created'));
			await loadTokens();
			resetForm();
		} catch (err) {
			toasts.add('error', err instanceof Error ? err.message : $t('admin.tokens.toast_create_failed'));
		} finally {
			creating = false;
		}
	}

	async function revokeToken(tokenId: string) {
		const confirmed = await confirm({
			title: $t('admin.tokens.confirm_revoke_title'),
			message: $t('admin.tokens.confirm_revoke_message'),
			confirmText: $t('admin.tokens.confirm_revoke_button'),
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

			toasts.add('success', $t('admin.tokens.toast_revoked'));
			await loadTokens();
		} catch (err) {
			toasts.add('error', $t('admin.tokens.toast_revoke_failed'));
		}
	}

	async function deleteToken(tokenId: string) {
		const confirmed = await confirm({
			title: $t('admin.tokens.confirm_delete_title'),
			message: $t('admin.tokens.confirm_delete_message'),
			confirmText: $t('admin.tokens.confirm_delete_button'),
			danger: true
		});
		if (!confirmed) {
			return;
		}

		try {
			await pb.collection('share_tokens').delete(tokenId);
			toasts.add('success', $t('admin.tokens.toast_deleted'));
			await loadTokens();
		} catch (err) {
			toasts.add('error', $t('admin.tokens.toast_delete_failed'));
		}
	}

	async function regenerateToken(token: ShareToken) {
		const viewName = getViewName(token);
		const confirmed = await confirm({
			title: $t('admin.tokens.confirm_regenerate_title'),
			message: $t('admin.tokens.confirm_regenerate_message', { values: { name: token.name || $t('admin.tokens.unnamed_token'), view: viewName } }),
			confirmText: $t('admin.tokens.confirm_regenerate_button'),
			danger: false
		});
		if (!confirmed) return;

		try {
			// First revoke the old token
			const revokeResponse = await fetch(`/api/share/revoke/${token.id}`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${pb.authStore.token}`
				}
			});

			if (!revokeResponse.ok) {
				throw new Error('Failed to revoke old token');
			}

			// Then create a new token with the same settings
			const response = await fetch('/api/share/generate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					view_id: token.view_id,
					name: token.name || undefined,
					expires_at: token.expires_at || undefined,
					max_uses: token.max_uses || 0
				})
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to create new token');
			}

			const data = await response.json();
			createdToken = {
				raw: data.token,
				url: `${window.location.origin}/s/${data.token}`
			};

			toasts.add('success', $t('admin.tokens.toast_regenerated'));
			await loadTokens();
		} catch (err) {
			toasts.add('error', err instanceof Error ? err.message : $t('admin.tokens.toast_regenerate_failed'));
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		toasts.add('success', $t('admin.tokens.toast_copied'));
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
			return { label: $t('admin.tokens.status_revoked'), class: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300' };
		}
		if (isExpired(token)) {
			return { label: $t('admin.tokens.status_expired'), class: 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400' };
		}
		if (isMaxUsesReached(token)) {
			return { label: $t('admin.tokens.status_max_uses'), class: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300' };
		}
		return { label: $t('admin.tokens.status_active'), class: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' };
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return $t('admin.tokens.never');
		return new Date(dateStr).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatRelativeDate(dateStr: string | undefined): string {
		if (!dateStr) return $t('admin.tokens.never');
		const date = new Date(dateStr);
		if (isNaN(date.getTime())) return $t('admin.tokens.never');
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

		if (diffDays === 0) return $t('admin.tokens.today');
		if (diffDays === 1) return $t('admin.tokens.yesterday');
		if (diffDays < 7) return $t('admin.tokens.days_ago', { values: { count: diffDays } });
		if (diffDays < 30) return $t('admin.tokens.weeks_ago', { values: { count: Math.floor(diffDays / 7) } });
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
	<title>{$t('admin.tokens.page_title')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<PageHelp pageKey="tokens">
		<p>{@html $t('admin.tokens.help_intro')}</p>
		<p>{@html $t('admin.tokens.help_description')}</p>
		<p>{@html $t('admin.tokens.help_tip')}</p>
	</PageHelp>

	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.tokens.title')}</h1>
		<button class="btn btn-primary" onclick={() => (showCreateModal = true)}>
			{@html icon('plus')} {$t('admin.tokens.generate_button')}
		</button>
	</div>

	<!-- Newly Created Token Banner -->
	{#if createdToken}
		<div class="card p-4 mb-6 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
			<div class="flex items-start justify-between">
				<div class="flex-1">
					<h3 class="font-medium text-green-800 dark:text-green-200 mb-2">
						{@html icon('check')} {$t('admin.tokens.token_created_title')}
					</h3>
					<p class="text-sm text-green-700 dark:text-green-300 mb-3">
						{$t('admin.tokens.token_created_message')}
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
								{@html icon('copy')} {$t('admin.tokens.copy_url')}
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
			<div class="animate-pulse">{$t('admin.tokens.loading_tokens')}</div>
		</div>
	{:else if tokens.length === 0}
		<div class="card p-8 text-center">
			<p class="text-gray-600 dark:text-gray-400 mb-2">{$t('admin.tokens.no_tokens')}</p>
			<p class="text-gray-500 dark:text-gray-300 text-sm mb-4">
				{$t('admin.tokens.no_tokens_description')}
			</p>
			{#if views.length === 0}
				<p class="text-gray-500 dark:text-gray-300 text-sm mb-4">
					{@html $t('admin.tokens.create_unlisted_first', { values: { link: `<a href="/admin/views/new" class="text-primary-600 hover:underline">${$t('admin.tokens.create_unlisted_link')}</a>` } })}
				</p>
			{:else}
				<button class="btn btn-primary" onclick={() => (showCreateModal = true)}>
					{$t('admin.tokens.generate_first')}
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
								{$t('admin.tokens.edit_view')}
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
												{token.name || $t('admin.tokens.unnamed_token')}
											</span>
											<span class="px-2 py-0.5 text-xs rounded {status.class}">
												{status.label}
											</span>
										</div>

										<div class="text-sm text-gray-500 dark:text-gray-400 space-y-1">
											<div class="flex items-center gap-4 flex-wrap">
												<span>
													{@html icon('clock')}
													{$t('admin.tokens.created_relative', { values: { date: formatRelativeDate(token.created) } })}
												</span>
												{#if token.expires_at}
													<span>
														{$t('admin.tokens.expires_label', { values: { date: formatDate(token.expires_at) } })}
													</span>
												{/if}
											</div>

											<div class="flex items-center gap-4 flex-wrap">
												<span>
													{#if token.max_uses}
														{$t('admin.tokens.uses_with_max', { values: { current: token.use_count, max: token.max_uses } })}
													{:else}
														{$t('admin.tokens.uses_unlimited', { values: { current: token.use_count } })}
													{/if}
												</span>
												{#if token.last_used_at}
													<span>
														{$t('admin.tokens.last_used', { values: { date: formatDate(token.last_used_at) } })}
													</span>
												{:else}
													<span>{$t('admin.tokens.never_used')}</span>
												{/if}
											</div>

											{#if token.token_prefix}
												<div>
													<span class="text-gray-400">{$t('admin.tokens.prefix_label')}</span>
													<code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">{token.token_prefix}...</code>
												</div>
											{/if}
										</div>
									</div>

									<div class="flex items-center gap-1 ml-4">
										{#if token.is_active && !isExpired(token) && !isMaxUsesReached(token)}
											<button
												class="btn btn-sm btn-ghost text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20"
												onclick={() => regenerateToken(token)}
												title={$t('admin.tokens.regenerate_title')}
											>
												<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2v6h-6"></path><path d="M3 12a9 9 0 0 1 15-6.7L21 8"></path><path d="M3 22v-6h6"></path><path d="M21 12a9 9 0 0 1-15 6.7L3 16"></path></svg>
											</button>
											<button
												class="btn btn-sm btn-ghost text-yellow-600 hover:bg-yellow-50 dark:hover:bg-yellow-900/20"
												onclick={() => revokeToken(token.id)}
												title={$t('admin.tokens.revoke_title')}
											>
												{@html icon('lock')}
											</button>
										{/if}
										<button
											class="btn btn-danger-ghost btn-sm"
											onclick={() => deleteToken(token.id)}
											title={$t('admin.tokens.delete_title')}
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
				<h2 class="text-lg font-bold text-gray-900 dark:text-white">{$t('admin.tokens.modal_title')}</h2>
			</div>

			<form onsubmit={preventDefault(createToken)} class="p-4 space-y-4">
				<div>
					<label for="view_id" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						{$t('admin.tokens.view_required')}
					</label>
					{#if views.length === 0}
						<p class="text-sm text-gray-500 dark:text-gray-400">
							{@html $t('admin.tokens.no_unlisted_views', { values: { link: `<a href="/admin/views/new" class="text-primary-600 hover:underline">${$t('admin.tokens.no_unlisted_link')}</a>` } })}
						</p>
					{:else}
						<select
							id="view_id"
							bind:value={newToken.view_id}
							class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
							required
						>
							<option value="">{$t('admin.tokens.select_view')}</option>
							{#each views as view}
								<option value={view.id}>{view.name} (/{view.slug})</option>
							{/each}
						</select>
					{/if}
				</div>

				<div>
					<label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						{$t('admin.tokens.name_label')}
					</label>
					<input
						type="text"
						id="name"
						bind:value={newToken.name}
						placeholder={$t('admin.tokens.name_placeholder')}
						class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
					/>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.tokens.name_help')}</p>
				</div>

				<div>
					<label for="expires_at" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						{$t('admin.tokens.expiration_label')}
					</label>
					<input
						type="datetime-local"
						id="expires_at"
						bind:value={newToken.expires_at}
						class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
					/>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.tokens.expiration_help')}</p>
				</div>

				<div>
					<label for="max_uses" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						{$t('admin.tokens.max_uses_label')}
					</label>
					<input
						type="number"
						id="max_uses"
						bind:value={newToken.max_uses}
						min="0"
						placeholder="0"
						class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600"
					/>
					<p class="text-xs text-gray-500 mt-1">{$t('admin.tokens.max_uses_help')}</p>
				</div>

				<div class="flex justify-end gap-2 pt-4">
					<button type="button" class="btn btn-ghost" onclick={resetForm}>
						{$t('admin.tokens.cancel')}
					</button>
					<button
						type="submit"
						class="btn btn-primary"
						disabled={creating || views.length === 0}
					>
						{#if creating}
							{$t('admin.tokens.generating')}
						{:else}
							{$t('admin.tokens.generate')}
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

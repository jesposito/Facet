<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { t } from 'svelte-i18n';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';

	// ---------------------------------------------------------------------------
	// Types
	// ---------------------------------------------------------------------------
	type ApiKey = {
		id: string;
		label: string;
		key_prefix: string;
		scopes: string[];
		allowed_origins: string[];
		expires_at: string | null;
		last_used_at: string | null;
		is_active: boolean;
		created: string;
	};

	// Scopes available for the v1 read API. Must match the backend's `validScopes`.
	const availableScopes = [
		'read:profile',
		'read:posts',
		'read:projects',
		'read:skills',
		'read:experience',
	] as const;
	type Scope = (typeof availableScopes)[number];

	// ---------------------------------------------------------------------------
	// State
	// ---------------------------------------------------------------------------
	let apiKeys: ApiKey[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let saving = $state(false);
	let helpOpen = $state(false);

	// Create form fields
	let label = $state('');
	let selectedScopes: Scope[] = $state(['read:profile']);
	let expiresAt = $state('');

	// One-time-reveal modal state
	let createdKey: string | null = $state(null);
	let copied = $state(false);
	let copyAnnouncement = $state('');
	let revealDialog: HTMLDialogElement | null = $state(null);

	// Element refs (focus restore)
	let createBtn: HTMLButtonElement | null = $state(null);
	let formLabelInput: HTMLInputElement | null = $state(null);

	// ---------------------------------------------------------------------------
	// Effects
	// ---------------------------------------------------------------------------
	onMount(() => {
		loadKeys();
	});

	// When the reveal modal opens, focus the copy button + trap focus.
	$effect(() => {
		if (createdKey && revealDialog && !revealDialog.open) {
			revealDialog.showModal();
		}
	});

	// ---------------------------------------------------------------------------
	// Data
	// ---------------------------------------------------------------------------
	async function loadKeys() {
		loading = true;
		try {
			const res = await fetch('/api/admin/api-keys', {
				headers: pb.authStore.isValid
					? { Authorization: `Bearer ${pb.authStore.token}` }
					: {},
			});
			if (!res.ok) throw new Error('failed');
			const json = await res.json();
			apiKeys = (json.data || []).map((k: Partial<ApiKey>) => ({
				...k,
				scopes: k.scopes || [],
				allowed_origins: k.allowed_origins || [],
			})) as ApiKey[];
		} catch (err) {
			console.error('[admin/api] load failed', err);
			toasts.add('error', $t('admin.api_keys.errors.load_failed'));
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		label = '';
		selectedScopes = ['read:profile'];
		expiresAt = '';
	}

	async function openForm() {
		resetForm();
		showForm = true;
		await tick();
		formLabelInput?.focus();
	}

	function closeForm() {
		showForm = false;
		resetForm();
		createBtn?.focus();
	}

	function toggleScope(scope: Scope) {
		if (selectedScopes.includes(scope)) {
			selectedScopes = selectedScopes.filter((s) => s !== scope);
		} else {
			selectedScopes = [...selectedScopes, scope];
		}
	}

	async function submitForm(e: Event) {
		e.preventDefault();
		const trimmed = label.trim();
		if (!trimmed) {
			toasts.add('error', $t('admin.api_keys.errors.label_required'));
			formLabelInput?.focus();
			return;
		}
		if (selectedScopes.length === 0) {
			toasts.add('error', $t('admin.api_keys.errors.scope_required'));
			return;
		}

		saving = true;
		try {
			const res = await fetch('/api/admin/api-keys', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid
						? { Authorization: `Bearer ${pb.authStore.token}` }
						: {}),
				},
				body: JSON.stringify({
					label: trimmed,
					scopes: selectedScopes,
					expires_at: expiresAt || null,
				}),
			});
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'failed');
			}
			const json = await res.json();
			createdKey = json.key;
			copied = false;
			showForm = false;
			resetForm();
			await loadKeys();
		} catch (err) {
			console.error('[admin/api] create failed', err);
			toasts.add('error', $t('admin.api_keys.errors.create_failed'));
		} finally {
			saving = false;
		}
	}

	async function revokeKey(key: ApiKey) {
		const ok = await confirm({
			title: $t('admin.api_keys.revoke_confirm_title'),
			message: $t('admin.api_keys.revoke_confirm_message', { values: { label: key.label } }),
			confirmText: $t('admin.api_keys.revoke'),
			danger: true,
		});
		if (!ok) return;

		try {
			const res = await fetch(`/api/admin/api-keys/${key.id}`, {
				method: 'DELETE',
				headers: pb.authStore.isValid
					? { Authorization: `Bearer ${pb.authStore.token}` }
					: {},
			});
			if (!res.ok) throw new Error('failed');
			toasts.add('success', $t('admin.api_keys.key_revoked'));
			await loadKeys();
		} catch (err) {
			console.error('[admin/api] revoke failed', err);
			toasts.add('error', $t('admin.api_keys.errors.revoke_failed'));
		}
	}

	async function copyToClipboard() {
		if (!createdKey) return;
		try {
			await navigator.clipboard.writeText(createdKey);
			copied = true;
			copyAnnouncement = $t('admin.api_keys.copy_success');
			setTimeout(() => {
				copied = false;
				copyAnnouncement = '';
			}, 2500);
		} catch {
			copyAnnouncement = $t('admin.api_keys.copy_failed');
		}
	}

	function dismissReveal() {
		createdKey = null;
		copyAnnouncement = '';
		revealDialog?.close();
		createBtn?.focus();
	}

	function onRevealKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			dismissReveal();
		}
	}

	function formatDate(s: string | null | undefined): string {
		if (!s) return $t('admin.api_keys.never');
		try {
			return new Date(s).toLocaleString();
		} catch {
			return s;
		}
	}

	function statusLabel(k: ApiKey): string {
		if (!k.is_active) return $t('admin.api_keys.status_revoked');
		if (k.expires_at && new Date(k.expires_at) < new Date())
			return $t('admin.api_keys.status_expired');
		return $t('admin.api_keys.status_active');
	}

	function statusClass(k: ApiKey): string {
		if (!k.is_active)
			return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
		if (k.expires_at && new Date(k.expires_at) < new Date())
			return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200';
		return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
	}
</script>

<svelte:head>
	<title>{$t('admin.api_keys.title')}</title>
</svelte:head>

<!-- Live region for clipboard announcements -->
<div class="sr-only" role="status" aria-live="polite">{copyAnnouncement}</div>

<div class="max-w-5xl mx-auto">
	<div class="flex items-start justify-between mb-6 gap-4">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
				{$t('admin.api_keys.title')}
			</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-2xl">
				{$t('admin.api_keys.intro')}
			</p>
		</div>
		{#if !showForm}
			<button
				bind:this={createBtn}
				type="button"
				class="btn btn-primary shrink-0"
				onclick={openForm}
			>
				{$t('admin.api_keys.new_key')}
			</button>
		{/if}
	</div>

	<!-- Help / docs panel -->
	<section class="mb-6 card p-4" aria-labelledby="api-keys-help-heading">
		<button
			type="button"
			class="flex items-center justify-between w-full text-left"
			onclick={() => (helpOpen = !helpOpen)}
			aria-expanded={helpOpen}
			aria-controls="api-keys-help-body"
		>
			<h2 id="api-keys-help-heading" class="text-sm font-semibold text-gray-900 dark:text-white">
				{$t('admin.api_keys.help.title')}
			</h2>
			<svg
				class="w-4 h-4 transition-transform {helpOpen ? '' : '-rotate-90'}"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				aria-hidden="true"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</button>
		{#if helpOpen}
			<div id="api-keys-help-body" class="mt-3 space-y-3 text-sm text-gray-700 dark:text-gray-300">
				<p>{$t('admin.api_keys.help.body')}</p>
				<div>
					<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-1">
						{$t('admin.api_keys.help.scopes_title')}
					</h3>
					<ul class="space-y-1">
						{#each availableScopes as scope}
							<li class="flex gap-2">
								<code class="font-mono text-xs px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded">{scope}</code>
								<span>{$t(`admin.api_keys.scope_descriptions.${scope}`)}</span>
							</li>
						{/each}
					</ul>
				</div>
				<div>
					<h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-1">
						{$t('admin.api_keys.help.curl_title')}
					</h3>
					<pre class="text-xs bg-gray-900 text-green-400 p-3 rounded overflow-x-auto"><code>curl -H "Authorization: Bearer facet_..." \
  https://your-site.example/api/v1/profile</code></pre>
				</div>
			</div>
		{/if}
	</section>

	<!-- Create form -->
	{#if showForm}
		<section class="card p-6 mb-6" aria-labelledby="api-key-form-heading">
			<h2 id="api-key-form-heading" class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
				{$t('admin.api_keys.new_key')}
			</h2>
			<form onsubmit={submitForm} class="space-y-4">
				<div>
					<label for="api-key-label" class="label">
						{$t('admin.api_keys.field_label')}
						<span aria-hidden="true"> *</span>
					</label>
					<input
						id="api-key-label"
						bind:this={formLabelInput}
						bind:value={label}
						type="text"
						class="input"
						required
						aria-required="true"
						aria-describedby="api-key-label-help"
						placeholder={$t('admin.api_keys.field_label_placeholder')}
					/>
					<p id="api-key-label-help" class="text-xs text-gray-500 dark:text-gray-400 mt-1">
						{$t('admin.api_keys.field_label_help')}
					</p>
				</div>

				<fieldset>
					<legend class="label">
						{$t('admin.api_keys.field_scopes')}
						<span aria-hidden="true"> *</span>
					</legend>
					<div class="space-y-2 mt-1">
						{#each availableScopes as scope}
							{@const id = `scope-${scope.replace(':', '-')}`}
							<label for={id} class="flex items-start gap-2 cursor-pointer">
								<input
									{id}
									type="checkbox"
									checked={selectedScopes.includes(scope)}
									onchange={() => toggleScope(scope)}
									class="mt-1 w-4 h-4 text-primary-600 rounded border-gray-300 dark:border-gray-600"
								/>
								<span class="text-sm">
									<code class="font-mono text-xs px-1 py-0.5 bg-gray-100 dark:bg-gray-700 rounded">{scope}</code>
									<span class="text-gray-700 dark:text-gray-300 ml-1">
										{$t(`admin.api_keys.scope_descriptions.${scope}`)}
									</span>
								</span>
							</label>
						{/each}
					</div>
				</fieldset>

				<div>
					<label for="api-key-expires" class="label">{$t('admin.api_keys.field_expires')}</label>
					<input
						id="api-key-expires"
						bind:value={expiresAt}
						type="date"
						class="input"
						aria-describedby="api-key-expires-help"
					/>
					<p id="api-key-expires-help" class="text-xs text-gray-500 dark:text-gray-400 mt-1">
						{$t('admin.api_keys.field_expires_help')}
					</p>
				</div>

				<div class="flex items-center justify-end gap-3 pt-2">
					<button type="button" class="btn btn-ghost" onclick={closeForm}>
						{$t('admin.api_keys.cancel')}
					</button>
					<button
						type="submit"
						class="btn btn-primary"
						disabled={saving || !label.trim() || selectedScopes.length === 0}
					>
						{saving ? $t('admin.api_keys.creating') : $t('admin.api_keys.create_key')}
					</button>
				</div>
			</form>
		</section>
	{/if}

	<!-- Keys table -->
	<section aria-labelledby="api-keys-list-heading">
		<h2 id="api-keys-list-heading" class="sr-only">{$t('admin.api_keys.list_heading')}</h2>
		{#if loading}
			<div class="card p-8 text-center text-sm text-gray-500 dark:text-gray-400">
				{$t('admin.api_keys.loading')}
			</div>
		{:else if apiKeys.length === 0}
			<div class="card p-12 text-center">
				<p class="text-gray-700 dark:text-gray-300 font-medium">
					{$t('admin.api_keys.empty_title')}
				</p>
				<p class="text-sm text-gray-500 dark:text-gray-400 mt-1 mb-4">
					{$t('admin.api_keys.empty_body')}
				</p>
				{#if !showForm}
					<button type="button" class="btn btn-primary" onclick={openForm}>
						{$t('admin.api_keys.new_key')}
					</button>
				{/if}
			</div>
		{:else}
			<div class="card overflow-hidden">
				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<caption class="sr-only">{$t('admin.api_keys.table_caption')}</caption>
						<thead class="bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700">
							<tr>
								<th scope="col" class="text-start py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									{$t('admin.api_keys.col_label')}
								</th>
								<th scope="col" class="text-start py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									{$t('admin.api_keys.col_prefix')}
								</th>
								<th scope="col" class="text-start py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									{$t('admin.api_keys.col_scopes')}
								</th>
								<th scope="col" class="text-start py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									{$t('admin.api_keys.col_status')}
								</th>
								<th scope="col" class="text-start py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									{$t('admin.api_keys.col_last_used')}
								</th>
								<th scope="col" class="text-start py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									{$t('admin.api_keys.col_created')}
								</th>
								<th scope="col" class="text-end py-3 px-4 font-medium text-gray-500 dark:text-gray-400">
									<span class="sr-only">{$t('admin.api_keys.col_actions')}</span>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each apiKeys as key (key.id)}
								<tr class="border-b border-gray-100 dark:border-gray-800 last:border-b-0 hover:bg-gray-50 dark:hover:bg-gray-800/30">
									<td class="py-3 px-4">
										<span class="font-medium text-gray-900 dark:text-white {key.is_active ? '' : 'line-through opacity-70'}">
											{key.label}
										</span>
									</td>
									<td class="py-3 px-4">
										<code class="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded font-mono">{key.key_prefix}…</code>
									</td>
									<td class="py-3 px-4">
										<div class="flex flex-wrap gap-1">
											{#each key.scopes as scope}
												<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-200">
													{scope}
												</span>
											{/each}
										</div>
									</td>
									<td class="py-3 px-4">
										<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {statusClass(key)}">
											{statusLabel(key)}
										</span>
									</td>
									<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
										{formatDate(key.last_used_at)}
									</td>
									<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
										{formatDate(key.created)}
									</td>
									<td class="py-3 px-4 text-end">
										{#if key.is_active}
											<button
												type="button"
												class="btn btn-ghost btn-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
												onclick={() => revokeKey(key)}
												aria-label={$t('admin.api_keys.revoke_aria', { values: { label: key.label } })}
											>
												{$t('admin.api_keys.revoke')}
											</button>
										{:else}
											<span class="text-xs text-gray-400 dark:text-gray-500">
												{$t('admin.api_keys.status_revoked')}
											</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</section>
</div>

<!-- One-time reveal modal (native <dialog>) -->
<dialog
	bind:this={revealDialog}
	class="rounded-lg p-0 bg-transparent backdrop:bg-black/50 max-w-2xl w-full"
	aria-labelledby="reveal-heading"
	aria-describedby="reveal-body"
	onkeydown={onRevealKeydown}
	oncancel={(e) => {
		e.preventDefault();
		dismissReveal();
	}}
>
	{#if createdKey}
		<div class="bg-white dark:bg-gray-800 rounded-lg p-6 border-2 border-amber-300 dark:border-amber-600">
			<div class="flex items-start gap-3">
				<svg
					class="w-6 h-6 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"
					/>
				</svg>
				<div class="flex-1 min-w-0">
					<h3 id="reveal-heading" class="text-lg font-semibold text-amber-900 dark:text-amber-100">
						{$t('admin.api_keys.reveal_heading')}
					</h3>
					<p id="reveal-body" class="text-sm text-amber-800 dark:text-amber-200 mt-1">
						{$t('admin.api_keys.reveal_warning')}
					</p>
					<div class="mt-4 flex items-stretch gap-2">
						<code class="flex-1 px-4 py-3 bg-gray-900 text-green-400 rounded font-mono text-sm break-all select-all">{createdKey}</code>
						<button
							type="button"
							class="btn {copied ? 'btn-success' : 'btn-primary'} shrink-0"
							onclick={copyToClipboard}
						>
							{copied ? $t('admin.api_keys.copied') : $t('admin.api_keys.copy')}
						</button>
					</div>
					<div class="mt-4 flex items-center justify-end">
						<button type="button" class="btn btn-primary" onclick={dismissReveal}>
							{$t('admin.api_keys.reveal_done')}
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</dialog>

<style>
	dialog::backdrop {
		background-color: rgba(0, 0, 0, 0.5);
	}
</style>

<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { toasts } from '$lib/stores';
	import { hasFeature } from '$lib/stores/plan';

	const hasPricing = hasFeature('pricing');

	type StripeStatus = {
		configured: boolean;
		secret_key_set: boolean;
		webhook_secret_set: boolean;
		source: 'database' | 'environment' | 'none';
	};

	let loading = $state(true);
	let status = $state<StripeStatus>({
		configured: false,
		secret_key_set: false,
		webhook_secret_set: false,
		source: 'none'
	});

	// Secret key state
	let secretKey = $state('');
	let secretKeyEditing = $state(false);
	let secretKeySaving = $state(false);
	let secretKeyClearing = $state(false);

	// Webhook secret state
	let webhookSecret = $state('');
	let webhookSecretEditing = $state(false);
	let webhookSecretSaving = $state(false);
	let webhookSecretClearing = $state(false);

	// Test connection state
	let testing = $state(false);

	onMount(async () => {
		if ($hasPricing) {
			await loadStatus();
		}
	});

	async function loadStatus() {
		loading = true;
		try {
			const response = await fetch('/api/stripe-settings', {
				headers: { Authorization: pb.authStore.token }
			});
			if (response.ok) {
				status = await response.json();
			}
		} catch (err) {
			console.error('Failed to load Stripe settings:', err);
		} finally {
			loading = false;
		}
	}

	async function saveSecretKey() {
		if (!secretKey.trim()) return;
		secretKeySaving = true;
		try {
			const response = await fetch('/api/stripe-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({ secret_key: secretKey })
			});
			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || $t('admin.settings_page.payments.save_error'));
				return;
			}
			secretKey = '';
			secretKeyEditing = false;
			toasts.add('success', $t('admin.settings_page.payments.save_success'));
			await loadStatus();
		} catch (err) {
			console.error('Failed to save Stripe secret key:', err);
			toasts.add('error', $t('admin.settings_page.payments.save_error'));
		} finally {
			secretKeySaving = false;
		}
	}

	async function saveWebhookSecret() {
		if (!webhookSecret.trim()) return;
		webhookSecretSaving = true;
		try {
			const response = await fetch('/api/stripe-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({ webhook_secret: webhookSecret })
			});
			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || $t('admin.settings_page.payments.save_error'));
				return;
			}
			webhookSecret = '';
			webhookSecretEditing = false;
			toasts.add('success', $t('admin.settings_page.payments.save_success'));
			await loadStatus();
		} catch (err) {
			console.error('Failed to save Stripe webhook secret:', err);
			toasts.add('error', $t('admin.settings_page.payments.save_error'));
		} finally {
			webhookSecretSaving = false;
		}
	}

	async function clearSettings() {
		secretKeyClearing = true;
		webhookSecretClearing = true;
		try {
			const response = await fetch('/api/stripe-settings', {
				method: 'DELETE',
				headers: { Authorization: pb.authStore.token }
			});
			if (!response.ok) {
				const result = await response.json();
				toasts.add('error', result.error || $t('admin.settings_page.payments.clear_error'));
				return;
			}
			secretKey = '';
			webhookSecret = '';
			secretKeyEditing = false;
			webhookSecretEditing = false;
			toasts.add('success', $t('admin.settings_page.payments.clear_success'));
			await loadStatus();
		} catch (err) {
			console.error('Failed to clear Stripe settings:', err);
			toasts.add('error', $t('admin.settings_page.payments.clear_error'));
		} finally {
			secretKeyClearing = false;
			webhookSecretClearing = false;
		}
	}

	async function testConnection() {
		testing = true;
		try {
			const response = await fetch('/api/stripe-settings/test', {
				method: 'POST',
				headers: { Authorization: pb.authStore.token }
			});
			const result = await response.json();
			if (result.success) {
				toasts.add('success', $t('admin.settings_page.payments.test_success'));
			} else {
				toasts.add('error', $t('admin.settings_page.payments.test_failed', { values: { error: result.error || 'Unknown error' } }));
			}
		} catch (err) {
			console.error('Failed to test Stripe connection:', err);
			toasts.add('error', $t('admin.settings_page.payments.test_failed', { values: { error: 'Network error' } }));
		} finally {
			testing = false;
		}
	}

	let statusColor = $derived.by(() => {
		if (status.source === 'database') return 'green';
		if (status.source === 'environment') return 'yellow';
		return 'gray';
	});
</script>

<svelte:head>
	<title>{$t('admin.settings_page.payments.title')} - {$t('admin.settings_page.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	{#if !$hasPricing}
		<!-- Feature not available -->
		<div class="card p-8 text-center">
			<div class="w-12 h-12 mx-auto mb-4 rounded-full bg-gray-100 dark:bg-gray-700 flex items-center justify-center">
				<svg class="w-6 h-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
				</svg>
			</div>
			<p class="text-gray-500 dark:text-gray-400">{$t('admin.settings_page.payments.feature_disabled')}</p>
		</div>
	{:else}
		<div class="space-y-6">
			<!-- Page header -->
			<div>
				<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.payments.title')}</p>
				<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.payments.description')}</p>
			</div>

			<!-- Status card -->
			<div class="card p-6">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.settings_page.payments.status')}</h2>

				{#if loading}
					<div class="animate-pulse h-6 bg-gray-200 dark:bg-gray-700 rounded w-48"></div>
				{:else}
					<div class="flex items-center gap-3">
						<span
							class="inline-block w-3 h-3 rounded-full shrink-0
								{statusColor === 'green' ? 'bg-green-500' : statusColor === 'yellow' ? 'bg-yellow-500' : 'bg-gray-400'}"
						></span>
						<span class="text-gray-700 dark:text-gray-300 font-medium">
							{#if status.source === 'database'}
								{$t('admin.settings_page.payments.configured')}
							{:else if status.source === 'environment'}
								{$t('admin.settings_page.payments.configured_via_env')}
							{:else}
								{$t('admin.settings_page.payments.not_configured')}
							{/if}
						</span>
					</div>
				{/if}
			</div>

			<!-- Keys configuration -->
			<div class="card p-6 space-y-6">
				<!-- Secret Key -->
				<div>
					<label for="stripe-secret-key" class="label">{$t('admin.settings_page.payments.secret_key')}</label>
					{#if status.secret_key_set && !secretKeyEditing}
						<div class="flex items-center gap-3">
							<span class="text-sm text-green-600 dark:text-green-400 flex items-center gap-1.5">
								<svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
								{$t('admin.settings_page.payments.secret_key_set')}
							</span>
							<button
								type="button"
								class="btn btn-sm btn-secondary"
								onclick={() => { secretKeyEditing = true; }}
							>
								{$t('admin.settings_page.payments.change')}
							</button>
						</div>
					{:else}
						<div class="flex gap-2">
							<input
								type="password"
								id="stripe-secret-key"
								bind:value={secretKey}
								class="input flex-1"
								placeholder={$t('admin.settings_page.payments.secret_key_placeholder')}
								disabled={secretKeySaving}
								autocomplete="off"
							/>
							<button
								type="button"
								class="btn btn-primary btn-sm"
								onclick={saveSecretKey}
								disabled={secretKeySaving || !secretKey.trim()}
							>
								{secretKeySaving ? '...' : $t('admin.settings_page.payments.save')}
							</button>
							{#if secretKeyEditing}
								<button
									type="button"
									class="btn btn-secondary btn-sm"
									onclick={() => { secretKeyEditing = false; secretKey = ''; }}
									disabled={secretKeySaving}
								>
									{$t('admin.settings_page.payments.change')}
								</button>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Webhook Secret -->
				<div>
					<label for="stripe-webhook-secret" class="label">{$t('admin.settings_page.payments.webhook_secret')}</label>
					{#if status.webhook_secret_set && !webhookSecretEditing}
						<div class="flex items-center gap-3">
							<span class="text-sm text-green-600 dark:text-green-400 flex items-center gap-1.5">
								<svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
								{$t('admin.settings_page.payments.webhook_secret_set')}
							</span>
							<button
								type="button"
								class="btn btn-sm btn-secondary"
								onclick={() => { webhookSecretEditing = true; }}
							>
								{$t('admin.settings_page.payments.change')}
							</button>
						</div>
					{:else}
						<div class="flex gap-2">
							<input
								type="password"
								id="stripe-webhook-secret"
								bind:value={webhookSecret}
								class="input flex-1"
								placeholder={$t('admin.settings_page.payments.webhook_secret_placeholder')}
								disabled={webhookSecretSaving}
								autocomplete="off"
							/>
							<button
								type="button"
								class="btn btn-primary btn-sm"
								onclick={saveWebhookSecret}
								disabled={webhookSecretSaving || !webhookSecret.trim()}
							>
								{webhookSecretSaving ? '...' : $t('admin.settings_page.payments.save')}
							</button>
							{#if webhookSecretEditing}
								<button
									type="button"
									class="btn btn-secondary btn-sm"
									onclick={() => { webhookSecretEditing = false; webhookSecret = ''; }}
									disabled={webhookSecretSaving}
								>
									{$t('admin.settings_page.payments.change')}
								</button>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Actions -->
				<div class="flex flex-wrap items-center gap-3 pt-2 border-t border-gray-200 dark:border-gray-700">
					{#if status.configured}
						<button
							type="button"
							class="btn btn-secondary"
							onclick={testConnection}
							disabled={testing}
						>
							{#if testing}
								<svg class="animate-spin h-4 w-4 mr-2 inline-block" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
							{/if}
							{$t('admin.settings_page.payments.test_connection')}
						</button>

						{#if status.source === 'database'}
							<button
								type="button"
								class="btn btn-danger-ghost"
								onclick={clearSettings}
								disabled={secretKeyClearing || webhookSecretClearing}
							>
								{$t('admin.settings_page.payments.clear')}
							</button>
						{/if}
					{/if}
				</div>
			</div>

			<!-- Info box -->
			<div class="rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 p-4">
				<div class="flex gap-3">
					<svg class="w-5 h-5 shrink-0 text-blue-500 dark:text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<p class="text-sm text-blue-700 dark:text-blue-300">
						{$t('admin.settings_page.payments.env_fallback_info')}
					</p>
				</div>
			</div>
		</div>
	{/if}
</div>

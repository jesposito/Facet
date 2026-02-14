<script lang="ts">
	import { run } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { toasts, confirm } from '$lib/stores';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import { icon } from '$lib/icons';
	import { preventDefault } from 'svelte/legacy';

	let loading = $state(true);
	let providers: Array<Record<string, unknown>> = $state([]);
	let showAddForm = $state(false);
	let testing: string | null = $state(null);

	// New provider form
	let newProvider = $state({
		name: '',
		type: 'openai' as 'openai' | 'anthropic' | 'ollama' | 'custom',
		api_key: '',
		base_url: '',
		model: '',
		is_active: true,
		is_default: false
	});

	// Fallback model options per provider type (used when API fetch fails or is unavailable)
	const fallbackModelOptions: Record<string, string[]> = {
		openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo', 'o1', 'o1-mini'],
		anthropic: ['claude-sonnet-4-20250514', 'claude-opus-4-20250514', 'claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022'],
		ollama: ['llama3.2', 'llama3.1', 'mistral', 'codellama', 'phi3'],
		custom: []
	};

	const defaultModels: Record<string, string> = {
		openai: 'gpt-4o-mini',
		anthropic: 'claude-sonnet-4-20250514',
		ollama: 'llama3.2',
		custom: ''
	};

	// Dynamic model state
	let fetchedModels = $state<string[]>([]);
	let fetchingModels = $state(false);
	let modelFetchError = $state('');

	// Computed: use fetched models if available, otherwise fallback
	let modelOptions = $derived.by(() => {
		if (fetchedModels.length > 0) {
			return fetchedModels;
		}
		return fallbackModelOptions[newProvider.type] || [];
	});

	// Fetch models when API key changes or provider type changes
	async function fetchModels() {
		// Don't fetch for custom without base_url, or if no API key for non-ollama
		if (newProvider.type === 'custom' && !newProvider.base_url) {
			fetchedModels = [];
			return;
		}
		if (newProvider.type !== 'ollama' && !newProvider.api_key) {
			fetchedModels = [];
			return;
		}

		fetchingModels = true;
		modelFetchError = '';

		try {
			const response = await fetch('/api/ai/models', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					type: newProvider.type,
					api_key: newProvider.api_key,
					base_url: newProvider.base_url || undefined
				})
			});

			const data = await response.json();
			if (data.error) {
				modelFetchError = data.error;
				fetchedModels = [];
			} else if (data.models && data.models.length > 0) {
				fetchedModels = data.models;
				// If current model not in list, reset to first or default
				if (!data.models.includes(newProvider.model)) {
					newProvider.model = data.models.includes(defaultModels[newProvider.type])
						? defaultModels[newProvider.type]
						: data.models[0];
				}
			} else {
				fetchedModels = [];
			}
		} catch (err) {
			modelFetchError = 'Failed to fetch models';
			fetchedModels = [];
		} finally {
			fetchingModels = false;
		}
	}

	// Reset model and fetch when provider type changes
	run(() => {
		if (newProvider.type) {
			const options = fallbackModelOptions[newProvider.type] || [];
			if (!options.includes(newProvider.model)) {
				newProvider.model = defaultModels[newProvider.type] || '';
			}
			// Reset fetched models when type changes
			fetchedModels = [];
			modelFetchError = '';
		}
	});

	onMount(async () => {
		await loadProviders();
	});

	async function loadProviders() {
		try {
			const result = await pb.collection('ai_providers').getList(1, 50);
			providers = result.items;
		} catch (err) {
			console.error('Failed to load providers:', err);
		} finally {
			loading = false;
		}
	}

	async function handleAddProvider() {
		try {
			// Build payload, excluding empty optional fields that might fail validation
			// Note: Don't send api_key_encrypted - it's a hidden field set by backend hook
			const payload: Record<string, unknown> = {
				name: newProvider.name,
				type: newProvider.type,
				api_key: newProvider.api_key,
				model: newProvider.model,
				is_active: newProvider.is_active,
				is_default: newProvider.is_default
			};
			// Only include base_url if it has a value (URLField rejects empty strings)
			if (newProvider.base_url) {
				payload.base_url = newProvider.base_url;
			}
			await pb.collection('ai_providers').create(payload);

			toasts.add('success', $t('admin.settings_page.integrations.ai_provider_added'));
			showAddForm = false;
			newProvider = {
				name: '',
				type: 'openai',
				api_key: '',
				base_url: '',
				model: '',
				is_active: true,
				is_default: false
			};
			await loadProviders();
		} catch (err: unknown) {
			console.error('Failed to add provider:', err);
			// Log full error for debugging
			console.error('[AI-PROVIDER] Full error object:', JSON.stringify(err, null, 2));
			// Extract detailed error from PocketBase ClientResponseError
			let message = $t('admin.settings_page.integrations.ai_add_error');
			if (err && typeof err === 'object' && 'data' in err) {
				const pbErr = err as { data?: { data?: Record<string, { message: string }>, message?: string } };
				console.error('[AI-PROVIDER] Error data:', pbErr.data);
				const fieldErrors = pbErr.data?.data;
				if (fieldErrors && Object.keys(fieldErrors).length > 0) {
					const details = Object.entries(fieldErrors)
						.map(([field, info]) => `${field}: ${info.message}`)
						.join(', ');
					message = `Validation error: ${details}`;
				} else if (pbErr.data?.message) {
					message = pbErr.data.message;
				}
			} else if (err instanceof Error) {
				message = err.message;
			}
			toasts.add('error', message);
		}
	}

	async function testConnection(id: string) {
		testing = id;
		try {
			const response = await fetch(`/api/ai/test/${id}`, {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token
				}
			});

			const result = await response.json();
			if (result.success) {
				toasts.add('success', $t('admin.settings_page.integrations.ai_test_success'));
			} else {
				toasts.add('error', $t('admin.settings_page.integrations.ai_test_failed', { values: { error: result.error } }));
			}
			await loadProviders();
		} catch (err) {
			toasts.add('error', $t('admin.settings_page.integrations.ai_test_error'));
		} finally {
			testing = null;
		}
	}

	async function deleteProvider(id: string) {
		const confirmed = await confirm({
			title: $t('admin.settings_page.integrations.ai_delete_title'),
			message: $t('admin.settings_page.integrations.ai_delete_message'),
			confirmText: $t('admin.settings_page.integrations.ai_delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('ai_providers').delete(id);
			toasts.add('success', $t('admin.settings_page.integrations.ai_provider_deleted'));
			await loadProviders();
		} catch (err) {
			toasts.add('error', $t('admin.settings_page.integrations.ai_add_error'));
		}
	}

	async function setDefault(id: string) {
		try {
			// Unset current defaults
			for (const p of providers) {
				if (p.is_default) {
					await pb.collection('ai_providers').update(p.id as string, { is_default: false });
				}
			}
			// Set new default
			await pb.collection('ai_providers').update(id, { is_default: true });
			toasts.add('success', $t('admin.settings_page.integrations.ai_default_updated'));
			await loadProviders();
		} catch (err) {
			toasts.add('error', $t('admin.settings_page.integrations.ai_add_error'));
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.settings_page.integrations.section_title')} - {$t('admin.settings_page.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<PageHelp pageKey="settings">
		<p>{@html $t('admin.settings_page.help_text')}</p>
		<p>{$t('admin.settings_page.help_tip_1')}</p>
		<p>{@html $t('admin.settings_page.help_tip_2')}</p>
	</PageHelp>

	<!-- Integrations section -->
	<div class="space-y-4">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.integrations.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.integrations.section_description')}</p>
		</div>

		<div class="card p-6">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.integrations.ai_providers_title')}</h2>
			<button class="btn btn-primary btn-sm" onclick={() => (showAddForm = !showAddForm)}>
				{showAddForm ? $t('admin.settings_page.integrations.ai_cancel') : $t('admin.settings_page.integrations.ai_add_provider')}
			</button>
		</div>

		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			{$t('admin.settings_page.integrations.ai_description')}
		</p>

		{#if showAddForm}
			<form onsubmit={preventDefault(handleAddProvider)} class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4 space-y-4">
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label for="name" class="label">{$t('admin.settings_page.integrations.ai_form_name_label')}</label>
						<input
							type="text"
							id="name"
							bind:value={newProvider.name}
							class="input"
							placeholder={$t('admin.settings_page.integrations.ai_form_name_placeholder')}
							required
						/>
					</div>
					<div>
						<label for="type" class="label">{$t('admin.settings_page.integrations.ai_form_type_label')}</label>
						<select id="type" bind:value={newProvider.type} class="input">
							<option value="openai">{$t('admin.settings_page.integrations.ai_form_type_openai')}</option>
							<option value="anthropic">{$t('admin.settings_page.integrations.ai_form_type_anthropic')}</option>
							<option value="ollama">{$t('admin.settings_page.integrations.ai_form_type_ollama')}</option>
							<option value="custom">{$t('admin.settings_page.integrations.ai_form_type_custom')}</option>
						</select>
					</div>
				</div>

				<div>
					<label for="api_key" class="label">
						{$t('admin.settings_page.integrations.ai_form_api_key_label')}
						{#if newProvider.type === 'ollama'}
							<span class="text-gray-500 font-normal">{$t('admin.settings_page.integrations.ai_form_api_key_not_required')}</span>
						{/if}
					</label>
					<input
						type="password"
						id="api_key"
						bind:value={newProvider.api_key}
						class="input"
						placeholder={newProvider.type === 'ollama' ? $t('admin.settings_page.integrations.ai_form_api_key_placeholder_optional') : $t('admin.settings_page.integrations.ai_form_api_key_placeholder')}
						required={newProvider.type !== 'ollama'}
					/>
				</div>

				{#if newProvider.type === 'ollama' || newProvider.type === 'custom'}
					<div>
						<label for="base_url" class="label">{$t('admin.settings_page.integrations.ai_form_base_url_label')}</label>
						<input
							type="url"
							id="base_url"
							bind:value={newProvider.base_url}
							class="input"
							placeholder={newProvider.type === 'ollama' ? 'http://localhost:11434' : 'https://api.example.com/v1'}
						/>
					</div>
				{/if}

				<div>
					<div class="flex items-center justify-between mb-1">
						<label for="model" class="label mb-0">{$t('admin.settings_page.integrations.ai_form_model_label')}</label>
						{#if newProvider.type !== 'custom' || newProvider.base_url}
							<button
								type="button"
								class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 flex items-center gap-1"
								onclick={fetchModels}
								disabled={fetchingModels || (newProvider.type !== 'ollama' && !newProvider.api_key)}
							>
								{#if fetchingModels}
									<svg class="animate-spin h-3 w-3" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
									{$t('admin.settings_page.integrations.ai_form_fetching')}
								{:else}
									<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
									</svg>
									{$t('admin.settings_page.integrations.ai_form_fetch_models')}
								{/if}
							</button>
						{/if}
					</div>
					{#if modelOptions.length > 0}
						<select id="model" bind:value={newProvider.model} class="input">
							{#each modelOptions as model}
								<option value={model}>{model}</option>
							{/each}
						</select>
						{#if fetchedModels.length > 0}
							<p class="text-xs text-green-600 dark:text-green-400 mt-1">
								{$t('admin.settings_page.integrations.ai_form_models_available', { values: { count: fetchedModels.length } })}
							</p>
						{/if}
					{:else}
						<input
							type="text"
							id="model"
							bind:value={newProvider.model}
							class="input"
							placeholder={$t('admin.settings_page.integrations.ai_form_model_placeholder')}
						/>
					{/if}
					{#if modelFetchError}
						<p class="text-xs text-amber-600 dark:text-amber-400 mt-1">
							{$t('admin.settings_page.integrations.ai_form_model_fetch_error', { values: { error: modelFetchError } })}
						</p>
					{/if}
				</div>

				<div class="flex items-center gap-4">
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_active} class="w-4 h-4" />
						<span>{$t('admin.settings_page.integrations.ai_form_active_label')}</span>
					</label>
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_default} class="w-4 h-4" />
						<span>{$t('admin.settings_page.integrations.ai_form_default_label')}</span>
					</label>
				</div>

				<button type="submit" class="btn btn-primary">{$t('admin.settings_page.integrations.ai_form_submit')}</button>
			</form>
		{/if}

		{#if loading}
			<div class="animate-pulse text-center py-4">{$t('admin.settings_page.integrations.ai_loading')}</div>
		{:else if providers.length === 0}
			<p class="text-gray-500 dark:text-gray-400 text-center py-8">
				{$t('admin.settings_page.integrations.ai_empty_message')}
			</p>
		{:else}
			<div class="space-y-3">
				{#each providers as provider}
					<div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
						<div class="flex items-center gap-3">
							<div
								class="w-10 h-10 rounded-lg flex items-center justify-center
								{provider.type === 'openai'
									? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
									: provider.type === 'anthropic'
										? 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300'
										: 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300'}"
							>
								{#if provider.type === 'openai'}
									<span class="text-lg font-bold">O</span>
								{:else if provider.type === 'anthropic'}
									<span class="text-lg font-bold">A</span>
								{:else if provider.type === 'ollama'}
									{@html icon('brain')}
								{:else}
									{@html icon('zap')}
								{/if}
							</div>
							<div>
								<div class="flex items-center gap-2">
									<span class="font-medium text-gray-900 dark:text-white">{provider.name}</span>
									{#if provider.is_default}
										<span class="px-2 py-0.5 text-xs bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300 rounded">
											{$t('admin.settings_page.integrations.ai_default_badge')}
										</span>
									{/if}
									{#if !provider.is_active}
										<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											{$t('admin.settings_page.integrations.ai_inactive_badge')}
										</span>
									{/if}
								</div>
								<p class="text-sm text-gray-500 dark:text-gray-400">
									{provider.type} • {provider.model || 'default model'}
									{#if provider.test_status}
										• Last test: {provider.test_status}
									{/if}
								</p>
							</div>
						</div>

						<div class="flex items-center gap-2">
							<button
								class="btn btn-sm btn-secondary"
								onclick={() => testConnection(String(provider.id))}
								disabled={testing === provider.id}
							>
								{#if testing === provider.id}
									<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
								{:else}
									{$t('admin.settings_page.integrations.ai_test_button')}
								{/if}
							</button>
							{#if !provider.is_default}
								<button
									class="btn btn-sm btn-secondary"
									onclick={() => setDefault(String(provider.id))}
								>
									{$t('admin.settings_page.integrations.ai_set_default_button')}
								</button>
							{/if}
							<button
								class="btn btn-danger-ghost btn-sm"
								onclick={() => deleteProvider(String(provider.id))}
							>
								{$t('admin.settings_page.integrations.ai_delete_button')}
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
		</div>
	</div>
</div>

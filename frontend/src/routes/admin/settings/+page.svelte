<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';

	let loading = true;
	let providers: Array<Record<string, unknown>> = [];
	let showAddForm = false;
	let testing: string | null = null;

	// New provider form
	let newProvider = {
		name: '',
		type: 'openai' as 'openai' | 'anthropic' | 'ollama' | 'custom',
		api_key: '',
		base_url: '',
		model: '',
		is_active: true,
		is_default: false
	};

	const defaultModels: Record<string, string> = {
		openai: 'gpt-4o-mini',
		anthropic: 'claude-3-haiku-20240307',
		ollama: 'llama3.2',
		custom: ''
	};

	$: newProvider.model = newProvider.model || defaultModels[newProvider.type] || '';

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
			await pb.collection('ai_providers').create({
				...newProvider,
				api_key_encrypted: '' // Will be encrypted by hook
			});

			toasts.add('success', 'AI provider added');
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
		} catch (err) {
			console.error('Failed to add provider:', err);
			toasts.add('error', 'Failed to add provider');
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
				toasts.add('success', 'Connection successful!');
			} else {
				toasts.add('error', `Connection failed: ${result.error}`);
			}
			await loadProviders();
		} catch (err) {
			toasts.add('error', 'Connection test failed');
		} finally {
			testing = null;
		}
	}

	async function deleteProvider(id: string) {
		if (!confirm('Are you sure you want to delete this provider?')) return;

		try {
			await pb.collection('ai_providers').delete(id);
			toasts.add('success', 'Provider deleted');
			await loadProviders();
		} catch (err) {
			toasts.add('error', 'Failed to delete provider');
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
			toasts.add('success', 'Default provider updated');
			await loadProviders();
		} catch (err) {
			toasts.add('error', 'Failed to update default');
		}
	}
</script>

<svelte:head>
	<title>Settings | Me.yaml</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Settings</h1>

	<!-- AI Providers -->
	<div class="card p-6">
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">AI Providers</h2>
			<button class="btn btn-primary btn-sm" on:click={() => (showAddForm = !showAddForm)}>
				{showAddForm ? 'Cancel' : '+ Add Provider'}
			</button>
		</div>

		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			Configure AI providers for enriching imported projects. Your API keys are encrypted at rest.
		</p>

		{#if showAddForm}
			<form on:submit|preventDefault={handleAddProvider} class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4 space-y-4">
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="name" class="label">Name</label>
						<input
							type="text"
							id="name"
							bind:value={newProvider.name}
							class="input"
							placeholder="My OpenAI"
							required
						/>
					</div>
					<div>
						<label for="type" class="label">Provider Type</label>
						<select id="type" bind:value={newProvider.type} class="input">
							<option value="openai">OpenAI</option>
							<option value="anthropic">Anthropic</option>
							<option value="ollama">Ollama</option>
							<option value="custom">Custom (OpenAI-compatible)</option>
						</select>
					</div>
				</div>

				<div>
					<label for="api_key" class="label">
						API Key
						{#if newProvider.type === 'ollama'}
							<span class="text-gray-500 font-normal">(not required for Ollama)</span>
						{/if}
					</label>
					<input
						type="password"
						id="api_key"
						bind:value={newProvider.api_key}
						class="input"
						placeholder={newProvider.type === 'ollama' ? 'Optional' : 'sk-...'}
						required={newProvider.type !== 'ollama'}
					/>
				</div>

				{#if newProvider.type === 'ollama' || newProvider.type === 'custom'}
					<div>
						<label for="base_url" class="label">Base URL</label>
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
					<label for="model" class="label">Model</label>
					<input
						type="text"
						id="model"
						bind:value={newProvider.model}
						class="input"
						placeholder={defaultModels[newProvider.type]}
					/>
				</div>

				<div class="flex items-center gap-4">
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_active} class="w-4 h-4" />
						<span>Active</span>
					</label>
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_default} class="w-4 h-4" />
						<span>Set as default</span>
					</label>
				</div>

				<button type="submit" class="btn btn-primary">Add Provider</button>
			</form>
		{/if}

		{#if loading}
			<div class="animate-pulse text-center py-4">Loading providers...</div>
		{:else if providers.length === 0}
			<p class="text-gray-500 dark:text-gray-400 text-center py-8">
				AI providers help generate project descriptions during import. Add one when you're ready.
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
											Default
										</span>
									{/if}
									{#if !provider.is_active}
										<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											Inactive
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
								on:click={() => testConnection(String(provider.id))}
								disabled={testing === provider.id}
							>
								{#if testing === provider.id}
									<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
								{:else}
									Test
								{/if}
							</button>
							{#if !provider.is_default}
								<button
									class="btn btn-sm btn-secondary"
									on:click={() => setDefault(String(provider.id))}
								>
									Set Default
								</button>
							{/if}
							<button
								class="btn btn-sm btn-ghost text-red-600"
								on:click={() => deleteProvider(String(provider.id))}
							>
								Delete
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Export Section -->
	<div class="card p-6 mt-6">
		<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Data Export</h2>
		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			Export your profile data in a portable format.
		</p>
		<button class="btn btn-secondary inline-flex items-center gap-2">
			{@html icon('download')}
			Download me.yaml
		</button>
	</div>
</div>

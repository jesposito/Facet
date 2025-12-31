<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { goto } from '$app/navigation';

	let step = 1;
	let loading = false;
	let error = '';

	// Step 1: Repo input
	let repoUrl = '';
	let githubToken = '';

	// Step 2: Preview
	let preview: Record<string, unknown> | null = null;

	// Step 3: AI enrichment
	let useAI = false;
	let selectedProvider = '';
	let privacyMode: 'full' | 'summary' | 'none' = 'summary';
	let providers: Array<{ id: string; name: string; type: string }> = [];

	// Step 4: Proposal
	let proposalId = '';

	async function loadProviders() {
		try {
			const result = await pb.collection('ai_providers').getList(1, 50, {
				filter: 'is_active = true'
			});
			providers = result.items.map((p) => ({
				id: p.id,
				name: p.name as string,
				type: p.type as string
			}));
			if (providers.length > 0) {
				const defaultProvider = result.items.find((p) => p.is_default);
				selectedProvider = defaultProvider?.id || providers[0].id;
			}
		} catch (err) {
			console.error('Failed to load AI providers:', err);
		}
	}

	async function handlePreview() {
		if (!repoUrl) {
			error = 'Please enter a repository URL';
			return;
		}

		loading = true;
		error = '';

		try {
			const response = await fetch('/api/github/preview', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					repo_url: repoUrl,
					token: githubToken
				})
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to fetch repository');
			}

			preview = await response.json();
			step = 2;
			await loadProviders();
		} catch (err) {
			error = (err as Error).message;
		} finally {
			loading = false;
		}
	}

	async function handleImport() {
		loading = true;
		error = '';

		try {
			const response = await fetch('/api/github/import', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					repo_url: repoUrl,
					token: githubToken,
					ai_enrich: useAI,
					ai_provider_id: useAI ? selectedProvider : '',
					privacy_mode: privacyMode
				})
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Import failed');
			}

			const result = await response.json();
			proposalId = result.proposal_id;
			toasts.add('success', 'Import proposal created! Review the changes below.');
			goto(`/admin/review/${proposalId}`);
		} catch (err) {
			error = (err as Error).message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Import from GitHub | Me.yaml</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Import from GitHub</h1>

	<!-- Progress steps -->
	<div class="flex items-center mb-8">
		{#each ['Repository', 'Preview', 'Enrich', 'Review'] as label, i}
			<div class="flex items-center">
				<div
					class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium
					{step > i + 1
						? 'bg-primary-600 text-white'
						: step === i + 1
							? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
							: 'bg-gray-200 text-gray-500 dark:bg-gray-700'}"
				>
					{i + 1}
				</div>
				{#if i < 3}
					<div
						class="w-12 h-1 mx-2 {step > i + 1 ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'}"
					></div>
				{/if}
			</div>
		{/each}
	</div>

	{#if error}
		<div class="mb-4 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300">
			{error}
		</div>
	{/if}

	<!-- Step 1: Repository URL -->
	{#if step === 1}
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Enter Repository</h2>

			<div class="space-y-4">
				<div>
					<label for="repo" class="label">Repository URL or owner/repo</label>
					<input
						type="text"
						id="repo"
						bind:value={repoUrl}
						class="input"
						placeholder="e.g., github.com/username/repo or username/repo"
					/>
				</div>

				<div>
					<label for="token" class="label">
						GitHub Token
						<span class="text-gray-500 font-normal">(optional, for private repos)</span>
					</label>
					<input
						type="password"
						id="token"
						bind:value={githubToken}
						class="input"
						placeholder="ghp_..."
					/>
					<p class="text-xs text-gray-500 mt-1">
						Token is used only for this import and not stored unless you choose to save it.
					</p>
				</div>

				<button class="btn btn-primary" on:click={handlePreview} disabled={loading}>
					{#if loading}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Parsing…
					{:else}
						Fetch Repository
					{/if}
				</button>
			</div>
		</div>
	{/if}

	<!-- Step 2: Preview -->
	{#if step === 2 && preview}
		<div class="card p-6 space-y-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Repository Preview</h2>

			<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
				<h3 class="font-medium text-gray-900 dark:text-white">{preview.name}</h3>
				{#if preview.description}
					<p class="text-gray-600 dark:text-gray-400 mt-1">{preview.description}</p>
				{/if}

				<div class="flex flex-wrap gap-4 mt-3 text-sm text-gray-500">
					{#if preview.stargazers_count}
						<span>⭐ {preview.stargazers_count} stars</span>
					{/if}
					{#if preview.forks_count}
						<span>🍴 {preview.forks_count} forks</span>
					{/if}
				</div>

				{#if preview.topics && (preview.topics as string[]).length > 0}
					<div class="flex flex-wrap gap-1 mt-3">
						{#each preview.topics as topic}
							<span class="px-2 py-0.5 text-xs bg-gray-200 dark:bg-gray-700 rounded">
								{topic}
							</span>
						{/each}
					</div>
				{/if}

				{#if preview.languages}
					<div class="mt-3">
						<span class="text-sm font-medium">Languages: </span>
						<span class="text-sm text-gray-600 dark:text-gray-400">
							{Object.keys(preview.languages as object).join(', ')}
						</span>
					</div>
				{/if}
			</div>

			<div class="flex gap-3">
				<button class="btn btn-secondary" on:click={() => (step = 1)}>Back</button>
				<button class="btn btn-primary" on:click={() => (step = 3)}>Continue</button>
			</div>
		</div>
	{/if}

	<!-- Step 3: AI Enrichment -->
	{#if step === 3}
		<div class="card p-6 space-y-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">AI Enrichment</h2>

			<p class="text-gray-600 dark:text-gray-400">
				Optionally use AI to generate a polished summary and highlight key features.
			</p>

			<div class="flex items-center gap-3">
				<input type="checkbox" id="useAI" bind:checked={useAI} class="w-4 h-4" />
				<label for="useAI" class="font-medium">Enable AI enrichment</label>
			</div>

			{#if useAI}
				<div class="pl-7 space-y-4">
					{#if providers.length === 0}
						<p class="text-gray-600 dark:text-gray-400 text-sm">
							You can <a href="/admin/settings" class="text-primary-600 dark:text-primary-400 underline">configure an AI provider</a> to automatically generate descriptions.
						</p>
					{:else}
						<div>
							<label for="provider" class="label">AI Provider</label>
							<select id="provider" bind:value={selectedProvider} class="input">
								{#each providers as provider}
									<option value={provider.id}>{provider.name} ({provider.type})</option>
								{/each}
							</select>
						</div>

						<div>
							<label for="privacy" class="label">README Privacy</label>
							<select id="privacy" bind:value={privacyMode} class="input">
								<option value="full">Send full README to AI</option>
								<option value="summary">Send only first 500 characters</option>
								<option value="none">Don't send README content</option>
							</select>
							<p class="text-xs text-gray-500 mt-1">
								Controls how much of your README is shared with the AI provider
							</p>
						</div>
					{/if}
				</div>
			{/if}

			<div class="flex gap-3 pt-4">
				<button class="btn btn-secondary" on:click={() => (step = 2)}>Back</button>
				<button
					class="btn btn-primary"
					on:click={handleImport}
					disabled={loading || (useAI && providers.length === 0)}
				>
					{#if loading}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Importing…
					{:else}
						Review & Import
					{/if}
				</button>
			</div>
		</div>
	{/if}
</div>

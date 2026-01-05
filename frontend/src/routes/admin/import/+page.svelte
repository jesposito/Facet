<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { icon } from '$lib/icons';

	// Resume upload state
	let resumeFile: File | null = null;
	let resumeUploading = false;
	let resumeError: { message: string; action: string; technical: string } | string = '';
	let dragActive = false;
	let resumeResult: any = null;
	let showTechnicalDetails = false;

	// GitHub import state
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

	// Resume upload handlers
	function handleResumeFileDrop(e: DragEvent) {
		e.preventDefault();
		dragActive = false;

		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			const file = files[0];
			if (file.type === 'application/pdf' ||
			    file.type === 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' ||
			    file.name.endsWith('.pdf') || file.name.endsWith('.docx')) {
				resumeFile = file;
				resumeError = '';
			} else {
				resumeError = 'Please upload a PDF or DOCX file';
			}
		}
	}

	function handleResumeFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		const files = target.files;
		if (files && files.length > 0) {
			resumeFile = files[0];
			resumeError = '';
		}
	}

	async function handleResumeUpload() {
		if (!resumeFile) {
			resumeError = 'Please select a resume file';
			return;
		}

		resumeUploading = true;
		resumeError = '';
		showTechnicalDetails = false;

		try {
			const formData = new FormData();
			formData.append('file', resumeFile);

			const response = await fetch('/api/resume/upload', {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token
				},
				body: formData
			});

			// Try to parse JSON response safely
			let data;
			try {
				const responseText = await response.text();
				if (!responseText || responseText.trim() === '') {
					throw new Error('Empty response from server');
				}
				data = JSON.parse(responseText);
			} catch (jsonErr) {
				// Response wasn't valid JSON or was empty
				resumeError = {
					message: 'The server response was incomplete or invalid.',
					action: 'This might be due to a timeout or server error. Try uploading your resume again, or try a shorter/simpler resume.',
					technical: `JSON parse error: ${jsonErr}. HTTP status: ${response.status}`
				};
				return;
			}

			if (!response.ok) {
				// Check if error is structured (with message, action, technical)
				if (data.error && typeof data.error === 'object' && data.error.message) {
					resumeError = data.error;
				} else {
					// Fallback to simple error message
					resumeError = data.error || 'Resume upload failed';
				}
				return;
			}

			resumeResult = data;
			toasts.add('success', `Resume imported! ${data.counts?.experience || 0} experiences, ${data.counts?.skills || 0} skills, and more.`);
		} catch (err) {
			// Network error or other unexpected error
			resumeError = {
				message: 'Upload failed unexpectedly.',
				action: 'Check your internet connection and try again.',
				technical: `Error: ${(err as Error).message}`
			};
		} finally {
			resumeUploading = false;
		}
	}

	function copyTechnicalDetails() {
		if (typeof resumeError === 'object' && resumeError.technical) {
			navigator.clipboard.writeText(resumeError.technical);
			toasts.add('success', 'Technical details copied to clipboard');
		}
	}

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

	function getLanguages(): string {
		if (preview?.languages && typeof preview.languages === 'object') {
			return Object.keys(preview.languages).join(', ');
		}
		return '';
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
	<title>Import | Facet</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Import Data</h1>

	<!-- Resume Upload Section -->
	<div class="card p-6 mb-8">
		<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
			Upload Resume
		</h2>

		<p class="text-gray-600 dark:text-gray-400 mb-4">
			Upload your resume (PDF or DOCX) and AI will automatically extract your experience, education, skills, projects, and more.
		</p>

		{#if resumeError}
			<div class="mb-4 p-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
				{#if typeof resumeError === 'object' && resumeError.message}
					<!-- Structured error with user-friendly message -->
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						<div class="flex-1">
							<p class="font-medium text-red-900 dark:text-red-100">
								{resumeError.message}
							</p>
							<p class="mt-1 text-sm text-red-700 dark:text-red-300">
								{resumeError.action}
							</p>
							{#if resumeError.technical}
								<button
									class="mt-2 text-xs text-red-600 dark:text-red-400 hover:underline flex items-center gap-1"
									on:click={() => (showTechnicalDetails = !showTechnicalDetails)}
								>
									{showTechnicalDetails ? 'Hide' : 'Show'} technical details
									<svg class="w-3 h-3 transition-transform {showTechnicalDetails ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
									</svg>
								</button>
								{#if showTechnicalDetails}
									<div class="mt-2 p-2 bg-red-100 dark:bg-red-900/40 rounded text-xs font-mono text-red-800 dark:text-red-200 relative">
										<button
											class="absolute top-1 right-1 p-1 hover:bg-red-200 dark:hover:bg-red-800 rounded"
											on:click={copyTechnicalDetails}
											title="Copy to clipboard"
										>
											<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
											</svg>
										</button>
										<pre class="whitespace-pre-wrap break-words pr-8">{resumeError.technical}</pre>
									</div>
								{/if}
							{/if}
						</div>
					</div>
				{:else}
					<!-- Simple error string (fallback) -->
					<div class="flex items-start gap-2">
						<svg class="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						<p class="text-sm text-red-700 dark:text-red-300">{resumeError}</p>
					</div>
				{/if}
			</div>
		{/if}

		{#if !resumeResult}
			<!-- File upload area -->
			<div
				class="border-2 border-dashed rounded-lg p-8 text-center transition-colors
				{dragActive
					? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
					: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
				on:drop={handleResumeFileDrop}
				on:dragover={(e) => {
					e.preventDefault();
					dragActive = true;
				}}
				on:dragleave={() => (dragActive = false)}
			>
				{#if resumeFile}
					<div class="space-y-3">
						<div class="w-16 h-16 mx-auto rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
							<svg class="w-8 h-8 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
							</svg>
						</div>
						<p class="font-medium text-gray-900 dark:text-white">{resumeFile.name}</p>
						<p class="text-sm text-gray-500">{(resumeFile.size / 1024).toFixed(1)} KB</p>
						<div class="flex gap-3 justify-center">
							<button
								class="btn btn-primary"
								on:click={handleResumeUpload}
								disabled={resumeUploading}
							>
								{#if resumeUploading}
									<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
									Parsing resume...
								{:else}
									Upload & Parse
								{/if}
							</button>
							<button class="btn btn-secondary" on:click={() => (resumeFile = null)}>
								Change File
							</button>
						</div>
					</div>
				{:else}
					<div class="space-y-3">
						<div class="w-16 h-16 mx-auto rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
							<svg class="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
							</svg>
						</div>
						<p class="text-gray-700 dark:text-gray-300">
							<label for="resumeFile" class="text-primary-600 dark:text-primary-400 hover:underline cursor-pointer">
								Choose a file
							</label>
							or drag and drop
						</p>
						<p class="text-sm text-gray-500">PDF or DOCX up to 5MB</p>
						<input
							type="file"
							id="resumeFile"
							class="hidden"
							accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
							on:change={handleResumeFileSelect}
						/>
					</div>
				{/if}
			</div>
		{:else}
			<!-- Import summary -->
			<div class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-6">
				<div class="flex items-start gap-3">
					<div class="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center flex-shrink-0">
						<svg class="w-5 h-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
					</div>
					<div class="flex-1">
						<h3 class="font-semibold text-green-900 dark:text-green-100 mb-2">
							Resume Imported Successfully!
						</h3>
						<p class="text-sm text-green-800 dark:text-green-200 mb-4">
							We extracted and imported the following from your resume:
						</p>
						<div class="grid grid-cols-2 gap-3 text-sm">
							{#if resumeResult.counts.experience}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.experience}</span>
									<span class="text-green-700 dark:text-green-300">Work Experience</span>
								</div>
							{/if}
							{#if resumeResult.counts.education}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.education}</span>
									<span class="text-green-700 dark:text-green-300">Education</span>
								</div>
							{/if}
							{#if resumeResult.counts.skills}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.skills}</span>
									<span class="text-green-700 dark:text-green-300">Skills</span>
								</div>
							{/if}
							{#if resumeResult.counts.projects}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.projects}</span>
									<span class="text-green-700 dark:text-green-300">Projects</span>
								</div>
							{/if}
							{#if resumeResult.counts.certifications}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.certifications}</span>
									<span class="text-green-700 dark:text-green-300">Certifications</span>
								</div>
							{/if}
							{#if resumeResult.counts.awards}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.awards}</span>
									<span class="text-green-700 dark:text-green-300">Awards</span>
								</div>
							{/if}
							{#if resumeResult.counts.talks}
								<div class="flex items-center gap-2">
									<span class="font-medium">{resumeResult.counts.talks}</span>
									<span class="text-green-700 dark:text-green-300">Talks</span>
								</div>
							{/if}
						</div>
						{#if resumeResult.warnings && resumeResult.warnings.length > 0}
							<div class="mt-3 text-sm text-yellow-700 dark:text-yellow-300">
								<p class="font-medium">Notes:</p>
								<ul class="list-disc list-inside mt-1">
									{#each resumeResult.warnings as warning}
										<li>{warning}</li>
									{/each}
								</ul>
							</div>
						{/if}
						<p class="mt-4 text-sm text-green-800 dark:text-green-200">
							All items have been imported as <strong>private</strong>. You can review and edit them in your admin sections.
						</p>
						<div class="flex gap-3 mt-4">
							<a href="/admin/experience" class="btn btn-sm btn-primary">
								Review Experience
							</a>
							<button class="btn btn-sm btn-secondary" on:click={() => {
								resumeFile = null;
								resumeResult = null;
								resumeError = '';
							}}>
								Upload Another
							</button>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- GitHub Import Section -->
	<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-4">Import from GitHub</h2>

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
						<span class="inline-flex items-center gap-1">
							{@html icon('star')}
							{preview.stargazers_count} stars
						</span>
					{/if}
					{#if preview.forks_count}
						<span class="inline-flex items-center gap-1">
							{@html icon('gitFork')}
							{preview.forks_count} forks
						</span>
					{/if}
				</div>

				{#if preview.topics && Array.isArray(preview.topics) && preview.topics.length > 0}
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
							{getLanguages()}
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

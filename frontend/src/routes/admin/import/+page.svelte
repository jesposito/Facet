<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { icon } from '$lib/icons';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';

	// ---------- Resume upload + review state ----------
	let resumeFile: File | null = $state(null);
	let resumeUploading = $state(false);
	let resumeError: { message: string; action: string; technical: string } | string = $state('');
	let dragActive = $state(false);
	let rawTextFallback = $state('');
	let parsedResume: any = $state(null);
	let parsedFilename = $state('');
	let showTechnicalDetails = $state(false);
	let importVisibility: 'private' | 'unlisted' | 'public' = $state('private');
	let resumeProviderId = $state('');
	let aiProviders: Array<{ id: string; name: string; type: string; is_default?: boolean }> = $state([]);
	let aiProvidersLoaded = $state(false);

	// Selection state: section name -> array of booleans aligned with parsed[section]
	let selected: Record<string, boolean[]> = $state({});

	// Accordion open state: section name -> boolean
	let sectionOpen: Record<string, boolean> = $state({});

	let importing = $state(false);
	let importResult: { created: number; deduplicated: number } | null = $state(null);

	const SECTIONS = [
		'experience',
		'education',
		'skills',
		'certifications',
		'projects',
		'awards',
		'talks'
	] as const;
	type Section = (typeof SECTIONS)[number];

	// ---------- GitHub import state (untouched logic, only strings i18n'd) ----------
	let step = $state(1);
	let loading = $state(false);
	let error = $state('');
	let repoUrl = $state('');
	let githubToken = $state('');
	let preview: Record<string, unknown> | null = $state(null);
	let useAI = $state(false);
	let selectedProvider = $state('');
	let privacyMode: 'full' | 'summary' | 'none' = $state('summary');
	let providers: Array<{ id: string; name: string; type: string }> = $state([]);
	let proposalId = '';

	$effect(() => {
		loadAIProviders();
	});

	async function loadAIProviders() {
		try {
			const result = await pb.collection('ai_providers').getList(1, 50, {
				filter: 'is_active = true'
			});
			aiProviders = result.items.map((p) => ({
				id: p.id,
				name: p.name as string,
				type: p.type as string,
				is_default: p.is_default as boolean
			}));
			if (aiProviders.length > 0) {
				const def = aiProviders.find((p) => p.is_default);
				resumeProviderId = def?.id || aiProviders[0].id;
			}
		} catch (err) {
			console.error('Failed to load AI providers:', err);
		} finally {
			aiProvidersLoaded = true;
		}
	}

	// ---------- Resume file handlers ----------
	function isAcceptableResumeFile(file: File): boolean {
		return (
			file.type === 'application/pdf' ||
			file.type ===
				'application/vnd.openxmlformats-officedocument.wordprocessingml.document' ||
			file.name.endsWith('.pdf') ||
			file.name.endsWith('.docx')
		);
	}

	function handleResumeFileDrop(e: DragEvent) {
		e.preventDefault();
		dragActive = false;
		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			const file = files[0];
			if (isAcceptableResumeFile(file)) {
				resumeFile = file;
				resumeError = '';
			} else {
				resumeError = $t('admin.import.error_unsupported_mime');
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
		if (!resumeFile) return;

		if (resumeFile.size > 10 * 1024 * 1024) {
			resumeError = $t('admin.import.error_oversized');
			return;
		}

		resumeUploading = true;
		resumeError = '';
		showTechnicalDetails = false;
		rawTextFallback = '';
		parsedResume = null;
		importResult = null;

		try {
			const formData = new FormData();
			formData.append('file', resumeFile);
			if (resumeProviderId) formData.append('provider_id', resumeProviderId);

			const response = await fetch('/api/resume/upload', {
				method: 'POST',
				headers: { Authorization: pb.authStore.token },
				body: formData
			});

			let data: any;
			try {
				const responseText = await response.text();
				data = responseText ? JSON.parse(responseText) : {};
			} catch (jsonErr) {
				resumeError = {
					message: $t('admin.import.error_ai_failed'),
					action: $t('admin.import.error_network'),
					technical: `JSON parse error: ${jsonErr}. HTTP status: ${response.status}`
				};
				return;
			}

			if (!response.ok) {
				if (data.error && typeof data.error === 'object' && data.error.message) {
					resumeError = data.error;
				} else {
					resumeError = data.error || $t('admin.import.error_ai_failed');
				}
				return;
			}

			if (data.status === 'partial' && data.raw_text) {
				// AI returned text but JSON parsing failed — raw-text fallback.
				rawTextFallback = data.raw_text;
				parsedFilename = data.filename || resumeFile.name;
				if (data.error) resumeError = data.error;
				return;
			}

			parsedResume = data.parsed;
			parsedFilename = data.filename || resumeFile.name;
			initializeSelection(parsedResume);
		} catch (err) {
			resumeError = {
				message: $t('admin.import.error_ai_failed'),
				action: $t('admin.import.error_network'),
				technical: `Error: ${(err as Error).message}`
			};
		} finally {
			resumeUploading = false;
		}
	}

	function initializeSelection(parsed: any) {
		const next: Record<string, boolean[]> = {};
		const open: Record<string, boolean> = {};
		for (const section of SECTIONS) {
			const items = Array.isArray(parsed?.[section]) ? parsed[section] : [];
			next[section] = items.map(() => true);
			open[section] = items.length > 0;
		}
		selected = next;
		sectionOpen = open;
	}

	function toggleSection(section: Section) {
		sectionOpen = { ...sectionOpen, [section]: !sectionOpen[section] };
	}

	function selectAll(section: Section) {
		selected = {
			...selected,
			[section]: selected[section].map(() => true)
		};
	}

	function clearAll(section: Section) {
		selected = {
			...selected,
			[section]: selected[section].map(() => false)
		};
	}

	function totalSelected(): number {
		let n = 0;
		for (const section of SECTIONS) n += (selected[section] || []).filter(Boolean).length;
		return n;
	}

	function sectionSelectedCount(section: Section): number {
		return (selected[section] || []).filter(Boolean).length;
	}

	function copyTechnicalDetails() {
		if (typeof resumeError === 'object' && resumeError.technical) {
			navigator.clipboard.writeText(resumeError.technical);
			toasts.add('success', $t('admin.import.copy_technical'));
		}
	}

	function copyRawText() {
		if (rawTextFallback) {
			navigator.clipboard.writeText(rawTextFallback);
			toasts.add('success', $t('admin.import.raw_fallback_copy'));
		}
	}

	function startOver() {
		resumeFile = null;
		parsedResume = null;
		parsedFilename = '';
		rawTextFallback = '';
		resumeError = '';
		importResult = null;
		selected = {};
		sectionOpen = {};
	}

	async function importSelected() {
		if (!parsedResume || totalSelected() === 0) return;
		importing = true;
		try {
			// Build trimmed parsed payload containing only the items the user kept.
			const trimmed: any = { profile: parsedResume.profile, metadata: parsedResume.metadata };
			for (const section of SECTIONS) {
				const items = Array.isArray(parsedResume[section]) ? parsedResume[section] : [];
				trimmed[section] = items.filter((_: any, i: number) => selected[section]?.[i]);
			}

			const response = await fetch('/api/resume/import', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					filename: parsedFilename,
					visibility: importVisibility,
					parsed: trimmed
				})
			});

			const data = await response.json();
			if (!response.ok) {
				const msg =
					(data.error && typeof data.error === 'object' && data.error.message) ||
					data.error ||
					$t('admin.import.error_ai_failed');
				toasts.add('error', typeof msg === 'string' ? msg : $t('admin.import.error_ai_failed'));
				return;
			}

			const created = Object.values(data.counts || {}).reduce(
				(a: number, b: any) => a + (typeof b === 'number' ? b : 0),
				0
			);
			importResult = { created, deduplicated: data.deduplicated || 0 };
			toasts.add(
				'success',
				$t('admin.import.import_success_body', {
					values: { created, deduplicated: data.deduplicated || 0 }
				})
			);
		} catch (err) {
			toasts.add('error', `${(err as Error).message}`);
		} finally {
			importing = false;
		}
	}

	function itemLabel(section: Section, item: any): string {
		switch (section) {
			case 'experience':
				return `${item.title || ''} @ ${item.company || ''}`.trim();
			case 'education':
				return `${item.degree || ''} ${item.field || ''} — ${item.institution || ''}`.trim();
			case 'skills':
				return item.name || '';
			case 'certifications':
				return `${item.name || ''} (${item.issuer || ''})`;
			case 'projects':
				return item.title || '';
			case 'awards':
				return `${item.title || ''} (${item.issuer || ''})`;
			case 'talks':
				return `${item.title || ''} @ ${item.event || ''}`;
			default:
				return '';
		}
	}

	function itemDetail(section: Section, item: any): string {
		switch (section) {
			case 'experience':
				return `${item.start_date || '?'} → ${item.end_date || 'present'}${item.location ? ' · ' + item.location : ''}`;
			case 'education':
				return `${item.start_date || ''} → ${item.end_date || ''}`;
			case 'skills':
				return [item.category, item.proficiency].filter(Boolean).join(' · ');
			case 'certifications':
				return item.issue_date || '';
			case 'projects':
				return Array.isArray(item.tech_stack) ? item.tech_stack.join(', ') : '';
			case 'awards':
				return item.awarded_at || '';
			case 'talks':
				return [item.date, item.location].filter(Boolean).join(' · ');
			default:
				return '';
		}
	}

	// ---------- GitHub flow (preserved) ----------
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
				headers: { 'Content-Type': 'application/json', Authorization: pb.authStore.token },
				body: JSON.stringify({ repo_url: repoUrl, token: githubToken })
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
				headers: { 'Content-Type': 'application/json', Authorization: pb.authStore.token },
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
	<title>{$t('admin.import.page_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<PageHelp pageKey="import">
		<p><strong>Import</strong> brings your existing data into Facet automatically.</p>
		<p>{$t('admin.import.resume_intro')}</p>
	</PageHelp>

	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{$t('admin.import.page_title')}</h1>

	<!-- Resume Upload Section -->
	<section class="card p-6 mb-8" aria-labelledby="resume-section-heading">
		<h2 id="resume-section-heading" class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
			{$t('admin.import.resume_section_title')}
		</h2>

		<p class="text-gray-600 dark:text-gray-400 mb-4">{$t('admin.import.resume_intro')}</p>

		{#if aiProvidersLoaded && aiProviders.length === 0}
			<!-- No AI provider configured: clear callout instead of silent failure. -->
			<div
				class="mb-4 p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800"
				role="alert"
			>
				<p class="font-medium text-yellow-900 dark:text-yellow-100">
					{$t('admin.import.no_provider_title')}
				</p>
				<p class="mt-1 text-sm text-yellow-800 dark:text-yellow-200">
					{$t('admin.import.no_provider_body')}
				</p>
				<a
					href="/admin/settings/integrations"
					class="mt-3 inline-block text-sm text-yellow-900 dark:text-yellow-100 underline"
				>
					{$t('admin.import.no_provider_link')} →
				</a>
			</div>
		{/if}

		{#if resumeError}
			<div
				class="mb-4 p-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800"
				role="alert"
				aria-live="assertive"
			>
				{#if typeof resumeError === 'object' && resumeError.message}
					<div class="flex items-start gap-3">
						<svg
							class="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<div class="flex-1">
							<p class="font-medium text-red-900 dark:text-red-100">{resumeError.message}</p>
							<p class="mt-1 text-sm text-red-700 dark:text-red-300">{resumeError.action}</p>
							{#if resumeError.technical}
								<button
									type="button"
									class="mt-2 text-xs text-red-600 dark:text-red-400 hover:underline flex items-center gap-1"
									aria-expanded={showTechnicalDetails}
									aria-controls="resume-technical-details"
									onclick={() => (showTechnicalDetails = !showTechnicalDetails)}
								>
									{showTechnicalDetails
										? $t('admin.import.hide_technical')
										: $t('admin.import.show_technical')}
								</button>
								{#if showTechnicalDetails}
									<div
										id="resume-technical-details"
										class="mt-2 p-2 bg-red-100 dark:bg-red-900/40 rounded text-xs font-mono text-red-800 dark:text-red-200 relative"
									>
										<button
											type="button"
											class="absolute top-1 right-1 p-1 hover:bg-red-200 dark:hover:bg-red-800 rounded"
											aria-label={$t('admin.import.copy_technical')}
											onclick={copyTechnicalDetails}
										>
											{@html icon('copy')}
										</button>
										<pre class="whitespace-pre-wrap break-words pr-8">{resumeError.technical}</pre>
									</div>
								{/if}
							{/if}
						</div>
					</div>
				{:else}
					<p class="text-sm text-red-700 dark:text-red-300">{resumeError}</p>
				{/if}
			</div>
		{/if}

		{#if rawTextFallback}
			<!-- Raw-text fallback (malformed JSON from AI) -->
			<div class="p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
				<h3 class="font-semibold text-blue-900 dark:text-blue-100">
					{$t('admin.import.raw_fallback_title')}
				</h3>
				<p class="mt-1 text-sm text-blue-800 dark:text-blue-200">
					{$t('admin.import.raw_fallback_body')}
				</p>
				<textarea
					readonly
					class="mt-3 w-full h-64 p-2 text-xs font-mono bg-white dark:bg-gray-900 border border-blue-200 dark:border-blue-700 rounded"
					value={rawTextFallback}
					aria-label={$t('admin.import.raw_fallback_title')}
				></textarea>
				<div class="mt-2 flex gap-2">
					<button type="button" class="btn btn-sm btn-secondary" onclick={copyRawText}>
						{$t('admin.import.raw_fallback_copy')}
					</button>
					<button type="button" class="btn btn-sm btn-secondary" onclick={startOver}>
						{$t('admin.import.start_over')}
					</button>
				</div>
			</div>
		{:else if !parsedResume && !importResult}
			<!-- File picker / dropzone -->
			<div
				class="border-2 border-dashed rounded-lg p-8 text-center transition-colors
				{dragActive
					? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
					: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
				role="region"
				aria-label={$t('admin.import.resume_dropzone_label')}
				ondrop={handleResumeFileDrop}
				ondragover={(e) => {
					e.preventDefault();
					dragActive = true;
				}}
				ondragleave={() => (dragActive = false)}
			>
				{#if resumeFile}
					<div class="space-y-3">
						<p class="font-medium text-gray-900 dark:text-white">{resumeFile.name}</p>
						<p class="text-sm text-gray-500">{(resumeFile.size / 1024).toFixed(1)} KB</p>
						<div class="flex gap-3 justify-center">
							<button
								type="button"
								class="btn btn-primary"
								onclick={handleResumeUpload}
								disabled={resumeUploading || aiProviders.length === 0}
								aria-disabled={resumeUploading || aiProviders.length === 0}
							>
								{#if resumeUploading}
									<svg
										class="animate-spin -ml-1 mr-2 h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										aria-hidden="true"
									>
										<circle
											class="opacity-25"
											cx="12"
											cy="12"
											r="10"
											stroke="currentColor"
											stroke-width="4"
										></circle>
										<path
											class="opacity-75"
											fill="currentColor"
											d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
										></path>
									</svg>
									{$t('admin.import.resume_parsing_short')}
								{:else}
									{$t('admin.import.resume_upload_button')}
								{/if}
							</button>
							<button type="button" class="btn btn-secondary" onclick={() => (resumeFile = null)}>
								{$t('admin.import.resume_change_file')}
							</button>
						</div>
					</div>
				{:else}
					<div class="space-y-3">
						<p class="text-gray-700 dark:text-gray-300">
							<label for="resumeFile" class="text-primary-600 dark:text-primary-400 hover:underline cursor-pointer">
								{$t('admin.import.resume_choose_file')}
							</label>
							{$t('admin.import.resume_or_drag')}
						</p>
						<p class="text-sm text-gray-500">{$t('admin.import.resume_dropzone_hint')}</p>
						<input
							type="file"
							id="resumeFile"
							class="sr-only"
							accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
							onchange={handleResumeFileSelect}
							aria-label={$t('admin.import.resume_choose_file')}
						/>
					</div>
				{/if}
			</div>

			{#if aiProviders.length > 1}
				<div class="mt-4">
					<label for="resumeProvider" class="label">{$t('admin.import.provider_select_label')}</label>
					<select id="resumeProvider" bind:value={resumeProviderId} class="input">
						{#each aiProviders as p}
							<option value={p.id}>{p.name} ({p.type})</option>
						{/each}
					</select>
				</div>
			{/if}

			<fieldset class="mt-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
				<legend class="label mb-2">{$t('admin.import.visibility_label')}</legend>
				<div class="flex flex-wrap gap-4">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="radio" bind:group={importVisibility} value="private" class="text-primary-600" />
						<span class="text-sm text-gray-700 dark:text-gray-300">{$t('admin.import.visibility_private')}</span>
						<span class="text-xs text-gray-500">{$t('admin.import.visibility_private_hint')}</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="radio" bind:group={importVisibility} value="unlisted" class="text-primary-600" />
						<span class="text-sm text-gray-700 dark:text-gray-300">{$t('admin.import.visibility_unlisted')}</span>
						<span class="text-xs text-gray-500">{$t('admin.import.visibility_unlisted_hint')}</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="radio" bind:group={importVisibility} value="public" class="text-primary-600" />
						<span class="text-sm text-gray-700 dark:text-gray-300">{$t('admin.import.visibility_public')}</span>
						<span class="text-xs text-gray-500">{$t('admin.import.visibility_public_hint')}</span>
					</label>
				</div>
				<p class="text-xs text-gray-500 mt-2">{$t('admin.import.visibility_note')}</p>
			</fieldset>
		{:else if resumeUploading}
			<!-- Loader between upload and parsed preview (handled by button label too) -->
			<div role="status" aria-live="polite" class="p-4 text-sm text-gray-600 dark:text-gray-400">
				{$t('admin.import.aria_loader')}
			</div>
		{:else if parsedResume && !importResult}
			<!-- Review parsed items -->
			<div>
				<h3 class="font-semibold text-gray-900 dark:text-white">
					{$t('admin.import.preview_heading')}
				</h3>
				<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
					{$t('admin.import.preview_intro')}
				</p>
				<div class="mt-2 text-xs text-gray-500">
					<span>{$t('admin.import.preview_filename')} <strong>{parsedFilename}</strong></span>
					{#if parsedResume?.metadata?.confidence}
						<span class="ml-3">{$t('admin.import.preview_confidence')} {parsedResume.metadata.confidence}</span>
					{/if}
				</div>
				{#if Array.isArray(parsedResume?.metadata?.warnings) && parsedResume.metadata.warnings.length > 0}
					<div class="mt-2 text-xs text-yellow-700 dark:text-yellow-300">
						<p class="font-medium">{$t('admin.import.preview_warnings')}</p>
						<ul class="list-disc list-inside">
							{#each parsedResume.metadata.warnings as w}
								<li>{w}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<!-- Accordion sections (APG disclosure pattern) -->
				<div class="mt-4 space-y-3">
					{#each SECTIONS as section}
						{@const items = Array.isArray(parsedResume?.[section]) ? parsedResume[section] : []}
						{#if items.length > 0}
							<div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
								<h4 class="m-0">
									<button
										type="button"
										class="w-full flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 text-left"
										aria-expanded={sectionOpen[section]}
										aria-controls={`resume-section-${section}`}
										onclick={() => toggleSection(section)}
									>
										<span class="font-medium text-gray-900 dark:text-white">
											{$t(`admin.import.section_${section}`)}
											<span class="ml-2 text-xs text-gray-500">
												({sectionSelectedCount(section)}/{items.length})
											</span>
										</span>
										<svg
											class="w-4 h-4 transition-transform {sectionOpen[section] ? 'rotate-180' : ''}"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
										>
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
										</svg>
									</button>
								</h4>
								{#if sectionOpen[section]}
									<div
										id={`resume-section-${section}`}
										role="region"
										aria-labelledby={`resume-section-${section}-label`}
										class="p-3 bg-white dark:bg-gray-900"
									>
										<span id={`resume-section-${section}-label`} class="sr-only">
											{$t(`admin.import.section_${section}`)}
										</span>
										<div class="flex gap-2 mb-2">
											<button
												type="button"
												class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
												onclick={() => selectAll(section)}
											>
												{$t('admin.import.preview_select_all')}
											</button>
											<button
												type="button"
												class="text-xs text-gray-600 dark:text-gray-400 hover:underline"
												onclick={() => clearAll(section)}
											>
												{$t('admin.import.preview_clear_all')}
											</button>
										</div>
										<ul class="space-y-2">
											{#each items as item, i}
												{@const label = itemLabel(section, item)}
												<li class="flex items-start gap-2">
													<input
														type="checkbox"
														id={`item-${section}-${i}`}
														bind:checked={selected[section][i]}
														class="mt-1"
														aria-label={$t('admin.import.aria_item_checkbox', {
															values: { label }
														})}
													/>
													<label for={`item-${section}-${i}`} class="text-sm flex-1 cursor-pointer">
														<span class="block font-medium text-gray-900 dark:text-white">{label}</span>
														{#if itemDetail(section, item)}
															<span class="block text-xs text-gray-500">{itemDetail(section, item)}</span>
														{/if}
													</label>
												</li>
											{/each}
										</ul>
									</div>
								{/if}
							</div>
						{/if}
					{/each}
				</div>

				<div class="mt-4 flex items-center gap-3">
					<button
						type="button"
						class="btn btn-primary"
						onclick={importSelected}
						disabled={importing || totalSelected() === 0}
						aria-disabled={importing || totalSelected() === 0}
					>
						{#if importing}
							{$t('admin.import.importing')}
						{:else}
							{$t('admin.import.import_selected_count', {
								values: { count: totalSelected() }
							})}
						{/if}
					</button>
					<button type="button" class="btn btn-secondary" onclick={startOver}>
						{$t('admin.import.start_over')}
					</button>
					{#if totalSelected() === 0}
						<span class="text-xs text-gray-500">{$t('admin.import.import_nothing_selected')}</span>
					{/if}
				</div>
			</div>
		{:else if importResult}
			<!-- Success -->
			<div
				class="p-4 rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800"
				role="status"
			>
				<h3 class="font-semibold text-green-900 dark:text-green-100">
					{$t('admin.import.import_success_title')}
				</h3>
				<p class="mt-1 text-sm text-green-800 dark:text-green-200">
					{$t('admin.import.import_success_body', {
						values: { created: importResult.created, deduplicated: importResult.deduplicated }
					})}
				</p>
				<div class="mt-3 flex gap-2">
					<a href="/admin/experience" class="btn btn-sm btn-primary">{$t('admin.import.review_experience')}</a>
					<button type="button" class="btn btn-sm btn-secondary" onclick={startOver}>
						{$t('admin.import.start_over')}
					</button>
				</div>
			</div>
		{/if}
	</section>

	<!-- GitHub Import Section (preserved) -->
	<div class="card p-6">
		<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Import from GitHub</h2>

		<p class="text-gray-600 dark:text-gray-400 mb-6">
			Import a GitHub repository as a project. Optionally use AI to generate a polished description.
		</p>

		<!-- Progress steps -->
		<div class="flex items-center mb-6">
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
						<div class="w-12 h-1 mx-2 {step > i + 1 ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'}"></div>
					{/if}
				</div>
			{/each}
		</div>

		{#if error}
			<div class="mb-4 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300">
				{error}
			</div>
		{/if}

		{#if step === 1}
			<div class="pt-4 border-t border-gray-200 dark:border-gray-700">
				<h3 class="text-base font-medium text-gray-900 dark:text-white mb-4">Enter Repository</h3>
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
						<input type="password" id="token" bind:value={githubToken} class="input" placeholder="ghp_..." />
						<p class="text-xs text-gray-500 mt-1">
							Token is used only for this import and not stored unless you choose to save it.
						</p>
					</div>
					<button class="btn btn-primary" onclick={handlePreview} disabled={loading}>
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

		{#if step === 2 && preview}
			<div class="pt-4 border-t border-gray-200 dark:border-gray-700 space-y-4">
				<h3 class="text-base font-medium text-gray-900 dark:text-white">Repository Preview</h3>
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
								<span class="px-2 py-0.5 text-xs bg-gray-200 dark:bg-gray-700 rounded">{topic}</span>
							{/each}
						</div>
					{/if}
					{#if preview.languages}
						<div class="mt-3">
							<span class="text-sm font-medium">Languages: </span>
							<span class="text-sm text-gray-600 dark:text-gray-400">{getLanguages()}</span>
						</div>
					{/if}
				</div>
				<div class="flex gap-3">
					<button class="btn btn-secondary" onclick={() => (step = 1)}>Back</button>
					<button class="btn btn-primary" onclick={() => (step = 3)}>Continue</button>
				</div>
			</div>
		{/if}

		{#if step === 3}
			<div class="pt-4 border-t border-gray-200 dark:border-gray-700 space-y-4">
				<h3 class="text-base font-medium text-gray-900 dark:text-white">AI Enrichment</h3>
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
								You can <a href="/admin/settings#integrations" class="text-primary-600 dark:text-primary-400 underline">configure an AI provider</a> to automatically generate descriptions.
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
					<button class="btn btn-secondary" onclick={() => (step = 2)}>Back</button>
					<button class="btn btn-primary" onclick={handleImport} disabled={loading || (useAI && providers.length === 0)}>
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
</div>

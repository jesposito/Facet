<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { focusTrap } from '$lib/actions/focusTrap';

	interface Props {
		onSuccess: (imported: Record<string, string[]>) => void;
		onClose: () => void;
	}

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
	type ResumeItem = Record<string, unknown>;
	type ParsedResume = {
		profile?: unknown;
		metadata?: { confidence?: string; warnings?: string[] };
	} & Partial<Record<Section, ResumeItem[]>>;
	type ApiError = string | { message?: string } | undefined;

	let { onSuccess, onClose }: Props = $props();

	let file: File | null = $state(null);
	let uploading = $state(false);
	let error = $state('');
	let dragActive = $state(false);
	let visibility: 'private' | 'unlisted' | 'public' = $state('private');
	let isMobile = $state(false);
	let headingEl = $state<HTMLHeadingElement | null>(null);
	let parsedResume: ParsedResume | null = $state(null);
	let parsedFilename = $state('');
	let selected: Record<Section, boolean[]> = $state({
		experience: [],
		education: [],
		skills: [],
		certifications: [],
		projects: [],
		awards: [],
		talks: []
	});
	let sectionOpen: Record<Section, boolean> = $state({
		experience: false,
		education: false,
		skills: false,
		certifications: false,
		projects: false,
		awards: false,
		talks: false
	});

	onMount(() => {
		headingEl?.focus();
		const mq = window.matchMedia('(max-width: 767px)');
		isMobile = mq.matches;
		const handler = (e: MediaQueryListEvent) => (isMobile = e.matches);
		mq.addEventListener('change', handler);
		return () => mq.removeEventListener('change', handler);
	});

	function handleFileDrop(e: DragEvent) {
		e.preventDefault();
		dragActive = false;
		const droppedFile = e.dataTransfer?.files[0];
		if (droppedFile) validateAndSetFile(droppedFile);
	}

	function handleFileSelect(e: Event) {
		const selectedFile = (e.target as HTMLInputElement).files?.[0];
		if (selectedFile) validateAndSetFile(selectedFile);
	}

	function validateAndSetFile(selectedFile: File) {
		if (
			selectedFile.type === 'application/pdf' ||
			selectedFile.type ===
				'application/vnd.openxmlformats-officedocument.wordprocessingml.document' ||
			selectedFile.name.endsWith('.pdf') ||
			selectedFile.name.endsWith('.docx')
		) {
			if (selectedFile.size > 5 * 1024 * 1024) {
				error = $t('admin.import.modal_error_oversized');
				file = null;
			} else {
				file = selectedFile;
				error = '';
			}
		} else {
			error = $t('admin.import.modal_error_unsupported');
		}
	}

	function apiError(errorValue: ApiError, fallback: string): string {
		if (typeof errorValue === 'string') return errorValue;
		return errorValue?.message || fallback;
	}

	function itemsFor(section: Section): ResumeItem[] {
		return parsedResume?.[section] || [];
	}

	function totalSelected(): number {
		return SECTIONS.reduce((total, section) => total + selected[section].filter(Boolean).length, 0);
	}

	function text(value: unknown): string {
		return typeof value === 'string' ? value : '';
	}

	function itemLabel(section: Section, item: ResumeItem): string {
		switch (section) {
			case 'experience':
				return `${text(item.title)} @ ${text(item.company)}`.trim();
			case 'education':
				return `${text(item.degree)} ${text(item.field)} — ${text(item.institution)}`.trim();
			case 'skills':
			case 'certifications':
				return text(item.name);
			case 'projects':
				return text(item.title);
			case 'awards':
				return `${text(item.title)} (${text(item.issuer)})`.trim();
			case 'talks':
				return `${text(item.title)} @ ${text(item.event)}`.trim();
		}
	}

	function itemDetail(section: Section, item: ResumeItem): string {
		switch (section) {
			case 'experience':
				return `${text(item.start_date) || '?'} → ${text(item.end_date) || 'present'}${text(item.location) ? ` · ${text(item.location)}` : ''}`;
			case 'education':
				return `${text(item.start_date)} → ${text(item.end_date)}`;
			case 'skills':
				return [text(item.category), text(item.proficiency)].filter(Boolean).join(' · ');
			case 'certifications':
				return text(item.issue_date);
			case 'projects':
				return Array.isArray(item.tech_stack) ? item.tech_stack.join(', ') : '';
			case 'awards':
				return text(item.awarded_at);
			case 'talks':
				return [text(item.date), text(item.location)].filter(Boolean).join(' · ');
		}
	}

	async function parseResume() {
		if (!file) {
			error = $t('admin.import.modal_error_no_file');
			return;
		}
		uploading = true;
		error = '';
		try {
			const formData = new FormData();
			formData.append('file', file);
			const response = await fetch('/api/resume/upload', {
				method: 'POST',
				headers: { Authorization: pb.authStore.token },
				body: formData
			});
			const data = (await response.json()) as {
				error?: ApiError;
				status?: string;
				filename?: string;
				parsed?: ParsedResume;
			};
			if (!response.ok) {
				throw new Error(apiError(data.error, $t('admin.import.modal_error_upload_failed')));
			}
			if (data.status !== 'success' || !data.parsed || typeof data.parsed !== 'object') {
				throw new Error(apiError(data.error, $t('admin.import.error_ai_failed')));
			}
			parsedResume = data.parsed;
			parsedFilename = data.filename || file.name;
			for (const section of SECTIONS) {
				selected[section] = itemsFor(section).map(() => true);
				sectionOpen[section] = itemsFor(section).length > 0;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : $t('admin.import.modal_error_unexpected');
		} finally {
			uploading = false;
		}
	}

	function startOver() {
		parsedResume = null;
		parsedFilename = '';
		error = '';
		for (const section of SECTIONS) {
			selected[section] = [];
			sectionOpen[section] = false;
		}
	}

	async function importSelected() {
		if (!parsedResume || totalSelected() === 0) return;
		uploading = true;
		error = '';
		try {
			const trimmed: ParsedResume = {
				profile: parsedResume.profile,
				metadata: parsedResume.metadata
			};
			for (const section of SECTIONS) {
				trimmed[section] = itemsFor(section).filter((_, index) => selected[section][index]);
			}
			const response = await fetch('/api/resume/import', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({ filename: parsedFilename, visibility, parsed: trimmed })
			});
			const data = (await response.json()) as {
				error?: ApiError;
				imported?: Record<string, string[]>;
			};
			if (!response.ok) {
				throw new Error(apiError(data.error, $t('admin.import.error_ai_failed')));
			}
			if (!data.imported || typeof data.imported !== 'object' || Array.isArray(data.imported)) {
				throw new Error($t('admin.import.modal_error_unexpected'));
			}
			onSuccess(data.imported);
			toasts.add('success', $t('admin.import.modal_success_toast'));
		} catch (err) {
			error = err instanceof Error ? err.message : $t('admin.import.modal_error_unexpected');
		} finally {
			uploading = false;
		}
	}
</script>

<div
	class="fixed inset-0 bg-black/50 flex {isMobile
		? 'flex-col justify-end'
		: 'items-center justify-center'} p-4 z-50"
	use:focusTrap={{ onEscape: onClose, autoFocus: false }}
>
	<button
		class="absolute inset-0 w-full h-full cursor-default border-0 p-0 m-0 bg-transparent"
		tabindex="-1"
		aria-label={$t('shared.aria.close_modal')}
		onclick={onClose}
		type="button"
	></button>

	<div
		role="dialog"
		aria-modal="true"
		aria-labelledby="resume-import-title"
		tabindex="-1"
		class="card w-full p-6 transform transition-transform relative z-10 {isMobile
			? 'rounded-t-2xl rounded-b-none max-h-[90vh] overflow-y-auto'
			: 'max-w-2xl'}"
	>
		<div class="flex justify-between items-start mb-6">
			<div>
				<h2
					id="resume-import-title"
					tabindex="-1"
					bind:this={headingEl}
					class="text-2xl font-bold text-gray-900 dark:text-white mb-2"
				>
					{$t('admin.import.modal_title')}
				</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					{$t('admin.import.modal_subtitle')}
				</p>
			</div>
			<button
				class="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300 transition-colors p-1"
				onclick={onClose}
				aria-label={$t('shared.aria.close_modal')}
			>
				<svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
		</div>

		{#if error}
			<div
				class="mb-4 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm flex items-start gap-2"
			>
				<svg class="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
				<span>{error}</span>
			</div>
		{/if}

		<div class="space-y-6">
			{#if !parsedResume}
				<div
					class="border-2 border-dashed rounded-lg p-8 text-center transition-colors
					{dragActive
						? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
						: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
					role="region"
					aria-label={$t('admin.import.resume_dropzone_label')}
					ondrop={handleFileDrop}
					ondragover={(e) => {
						e.preventDefault();
						dragActive = true;
					}}
					ondragleave={() => (dragActive = false)}
				>
					{#if file}
						<div class="space-y-3">
							<p class="font-medium text-gray-900 dark:text-white">{file.name}</p>
							<p class="text-sm text-gray-500">{(file.size / 1024).toFixed(1)} KB</p>
							<button class="btn btn-sm btn-secondary mt-2" onclick={() => (file = null)}>
								{$t('admin.import.resume_change_file')}
							</button>
						</div>
					{:else}
						<div class="space-y-3">
							<p class="text-gray-700 dark:text-gray-300">
								<label
									for="resumeFileModal"
									class="text-primary-600 dark:text-primary-400 hover:underline cursor-pointer"
								>
									{$t('admin.import.resume_choose_file')}
								</label>
								{$t('admin.import.resume_or_drag')}
							</p>
							<p class="text-sm text-gray-500">{$t('admin.import.modal_dropzone_hint')}</p>
							<input
								type="file"
								id="resumeFileModal"
								class="hidden"
								accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
								onchange={handleFileSelect}
							/>
						</div>
					{/if}
				</div>
			{:else}
				<div>
					<h3 class="font-semibold text-gray-900 dark:text-white">
						{$t('admin.import.preview_heading')}
					</h3>
					<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
						{$t('admin.import.preview_intro')}
					</p>
					<p class="mt-2 text-xs text-gray-500">
						{$t('admin.import.preview_filename')} <strong>{parsedFilename}</strong>
					</p>
					<div class="mt-4 space-y-3">
						{#each SECTIONS as section}
							{@const items = itemsFor(section)}
							{#if items.length > 0}
								<div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
									<button
										type="button"
										class="w-full flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-800 text-left"
										aria-expanded={sectionOpen[section]}
										aria-controls={`resume-modal-section-${section}`}
										onclick={() => (sectionOpen[section] = !sectionOpen[section])}
									>
										<span class="font-medium text-gray-900 dark:text-white">
											{$t(`admin.import.section_${section}`)}
											<span class="ml-2 text-xs text-gray-500">
												({selected[section].filter(Boolean).length}/{items.length})
											</span>
										</span>
										<svg
											class="w-4 h-4 transition-transform {sectionOpen[section]
												? 'rotate-180'
												: ''}"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 9l-7 7-7-7"
											/>
										</svg>
									</button>
									{#if sectionOpen[section]}
										<div id={`resume-modal-section-${section}`} class="p-3 space-y-3">
											<div class="flex gap-3">
												<button
													type="button"
													class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
													onclick={() => (selected[section] = items.map(() => true))}
												>
													{$t('admin.import.preview_select_all')}
												</button>
												<button
													type="button"
													class="text-xs text-gray-600 dark:text-gray-400 hover:underline"
													onclick={() => (selected[section] = items.map(() => false))}
												>
													{$t('admin.import.preview_clear_all')}
												</button>
											</div>
											<ul class="space-y-2">
												{#each items as item, index}
													{@const label = itemLabel(section, item)}
													<li class="flex items-start gap-2">
														<input
															type="checkbox"
															id={`resume-modal-item-${section}-${index}`}
															bind:checked={selected[section][index]}
															class="mt-1"
															aria-label={$t('admin.import.aria_item_checkbox', {
																values: { label }
															})}
														/>
														<label
															for={`resume-modal-item-${section}-${index}`}
															class="text-sm flex-1 cursor-pointer"
														>
															<span class="block font-medium text-gray-900 dark:text-white"
																>{label}</span
															>
															{#if itemDetail(section, item)}
																<span class="block text-xs text-gray-500"
																	>{itemDetail(section, item)}</span
																>
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
				</div>
			{/if}

			<fieldset class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg" disabled={uploading}>
				<legend class="label mb-2">{$t('admin.import.visibility_label')}</legend>
				<div class="flex flex-wrap gap-4">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="radio" bind:group={visibility} value="private" class="text-primary-600" />
						<span class="text-sm text-gray-700 dark:text-gray-300"
							>{$t('admin.import.visibility_private')}</span
						>
						<span class="text-xs text-gray-500">{$t('admin.import.visibility_private_hint')}</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="radio" bind:group={visibility} value="unlisted" class="text-primary-600" />
						<span class="text-sm text-gray-700 dark:text-gray-300"
							>{$t('admin.import.visibility_unlisted')}</span
						>
						<span class="text-xs text-gray-500">{$t('admin.import.visibility_unlisted_hint')}</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="radio" bind:group={visibility} value="public" class="text-primary-600" />
						<span class="text-sm text-gray-700 dark:text-gray-300"
							>{$t('admin.import.visibility_public')}</span
						>
						<span class="text-xs text-gray-500">{$t('admin.import.visibility_public_hint')}</span>
					</label>
				</div>
				<p class="text-xs text-gray-500 mt-2">{$t('admin.import.visibility_note')}</p>
			</fieldset>

			<div class="flex flex-wrap gap-3 justify-end pt-2">
				<button class="btn btn-secondary" onclick={onClose} disabled={uploading}>
					{$t('shared.actions.cancel')}
				</button>
				{#if parsedResume}
					<button class="btn btn-secondary" onclick={startOver} disabled={uploading}>
						{$t('admin.import.start_over')}
					</button>
					<button
						class="btn btn-primary"
						onclick={importSelected}
						disabled={uploading || totalSelected() === 0}
					>
						{uploading
							? $t('admin.import.importing')
							: $t('admin.import.import_selected_count', { values: { count: totalSelected() } })}
					</button>
				{:else}
					<button class="btn btn-primary" onclick={parseResume} disabled={uploading || !file}>
						{uploading
							? $t('admin.import.resume_parsing_short')
							: $t('admin.import.resume_upload_button')}
					</button>
				{/if}
			</div>
		</div>
	</div>
</div>

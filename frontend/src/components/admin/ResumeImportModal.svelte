<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { onMount } from 'svelte';

	interface Props {
		onSuccess: (imported: Record<string, string[]>) => void;
		onClose: () => void;
	}

	let { onSuccess, onClose }: Props = $props();

	let file: File | null = $state(null);
	let uploading = $state(false);
	let error = $state('');
	let dragActive = $state(false);
	let visibility: 'private' | 'unlisted' | 'public' = $state('private');
	let isMobile = $state(false);

	function handleFileDrop(e: DragEvent) {
		e.preventDefault();
		dragActive = false;

		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			const droppedFile = files[0];
			validateAndSetFile(droppedFile);
		}
	}

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		const files = target.files;
		if (files && files.length > 0) {
			validateAndSetFile(files[0]);
		}
	}

	function validateAndSetFile(selectedFile: File) {
		if (
			selectedFile.type === 'application/pdf' ||
			selectedFile.type === 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' ||
			selectedFile.name.endsWith('.pdf') ||
			selectedFile.name.endsWith('.docx')
		) {
			if (selectedFile.size > 5 * 1024 * 1024) {
				error = 'File size exceeds 5MB limit';
				file = null;
			} else {
				file = selectedFile;
				error = '';
			}
		} else {
			error = 'Please upload a PDF or DOCX file';
		}
	}

	async function handleSubmit() {
		if (!file) {
			error = 'Please select a resume file';
			return;
		}

		uploading = true;
		error = '';

		try {
			const formData = new FormData();
			formData.append('file', file);
			formData.append('visibility', visibility);

			const response = await fetch('/api/resume/upload', {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token
				},
				body: formData
			});

			const data = await response.json();

			if (!response.ok) {
				if (data.error && typeof data.error === 'object' && data.error.message) {
					throw new Error(data.error.message);
				} else {
					throw new Error(data.error || 'Resume upload failed');
				}
			}

			toasts.add('success', 'Resume imported! Select items to include in your facet.');
			onSuccess(data.imported);
		} catch (err: any) {
			error = err.message || 'Upload failed unexpectedly';
		} finally {
			uploading = false;
		}
	}

	onMount(() => {
		const mq = window.matchMedia('(max-width: 767px)');
		isMobile = mq.matches;
		const handler = (e: MediaQueryListEvent) => (isMobile = e.matches);
		mq.addEventListener('change', handler);

		return () => mq.removeEventListener('change', handler);
	});
</script>

<div
	class="fixed inset-0 bg-black/50 flex {isMobile
		? 'flex-col justify-end'
		: 'items-center justify-center'} p-4 z-50"
>
	<button
		class="absolute inset-0 w-full h-full cursor-default border-0 p-0 m-0 bg-transparent"
		aria-label="Close modal"
		onclick={onClose}
		type="button"
	></button>

	<div
		class="card w-full p-6 transform transition-transform relative z-10 {isMobile
			? 'rounded-t-2xl rounded-b-none max-h-[90vh] overflow-y-auto'
			: 'max-w-2xl'}"
	>
		<div class="flex justify-between items-start mb-6">
			<div>
				<h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Import Resume</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Upload your resume to automatically extract experience, education, and skills.
				</p>
			</div>
			<button
				class="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300 transition-colors p-1"
				onclick={onClose}
				aria-label="Close modal"
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
				<svg
					class="w-5 h-5 flex-shrink-0"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
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
			<div
				class="border-2 border-dashed rounded-lg p-8 text-center transition-colors
                {dragActive
					? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
					: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
				role="region"
				aria-label="Resume file drop zone"
				ondrop={handleFileDrop}
				ondragover={(e) => {
					e.preventDefault();
					dragActive = true;
				}}
				ondragleave={() => (dragActive = false)}
			>
				{#if file}
					<div class="space-y-3">
						<div
							class="w-16 h-16 mx-auto rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center"
						>
							<svg
								class="w-8 h-8 text-primary-600 dark:text-primary-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
								/>
							</svg>
						</div>
						<p class="font-medium text-gray-900 dark:text-white">{file.name}</p>
						<p class="text-sm text-gray-500">{(file.size / 1024).toFixed(1)} KB</p>
						<button class="btn btn-sm btn-secondary mt-2" onclick={() => (file = null)}>
							Change File
						</button>
					</div>
				{:else}
					<div class="space-y-3">
						<div
							class="w-16 h-16 mx-auto rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center"
						>
							<svg class="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
								/>
							</svg>
						</div>
						<p class="text-gray-700 dark:text-gray-300">
							<label
								for="resumeFileModal"
								class="text-primary-600 dark:text-primary-400 hover:underline cursor-pointer"
							>
								Choose a file
							</label>
							or drag and drop
						</p>
						<p class="text-sm text-gray-500">PDF or DOCX up to 5MB</p>
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

			<fieldset class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
				<legend class="label mb-2">Import visibility</legend>
				<div class="flex flex-wrap gap-4">
					<label class="flex items-center gap-2 cursor-pointer">
						<input
							type="radio"
							bind:group={visibility}
							value="private"
							class="text-primary-600"
						/>
						<span class="text-sm text-gray-700 dark:text-gray-300">Private</span>
						<span class="text-xs text-gray-500">(recommended)</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input
							type="radio"
							bind:group={visibility}
							value="unlisted"
							class="text-primary-600"
						/>
						<span class="text-sm text-gray-700 dark:text-gray-300">Unlisted</span>
						<span class="text-xs text-gray-500">(hidden but shareable)</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input
							type="radio"
							bind:group={visibility}
							value="public"
							class="text-primary-600"
						/>
						<span class="text-sm text-gray-700 dark:text-gray-300">Public</span>
						<span class="text-xs text-gray-500">(visible immediately)</span>
					</label>
				</div>
				<p class="text-xs text-gray-500 mt-2">
					Private items can be made visible per-view when you add them to a view.
				</p>
			</fieldset>

			<div class="flex gap-3 justify-end pt-2">
				<button class="btn btn-secondary" onclick={onClose} disabled={uploading}>
					Cancel
				</button>
				<button
					class="btn btn-primary"
					onclick={handleSubmit}
					disabled={uploading || !file}
				>
					{#if uploading}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
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
						Uploading...
					{:else}
						Upload & Import
					{/if}
				</button>
			</div>
		</div>
	</div>
</div>

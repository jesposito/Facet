<script lang="ts">
	/**
	 * ViewResumeGenerator - Modal for generating AI-powered resumes from view content
	 * 
	 * Allows users to configure and generate PDF/DOCX resumes tailored to specific
	 * roles, with different styles and lengths.
	 */
	import { self } from 'svelte/legacy';
	import type { ResumeGenerationConfig, ExportRecord } from '$lib/view-editor/types';

	// Props
	let {
		slug,
		authToken,
		exports = [],
		onGenerate,
		onDeleteExport,
		onClose
	}: {
		slug: string;
		authToken: string;
		exports?: ExportRecord[];
		onGenerate: (config: ResumeGenerationConfig) => Promise<void>;
		onDeleteExport: (exportId: string) => Promise<void>;
		onClose: () => void;
	} = $props();

	// Local state
	let generating = $state(false);
	let config = $state<ResumeGenerationConfig>({
		format: 'pdf',
		target_role: '',
		style: 'chronological',
		length: 'two-page',
		emphasis: []
	});

	/**
	 * Handle generate button click
	 */
	async function handleGenerate() {
		generating = true;
		try {
			await onGenerate(config);
			// Reset config for next time
			config = {
				format: 'pdf',
				target_role: '',
				style: 'chronological',
				length: 'two-page',
				emphasis: []
			};
		} finally {
			generating = false;
		}
	}

	/**
	 * Handle delete export
	 */
	async function handleDeleteExport(exportId: string) {
		await onDeleteExport(exportId);
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={self(() => onClose())}>
	<div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[90vh] overflow-hidden">
		<!-- Header -->
		<div class="p-4 border-b border-gray-200 dark:border-gray-700">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Generate Resume</h2>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
				AI will create a professional resume from this view's content.
			</p>
		</div>

		<!-- Content -->
		<div class="p-4 space-y-4 overflow-y-auto">
			<!-- Format -->
			<div>
				<label for="format" class="label">Format</label>
				<select id="format" bind:value={config.format} class="input">
					<option value="pdf">PDF</option>
					<option value="docx">Word Document (DOCX)</option>
				</select>
			</div>

			<!-- Target Role -->
			<div>
				<label for="target_role" class="label">Target Role (optional)</label>
				<input
					type="text"
					id="target_role"
					bind:value={config.target_role}
					class="input"
					placeholder="e.g., Senior Software Engineer at FAANG"
				/>
				<p class="text-xs text-gray-500 mt-1">AI will tailor content for this role</p>
			</div>

			<!-- Style -->
			<div>
				<label for="style" class="label">Resume Style</label>
				<select id="style" bind:value={config.style} class="input">
					<option value="chronological">Chronological (most common)</option>
					<option value="functional">Functional (skills-focused)</option>
					<option value="hybrid">Hybrid (combination)</option>
				</select>
			</div>

			<!-- Length -->
			<div>
				<label for="length" class="label">Length</label>
				<select id="length" bind:value={config.length} class="input">
					<option value="one-page">One Page</option>
					<option value="two-page">Two Pages</option>
					<option value="full">Full (no limit)</option>
				</select>
			</div>

			<!-- Previous Exports -->
			{#if exports.length > 0}
				<div class="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
					<h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Previous Exports</h3>
					<div class="space-y-2 max-h-32 overflow-y-auto">
						{#each exports as exp}
							<div class="flex items-center justify-between text-sm bg-gray-50 dark:bg-gray-700/50 rounded p-2">
								<div class="flex items-center gap-2">
									<span class="uppercase text-xs font-medium px-1.5 py-0.5 rounded {exp.format === 'pdf' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'}">
										{exp.format}
									</span>
									<span class="text-gray-500 dark:text-gray-400">
										{new Date(exp.generated_at).toLocaleDateString()}
									</span>
								</div>
								<div class="flex items-center gap-2">
									{#if exp.download_url}
										<a
											href={exp.download_url}
											class="text-primary-600 hover:text-primary-700 dark:text-primary-400"
											download
										>
											Download
										</a>
									{/if}
									<button
										type="button"
										class="text-red-500 hover:text-red-700"
										onclick={() => handleDeleteExport(exp.id)}
									>
										Delete
									</button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
			<button type="button" class="btn btn-ghost" onclick={onClose}>
				Cancel
			</button>
			<button
				type="button"
				class="btn btn-primary"
				onclick={handleGenerate}
				disabled={generating}
			>
				{#if generating}
					<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Generating...
				{:else}
					Generate
				{/if}
			</button>
		</div>
	</div>
</div>

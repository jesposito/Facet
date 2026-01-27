<script lang="ts">
	import { run, createBubbler, stopPropagation } from 'svelte/legacy';

	const bubble = createBubbler();
	/**
	 * AIContentHelper - Enhanced AI writing assistant
	 *
	 * Features:
	 * - Multiple rewrite tones (executive, professional, technical, conversational, creative)
	 * - Critique mode with inline feedback
	 * - Context-aware prompts
	 * - Replace or preview results
	 */

	import { createEventDispatcher } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';
	import { t } from 'svelte-i18n';

	interface Props {
		content?: string;
		fieldType?: 'headline' | 'summary' | 'description' | 'bullets' | 'content';
		context?: Record<string, string>;
		disabled?: boolean;
		size?: 'sm' | 'md';
	}

	let {
		content = '',
		fieldType = 'description',
		context = {},
		disabled = false,
		size = 'md'
	}: Props = $props();

	const dispatch = createEventDispatcher<{
		apply: { content: string };
	}>();

	type Mode = 'rewrite' | 'critique';
	type Tone = 'executive' | 'professional' | 'technical' | 'conversational' | 'creative';

	let loading = $state(false);
	let aiAvailable: boolean | null = $state(null);
	let showMenu = $state(false);
	let mode: Mode = $state('rewrite');
	let selectedTone: Tone = 'professional';
	let previewContent = $state('');
	let showPreview = $state(false);

	// Tone definitions with icons
	const toneIcons: Record<Tone, string> = {
		executive: '👔',
		professional: '💼',
		technical: '⚙️',
		conversational: '💬',
		creative: '🎨'
	};
	const toneValues: Tone[] = ['executive', 'professional', 'technical', 'conversational', 'creative'];

	// Derive translated tones
	let tones = $derived(
		toneValues.map((value) => ({
			value,
			label: $t(`admin.ai_assistant.tone_${value}`),
			description: $t(`admin.ai_assistant.tone_${value}_desc`),
			icon: toneIcons[value]
		}))
	);

	// Check AI availability
	async function checkAIStatus() {
		try {
			const response = await fetch('/api/ai/status');
			if (response.ok) {
				const data = await response.json();
				aiAvailable = data.available;
			} else {
				aiAvailable = false;
			}
		} catch {
			aiAvailable = false;
		}
	}

	run(() => {
		if (aiAvailable === null) {
			checkAIStatus();
		}
	});

	async function handleRewrite(tone: Tone) {
		if (!content.trim()) {
			toasts.add('error', $t('admin.ai_assistant.enter_content_first'));
			return;
		}

		loading = true;
		showMenu = false;

		try {
			const response = await fetch('/api/ai/rewrite', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					content,
					field_type: fieldType,
					context,
					tone
				})
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'AI request failed');
			}

			const result = await response.json();
			previewContent = result.content;
			showPreview = true;
			toasts.add('success', $t('admin.ai_assistant.rewrite_success', { values: { tone: $t(`admin.ai_assistant.tone_${tone}`) } }));
		} catch (err) {
			console.error('AI rewrite failed:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.ai_assistant.rewrite_failed'));
		} finally {
			loading = false;
		}
	}

	async function handleCritique() {
		if (!content.trim()) {
			toasts.add('error', $t('admin.ai_assistant.enter_content_first'));
			return;
		}

		loading = true;
		showMenu = false;

		try {
			const response = await fetch('/api/ai/critique', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					content,
					field_type: fieldType,
					context
				})
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'AI request failed');
			}

			const result = await response.json();
			previewContent = result.content;
			showPreview = true;
			toasts.add('success', $t('admin.ai_assistant.feedback_provided'));
		} catch (err) {
			console.error('AI critique failed:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.ai_assistant.feedback_failed'));
		} finally {
			loading = false;
		}
	}

	function applyPreview() {
		dispatch('apply', { content: previewContent });
		showPreview = false;
		previewContent = '';
		toasts.add('success', $t('admin.ai_assistant.changes_applied'));
	}

	function cancelPreview() {
		showPreview = false;
		previewContent = '';
	}

	function toggleMenu() {
		showMenu = !showMenu;
	}
</script>

{#if aiAvailable}
	<div class="ai-content-helper relative inline-block">
		<!-- Main Button -->
		<button
			type="button"
			class="ai-button inline-flex items-center gap-1.5 px-2 sm:px-3 py-1.5 rounded-md border border-purple-200 dark:border-purple-800 bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-950 dark:to-blue-950 text-purple-700 dark:text-purple-300 hover:from-purple-100 hover:to-blue-100 dark:hover:from-purple-900 dark:hover:to-blue-900 transition-all whitespace-nowrap
				{size === 'sm' ? 'text-xs' : 'text-xs sm:text-sm'}
				{disabled || loading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer shadow-sm hover:shadow-md'}"
			onclick={toggleMenu}
			disabled={disabled || loading}
			title={$t('admin.ai_assistant.title')}
		>
			{#if loading}
				<svg class="animate-spin h-3.5 w-3.5" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
			{:else}
				{@html icon('sparkles')}
			{/if}
			<span class="font-medium hidden sm:inline">{$t('admin.ai_assistant.button_label')}</span>
			<span class="font-medium sm:hidden">{$t('admin.ai_assistant.button_label_short')}</span>
			<svg
				class="h-4 w-4 transition-transform {showMenu ? 'rotate-180' : ''}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
			</svg>
		</button>

		<!-- Dropdown Menu -->
		{#if showMenu}
			<div
				class="absolute z-50 mt-2 w-80 max-w-[calc(100vw-2rem)] rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-xl right-0 sm:{size === 'sm' ? 'right-0' : 'left-0'}"
			>
				<div class="p-3">
					<!-- Mode Tabs -->
					<div class="flex gap-2 mb-3 border-b border-gray-200 dark:border-gray-700 pb-2">
						<button
							type="button"
							class="flex-1 px-3 py-2 rounded-md text-sm font-medium transition-colors {mode === 'rewrite'
								? 'bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300'
								: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}"
							onclick={() => (mode = 'rewrite')}
						>
							✨ {$t('admin.ai_assistant.mode_rewrite')}
						</button>
						<button
							type="button"
							class="flex-1 px-3 py-2 rounded-md text-sm font-medium transition-colors {mode === 'critique'
								? 'bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300'
								: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}"
							onclick={() => (mode = 'critique')}
						>
							💭 {$t('admin.ai_assistant.mode_feedback')}
						</button>
					</div>

					{#if mode === 'rewrite'}
						<!-- Tone Options -->
						<div class="space-y-2">
							<p class="text-xs text-gray-600 dark:text-gray-400 mb-2">{$t('admin.ai_assistant.choose_tone')}</p>
							{#each tones as tone}
								<button
									type="button"
									class="w-full text-left px-3 py-2 rounded-md border transition-all {selectedTone === tone.value
										? 'border-purple-500 bg-purple-50 dark:bg-purple-900/30'
										: 'border-gray-200 dark:border-gray-700 hover:border-purple-300 dark:hover:border-purple-700 hover:bg-gray-50 dark:hover:bg-gray-750'}"
									onclick={() => handleRewrite(tone.value)}
									disabled={loading}
								>
									<div class="flex items-start gap-2">
										<span class="text-xl" aria-hidden="true">{tone.icon}</span>
										<div class="flex-1 min-w-0">
											<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
												{tone.label}
											</div>
											<div class="text-xs text-gray-600 dark:text-gray-400">
												{tone.description}
											</div>
										</div>
									</div>
								</button>
							{/each}
						</div>
					{:else}
						<!-- Critique Mode -->
						<div class="space-y-3">
							<p class="text-sm text-gray-700 dark:text-gray-300">
								{$t('admin.ai_assistant.critique_description')}
							</p>
							<button
								type="button"
								class="w-full px-4 py-2 rounded-md bg-purple-600 hover:bg-purple-700 text-white font-medium transition-colors disabled:opacity-50"
								onclick={handleCritique}
								disabled={loading}
							>
								{loading ? $t('admin.ai_assistant.getting_feedback') : `💭 ${$t('admin.ai_assistant.get_feedback_button')}`}
							</button>
						</div>
					{/if}
				</div>

				<!-- Info Footer -->
				<div class="px-3 py-2 bg-gray-50 dark:bg-gray-750 rounded-b-lg border-t border-gray-200 dark:border-gray-700">
					<p class="text-xs text-gray-600 dark:text-gray-400">
						<strong>{$t('admin.ai_assistant.tip_label')}</strong> {$t('admin.ai_assistant.tip')}
					</p>
				</div>
			</div>
		{/if}

		<!-- Preview Modal -->
		{#if showPreview}
			<div
				class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
				onclick={cancelPreview}
				onkeydown={(e) => e.key === 'Escape' && cancelPreview()}
				role="button"
				tabindex="-1"
			>
				<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
				<div
					class="bg-white dark:bg-gray-800 rounded-lg shadow-2xl max-w-3xl w-full max-h-[90vh] sm:max-h-[80vh] overflow-hidden flex flex-col"
					onclick={stopPropagation(bubble('click'))}
					onkeydown={stopPropagation(bubble('keydown'))}
					role="dialog"
					aria-modal="true"
					aria-labelledby="ai-dialog-title"
					tabindex="-1"
				>
					<!-- Header -->
					<div
						class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between"
					>
						<h3 id="ai-dialog-title" class="text-lg font-semibold text-gray-900 dark:text-gray-100">
							{mode === 'critique' ? `💭 ${$t('admin.ai_assistant.preview_title_feedback')}` : `✨ ${$t('admin.ai_assistant.preview_title_rewrite')}`}
						</h3>
						<button
							type="button"
							class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
							onclick={cancelPreview}
							aria-label={$t('admin.ai_assistant.close')}
						>
							<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M6 18L18 6M6 6l12 12"
								></path>
							</svg>
						</button>
					</div>

					<!-- Content -->
					<div class="px-4 sm:px-6 py-4 overflow-y-auto flex-1">
						{#if mode === 'critique'}
							<div class="prose dark:prose-invert max-w-none">
								<div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-4">
									<p class="text-sm text-yellow-800 dark:text-yellow-300 m-0">
										<strong>{$t('admin.ai_assistant.note_label')}</strong>
										{@html $t('admin.ai_assistant.feedback_note', { values: { brackets: '<span class="text-purple-600 dark:text-purple-400">[brackets]</span>' } })}
									</p>
								</div>
								<div class="whitespace-pre-wrap font-mono text-sm bg-gray-50 dark:bg-gray-900 p-4 rounded border border-gray-200 dark:border-gray-700">
									{previewContent}
								</div>
							</div>
						{:else}
							<div class="space-y-4">
								<div>
									<p class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
										{$t('admin.ai_assistant.original_label')}
									</p>
									<div class="bg-gray-50 dark:bg-gray-900 p-4 rounded border border-gray-200 dark:border-gray-700 whitespace-pre-wrap">
										{content}
									</div>
								</div>

								<div>
									<p class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
										{$t('admin.ai_assistant.rewrite_label')}
									</p>
									<div class="bg-purple-50 dark:bg-purple-900/20 p-4 rounded border border-purple-200 dark:border-purple-700 whitespace-pre-wrap">
										{previewContent}
									</div>
								</div>
							</div>
						{/if}
					</div>

					<!-- Actions -->
					<div class="px-4 sm:px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row gap-3 flex-shrink-0">
						{#if mode === 'rewrite'}
							<button
								type="button"
								class="flex-1 px-4 py-2 rounded-md bg-purple-600 hover:bg-purple-700 text-white font-medium transition-colors"
								onclick={applyPreview}
							>
								✓ {$t('admin.ai_assistant.apply_changes')}
							</button>
						{/if}
						<button
							type="button"
							class="flex-1 px-4 py-2 rounded-md border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-750 font-medium transition-colors"
							onclick={cancelPreview}
						>
							{mode === 'critique' ? $t('admin.ai_assistant.close') : $t('admin.ai_assistant.cancel')}
						</button>
					</div>
				</div>
			</div>
		{/if}
	</div>
{/if}

<style>
	.ai-button {
		font-variant-numeric: tabular-nums;
	}

	/* Click outside to close menu */
	:global(body:has(.ai-content-helper [data-menu-open])) {
		overflow: hidden;
	}
</style>

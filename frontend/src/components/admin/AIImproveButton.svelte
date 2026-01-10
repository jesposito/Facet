<script lang="ts">
	import { run } from 'svelte/legacy';

	import { createEventDispatcher } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';

	
	interface Props {
		// Props
		contentType?: 'headline' | 'summary' | 'description' | 'bullets' | 'experience' | 'project' | 'education';
		content?: string;
		context?: Record<string, string>;
		action?: 'improve' | 'generate' | 'expand' | 'shorten';
		label?: string;
		size?: 'sm' | 'md';
		disabled?: boolean;
	}

	let {
		contentType = 'description',
		content = '',
		context = {},
		action = 'improve',
		label = '',
		size = 'sm',
		disabled = false
	}: Props = $props();

	const dispatch = createEventDispatcher<{
		result: { content: string };
	}>();

	let loading = $state(false);
	let aiAvailable: boolean | null = $state(null);

	// Check AI availability on mount
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

	// Call on mount
	run(() => {
		if (aiAvailable === null) {
			checkAIStatus();
		}
	});

	async function handleImprove() {
		if (loading || !aiAvailable) return;

		loading = true;
		try {
			const response = await fetch('/api/ai/improve', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					content_type: contentType,
					content,
					context,
					action
				})
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'AI request failed');
			}

			const result = await response.json();
			dispatch('result', { content: result.improved_content });
			toasts.add('success', 'Content improved with AI');
		} catch (err) {
			console.error('AI improve failed:', err);
			toasts.add('error', err instanceof Error ? err.message : 'Failed to improve content');
		} finally {
			loading = false;
		}
	}

	// Determine button label
	let buttonLabel = $derived(label || (action === 'generate' ? 'Generate' : action === 'expand' ? 'Expand' : action === 'shorten' ? 'Shorten' : 'Improve'));
</script>

{#if aiAvailable}
	<button
		type="button"
		class="inline-flex items-center gap-1.5 text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 transition-colors
			{size === 'sm' ? 'text-xs' : 'text-sm'}
			{disabled || loading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}"
		onclick={handleImprove}
		disabled={disabled || loading}
		title="{buttonLabel} with AI"
	>
		{#if loading}
			<svg class="animate-spin h-3.5 w-3.5" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
		{:else}
			{@html icon('sparkles')}
		{/if}
		<span>{buttonLabel}</span>
	</button>
{/if}

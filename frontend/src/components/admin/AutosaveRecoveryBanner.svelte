<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	interface Props {
		savedAt: number;
		isEditing: boolean;
		visible: boolean;
	}

	let {
		savedAt,
		isEditing,
		visible
	}: Props = $props();

	const dispatch = createEventDispatcher<{
		restore: void;
		dismiss: void;
	}>();

	function formatTimeAgo(timestamp: number): string {
		const seconds = Math.floor((Date.now() - timestamp) / 1000);
		if (seconds < 60) return 'just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`;
		const days = Math.floor(hours / 24);
		return `${days} day${days === 1 ? '' : 's'} ago`;
	}

	const timeAgo = $derived(formatTimeAgo(savedAt));
	const actionText = $derived(isEditing ? 'editing session' : 'new item draft');
</script>

{#if visible}
	<div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md p-4 mb-4">
		<div class="flex items-start">
			<div class="flex-shrink-0">
				<svg class="w-5 h-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<div class="ml-3 flex-1">
				<h3 class="text-sm font-medium text-blue-800 dark:text-blue-200">
					Draft Available
				</h3>
				<div class="mt-1 text-sm text-blue-700 dark:text-blue-300">
					You have an unsaved {actionText} from {timeAgo}.
				</div>
				<div class="mt-3 flex space-x-3">
					<button
						type="button"
						class="bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200 px-3 py-1.5 rounded-md text-sm font-medium hover:bg-blue-200 dark:hover:bg-blue-700 transition-colors"
						onclick={() => dispatch('restore')}
					>
						Restore Draft
					</button>
					<button
						type="button"
						class="text-blue-800 dark:text-blue-200 text-sm font-medium hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
						onclick={() => dispatch('dismiss')}
					>
						Dismiss
					</button>
				</div>
			</div>
			<div class="ml-3">
				<button
					type="button"
					class="text-blue-400 hover:text-blue-500 dark:hover:text-blue-300 transition-colors"
					onclick={() => dispatch('dismiss')}
					aria-label="Close banner"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>
	</div>
{/if}

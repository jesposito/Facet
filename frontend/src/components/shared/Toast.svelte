<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Toast } from '$lib/stores';

	export let toast: Toast;

	const dispatch = createEventDispatcher();

	const icons = {
		success: '✓',
		error: '✕',
		info: 'ℹ',
		warning: '⚠'
	};

	const colors = {
		success: 'bg-green-500',
		error: 'bg-red-500',
		info: 'bg-blue-500',
		warning: 'bg-yellow-500'
	};
</script>

<div
	class="animate-fade-in flex items-center gap-3 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-4 min-w-[300px]"
	role="alert"
>
	<span
		class="{colors[toast.type]} text-white w-6 h-6 rounded-full flex items-center justify-center text-sm font-bold"
	>
		{icons[toast.type]}
	</span>
	<span class="flex-1 text-gray-900 dark:text-gray-100">{toast.message}</span>
	<button
		class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
		on:click={() => dispatch('dismiss')}
		aria-label="Dismiss"
	>
		✕
	</button>
</div>

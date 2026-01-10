<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { icon } from '$lib/icons';
	import type { Toast } from '$lib/stores';

	interface Props {
		toast: Toast;
	}

	let { toast }: Props = $props();

	const dispatch = createEventDispatcher();

	const iconMap = {
		success: 'check',
		error: 'x',
		info: 'info',
		warning: 'warning'
	} as const;

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
		class="{colors[toast.type]} text-white w-6 h-6 rounded-full flex items-center justify-center"
	>
		{@html icon(iconMap[toast.type])}
	</span>
	<span class="flex-1 text-gray-900 dark:text-gray-100">{toast.message}</span>
	<button
		class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 p-1"
		onclick={() => dispatch('dismiss')}
		aria-label="Dismiss"
	>
		{@html icon('x')}
	</button>
</div>

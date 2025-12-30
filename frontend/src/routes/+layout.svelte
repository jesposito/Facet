<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { theme, toasts } from '$lib/stores';
	import Toast from '$components/shared/Toast.svelte';

	onMount(() => {
		theme.initialize();
	});
</script>

<svelte:head>
	<meta name="theme-color" content="#0ea5e9" />
</svelte:head>

<slot />

<!-- Toast notifications -->
<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
	{#each $toasts as toast (toast.id)}
		<Toast {toast} on:dismiss={() => toasts.remove(toast.id)} />
	{/each}
</div>

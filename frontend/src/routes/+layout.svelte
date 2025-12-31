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

<!-- Skip link for keyboard navigation -->
<a href="#main-content" class="skip-link">
	Skip to main content
</a>

<slot />

<!-- Toast notifications - live region for screen readers -->
<div
	class="fixed bottom-4 right-4 z-50 flex flex-col gap-2"
	role="region"
	aria-label="Notifications"
	aria-live="polite"
>
	{#each $toasts as toast (toast.id)}
		<Toast {toast} on:dismiss={() => toasts.remove(toast.id)} />
	{/each}
</div>

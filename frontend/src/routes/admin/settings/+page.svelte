<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	// Redirect to the default settings sub-page
	onMount(() => {
		goto('/admin/settings/account', { replaceState: true });
	});
</script>

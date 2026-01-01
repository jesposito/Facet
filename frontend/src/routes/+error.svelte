<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	onMount(() => {
		console.error('[ERROR PAGE] Displayed error:', {
			status: $page.status,
			message: $page.error?.message,
			url: $page.url.href
		});
	});
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="text-center p-8 max-w-lg">
		<h1 class="text-6xl font-bold text-gray-900 dark:text-white mb-4">
			{$page.status}
		</h1>
		<p class="text-xl text-gray-600 dark:text-gray-400 mb-8">
			{#if $page.error?.message}
				{$page.error.message}
			{:else if $page.status === 404}
				Page not found
			{:else}
				Something went wrong
			{/if}
		</p>

		<!-- Debug info for development -->
		<div class="text-left text-sm bg-gray-100 dark:bg-gray-800 p-4 rounded-lg mb-8">
			<p class="text-gray-500 dark:text-gray-400 mb-2">Debug info:</p>
			<ul class="text-gray-600 dark:text-gray-400 space-y-1">
				<li><strong>URL:</strong> {$page.url.pathname}</li>
				<li><strong>Status:</strong> {$page.status}</li>
				<li><strong>Error:</strong> {$page.error?.message || 'None'}</li>
			</ul>
		</div>

		<div class="space-x-4">
			<a
				href="/"
				class="inline-flex items-center gap-2 px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
				</svg>
				Go Home
			</a>
			<button
				on:click={() => window.location.reload()}
				class="inline-flex items-center gap-2 px-6 py-3 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Reload Page
			</button>
		</div>
	</div>
</div>

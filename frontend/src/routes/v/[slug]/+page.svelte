<!--
	Legacy route: /v/<slug>

	This route always redirects to /<slug> (the canonical URL).
	If you see this page, the redirect failed unexpectedly.
-->
<script lang="ts">
	// This page should never render - the load function always redirects
</script>

<svelte:head>
	<title>Redirecting...</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center">
	<div class="text-center">
		<p class="text-gray-600 dark:text-gray-400">Redirecting...</p>
		<noscript>
			<p class="mt-2 text-sm text-gray-500">
				If you're not redirected, <a href="/" class="text-primary-600 hover:underline">click here</a>.
			</p>
		</noscript>
	</div>
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';

	let loading = $state(true);
	let success = $state(false);
	let error = $state('');

	onMount(async () => {
		const token = $page.params.token;
		try {
			const response = await fetch(`/api/testimonials/verify/email/${token}`);
			const data = await response.json();
			
			if (response.ok && data.status === 'verified') {
				success = true;
			} else {
				error = data.error || 'Verification failed. The link may have expired.';
			}
		} catch {
			error = 'Verification failed. Please try again.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Verify Testimonial</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center py-12 px-4">
	<div class="max-w-md w-full">
		{#if loading}
			<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm p-8 text-center">
				<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto mb-4"></div>
				<p class="text-gray-600 dark:text-gray-400">Verifying your email...</p>
			</div>
		{:else if success}
			<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm p-8 text-center">
				<div class="w-16 h-16 mx-auto mb-4 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center">
					<svg class="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Email Verified!</h1>
				<p class="text-gray-600 dark:text-gray-400 mb-4">
					Your testimonial is now verified. Thank you for taking the time to provide feedback.
				</p>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					You can close this window.
				</p>
			</div>
		{:else}
			<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm p-8 text-center">
				<div class="w-16 h-16 mx-auto mb-4 bg-red-100 dark:bg-red-900 rounded-full flex items-center justify-center">
					<svg class="w-8 h-8 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Verification Failed</h1>
				<p class="text-gray-600 dark:text-gray-400">
					{error}
				</p>
			</div>
		{/if}
	</div>
</div>

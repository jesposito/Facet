<script lang="ts">
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pocketbase';

	let loading = false;
	let error = '';

	async function handleDemoLogin() {
		loading = true;
		error = '';

		try {
			const response = await fetch('/api/demo/login', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to login to demo');
			}

			const data = await response.json();

			// Store auth data in PocketBase
			pb.authStore.save(data.token, data.record);

			// Redirect to home page (will show The Doctor's profile)
			goto('/');
		} catch (err: any) {
			error = err.message || 'Something went wrong';
			loading = false;
		}
	}
</script>

<div class="min-h-screen bg-gradient-to-br from-primary-50 via-white to-accent-50 dark:from-gray-900 dark:via-gray-900 dark:to-gray-800 flex items-center justify-center p-4">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40">
		<ThemeToggle />
	</div>

	<div class="max-w-3xl w-full bg-white dark:bg-gray-800 rounded-2xl shadow-xl border border-gray-200 dark:border-gray-700 p-8 md:p-12 space-y-8">
		<!-- Header -->
		<div class="text-center space-y-4">
			<div class="inline-block p-4 bg-primary-100 dark:bg-primary-900/30 rounded-full">
				<svg class="w-16 h-16 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
					<path stroke-linecap="round" stroke-linejoin="round" d="M17.982 18.725A7.488 7.488 0 0012 15.75a7.488 7.488 0 00-5.982 2.975m11.963 0a9 9 0 10-11.963 0m11.963 0A8.966 8.966 0 0112 21a8.966 8.966 0 01-5.982-2.275M15 9.75a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
			</div>
			<h1 class="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white">
				Welcome to Facet
			</h1>
			<p class="text-xl text-gray-600 dark:text-gray-300">
				Your personal profile platform. Self-hosted, private, and totally yours.
			</p>
		</div>

		<!-- What you can do -->
		<div class="space-y-6 pt-4">
			<h2 class="text-2xl font-semibold text-gray-900 dark:text-white text-center">
				What can you do with Facet?
			</h2>

			<div class="grid gap-4 sm:grid-cols-2">
				<!-- Feature 1 -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Create Multiple Views</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Show different audiences what matters to them</p>
					</div>
				</div>

				<!-- Feature 2 -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Privacy Controls</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Public, unlisted, password-protected, or private</p>
					</div>
				</div>

				<!-- Feature 3 -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Own Your Data</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">SQLite database you control, easy to backup</p>
					</div>
				</div>

				<!-- Feature 4 -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">AI-Powered Features</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Optional AI writing assistant and resume generation</p>
					</div>
				</div>
			</div>
		</div>

		<!-- CTA -->
		<div class="pt-8 border-t border-gray-200 dark:border-gray-700 space-y-6 text-center">
			<!-- Error message -->
			{#if error}
				<div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
					<p class="text-sm text-red-600 dark:text-red-400">{error}</p>
				</div>
			{/if}

			<!-- Primary CTA - Try Demo -->
			<div class="space-y-3">
				<button
					on:click={handleDemoLogin}
					disabled={loading}
					class="inline-flex items-center gap-2 px-8 py-4 text-lg font-semibold text-white bg-accent-600 hover:bg-accent-700 disabled:bg-accent-400 rounded-lg shadow-sm transition-colors"
				>
					{#if loading}
						<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{:else}
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
							<path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
					{/if}
					{loading ? 'Loading...' : 'Try Demo'}
				</button>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					One-click access to The Doctor's hilarious profile
				</p>
			</div>

			<!-- Secondary CTA - Sign In -->
			<div class="pt-4">
				<a
					href="/admin"
					class="inline-flex items-center gap-2 px-6 py-3 text-base font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 border-2 border-primary-600 dark:border-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 rounded-lg transition-colors"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
					</svg>
					Sign In to Build Your Own
				</a>
				<p class="text-sm text-gray-500 dark:text-gray-400 mt-2">
					Create your own profile after seeing the demo
				</p>
			</div>
		</div>

		<!-- Footer note -->
		<div class="pt-4 text-center text-sm text-gray-500 dark:text-gray-400">
			<p>No tracking. No ads. Just your profile, your way.</p>
		</div>
	</div>
</div>

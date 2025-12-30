<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let type: 'view' | 'project' | 'experience';
	export let id: string;

	const dispatch = createEventDispatcher();

	let password = '';
	let error = '';
	let loading = false;

	async function handleSubmit() {
		if (!password) {
			error = 'Please enter a password';
			return;
		}

		loading = true;
		error = '';

		try {
			const response = await fetch('/api/password/check', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ type, id, password })
			});

			if (!response.ok) {
				const data = await response.json();
				error = data.error || 'Incorrect password';
				return;
			}

			dispatch('verified');
		} catch (err) {
			error = 'Failed to verify password';
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
	<div class="card p-8 max-w-md w-full">
		<div class="text-center mb-6">
			<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
				<svg class="w-8 h-8 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
				</svg>
			</div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
				Password Protected
			</h1>
			<p class="text-gray-600 dark:text-gray-400 mt-2">
				This content requires a password to view.
			</p>
		</div>

		<form on:submit|preventDefault={handleSubmit}>
			<div class="mb-4">
				<label for="password" class="label">Password</label>
				<input
					type="password"
					id="password"
					bind:value={password}
					class="input"
					placeholder="Enter password"
					disabled={loading}
				/>
			</div>

			{#if error}
				<p class="text-red-600 dark:text-red-400 text-sm mb-4">{error}</p>
			{/if}

			<button
				type="submit"
				class="btn btn-primary w-full"
				disabled={loading}
			>
				{#if loading}
					<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Verifying...
				{:else}
					Continue
				{/if}
			</button>
		</form>

		<div class="mt-6 text-center">
			<a href="/" class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300">
				← Back to main profile
			</a>
		</div>
	</div>
</div>

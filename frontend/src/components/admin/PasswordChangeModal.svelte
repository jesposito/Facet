<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { onMount } from 'svelte';

	export let onPasswordChanged: () => void;

	let currentPassword = 'changeme123'; // Default password pre-filled
	let newPassword = '';
	let confirmPassword = '';
	let loading = false;
	let error = '';
	let passwordStrength = '';

	// Check password strength
	$: {
		if (newPassword.length === 0) {
			passwordStrength = '';
		} else if (newPassword.length < 8) {
			passwordStrength = 'weak';
		} else if (newPassword.length < 12) {
			passwordStrength = 'medium';
		} else {
			passwordStrength = 'strong';
		}
	}

	async function handleSubmit() {
		error = '';

		// Validate inputs
		if (!currentPassword) {
			error = 'Current password is required';
			return;
		}
		if (newPassword.length < 8) {
			error = 'New password must be at least 8 characters';
			return;
		}
		if (newPassword === currentPassword) {
			error = 'New password must be different from current password';
			return;
		}
		if (newPassword !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;

		try {
			const response = await fetch('/api/auth/change-password', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					currentPassword,
					newPassword
				})
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || data.message || 'Failed to change password');
			}

			// Success! Call the callback
			onPasswordChanged();
		} catch (err: any) {
			error = err.message || 'Failed to change password';
			loading = false;
		}
	}

	onMount(() => {
		// Focus the new password field
		setTimeout(() => {
			document.getElementById('new-password')?.focus();
		}, 100);
	});
</script>

<!-- Modal overlay (cannot be dismissed) -->
<div class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
	<div class="card max-w-md w-full p-6">
		<div class="mb-6">
			<h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">
				Change Your Password
			</h2>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				You're currently using the default password from the documentation. Please set a secure password before continuing.
			</p>
		</div>

		{#if error}
			<div class="mb-4 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">
				{error}
			</div>
		{/if}

		<form on:submit|preventDefault={handleSubmit} class="space-y-4">
			<!-- Current Password -->
			<div>
				<label for="current-password" class="label">Current Password</label>
				<input
					type="password"
					id="current-password"
					bind:value={currentPassword}
					class="input"
					disabled={loading}
					placeholder="changeme123"
				/>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					The default password from the documentation
				</p>
			</div>

			<!-- New Password -->
			<div>
				<label for="new-password" class="label">New Password</label>
				<input
					type="password"
					id="new-password"
					bind:value={newPassword}
					class="input"
					disabled={loading}
					placeholder="Enter a secure password"
					minlength="8"
				/>
				{#if passwordStrength}
					<div class="mt-2 flex items-center gap-2">
						<div class="flex-1 h-1 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
							<div
								class="h-full transition-all duration-300 {passwordStrength === 'weak'
									? 'w-1/3 bg-red-500'
									: passwordStrength === 'medium'
										? 'w-2/3 bg-yellow-500'
										: 'w-full bg-green-500'}"
							/>
						</div>
						<span
							class="text-xs {passwordStrength === 'weak'
								? 'text-red-600 dark:text-red-400'
								: passwordStrength === 'medium'
									? 'text-yellow-600 dark:text-yellow-400'
									: 'text-green-600 dark:text-green-400'}"
						>
							{passwordStrength === 'weak' ? 'Weak' : passwordStrength === 'medium' ? 'Medium' : 'Strong'}
						</span>
					</div>
				{/if}
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					Minimum 8 characters. Longer is better!
				</p>
			</div>

			<!-- Confirm Password -->
			<div>
				<label for="confirm-password" class="label">Confirm New Password</label>
				<input
					type="password"
					id="confirm-password"
					bind:value={confirmPassword}
					class="input"
					disabled={loading}
					placeholder="Re-enter your new password"
				/>
			</div>

			<!-- Submit Button -->
			<div class="pt-4">
				<button type="submit" class="btn btn-primary w-full" disabled={loading || !newPassword || !confirmPassword}>
					{#if loading}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Changing Password...
					{:else}
						Change Password & Continue
					{/if}
				</button>
			</div>
		</form>

		<div class="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
			<p class="text-xs text-blue-700 dark:text-blue-300">
				<strong>💡 Tip:</strong> Use a password manager to generate and store a strong, unique password for your Facet instance.
			</p>
		</div>
	</div>
</div>

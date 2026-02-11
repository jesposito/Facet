<script lang="ts">
	import { run, preventDefault } from 'svelte/legacy';

	import { pb } from '$lib/pocketbase';
	import { onMount, onDestroy } from 'svelte';
	import { t } from 'svelte-i18n';

	interface Props {
		onPasswordChanged: () => void;
	}

	let { onPasswordChanged }: Props = $props();

	let currentPassword = $state('changeme123'); // Default password pre-filled
	let newPassword = $state('');
	let confirmPassword = $state('');
	let loading = $state(false);
	let error = $state('');
	let passwordStrength = $state('');
	let isMobile = $state(false);

	let dialogEl: HTMLDivElement | undefined = $state();
	let previousActiveElement: HTMLElement | null = $state(null);

	// Check password strength
	run(() => {
		if (newPassword.length === 0) {
			passwordStrength = '';
		} else if (newPassword.length < 8) {
			passwordStrength = 'weak';
		} else if (newPassword.length < 12) {
			passwordStrength = 'medium';
		} else {
			passwordStrength = 'strong';
		}
	});

	async function handleSubmit() {
		error = '';

		// Validate inputs
		if (!currentPassword) {
			error = $t('admin.password_modal.error_current_required');
			return;
		}
		if (newPassword.length < 8) {
			error = $t('admin.password_modal.error_min_length');
			return;
		}
		if (newPassword === currentPassword) {
			error = $t('admin.password_modal.error_same_password');
			return;
		}
		if (newPassword !== confirmPassword) {
			error = $t('admin.password_modal.error_mismatch');
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

			// Update auth store with new token to stay logged in
			if (data.token && data.record) {
				pb.authStore.save(data.token, data.record);
			}

			// Success! Call the callback
			onPasswordChanged();
		} catch (err: any) {
			error = err.message || 'Failed to change password';
			loading = false;
		}
	}

	// Focus trap and keyboard handling
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Tab') {
			const focusableElements = dialogEl?.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			);

			if (!focusableElements?.length) return;

			const firstElement = focusableElements[0];
			const lastElement = focusableElements[focusableElements.length - 1];

			if (event.shiftKey && document.activeElement === firstElement) {
				event.preventDefault();
				lastElement.focus();
			} else if (!event.shiftKey && document.activeElement === lastElement) {
				event.preventDefault();
				firstElement.focus();
			}
		}
	}

	onMount(() => {
		const mq = window.matchMedia('(max-width: 767px)');
		isMobile = mq.matches;
		const handler = (e: MediaQueryListEvent) => isMobile = e.matches;
		mq.addEventListener('change', handler);

		previousActiveElement = document.activeElement as HTMLElement;
		document.body.style.overflow = 'hidden';

		// Focus the new password field
		setTimeout(() => {
			document.getElementById('new-password')?.focus();
		}, 100);

		return () => {
			mq.removeEventListener('change', handler);
			document.body.style.overflow = '';
			if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
				previousActiveElement.focus();
			}
		};
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Modal overlay (cannot be dismissed) -->
<div
	class="fixed inset-0 bg-black/50 flex {isMobile ? 'flex-col justify-end' : 'items-center justify-center'} p-4 z-50"
	role="dialog"
	aria-modal="true"
	aria-labelledby="password-change-title"
>
	<div
		bind:this={dialogEl}
		class="card w-full p-6 transform transition-transform {isMobile ? 'rounded-t-2xl rounded-b-none max-h-[90vh] overflow-y-auto' : 'max-w-md'}"
	>
		<div class="mb-6">
			<h2 id="password-change-title" class="text-2xl font-bold text-gray-900 dark:text-white mb-2">
				{$t('admin.password_modal.title')}
			</h2>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				{$t('admin.password_modal.description')}
			</p>
		</div>

		{#if error}
			<div role="alert" class="mb-4 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">
				{error}
			</div>
		{/if}

		<form onsubmit={preventDefault(handleSubmit)} class="space-y-4">
			<!-- Current Password -->
			<div>
				<label for="current-password" class="label">{$t('admin.password_modal.current_password_label')}</label>
				<input
					type="password"
					id="current-password"
					bind:value={currentPassword}
					class="input"
					disabled={loading}
					placeholder="changeme123"
				/>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.password_modal.current_password_hint')}
				</p>
			</div>

			<!-- New Password -->
			<div>
				<label for="new-password" class="label">{$t('admin.password_modal.new_password_label')}</label>
				<input
					type="password"
					id="new-password"
					bind:value={newPassword}
					class="input"
					disabled={loading}
					placeholder={$t('admin.password_modal.new_password_placeholder')}
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
							></div>
						</div>
						<span
							class="text-xs {passwordStrength === 'weak'
								? 'text-red-600 dark:text-red-400'
								: passwordStrength === 'medium'
									? 'text-yellow-600 dark:text-yellow-400'
									: 'text-green-600 dark:text-green-400'}"
						>
							{passwordStrength === 'weak' ? $t('admin.password_modal.strength_weak') : passwordStrength === 'medium' ? $t('admin.password_modal.strength_medium') : $t('admin.password_modal.strength_strong')}
						</span>
					</div>
				{/if}
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.password_modal.min_length_hint')}
				</p>
			</div>

			<!-- Confirm Password -->
			<div>
				<label for="confirm-password" class="label">{$t('admin.password_modal.confirm_password_label')}</label>
				<input
					type="password"
					id="confirm-password"
					bind:value={confirmPassword}
					class="input"
					disabled={loading}
					placeholder={$t('admin.password_modal.confirm_password_placeholder')}
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
						{$t('admin.password_modal.changing')}
					{:else}
						{$t('admin.password_modal.submit_button')}
					{/if}
				</button>
			</div>
		</form>

		<div class="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
			<p class="text-xs text-blue-700 dark:text-blue-300">
				{$t('admin.password_modal.tip')}
			</p>
		</div>
	</div>
</div>

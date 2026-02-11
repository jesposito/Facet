<script lang="ts">
	import { preventDefault } from 'svelte/legacy';
	import { pb } from '$lib/pocketbase';
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';

	interface Props {
		onVerified: () => void;
		onLogout: () => void;
	}

	let { onVerified, onLogout }: Props = $props();

	let code = $state('');
	let loading = $state(false);
	let error = $state('');
	let useRecoveryCode = $state(false);
	let isMobile = $state(false);

	function handleInput(e: Event) {
		const input = e.target as HTMLInputElement;
		let val = input.value;

		if (useRecoveryCode) {
			val = val.replace(/[^a-fA-F0-9-]/g, '').toLowerCase();
			if (val.length === 4 && !val.includes('-')) {
				val = val + '-';
			}
			if (val.length > 9) val = val.slice(0, 9);
		} else {
			val = val.replace(/\D/g, '');
			if (val.length > 6) val = val.slice(0, 6);
		}

		code = val;
	}

	async function handleSubmit() {
		error = '';
		const trimmed = code.trim();

		if (!useRecoveryCode && trimmed.length !== 6) {
			error = $t('admin.two_factor.error_code_length');
			return;
		}
		if (useRecoveryCode && (trimmed.length !== 9 || trimmed[4] !== '-')) {
			error = $t('admin.two_factor.error_recovery_format');
			return;
		}

		loading = true;
		try {
			const response = await fetch('/api/totp/verify', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({ code: trimmed })
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || $t('admin.two_factor.error_invalid'));
			}

			onVerified();
		} catch (err: any) {
			error = err.message || $t('admin.two_factor.error_invalid');
			loading = false;
		}
	}

	function toggleMode() {
		useRecoveryCode = !useRecoveryCode;
		code = '';
		error = '';
	}

	onMount(() => {
		const mq = window.matchMedia('(max-width: 767px)');
		isMobile = mq.matches;
		const handler = (e: MediaQueryListEvent) => (isMobile = e.matches);
		mq.addEventListener('change', handler);

		setTimeout(() => {
			document.getElementById('totp-code')?.focus();
		}, 100);

		return () => mq.removeEventListener('change', handler);
	});
</script>

<div class="fixed inset-0 bg-black/50 flex {isMobile ? 'flex-col justify-end' : 'items-center justify-center'} p-4 z-50">
	<div
		class="card w-full p-6 transform transition-transform {isMobile
			? 'rounded-t-2xl rounded-b-none max-h-[90vh] overflow-y-auto'
			: 'max-w-md'}"
	>
		<div class="mb-6">
			<div class="flex items-center gap-3 mb-2">
				<div class="w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
					<svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
					</svg>
				</div>
				<h2 class="text-2xl font-bold text-gray-900 dark:text-white">
					{$t('admin.two_factor.title')}
				</h2>
			</div>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				{useRecoveryCode ? $t('admin.two_factor.recovery_description') : $t('admin.two_factor.description')}
			</p>
		</div>

		{#if error}
			<div class="mb-4 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">
				{error}
			</div>
		{/if}

		<form onsubmit={preventDefault(handleSubmit)} class="space-y-4">
			<div>
				<label for="totp-code" class="label">
					{useRecoveryCode ? $t('admin.two_factor.recovery_label') : $t('admin.two_factor.code_label')}
				</label>
				<input
					type="text"
					id="totp-code"
					value={code}
					oninput={handleInput}
					class="input text-center text-2xl tracking-widest font-mono"
					disabled={loading}
					placeholder={useRecoveryCode ? 'xxxx-xxxx' : '000000'}
					autocomplete="one-time-code"
					inputmode={useRecoveryCode ? 'text' : 'numeric'}
				/>
			</div>

			<div class="pt-2">
				<button type="submit" class="btn btn-primary w-full" disabled={loading || !code.trim()}>
					{#if loading}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						{$t('admin.two_factor.verifying')}
					{:else}
						{$t('admin.two_factor.verify_button')}
					{/if}
				</button>
			</div>
		</form>

		<div class="mt-4 flex items-center justify-between">
			<button type="button" class="text-sm text-primary-600 dark:text-primary-400 hover:underline" onclick={toggleMode}>
				{useRecoveryCode ? $t('admin.two_factor.use_totp') : $t('admin.two_factor.use_recovery')}
			</button>
			<button type="button" class="text-sm text-gray-500 dark:text-gray-400 hover:underline" onclick={onLogout}>
				{$t('admin.two_factor.logout_link')}
			</button>
		</div>
	</div>
</div>

<script lang="ts">
	import { preventDefault } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { toasts } from '$lib/stores';
	import QRCode from 'qrcode';
	import PageHelp from '$components/admin/PageHelp.svelte';

	// Two-Factor Authentication state
	let totpEnabled = $state(false);
	let totpLoading = $state(false);
	let totpSetupData: { secret: string; url: string } | null = $state(null);
	let totpQrDataUrl: string = $state('');
	let totpSetupCode = $state('');
	let totpSetupError = $state('');
	let totpRecoveryCodes: string[] | null = $state(null);
	let totpDisableCode = $state('');
	let totpDisableError = $state('');
	let showDisable2FA = $state(false);
	let showRegenerateCodes = $state(false);
	let totpRegenerateCode = $state('');
	let totpRegenerateError = $state('');

	// Password change form
	let passwordForm = $state({
		currentPassword: '',
		newPassword: '',
		confirmPassword: '',
		loading: false,
		error: '',
		success: false
	});

	onMount(async () => {
		await loadTOTPStatus();
	});

	async function changePassword() {
		// Reset states
		passwordForm.error = '';
		passwordForm.success = false;

		// Validate
		if (passwordForm.newPassword.length < 8) {
			passwordForm.error = $t('admin.settings_page.account.error_min_length');
			return;
		}

		if (passwordForm.newPassword !== passwordForm.confirmPassword) {
			passwordForm.error = $t('admin.settings_page.account.error_mismatch');
			return;
		}

		if (passwordForm.newPassword === passwordForm.currentPassword) {
			passwordForm.error = $t('admin.settings_page.account.error_same_password');
			return;
		}

		passwordForm.loading = true;

		try {
			const response = await fetch('/api/auth/change-password', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					currentPassword: passwordForm.currentPassword,
					newPassword: passwordForm.newPassword
				})
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || $t('admin.settings_page.account.change_password_button'));
			}

			// Success!
			passwordForm.success = true;
			passwordForm.currentPassword = '';
			passwordForm.newPassword = '';
			passwordForm.confirmPassword = '';
			toasts.add('success', $t('admin.settings_page.account.password_changed'));
		} catch (err: any) {
			passwordForm.error = err.message || $t('admin.settings_page.account.change_password_button');
			toasts.add('error', $t('admin.settings_page.account.change_password_button'));
		} finally {
			passwordForm.loading = false;
		}
	}

	async function loadTOTPStatus() {
		try {
			const response = await fetch('/api/totp/status', {
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			if (response.ok) {
				const data = await response.json();
				totpEnabled = data.enabled;
			}
		} catch (err) {
			console.error('Failed to load TOTP status:', err);
		}
	}

	async function beginTOTPSetup() {
		totpLoading = true;
		totpSetupError = '';
		try {
			const response = await fetch('/api/totp/begin-setup', {
				method: 'POST',
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			const data = await response.json();
			if (!response.ok) throw new Error(data.error);
			totpSetupData = { secret: data.secret, url: data.url };
			// Generate QR code locally — never send TOTP secret to third-party services
			try {
				totpQrDataUrl = await QRCode.toDataURL(data.url, { width: 200, margin: 1 });
			} catch {
				totpQrDataUrl = '';
			}
		} catch (err: any) {
			totpSetupError = err.message || 'Failed to begin setup';
			toasts.add('error', totpSetupError);
		} finally {
			totpLoading = false;
		}
	}

	async function confirmTOTPSetup() {
		totpLoading = true;
		totpSetupError = '';
		try {
			const response = await fetch('/api/totp/confirm-setup', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({ code: totpSetupCode })
			});
			const data = await response.json();
			if (!response.ok) throw new Error(data.error);
			totpRecoveryCodes = data.recovery_codes;
			totpEnabled = true;
			totpSetupData = null;
			totpSetupCode = '';
			totpQrDataUrl = '';
			toasts.add('success', $t('admin.settings_page.two_factor.enabled_toast'));
		} catch (err: any) {
			totpSetupError = err.message || 'Invalid code';
		} finally {
			totpLoading = false;
		}
	}

	async function disableTOTP() {
		totpLoading = true;
		totpDisableError = '';
		try {
			const response = await fetch('/api/totp/disable', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({ code: totpDisableCode })
			});
			const data = await response.json();
			if (!response.ok) throw new Error(data.error);
			totpEnabled = false;
			totpDisableCode = '';
			showDisable2FA = false;
			toasts.add('success', $t('admin.settings_page.two_factor.disabled_toast'));
		} catch (err: any) {
			totpDisableError = err.message || 'Invalid code';
		} finally {
			totpLoading = false;
		}
	}

	async function regenerateRecoveryCodes() {
		totpLoading = true;
		totpRegenerateError = '';
		try {
			const response = await fetch('/api/totp/regenerate-codes', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({ code: totpRegenerateCode })
			});
			const data = await response.json();
			if (!response.ok) throw new Error(data.error);
			totpRecoveryCodes = data.recovery_codes;
			totpRegenerateCode = '';
			showRegenerateCodes = false;
			toasts.add('success', $t('admin.settings_page.two_factor.codes_regenerated_toast'));
		} catch (err: any) {
			totpRegenerateError = err.message || 'Invalid code';
		} finally {
			totpLoading = false;
		}
	}

	function dismissRecoveryCodes() {
		totpRecoveryCodes = null;
	}

	function cancelTOTPSetup() {
		totpSetupData = null;
		totpSetupCode = '';
		totpSetupError = '';
		totpQrDataUrl = '';
	}
</script>

<svelte:head>
	<title>{$t('admin.settings_page.account.section_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<PageHelp pageKey="settings">
		<p>{@html $t('admin.settings_page.help_text')}</p>
		<p>{$t('admin.settings_page.help_tip_1')}</p>
		<p>{@html $t('admin.settings_page.help_tip_2')}</p>
	</PageHelp>

	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{$t('admin.settings_page.account.section_title')}</h1>
	<p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
		{$t('admin.settings_page.account.section_description')}
	</p>

	<!-- Account & Security section -->
	<div class="space-y-4">
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.settings_page.account.change_password_title')}</h2>

			<form onsubmit={preventDefault(changePassword)} class="space-y-4 max-w-md">
				<div>
					<label for="current-password-settings" class="label">{$t('admin.settings_page.account.current_password_label')}</label>
					<input
						type="password"
						id="current-password-settings"
						bind:value={passwordForm.currentPassword}
						class="input"
						disabled={passwordForm.loading}
						required
					/>
				</div>

				<div>
					<label for="new-password-settings" class="label">{$t('admin.settings_page.account.new_password_label')}</label>
					<input
						type="password"
						id="new-password-settings"
						bind:value={passwordForm.newPassword}
						class="input"
						disabled={passwordForm.loading}
						minlength="8"
						required
					/>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
						{$t('admin.settings_page.account.new_password_help')}
					</p>
				</div>

				<div>
					<label for="confirm-password-settings" class="label">{$t('admin.settings_page.account.confirm_password_label')}</label>
					<input
						type="password"
						id="confirm-password-settings"
						bind:value={passwordForm.confirmPassword}
						class="input"
						disabled={passwordForm.loading}
						required
					/>
				</div>

				{#if passwordForm.error}
					<div class="p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">
						{passwordForm.error}
					</div>
				{/if}

				{#if passwordForm.success}
					<div class="p-3 rounded-lg bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 text-sm">
						{$t('admin.settings_page.account.password_changed')}
					</div>
				{/if}

				<button
					type="submit"
					class="btn btn-primary"
					disabled={passwordForm.loading}
				>
					{#if passwordForm.loading}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						{$t('admin.settings_page.account.changing_password')}
					{:else}
						{$t('admin.settings_page.account.change_password_button')}
					{/if}
				</button>
			</form>
		</div>

		<!-- Two-Factor Authentication -->
		<div class="card p-6">
			<div class="flex items-center justify-between mb-4">
				<div>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.two_factor.title')}</h2>
					<p class="text-gray-600 dark:text-gray-400 text-sm mt-1">{$t('admin.settings_page.two_factor.description')}</p>
				</div>
				{#if totpEnabled}
					<span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300">
						<span class="w-1.5 h-1.5 rounded-full bg-green-500"></span>
						{$t('admin.settings_page.two_factor.status_enabled')}
					</span>
				{:else}
					<span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
						{$t('admin.settings_page.two_factor.status_disabled')}
					</span>
				{/if}
			</div>

			{#if totpRecoveryCodes}
				<!-- Recovery codes display -->
				<div class="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
					<div class="flex items-start gap-3 mb-3">
						<svg class="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
						</svg>
						<div>
							<p class="text-sm font-medium text-amber-800 dark:text-amber-200">{$t('admin.settings_page.two_factor.recovery_warning')}</p>
							<p class="text-xs text-amber-700 dark:text-amber-300 mt-1">{$t('admin.settings_page.two_factor.recovery_save_hint')}</p>
						</div>
					</div>
					<div class="grid grid-cols-2 gap-2 mb-3">
						{#each totpRecoveryCodes as code}
							<div class="font-mono text-sm bg-white dark:bg-gray-800 px-3 py-2 rounded text-center border border-amber-200 dark:border-amber-700">
								{code}
							</div>
						{/each}
					</div>
					<button type="button" class="btn btn-secondary btn-sm w-full" onclick={dismissRecoveryCodes}>
						{$t('admin.settings_page.two_factor.recovery_dismiss')}
					</button>
				</div>
			{/if}

			{#if !totpEnabled}
				{#if totpSetupData}
					<!-- Setup flow: QR code + confirm -->
					<div class="space-y-4">
						<div class="text-center">
							<p class="text-sm text-gray-600 dark:text-gray-400 mb-3">{$t('admin.settings_page.two_factor.scan_qr')}</p>
							<div class="inline-block p-4 bg-white rounded-lg border border-gray-200 dark:border-gray-600">
								{#if totpQrDataUrl}
									<img src={totpQrDataUrl} alt="QR Code" width="200" height="200" />
								{:else}
									<div class="w-[200px] h-[200px] flex items-center justify-center text-sm text-gray-500">
										QR code unavailable — use manual entry below
									</div>
								{/if}
							</div>
						</div>

						<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3">
							<p class="text-xs text-gray-500 dark:text-gray-400 mb-1">{$t('admin.settings_page.two_factor.manual_entry')}</p>
							<code class="text-sm font-mono text-gray-900 dark:text-white break-all">{totpSetupData.secret}</code>
						</div>

						{#if totpSetupError}
							<div class="p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">
								{totpSetupError}
							</div>
						{/if}

						<div>
							<label for="totp-setup-code" class="label">{$t('admin.settings_page.two_factor.confirm_code_label')}</label>
							<input
								type="text"
								id="totp-setup-code"
								bind:value={totpSetupCode}
								class="input text-center text-xl tracking-widest font-mono"
								placeholder="000000"
								maxlength="6"
								inputmode="numeric"
								disabled={totpLoading}
							/>
						</div>

						<div class="flex gap-3">
							<button type="button" class="btn btn-secondary flex-1" onclick={cancelTOTPSetup} disabled={totpLoading}>
								{$t('shared.actions.cancel')}
							</button>
							<button type="button" class="btn btn-primary flex-1" onclick={confirmTOTPSetup} disabled={totpLoading || totpSetupCode.length !== 6}>
								{#if totpLoading}
									<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
								{/if}
								{$t('admin.settings_page.two_factor.confirm_button')}
							</button>
						</div>
					</div>
				{:else}
					<!-- Enable button -->
					<button type="button" class="btn btn-primary" onclick={beginTOTPSetup} disabled={totpLoading}>
						{#if totpLoading}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
						{/if}
						{$t('admin.settings_page.two_factor.enable_button')}
					</button>
				{/if}
			{:else}
				<!-- 2FA is enabled — management actions -->
				<div class="space-y-3">
					{#if showDisable2FA}
						<div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
							<p class="text-sm text-red-700 dark:text-red-300 mb-3">{$t('admin.settings_page.two_factor.disable_confirm')}</p>
							{#if totpDisableError}
								<div class="mb-3 p-2 rounded bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300 text-sm">
									{totpDisableError}
								</div>
							{/if}
							<input
								type="text"
								bind:value={totpDisableCode}
								class="input text-center text-xl tracking-widest font-mono mb-3"
								placeholder="000000"
								maxlength="9"
								disabled={totpLoading}
							/>
							<div class="flex gap-3">
								<button type="button" class="btn btn-secondary btn-sm flex-1" onclick={() => { showDisable2FA = false; totpDisableCode = ''; totpDisableError = ''; }}>
									{$t('shared.actions.cancel')}
								</button>
								<button type="button" class="btn btn-sm flex-1 bg-red-600 hover:bg-red-700 text-white" onclick={disableTOTP} disabled={totpLoading || !totpDisableCode.trim()}>
									{$t('admin.settings_page.two_factor.disable_button')}
								</button>
							</div>
						</div>
					{:else if showRegenerateCodes}
						<div class="p-4 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
							<p class="text-sm text-gray-700 dark:text-gray-300 mb-3">{$t('admin.settings_page.two_factor.regenerate_confirm')}</p>
							{#if totpRegenerateError}
								<div class="mb-3 p-2 rounded bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300 text-sm">
									{totpRegenerateError}
								</div>
							{/if}
							<input
								type="text"
								bind:value={totpRegenerateCode}
								class="input text-center text-xl tracking-widest font-mono mb-3"
								placeholder="000000"
								maxlength="6"
								inputmode="numeric"
								disabled={totpLoading}
							/>
							<div class="flex gap-3">
								<button type="button" class="btn btn-secondary btn-sm flex-1" onclick={() => { showRegenerateCodes = false; totpRegenerateCode = ''; totpRegenerateError = ''; }}>
									{$t('shared.actions.cancel')}
								</button>
								<button type="button" class="btn btn-primary btn-sm flex-1" onclick={regenerateRecoveryCodes} disabled={totpLoading || totpRegenerateCode.length !== 6}>
									{$t('admin.settings_page.two_factor.regenerate_button')}
								</button>
							</div>
						</div>
					{:else}
						<div class="flex flex-wrap gap-3">
							<button type="button" class="btn btn-secondary btn-sm" onclick={() => showRegenerateCodes = true}>
								{$t('admin.settings_page.two_factor.regenerate_codes')}
							</button>
							<button type="button" class="btn btn-sm bg-red-600 hover:bg-red-700 text-white" onclick={() => showDisable2FA = true}>
								{$t('admin.settings_page.two_factor.disable_2fa')}
							</button>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>

</div>

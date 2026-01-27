<script lang="ts">
	import { run, preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { pb, type Profile } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import {
		ACCENT_COLORS,
		ACCENT_COLOR_LIST,
		DEFAULT_ACCENT_COLOR,
		type AccentColor
	} from '$lib/colors';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import LanguageSwitcher from '$components/admin/LanguageSwitcher.svelte';

	// App version from Vite config
	const appVersion = __APP_VERSION__;

	let loading = $state(true);
	let providers: Array<Record<string, unknown>> = $state([]);
	let showAddForm = $state(false);
	let testing: string | null = $state(null);

	// Site settings (custom CSS, analytics)
	let siteSettingsLoading = $state(true);
	let siteSettingsSaving = $state(false);
	let customCSS = $state('');
	let gaMeasurementId = $state('');
	let hideDemoToggle = $state(false);
	let showCSSHelp = $state(false);

	// Favicon state
	let faviconUrl: string | null = $state(null);
	let faviconFile: File | null = $state(null);
	let faviconBlobUrl: string | null = $state(null);
	let faviconSaving = $state(false);

	// Export state
	let exporting: string | null = $state(null);

	// Appearance state
	let profile: Profile | null = $state(null);
	let selectedAccentColor: AccentColor = $state(DEFAULT_ACCENT_COLOR);
	let savingAppearance = $state(false);

	// Password change form
	let passwordForm = $state({
		currentPassword: '',
		newPassword: '',
		confirmPassword: '',
		loading: false,
		error: '',
		success: false
	});

	// New provider form
	let newProvider = $state({
		name: '',
		type: 'openai' as 'openai' | 'anthropic' | 'ollama' | 'custom',
		api_key: '',
		base_url: '',
		model: '',
		is_active: true,
		is_default: false
	});

	// Fallback model options per provider type (used when API fetch fails or is unavailable)
	const fallbackModelOptions: Record<string, string[]> = {
		openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo', 'o1', 'o1-mini'],
		anthropic: ['claude-sonnet-4-20250514', 'claude-opus-4-20250514', 'claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022'],
		ollama: ['llama3.2', 'llama3.1', 'mistral', 'codellama', 'phi3'],
		custom: []
	};

	const defaultModels: Record<string, string> = {
		openai: 'gpt-4o-mini',
		anthropic: 'claude-sonnet-4-20250514',
		ollama: 'llama3.2',
		custom: ''
	};

	// Dynamic model state
	let fetchedModels = $state<string[]>([]);
	let fetchingModels = $state(false);
	let modelFetchError = $state('');

	// Computed: use fetched models if available, otherwise fallback
	let modelOptions = $derived.by(() => {
		if (fetchedModels.length > 0) {
			return fetchedModels;
		}
		return fallbackModelOptions[newProvider.type] || [];
	});

	// Fetch models when API key changes or provider type changes
	async function fetchModels() {
		// Don't fetch for custom without base_url, or if no API key for non-ollama
		if (newProvider.type === 'custom' && !newProvider.base_url) {
			fetchedModels = [];
			return;
		}
		if (newProvider.type !== 'ollama' && !newProvider.api_key) {
			fetchedModels = [];
			return;
		}

		fetchingModels = true;
		modelFetchError = '';

		try {
			const response = await fetch('/api/ai/models', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					type: newProvider.type,
					api_key: newProvider.api_key,
					base_url: newProvider.base_url || undefined
				})
			});

			const data = await response.json();
			if (data.error) {
				modelFetchError = data.error;
				fetchedModels = [];
			} else if (data.models && data.models.length > 0) {
				fetchedModels = data.models;
				// If current model not in list, reset to first or default
				if (!data.models.includes(newProvider.model)) {
					newProvider.model = data.models.includes(defaultModels[newProvider.type])
						? defaultModels[newProvider.type]
						: data.models[0];
				}
			} else {
				fetchedModels = [];
			}
		} catch (err) {
			modelFetchError = 'Failed to fetch models';
			fetchedModels = [];
		} finally {
			fetchingModels = false;
		}
	}

	// Reset model and fetch when provider type changes
	run(() => {
		if (newProvider.type) {
			const options = fallbackModelOptions[newProvider.type] || [];
			if (!options.includes(newProvider.model)) {
				newProvider.model = defaultModels[newProvider.type] || '';
			}
			// Reset fetched models when type changes
			fetchedModels = [];
			modelFetchError = '';
		}
	});

	onMount(async () => {
		await Promise.all([loadProviders(), loadProfile(), loadSiteSettings()]);
	});

	async function loadProfile() {
		try {
			const records = await await collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				profile = records.items[0] as unknown as Profile;
				selectedAccentColor = (profile.accent_color as AccentColor) || DEFAULT_ACCENT_COLOR;
			}
		} catch (err) {
			console.error('Failed to load profile:', err);
		}
	}

	async function saveAccentColor(color: AccentColor) {
		if (!profile) return;

		savingAppearance = true;
		try {
			await await collection('profile').update(profile.id, {
				accent_color: color
			});
			selectedAccentColor = color;
			profile.accent_color = color;
			toasts.add('success', $t('admin.settings_page.appearance.accent_color_updated'));

			// Dispatch event to notify layout of color change
			window.dispatchEvent(new CustomEvent('accent-color-changed', { detail: color }));
		} catch (err) {
			console.error('Failed to save accent color:', err);
			toasts.add('error', $t('admin.settings_page.appearance.accent_color_error'));
		} finally {
			savingAppearance = false;
		}
	}

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

	async function loadProviders() {
		try {
			const result = await pb.collection('ai_providers').getList(1, 50);
			providers = result.items;
		} catch (err) {
			console.error('Failed to load providers:', err);
		} finally {
			loading = false;
		}
	}

	async function loadSiteSettings() {
		try {
			const response = await fetch('/api/site-settings');
			if (response.ok) {
				const data = await response.json();
				customCSS = data.custom_css || '';
				gaMeasurementId = data.ga_measurement_id || '';
				hideDemoToggle = data.hide_demo_toggle || false;
				faviconUrl = data.favicon || null;
			}
		} catch (err) {
			console.error('Failed to load site settings:', err);
		} finally {
			siteSettingsLoading = false;
		}
	}

	async function saveSiteSettings() {
		siteSettingsSaving = true;
		try {
			const response = await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					custom_css: customCSS,
					ga_measurement_id: gaMeasurementId
				})
			});

			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || $t('admin.settings_page.settings_error'));
				return;
			}

			customCSS = result.custom_css || '';
			gaMeasurementId = result.ga_measurement_id || '';
			toasts.add('success', $t('admin.settings_page.settings_saved'));
		} catch (err) {
			console.error('Failed to save site settings:', err);
			toasts.add('error', $t('admin.settings_page.settings_error'));
		} finally {
			siteSettingsSaving = false;
		}
	}

	const MAX_FAVICON_SIZE = 524288; // 512KB

	function handleFaviconSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files?.length) return;

		const file = input.files[0];

		// Validate file size
		if (file.size > MAX_FAVICON_SIZE) {
			toasts.add('error', $t('admin.settings_page.appearance.favicon_size_error', { values: { size: Math.round(file.size / 1024) } }));
			input.value = ''; // Reset input
			return;
		}

		if (faviconBlobUrl) URL.revokeObjectURL(faviconBlobUrl);
		faviconFile = file;
		faviconBlobUrl = URL.createObjectURL(faviconFile);
		faviconUrl = faviconBlobUrl;
	}

	function cancelFaviconUpload() {
		if (faviconBlobUrl) {
			URL.revokeObjectURL(faviconBlobUrl);
			faviconBlobUrl = null;
		}
		faviconFile = null;
		// Restore original favicon URL from server
		loadSiteSettings();
	}

	async function saveFavicon() {
		if (!faviconFile) return;

		faviconSaving = true;
		try {
			const formData = new FormData();
			formData.append('favicon', faviconFile);

			const response = await fetch('/api/site-settings/favicon', {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token || ''
				},
				body: formData
			});

			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || $t('admin.settings_page.appearance.favicon_error'));
				return;
			}

			// Clean up blob URL
			if (faviconBlobUrl) {
				URL.revokeObjectURL(faviconBlobUrl);
				faviconBlobUrl = null;
			}
			faviconFile = null;

			// Update URL with cache buster
			if (result.favicon) {
				const newUrl = `${result.favicon}?${Date.now()}`;
				faviconUrl = newUrl;
				// Notify layout to update favicon in browser tab
				window.dispatchEvent(new CustomEvent('favicon-changed', { detail: newUrl }));
			}

			toasts.add('success', $t('admin.settings_page.appearance.favicon_updated'));
		} catch (err) {
			console.error('Failed to save favicon:', err);
			toasts.add('error', $t('admin.settings_page.appearance.favicon_error'));
		} finally {
			faviconSaving = false;
		}
	}

	async function removeFavicon() {
		const confirmed = await confirm({
			title: $t('admin.settings_page.appearance.favicon_remove_title'),
			message: $t('admin.settings_page.appearance.favicon_remove_message'),
			confirmText: $t('admin.settings_page.appearance.favicon_remove_button'),
			danger: true
		});
		if (!confirmed) return;

		faviconSaving = true;
		try {
			const response = await fetch('/api/site-settings/favicon', {
				method: 'DELETE',
				headers: {
					Authorization: pb.authStore.token || ''
				}
			});

			if (!response.ok) {
				const result = await response.json();
				toasts.add('error', result.error || $t('admin.settings_page.appearance.favicon_remove_error'));
				return;
			}

			if (faviconBlobUrl) {
				URL.revokeObjectURL(faviconBlobUrl);
				faviconBlobUrl = null;
			}
			faviconFile = null;
			faviconUrl = null;

			// Notify layout to revert to default favicon
			window.dispatchEvent(new CustomEvent('favicon-changed', { detail: null }));

			toasts.add('success', $t('admin.settings_page.appearance.favicon_removed'));
		} catch (err) {
			console.error('Failed to remove favicon:', err);
			toasts.add('error', $t('admin.settings_page.appearance.favicon_remove_error'));
		} finally {
			faviconSaving = false;
		}
	}

	async function toggleHideDemoToggle() {
		siteSettingsSaving = true;
		try {
			const response = await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					hide_demo_toggle: !hideDemoToggle
				})
			});

			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || $t('admin.settings_page.general.demo_toggle_error'));
				return;
			}

			hideDemoToggle = result.hide_demo_toggle || false;
			toasts.add('success', hideDemoToggle ? $t('admin.settings_page.general.demo_hidden_toast') : $t('admin.settings_page.general.demo_visible_toast'));

			// Invalidate all data to trigger component updates (e.g., AdminHeader)
			await invalidateAll();
		} catch (err) {
			console.error('Failed to toggle demo setting:', err);
			toasts.add('error', $t('admin.settings_page.general.demo_toggle_error'));
		} finally {
			siteSettingsSaving = false;
		}
	}

	async function handleAddProvider() {
		try {
			// Build payload, excluding empty optional fields that might fail validation
			// Note: Don't send api_key_encrypted - it's a hidden field set by backend hook
			const payload: Record<string, unknown> = {
				name: newProvider.name,
				type: newProvider.type,
				api_key: newProvider.api_key,
				model: newProvider.model,
				is_active: newProvider.is_active,
				is_default: newProvider.is_default
			};
			// Only include base_url if it has a value (URLField rejects empty strings)
			if (newProvider.base_url) {
				payload.base_url = newProvider.base_url;
			}
			await pb.collection('ai_providers').create(payload);

			toasts.add('success', $t('admin.settings_page.integrations.ai_provider_added'));
			showAddForm = false;
			newProvider = {
				name: '',
				type: 'openai',
				api_key: '',
				base_url: '',
				model: '',
				is_active: true,
				is_default: false
			};
			await loadProviders();
		} catch (err: unknown) {
			console.error('Failed to add provider:', err);
			// Log full error for debugging
			console.error('[AI-PROVIDER] Full error object:', JSON.stringify(err, null, 2));
			// Extract detailed error from PocketBase ClientResponseError
			let message = $t('admin.settings_page.integrations.ai_add_error');
			if (err && typeof err === 'object' && 'data' in err) {
				const pbErr = err as { data?: { data?: Record<string, { message: string }>, message?: string } };
				console.error('[AI-PROVIDER] Error data:', pbErr.data);
				const fieldErrors = pbErr.data?.data;
				if (fieldErrors && Object.keys(fieldErrors).length > 0) {
					const details = Object.entries(fieldErrors)
						.map(([field, info]) => `${field}: ${info.message}`)
						.join(', ');
					message = `Validation error: ${details}`;
				} else if (pbErr.data?.message) {
					message = pbErr.data.message;
				}
			} else if (err instanceof Error) {
				message = err.message;
			}
			toasts.add('error', message);
		}
	}

	async function testConnection(id: string) {
		testing = id;
		try {
			const response = await fetch(`/api/ai/test/${id}`, {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token
				}
			});

			const result = await response.json();
			if (result.success) {
				toasts.add('success', $t('admin.settings_page.integrations.ai_test_success'));
			} else {
				toasts.add('error', $t('admin.settings_page.integrations.ai_test_failed', { values: { error: result.error } }));
			}
			await loadProviders();
		} catch (err) {
			toasts.add('error', $t('admin.settings_page.integrations.ai_test_error'));
		} finally {
			testing = null;
		}
	}

	async function deleteProvider(id: string) {
		const confirmed = await confirm({
			title: $t('admin.settings_page.integrations.ai_delete_title'),
			message: $t('admin.settings_page.integrations.ai_delete_message'),
			confirmText: $t('admin.settings_page.integrations.ai_delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('ai_providers').delete(id);
			toasts.add('success', $t('admin.settings_page.integrations.ai_provider_deleted'));
			await loadProviders();
		} catch (err) {
			toasts.add('error', $t('admin.settings_page.integrations.ai_add_error'));
		}
	}

	async function setDefault(id: string) {
		try {
			// Unset current defaults
			for (const p of providers) {
				if (p.is_default) {
					await pb.collection('ai_providers').update(p.id as string, { is_default: false });
				}
			}
			// Set new default
			await pb.collection('ai_providers').update(id, { is_default: true });
			toasts.add('success', $t('admin.settings_page.integrations.ai_default_updated'));
			await loadProviders();
		} catch (err) {
			toasts.add('error', $t('admin.settings_page.integrations.ai_add_error'));
		}
	}

	async function handleExport(format: 'json' | 'yaml') {
		exporting = format;
		try {
			const response = await fetch(`/api/export?format=${format}`, {
				headers: { Authorization: pb.authStore.token }
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || $t('admin.settings_page.general.export_error'));
			}

			// Get filename from Content-Disposition header or use default
			const disposition = response.headers.get('Content-Disposition');
			let filename = `facet-export.${format}`;
			if (disposition) {
				const match = disposition.match(/filename="?([^"]+)"?/);
				if (match) filename = match[1];
			}

			// Download the file
			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = filename;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);

			toasts.add('success', $t('admin.settings_page.general.export_downloaded', { values: { filename } }));
		} catch (err) {
			console.error('Export failed:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.settings_page.general.export_error'));
		} finally {
			exporting = null;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.settings_page.title')} {$t('admin.settings_page.page_title_suffix')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<PageHelp pageKey="settings">
		<p>{@html $t('admin.settings_page.help_text')}</p>
		<p>{$t('admin.settings_page.help_tip_1')}</p>
		<p>{@html $t('admin.settings_page.help_tip_2')}</p>
	</PageHelp>

	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{$t('admin.settings_page.title')}</h1>
	<p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
		{$t('admin.settings_page.subtitle')}
	</p>

	<!-- Account & Security section -->
	<div id="account" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.account.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.account.section_description')}</p>
		</div>

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
	</div>

	<!-- Appearance section -->
	<div id="appearance" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.appearance.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.appearance.section_description')}</p>
		</div>

		<!-- Accent Color -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.settings_page.appearance.accent_color_title')}</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				{$t('admin.settings_page.appearance.accent_color_description')}
			</p>

		{#if profile}
			<!-- Color Swatches -->
			<div class="mb-6">
				<span class="label mb-3 block">{$t('admin.settings_page.appearance.accent_color_label')}</span>
				<div class="flex flex-wrap gap-3">
					{#each ACCENT_COLOR_LIST as color}
						{@const colorInfo = ACCENT_COLORS[color]}
						<button
							type="button"
							class="relative group"
							onclick={() => saveAccentColor(color)}
							disabled={savingAppearance}
							title={colorInfo.label}
						>
							<div
								class="w-12 h-12 rounded-xl transition-all duration-200 ring-offset-2 ring-offset-white dark:ring-offset-gray-900
									{selectedAccentColor === color
									? 'ring-2 ring-gray-900 dark:ring-white scale-110'
									: 'hover:scale-105'}"
								style="background-color: {colorInfo.scale[500]}"
							>
								{#if selectedAccentColor === color}
									<div class="absolute inset-0 flex items-center justify-center">
										<svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
											<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
										</svg>
									</div>
								{/if}
							</div>
							<span class="block text-xs text-center mt-1 text-gray-600 dark:text-gray-400">
								{colorInfo.label}
							</span>
						</button>
					{/each}
				</div>
			</div>

			<!-- Preview Section -->
			<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
				<span class="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide font-medium mb-3 block">
					{$t('admin.settings_page.appearance.preview_label')}
				</span>
				<div class="flex flex-wrap items-center gap-4">
					<button
						type="button"
						class="px-4 py-2 rounded-lg font-medium text-white transition-colors"
						style="background-color: {ACCENT_COLORS[selectedAccentColor].scale[600]}"
					>
						{$t('admin.settings_page.appearance.primary_button')}
					</button>
					<button
						type="button"
						class="px-4 py-2 rounded-lg font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200"
					>
						{$t('admin.settings_page.appearance.secondary_button')}
					</button>
					<a
						href="#appearance"
						class="font-medium underline underline-offset-2"
						style="color: {ACCENT_COLORS[selectedAccentColor].scale[600]}"
					>
						{$t('admin.settings_page.appearance.link_example')}
					</a>
					<span
						class="px-2 py-1 rounded text-sm font-medium"
						style="background-color: {ACCENT_COLORS[selectedAccentColor].scale[100]}; color: {ACCENT_COLORS[selectedAccentColor].scale[700]}"
					>
						{$t('admin.settings_page.appearance.badge')}
					</span>
				</div>
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
					{ACCENT_COLORS[selectedAccentColor].description}
				</p>
			</div>
		{:else}
			<div class="text-gray-500 dark:text-gray-400 text-center py-4">
				<p>{$t('admin.settings_page.appearance.no_profile_message')}</p>
				<a href="/admin/homepage" class="text-primary-600 dark:text-primary-400 hover:underline mt-2 inline-block">
					{$t('admin.settings_page.appearance.go_to_homepage')}
				</a>
			</div>
		{/if}
		</div>

		<!-- Favicon -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">{$t('admin.settings_page.appearance.favicon_title')}</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				{$t('admin.settings_page.appearance.favicon_description')}
			</p>

			<div class="flex items-center gap-4">
				{#if faviconUrl}
					<div class="relative">
						<img
							src={faviconUrl}
							alt={$t('admin.settings_page.appearance.favicon_preview_alt')}
							class="w-16 h-16 object-contain rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-2"
						/>
						{#if !faviconFile}
							<!-- Only show remove button when not in pending upload state -->
							<button
								type="button"
								onclick={removeFavicon}
								disabled={faviconSaving || siteSettingsLoading}
								class="absolute -top-2 -right-2 p-1 bg-red-500 hover:bg-red-600 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-full shadow"
								aria-label={$t('admin.settings_page.appearance.favicon_remove_label')}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						{/if}
					</div>
				{/if}

				<div class="flex flex-col gap-2">
					<input
						type="file"
						accept="image/png,image/x-icon,image/vnd.microsoft.icon,image/svg+xml,image/gif"
						id="favicon"
						class="hidden"
						onchange={handleFaviconSelect}
						disabled={siteSettingsLoading || faviconSaving}
					/>
					<label
						for="favicon"
						class="btn btn-secondary btn-sm cursor-pointer {siteSettingsLoading || faviconSaving ? 'opacity-50 pointer-events-none' : ''}"
					>
						{faviconUrl ? $t('admin.settings_page.appearance.favicon_change_button') : $t('admin.settings_page.appearance.favicon_upload_button')}
					</label>
					{#if faviconFile}
						<div class="flex gap-2">
							<button
								type="button"
								class="btn btn-primary btn-sm"
								onclick={saveFavicon}
								disabled={faviconSaving}
							>
								{#if faviconSaving}
									<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" aria-hidden="true">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
									{$t('admin.settings_page.appearance.favicon_saving')}
								{:else}
									{$t('admin.settings_page.appearance.favicon_save')}
								{/if}
							</button>
							<button
								type="button"
								class="btn btn-secondary btn-sm"
								onclick={cancelFaviconUpload}
								disabled={faviconSaving}
							>
								{$t('admin.settings_page.appearance.favicon_cancel')}
							</button>
						</div>
					{/if}
				</div>
			</div>

			<p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
				{$t('admin.settings_page.appearance.favicon_supported_formats')}
			</p>
		</div>

		<!-- Custom CSS -->
		<div class="card p-6">
			<div class="flex items-start justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">{$t('admin.settings_page.appearance.custom_css_title')}</h2>
					<p class="text-gray-600 dark:text-gray-400 text-sm">
						{$t('admin.settings_page.appearance.custom_css_description')}
					</p>
				</div>
				<div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400 mt-1">
					<span>{$t('admin.settings_page.appearance.custom_css_char_count', { values: { count: customCSS.length } })}</span>
					<button class="btn btn-ghost btn-sm px-2" onclick={() => (showCSSHelp = true)}>
						{@html icon('info', 'w-4 h-4 mr-1')}
						<span class="align-middle">{$t('admin.settings_page.appearance.custom_css_selectors')}</span>
					</button>
				</div>
			</div>

			<div class="mt-4 space-y-3">
				<textarea
					class="input font-mono text-sm h-48"
					placeholder={$t('admin.settings_page.appearance.custom_css_placeholder')}
					bind:value={customCSS}
					disabled={siteSettingsLoading || siteSettingsSaving}
					maxlength="20000"
				></textarea>
				<div class="flex justify-end">
					<button class="btn btn-primary" onclick={saveSiteSettings} disabled={siteSettingsSaving || siteSettingsLoading}>
						{siteSettingsSaving ? $t('admin.settings_page.appearance.custom_css_saving') : $t('admin.settings_page.appearance.custom_css_save')}
					</button>
				</div>
			</div>
		</div>
	</div>

	{#if showCSSHelp}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
			<div class="bg-white dark:bg-gray-900 rounded-xl shadow-lg w-full max-w-2xl p-6 border border-gray-200 dark:border-gray-700">
				<div class="flex items-start justify-between gap-3 mb-4">
					<div>
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.appearance.css_help_title')}</h3>
						<p class="text-sm text-gray-600 dark:text-gray-400">
							{$t('admin.settings_page.appearance.css_help_description')}
						</p>
					</div>
					<button class="btn btn-ghost btn-sm px-2" onclick={() => (showCSSHelp = false)}>
						{@html icon('x', 'w-4 h-4')}
					</button>
				</div>

				<div class="space-y-3 text-sm text-gray-800 dark:text-gray-200">
					<div>
						<p class="font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.appearance.css_help_base')}</p>
						<ul class="list-disc list-inside text-gray-700 dark:text-gray-300">
							<li><code>:root</code> — accent palette vars <code>--color-primary-50..950</code></li>
							<li><code>body</code>, <code>main</code>, <code>header</code>, <code>footer</code></li>
							<li><code>h1</code>–<code>h4</code>, <code>p</code>, <code>a</code>, <code>ul</code>/<code>ol</code>, <code>li</code></li>
						</ul>
					</div>

					<div>
						<p class="font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.appearance.css_help_components')}</p>
						<ul class="list-disc list-inside text-gray-700 dark:text-gray-300">
							<li><code>.card</code> — section cards</li>
							<li><code>.section-title</code> — section headings</li>
							<li><code>.prose-custom</code> — rich text blocks (posts/talks)</li>
							<li><code>.btn</code>, <code>.btn-primary</code>, <code>.btn-secondary</code></li>
							<li><code>.input</code>, <code>.label</code></li>
						</ul>
					</div>

					<div class="bg-gray-50 dark:bg-gray-800/60 rounded-lg p-3 border border-gray-200 dark:border-gray-700 font-mono text-xs text-gray-800 dark:text-gray-200">
						<pre class="whitespace-pre-wrap">{`:root { --color-primary-500: #6366f1; } /* swap accent */
body { font-family: 'Inter', sans-serif; }
.card { border-radius: 1rem; box-shadow: 0 10px 30px rgba(0,0,0,0.08); }
.section-title { letter-spacing: 0.01em; }
.btn-primary { text-transform: uppercase; }`}</pre>
					</div>

					<p class="text-xs text-gray-500 dark:text-gray-400">
						{$t('admin.settings_page.appearance.css_help_tip')}
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- General section -->
	<div id="general" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.general.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.general.section_description')}</p>
		</div>

		<!-- Language Switcher -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">{$t('admin.settings_page.language.section_title')}</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				{$t('admin.settings_page.language.section_description')}
			</p>
			<LanguageSwitcher />
			<p class="text-xs text-gray-500 dark:text-gray-400 mt-4">
				{$t('admin.settings_page.language.more_coming')}
			</p>
		</div>

		<!-- Demo Toggle -->
		<div class="card p-6">
			<div class="flex items-center justify-between gap-4">
				<div>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">{$t('admin.settings_page.general.demo_toggle_title')}</h2>
					<p class="text-gray-600 dark:text-gray-400 text-sm">
						{$t('admin.settings_page.general.demo_toggle_description')}
					</p>
				</div>
				<button
					onclick={toggleHideDemoToggle}
					disabled={siteSettingsLoading || siteSettingsSaving}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed
						{hideDemoToggle ? 'bg-gray-300 dark:bg-gray-600' : 'bg-primary-600'}"
					role="switch"
					aria-checked={!hideDemoToggle}
					aria-label="Toggle demo mode visibility"
				>
					<span
						class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform
							{hideDemoToggle ? 'translate-x-1' : 'translate-x-6'}"
					></span>
				</button>
			</div>
			<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
				{hideDemoToggle ? $t('admin.settings_page.general.demo_toggle_hidden') : $t('admin.settings_page.general.demo_toggle_visible')}
			</p>
		</div>

		<!-- Data Export -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.settings_page.general.export_title')}</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				{$t('admin.settings_page.general.export_description')}
			</p>
			<div class="flex flex-wrap gap-3">
				<button
					class="btn btn-secondary inline-flex items-center gap-2"
					onclick={() => handleExport('yaml')}
					disabled={exporting !== null}
				>
					{#if exporting === 'yaml'}
						<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{:else}
						{@html icon('download')}
					{/if}
					{$t('admin.settings_page.general.export_yaml')}
				</button>
				<button
					class="btn btn-secondary inline-flex items-center gap-2"
					onclick={() => handleExport('json')}
					disabled={exporting !== null}
				>
					{#if exporting === 'json'}
						<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{:else}
						{@html icon('download')}
					{/if}
					{$t('admin.settings_page.general.export_json')}
				</button>
			</div>
			<p class="text-gray-500 dark:text-gray-500 text-xs mt-3">
				{$t('admin.settings_page.general.export_yaml_help')}
			</p>
		</div>
	</div>

	<!-- Analytics section -->
	<div id="analytics" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.analytics.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.analytics.section_description')}</p>
		</div>

		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">{$t('admin.settings_page.analytics.ga_title')}</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				{$t('admin.settings_page.analytics.ga_description')}
			</p>

			<div class="space-y-3">
				<label class="label" for="ga-id">{$t('admin.settings_page.analytics.ga_label')}</label>
				<input
					id="ga-id"
					class="input"
					placeholder={$t('admin.settings_page.analytics.ga_placeholder')}
					bind:value={gaMeasurementId}
					disabled={siteSettingsLoading || siteSettingsSaving}
					maxlength="100"
				/>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.settings_page.analytics.ga_help')}
				</p>
				<div class="flex justify-end">
					<button class="btn btn-primary" onclick={saveSiteSettings} disabled={siteSettingsSaving || siteSettingsLoading}>
						{siteSettingsSaving ? $t('admin.settings_page.analytics.ga_saving') : $t('admin.settings_page.analytics.ga_save')}
					</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Integrations section -->
	<div id="integrations" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.integrations.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.integrations.section_description')}</p>
		</div>

		<div class="card p-6">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.integrations.ai_providers_title')}</h2>
			<button class="btn btn-primary btn-sm" onclick={() => (showAddForm = !showAddForm)}>
				{showAddForm ? $t('admin.settings_page.integrations.ai_cancel') : $t('admin.settings_page.integrations.ai_add_provider')}
			</button>
		</div>

		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			{$t('admin.settings_page.integrations.ai_description')}
		</p>

		{#if showAddForm}
			<form onsubmit={preventDefault(handleAddProvider)} class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4 space-y-4">
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label for="name" class="label">{$t('admin.settings_page.integrations.ai_form_name_label')}</label>
						<input
							type="text"
							id="name"
							bind:value={newProvider.name}
							class="input"
							placeholder={$t('admin.settings_page.integrations.ai_form_name_placeholder')}
							required
						/>
					</div>
					<div>
						<label for="type" class="label">{$t('admin.settings_page.integrations.ai_form_type_label')}</label>
						<select id="type" bind:value={newProvider.type} class="input">
							<option value="openai">{$t('admin.settings_page.integrations.ai_form_type_openai')}</option>
							<option value="anthropic">{$t('admin.settings_page.integrations.ai_form_type_anthropic')}</option>
							<option value="ollama">{$t('admin.settings_page.integrations.ai_form_type_ollama')}</option>
							<option value="custom">{$t('admin.settings_page.integrations.ai_form_type_custom')}</option>
						</select>
					</div>
				</div>

				<div>
					<label for="api_key" class="label">
						{$t('admin.settings_page.integrations.ai_form_api_key_label')}
						{#if newProvider.type === 'ollama'}
							<span class="text-gray-500 font-normal">{$t('admin.settings_page.integrations.ai_form_api_key_not_required')}</span>
						{/if}
					</label>
					<input
						type="password"
						id="api_key"
						bind:value={newProvider.api_key}
						class="input"
						placeholder={newProvider.type === 'ollama' ? $t('admin.settings_page.integrations.ai_form_api_key_placeholder_optional') : $t('admin.settings_page.integrations.ai_form_api_key_placeholder')}
						required={newProvider.type !== 'ollama'}
					/>
				</div>

				{#if newProvider.type === 'ollama' || newProvider.type === 'custom'}
					<div>
						<label for="base_url" class="label">{$t('admin.settings_page.integrations.ai_form_base_url_label')}</label>
						<input
							type="url"
							id="base_url"
							bind:value={newProvider.base_url}
							class="input"
							placeholder={newProvider.type === 'ollama' ? 'http://localhost:11434' : 'https://api.example.com/v1'}
						/>
					</div>
				{/if}

				<div>
					<div class="flex items-center justify-between mb-1">
						<label for="model" class="label mb-0">{$t('admin.settings_page.integrations.ai_form_model_label')}</label>
						{#if newProvider.type !== 'custom' || newProvider.base_url}
							<button
								type="button"
								class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 flex items-center gap-1"
								onclick={fetchModels}
								disabled={fetchingModels || (newProvider.type !== 'ollama' && !newProvider.api_key)}
							>
								{#if fetchingModels}
									<svg class="animate-spin h-3 w-3" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
									{$t('admin.settings_page.integrations.ai_form_fetching')}
								{:else}
									<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
									</svg>
									{$t('admin.settings_page.integrations.ai_form_fetch_models')}
								{/if}
							</button>
						{/if}
					</div>
					{#if modelOptions.length > 0}
						<select id="model" bind:value={newProvider.model} class="input">
							{#each modelOptions as model}
								<option value={model}>{model}</option>
							{/each}
						</select>
						{#if fetchedModels.length > 0}
							<p class="text-xs text-green-600 dark:text-green-400 mt-1">
								{$t('admin.settings_page.integrations.ai_form_models_available', { values: { count: fetchedModels.length } })}
							</p>
						{/if}
					{:else}
						<input
							type="text"
							id="model"
							bind:value={newProvider.model}
							class="input"
							placeholder={$t('admin.settings_page.integrations.ai_form_model_placeholder')}
						/>
					{/if}
					{#if modelFetchError}
						<p class="text-xs text-amber-600 dark:text-amber-400 mt-1">
							{$t('admin.settings_page.integrations.ai_form_model_fetch_error', { values: { error: modelFetchError } })}
						</p>
					{/if}
				</div>

				<div class="flex items-center gap-4">
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_active} class="w-4 h-4" />
						<span>{$t('admin.settings_page.integrations.ai_form_active_label')}</span>
					</label>
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_default} class="w-4 h-4" />
						<span>{$t('admin.settings_page.integrations.ai_form_default_label')}</span>
					</label>
				</div>

				<button type="submit" class="btn btn-primary">{$t('admin.settings_page.integrations.ai_form_submit')}</button>
			</form>
		{/if}

		{#if loading}
			<div class="animate-pulse text-center py-4">{$t('admin.settings_page.integrations.ai_loading')}</div>
		{:else if providers.length === 0}
			<p class="text-gray-500 dark:text-gray-400 text-center py-8">
				{$t('admin.settings_page.integrations.ai_empty_message')}
			</p>
		{:else}
			<div class="space-y-3">
				{#each providers as provider}
					<div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
						<div class="flex items-center gap-3">
							<div
								class="w-10 h-10 rounded-lg flex items-center justify-center
								{provider.type === 'openai'
									? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
									: provider.type === 'anthropic'
										? 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300'
										: 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300'}"
							>
								{#if provider.type === 'openai'}
									<span class="text-lg font-bold">O</span>
								{:else if provider.type === 'anthropic'}
									<span class="text-lg font-bold">A</span>
								{:else if provider.type === 'ollama'}
									{@html icon('brain')}
								{:else}
									{@html icon('zap')}
								{/if}
							</div>
							<div>
								<div class="flex items-center gap-2">
									<span class="font-medium text-gray-900 dark:text-white">{provider.name}</span>
									{#if provider.is_default}
										<span class="px-2 py-0.5 text-xs bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300 rounded">
											{$t('admin.settings_page.integrations.ai_default_badge')}
										</span>
									{/if}
									{#if !provider.is_active}
										<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											{$t('admin.settings_page.integrations.ai_inactive_badge')}
										</span>
									{/if}
								</div>
								<p class="text-sm text-gray-500 dark:text-gray-400">
									{provider.type} • {provider.model || 'default model'}
									{#if provider.test_status}
										• Last test: {provider.test_status}
									{/if}
								</p>
							</div>
						</div>

						<div class="flex items-center gap-2">
							<button
								class="btn btn-sm btn-secondary"
								onclick={() => testConnection(String(provider.id))}
								disabled={testing === provider.id}
							>
								{#if testing === provider.id}
									<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
								{:else}
									{$t('admin.settings_page.integrations.ai_test_button')}
								{/if}
							</button>
							{#if !provider.is_default}
								<button
									class="btn btn-sm btn-secondary"
									onclick={() => setDefault(String(provider.id))}
								>
									{$t('admin.settings_page.integrations.ai_set_default_button')}
								</button>
							{/if}
							<button
								class="btn btn-danger-ghost btn-sm"
								onclick={() => deleteProvider(String(provider.id))}
							>
								{$t('admin.settings_page.integrations.ai_delete_button')}
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
		</div>
	</div>

	<!-- About / Changelog Section -->
	<div id="about" class="card p-6 mt-6">
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.settings_page.about.section_title')}</h2>
			<span class="px-3 py-1 text-sm font-mono bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg">
				{appVersion}
			</span>
		</div>

		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			{$t('admin.settings_page.about.description')}
		</p>

		<div class="flex flex-wrap gap-3 mb-6">
			<a
				href="https://github.com/jesposito/Facet/issues/new"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-secondary inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
				{$t('admin.settings_page.about.report_issue')}
			</a>
			<a
				href="https://github.com/jesposito/Facet"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-ghost inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
				{$t('admin.settings_page.about.view_github')}
			</a>
		</div>
	</div>
</div>

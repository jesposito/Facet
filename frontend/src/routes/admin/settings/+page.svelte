<script lang="ts">
	import { run, preventDefault } from 'svelte/legacy';

	import { onMount } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { pb, type Profile } from '$lib/pocketbase';
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

	// App version from Vite config
	const appVersion = __APP_VERSION__;

	// Changelog state
	interface ChangelogEntry {
		version: string;
		date: string;
		bugs: string[];
		features: string[];
		other: string[];
		prLinks: string;
	}
	let allChangelogEntries: ChangelogEntry[] = $state([]);
	let visibleChangelogCount = $state(3);
	let changelogLoading = $state(true);

	// Derived: visible changelog entries
	let changelogEntries = $derived(allChangelogEntries.slice(0, visibleChangelogCount));
	let hasMoreChangelog = $derived(visibleChangelogCount < allChangelogEntries.length);

	function showMoreChangelog() {
		visibleChangelogCount += 3;
	}

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
		await Promise.all([loadProviders(), loadProfile(), loadSiteSettings(), loadChangelog()]);
	});

	async function loadChangelog() {
		try {
			const response = await fetch('/CHANGELOG.md');
			if (!response.ok) {
				changelogLoading = false;
				return;
			}
			const text = await response.text();
			allChangelogEntries = parseChangelog(text);
		} catch (err) {
			console.error('Failed to load changelog:', err);
		} finally {
			changelogLoading = false;
		}
	}

	function parseChangelog(markdown: string): ChangelogEntry[] {
		const entries: ChangelogEntry[] = [];
		// Split by version headers (## vX.X.X - Date or ## Unreleased - Date)
		const sections = markdown.split(/^## /m).filter(s => s.trim());

		for (const section of sections) {
			if (section.startsWith('Changelog') || section.startsWith('#')) continue;

			const lines = section.split('\n');
			const headerLine = lines[0]?.trim() || '';

			// Parse header: "v1.0.0 - January 26, 2026" or "Unreleased - January 26, 2026"
			const headerMatch = headerLine.match(/^(v?\d+\.\d+\.\d+|Unreleased)\s*-\s*(.+)$/i);
			if (!headerMatch) continue;

			const entry: ChangelogEntry = {
				version: headerMatch[1],
				date: headerMatch[2].trim(),
				bugs: [],
				features: [],
				other: [],
				prLinks: ''
			};

			let currentSection = '';
			for (let i = 1; i < lines.length; i++) {
				const line = lines[i].trim();
				if (!line || line === '---') continue;

				if (line.startsWith('**Bugs Fixed:**')) {
					currentSection = 'bugs';
				} else if (line.startsWith('**New Features:**')) {
					currentSection = 'features';
				} else if (line.startsWith('**Other Changes:**')) {
					currentSection = 'other';
				} else if (line.startsWith('**Pull Requests:**')) {
					entry.prLinks = line.replace('**Pull Requests:**', '').trim();
					currentSection = '';
				} else if (line.startsWith('- ') && currentSection) {
					const item = line.substring(2).trim();
					if (currentSection === 'bugs') entry.bugs.push(item);
					else if (currentSection === 'features') entry.features.push(item);
					else if (currentSection === 'other') entry.other.push(item);
				}
			}

			if (entry.bugs.length > 0 || entry.features.length > 0 || entry.other.length > 0) {
				entries.push(entry);
			}
		}

		return entries;
	}

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
			toasts.add('success', 'Accent color updated');

			// Dispatch event to notify layout of color change
			window.dispatchEvent(new CustomEvent('accent-color-changed', { detail: color }));
		} catch (err) {
			console.error('Failed to save accent color:', err);
			toasts.add('error', 'Failed to update accent color');
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
			passwordForm.error = 'New password must be at least 8 characters';
			return;
		}

		if (passwordForm.newPassword !== passwordForm.confirmPassword) {
			passwordForm.error = 'Passwords do not match';
			return;
		}

		if (passwordForm.newPassword === passwordForm.currentPassword) {
			passwordForm.error = 'New password must be different from current password';
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
				throw new Error(data.error || 'Failed to change password');
			}

			// Success!
			passwordForm.success = true;
			passwordForm.currentPassword = '';
			passwordForm.newPassword = '';
			passwordForm.confirmPassword = '';
			toasts.add('success', 'Password changed successfully');
		} catch (err: any) {
			passwordForm.error = err.message || 'Failed to change password';
			toasts.add('error', 'Failed to change password');
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
				toasts.add('error', result.error || 'Failed to save settings');
				return;
			}

			customCSS = result.custom_css || '';
			gaMeasurementId = result.ga_measurement_id || '';
			toasts.add('success', 'Settings saved');
		} catch (err) {
			console.error('Failed to save site settings:', err);
			toasts.add('error', 'Failed to save settings');
		} finally {
			siteSettingsSaving = false;
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
				toasts.add('error', result.error || 'Failed to update setting');
				return;
			}

			hideDemoToggle = result.hide_demo_toggle || false;
			toasts.add('success', hideDemoToggle ? 'Demo toggle hidden' : 'Demo toggle visible');
			
			// Invalidate all data to trigger component updates (e.g., AdminHeader)
			await invalidateAll();
		} catch (err) {
			console.error('Failed to toggle demo setting:', err);
			toasts.add('error', 'Failed to update setting');
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

			toasts.add('success', 'AI provider added');
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
			let message = 'Failed to add provider';
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
				toasts.add('success', 'Connection successful!');
			} else {
				toasts.add('error', `Connection failed: ${result.error}`);
			}
			await loadProviders();
		} catch (err) {
			toasts.add('error', 'Connection test failed');
		} finally {
			testing = null;
		}
	}

	async function deleteProvider(id: string) {
		const confirmed = await confirm({
			title: 'Delete AI Provider',
			message: 'Are you sure you want to delete this AI provider? This action cannot be undone.',
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('ai_providers').delete(id);
			toasts.add('success', 'Provider deleted');
			await loadProviders();
		} catch (err) {
			toasts.add('error', 'Failed to delete provider');
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
			toasts.add('success', 'Default provider updated');
			await loadProviders();
		} catch (err) {
			toasts.add('error', 'Failed to update default');
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
				throw new Error(error.error || 'Export failed');
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

			toasts.add('success', `Export downloaded: ${filename}`);
		} catch (err) {
			console.error('Export failed:', err);
			toasts.add('error', err instanceof Error ? err.message : 'Export failed');
		} finally {
			exporting = null;
		}
	}
</script>

<svelte:head>
	<title>Settings | Facet</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<PageHelp pageKey="settings">
		<p><strong>Settings</strong> controls your site's appearance, AI providers, analytics, and data export.</p>
		<p>Configure AI providers to enable smart features like resume parsing, content rewriting, and GitHub project summaries. Add custom CSS to style your public profile.</p>
		<p><strong>Tip:</strong> Export your data regularly as JSON or YAML for backup. It includes everything.</p>
	</PageHelp>

	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Settings</h1>
	<p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
		Control what visitors see on your public site, enable analytics, and manage advanced options.
	</p>

	<!-- Account & Security section -->
	<div id="account" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Account & Security</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">Manage your account password.</p>
		</div>

		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Change Password</h2>

			<form onsubmit={preventDefault(changePassword)} class="space-y-4 max-w-md">
				<div>
					<label for="current-password-settings" class="label">Current Password</label>
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
					<label for="new-password-settings" class="label">New Password</label>
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
						Minimum 8 characters
					</p>
				</div>

				<div>
					<label for="confirm-password-settings" class="label">Confirm New Password</label>
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
						Password changed successfully!
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
						Changing Password...
					{:else}
						Change Password
					{/if}
				</button>
			</form>
		</div>
	</div>

	<!-- Appearance section -->
	<div id="appearance" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Appearance</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">Customize colors and styling for your profile.</p>
		</div>

		<!-- Accent Color -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Accent Color</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				Choose an accent color for buttons, links, and highlights across your profile.
			</p>

		{#if profile}
			<!-- Color Swatches -->
			<div class="mb-6">
				<span class="label mb-3 block">Accent Color</span>
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
					Preview
				</span>
				<div class="flex flex-wrap items-center gap-4">
					<button
						type="button"
						class="px-4 py-2 rounded-lg font-medium text-white transition-colors"
						style="background-color: {ACCENT_COLORS[selectedAccentColor].scale[600]}"
					>
						Primary Button
					</button>
					<button
						type="button"
						class="px-4 py-2 rounded-lg font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200"
					>
						Secondary
					</button>
					<a
						href="#appearance"
						class="font-medium underline underline-offset-2"
						style="color: {ACCENT_COLORS[selectedAccentColor].scale[600]}"
					>
						Link Example
					</a>
					<span
						class="px-2 py-1 rounded text-sm font-medium"
						style="background-color: {ACCENT_COLORS[selectedAccentColor].scale[100]}; color: {ACCENT_COLORS[selectedAccentColor].scale[700]}"
					>
						Badge
					</span>
				</div>
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
					{ACCENT_COLORS[selectedAccentColor].description}
				</p>
			</div>
		{:else}
			<div class="text-gray-500 dark:text-gray-400 text-center py-4">
				<p>Create a profile first to customize appearance.</p>
				<a href="/admin/homepage" class="text-primary-600 dark:text-primary-400 hover:underline mt-2 inline-block">
					Go to Homepage
				</a>
			</div>
		{/if}
		</div>

		<!-- Custom CSS -->
		<div class="card p-6">
			<div class="flex items-start justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Custom CSS</h2>
					<p class="text-gray-600 dark:text-gray-400 text-sm">
						Optional styles applied to public pages. Keep it minimal; you own the result.
					</p>
				</div>
				<div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400 mt-1">
					<span>{customCSS.length}/20000</span>
					<button class="btn btn-ghost btn-sm px-2" onclick={() => (showCSSHelp = true)}>
						{@html icon('info', 'w-4 h-4 mr-1')}
						<span class="align-middle">Selectors</span>
					</button>
				</div>
			</div>

			<div class="mt-4 space-y-3">
				<textarea
					class="input font-mono text-sm h-48"
					placeholder="/* Custom CSS (e.g., tweak fonts, spacing, colors) */"
					bind:value={customCSS}
					disabled={siteSettingsLoading || siteSettingsSaving}
					maxlength="20000"
				></textarea>
				<div class="flex justify-end">
					<button class="btn btn-primary" onclick={saveSiteSettings} disabled={siteSettingsSaving || siteSettingsLoading}>
						{siteSettingsSaving ? 'Saving...' : 'Save'}
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
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Public CSS hooks</h3>
						<p class="text-sm text-gray-600 dark:text-gray-400">
							These selectors match the public site (not the admin UI). Start small and test on mobile.
						</p>
					</div>
					<button class="btn btn-ghost btn-sm px-2" onclick={() => (showCSSHelp = false)}>
						{@html icon('x', 'w-4 h-4')}
					</button>
				</div>

				<div class="space-y-3 text-sm text-gray-800 dark:text-gray-200">
					<div>
						<p class="font-semibold text-gray-900 dark:text-white">Base</p>
						<ul class="list-disc list-inside text-gray-700 dark:text-gray-300">
							<li><code>:root</code> — accent palette vars <code>--color-primary-50..950</code></li>
							<li><code>body</code>, <code>main</code>, <code>header</code>, <code>footer</code></li>
							<li><code>h1</code>–<code>h4</code>, <code>p</code>, <code>a</code>, <code>ul</code>/<code>ol</code>, <code>li</code></li>
						</ul>
					</div>

					<div>
						<p class="font-semibold text-gray-900 dark:text-white">Components</p>
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
						Tip: Keep CSS small; avoid hiding structural elements that power accessibility and layout.
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- General section -->
	<div id="general" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">General</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">Admin preferences and data management.</p>
		</div>

		<!-- Demo Toggle -->
		<div class="card p-6">
			<div class="flex items-center justify-between gap-4">
				<div>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">Demo Mode Toggle</h2>
					<p class="text-gray-600 dark:text-gray-400 text-sm">
						Show or hide the demo toggle in the admin header. Useful once you've set up your profile.
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
				{hideDemoToggle ? 'Demo toggle is hidden' : 'Demo toggle is visible'}
			</p>
		</div>

		<!-- Data Export -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Data Export</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				Download your complete profile data for backup or migration. All your content (profile, experience,
				projects, education, skills, posts, talks, and views) is included.
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
					Download YAML
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
					Download JSON
				</button>
			</div>
			<p class="text-gray-500 dark:text-gray-500 text-xs mt-3">
				YAML is human-readable and easy to edit. JSON is useful for programmatic access.
			</p>
		</div>
	</div>

	<!-- Analytics section -->
	<div id="analytics" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Analytics</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">Track visitor activity on your public profile.</p>
		</div>

		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Google Analytics</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				GA4 measurement ID (public). Leave blank to disable tracking.
			</p>

			<div class="space-y-3">
				<label class="label" for="ga-id">GA4 Measurement ID</label>
				<input
					id="ga-id"
					class="input"
					placeholder="G-XXXXXXXXXX"
					bind:value={gaMeasurementId}
					disabled={siteSettingsLoading || siteSettingsSaving}
					maxlength="100"
				/>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					We only load GA on public pages when this is set. Do not use sensitive values.
				</p>
				<div class="flex justify-end">
					<button class="btn btn-primary" onclick={saveSiteSettings} disabled={siteSettingsSaving || siteSettingsLoading}>
						{siteSettingsSaving ? 'Saving...' : 'Save'}
					</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Integrations section -->
	<div id="integrations" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Integrations</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">Connect external services to enhance your profile.</p>
		</div>

		<div class="card p-6">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">AI Providers</h2>
			<button class="btn btn-primary btn-sm" onclick={() => (showAddForm = !showAddForm)}>
				{showAddForm ? 'Cancel' : '+ Add Provider'}
			</button>
		</div>

		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			Configure AI providers for enriching imported projects. Your API keys are encrypted at rest.
		</p>

		{#if showAddForm}
			<form onsubmit={preventDefault(handleAddProvider)} class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4 space-y-4">
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label for="name" class="label">Name</label>
						<input
							type="text"
							id="name"
							bind:value={newProvider.name}
							class="input"
							placeholder="My OpenAI"
							required
						/>
					</div>
					<div>
						<label for="type" class="label">Provider Type</label>
						<select id="type" bind:value={newProvider.type} class="input">
							<option value="openai">OpenAI</option>
							<option value="anthropic">Anthropic</option>
							<option value="ollama">Ollama</option>
							<option value="custom">Custom (OpenAI-compatible)</option>
						</select>
					</div>
				</div>

				<div>
					<label for="api_key" class="label">
						API Key
						{#if newProvider.type === 'ollama'}
							<span class="text-gray-500 font-normal">(not required for Ollama)</span>
						{/if}
					</label>
					<input
						type="password"
						id="api_key"
						bind:value={newProvider.api_key}
						class="input"
						placeholder={newProvider.type === 'ollama' ? 'Optional' : 'sk-...'}
						required={newProvider.type !== 'ollama'}
					/>
				</div>

				{#if newProvider.type === 'ollama' || newProvider.type === 'custom'}
					<div>
						<label for="base_url" class="label">Base URL</label>
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
						<label for="model" class="label mb-0">Model</label>
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
									Fetching...
								{:else}
									<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
									</svg>
									Fetch available models
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
								{fetchedModels.length} models available from API
							</p>
						{/if}
					{:else}
						<input
							type="text"
							id="model"
							bind:value={newProvider.model}
							class="input"
							placeholder="Enter model name"
						/>
					{/if}
					{#if modelFetchError}
						<p class="text-xs text-amber-600 dark:text-amber-400 mt-1">
							{modelFetchError} — using default list
						</p>
					{/if}
				</div>

				<div class="flex items-center gap-4">
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_active} class="w-4 h-4" />
						<span>Active</span>
					</label>
					<label class="flex items-center gap-2">
						<input type="checkbox" bind:checked={newProvider.is_default} class="w-4 h-4" />
						<span>Set as default</span>
					</label>
				</div>

				<button type="submit" class="btn btn-primary">Add Provider</button>
			</form>
		{/if}

		{#if loading}
			<div class="animate-pulse text-center py-4">Loading providers...</div>
		{:else if providers.length === 0}
			<p class="text-gray-500 dark:text-gray-400 text-center py-8">
				AI providers help generate project descriptions during import. Add one when you're ready.
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
											Default
										</span>
									{/if}
									{#if !provider.is_active}
										<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											Inactive
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
									Test
								{/if}
							</button>
							{#if !provider.is_default}
								<button
									class="btn btn-sm btn-secondary"
									onclick={() => setDefault(String(provider.id))}
								>
									Set Default
								</button>
							{/if}
							<button
								class="btn btn-sm btn-ghost text-red-600"
								onclick={() => deleteProvider(String(provider.id))}
							>
								Delete
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
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">About Facet</h2>
			<span class="px-3 py-1 text-sm font-mono bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg">
				{appVersion}
			</span>
		</div>

		<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
			Facet is an open-source professional portfolio platform. Report bugs or request features on GitHub.
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
				Report an Issue
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
				View on GitHub
			</a>
		</div>

		<!-- Recent Changes -->
		<div class="border-t border-gray-200 dark:border-gray-700 pt-4">
			<h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Recent Changes</h3>

			{#if changelogLoading}
				<div class="animate-pulse text-sm text-gray-500 dark:text-gray-400">Loading changelog...</div>
			{:else if changelogEntries.length === 0}
				<p class="text-sm text-gray-500 dark:text-gray-400">
					No changelog available yet. Check <a href="https://github.com/jesposito/Facet/releases" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">GitHub releases</a> for updates.
				</p>
			{:else}
				<div class="space-y-4">
					{#each changelogEntries as entry}
						<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
							<div class="flex items-center justify-between mb-2">
								<span class="font-medium text-gray-900 dark:text-white">{entry.version}</span>
								<span class="text-xs text-gray-500 dark:text-gray-400">{entry.date}</span>
							</div>

							{#if entry.bugs.length > 0}
								<div class="mb-2">
									<span class="text-xs font-semibold text-red-600 dark:text-red-400 uppercase">Bug Fixes</span>
									<ul class="mt-1 text-sm text-gray-600 dark:text-gray-400 space-y-0.5">
										{#each entry.bugs.slice(0, 3) as bug}
											<li class="flex items-start gap-2">
												<span class="text-red-500 mt-0.5">•</span>
												<span>{bug}</span>
											</li>
										{/each}
										{#if entry.bugs.length > 3}
											<li class="text-gray-500 dark:text-gray-500 text-xs">+{entry.bugs.length - 3} more</li>
										{/if}
									</ul>
								</div>
							{/if}

							{#if entry.features.length > 0}
								<div class="mb-2">
									<span class="text-xs font-semibold text-green-600 dark:text-green-400 uppercase">New Features</span>
									<ul class="mt-1 text-sm text-gray-600 dark:text-gray-400 space-y-0.5">
										{#each entry.features.slice(0, 3) as feature}
											<li class="flex items-start gap-2">
												<span class="text-green-500 mt-0.5">•</span>
												<span>{feature}</span>
											</li>
										{/each}
										{#if entry.features.length > 3}
											<li class="text-gray-500 dark:text-gray-500 text-xs">+{entry.features.length - 3} more</li>
										{/if}
									</ul>
								</div>
							{/if}

							{#if entry.other.length > 0}
								<div class="mb-2">
									<span class="text-xs font-semibold text-blue-600 dark:text-blue-400 uppercase">Other</span>
									<ul class="mt-1 text-sm text-gray-600 dark:text-gray-400 space-y-0.5">
										{#each entry.other.slice(0, 2) as item}
											<li class="flex items-start gap-2">
												<span class="text-blue-500 mt-0.5">•</span>
												<span>{item}</span>
											</li>
										{/each}
										{#if entry.other.length > 2}
											<li class="text-gray-500 dark:text-gray-500 text-xs">+{entry.other.length - 2} more</li>
										{/if}
									</ul>
								</div>
							{/if}
						</div>
					{/each}
				</div>

				<div class="mt-4 flex flex-col items-center gap-2">
					{#if hasMoreChangelog}
						<button
							onclick={showMoreChangelog}
							class="btn btn-ghost text-sm"
						>
							Show more ({allChangelogEntries.length - visibleChangelogCount} remaining)
						</button>
					{/if}
					<a
						href="https://github.com/jesposito/Facet/blob/main/CHANGELOG.md"
						target="_blank"
						rel="noopener noreferrer"
						class="text-sm text-primary-600 dark:text-primary-400 hover:underline"
					>
						View full changelog on GitHub →
					</a>
				</div>
			{/if}
		</div>
	</div>
</div>

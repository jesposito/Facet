<script lang="ts">
	import { run } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { pb, type Profile } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';
	import {
		DEFAULT_ACCENT_COLOR,
		type AccentColor
	} from '$lib/colors';
	import {
		FONT_PACKS,
		FONT_PACK_LIST,
		DEFAULT_FONT_PACK,
		type FontPack
	} from '$lib/fonts';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import LanguageSwitcher from '$components/admin/LanguageSwitcher.svelte';
import AccentPicker from '$components/admin/AccentPicker.svelte';
	import AccordionSection from '$components/admin/forms/AccordionSection.svelte';

	// Site settings (custom CSS, analytics)
	let siteSettingsLoading = $state(true);
	let siteSettingsSaving = $state(false);
	let customCSS = $state('');
	let gaMeasurementId = $state('');
	let showCSSHelp = $state(false);

	// Favicon state
	let faviconUrl: string | null = $state(null);
	let faviconFile: File | null = $state(null);
	let faviconBlobUrl: string | null = $state(null);
	let faviconSaving = $state(false);

	// SMTP/Email state
	let smtpLoading = $state(true);
	let smtpSaving = $state(false);
	let smtpTesting = $state(false);
	let smtpSettings = $state({
		enabled: false,
		host: '',
		port: 587,
		username: '',
		password: '',
		password_set: false,
		auth_method: 'PLAIN',
		tls: true,
		sender_name: '',
		sender_address: ''
	});
	let smtpTestRecipient = $state('');

	// Export state
	let exporting: string | null = $state(null);

	// Appearance state
	let profile: Profile | null = $state(null);
	let selectedAccentColor: AccentColor = $state(DEFAULT_ACCENT_COLOR);
	let customHexColor: string = $state('');
	let savingAppearance = $state(false);
	let selectedFontPack: FontPack = $state(DEFAULT_FONT_PACK);
	let savingFontPack = $state(false);
	let selectedHeroLayout = $state('standard');
	let savingHeroLayout = $state(false);

	const HERO_LAYOUTS = [
		{ id: 'standard', label: 'Image & Gradient', description: 'Background image with gradient overlay' },
		{ id: 'centered', label: 'Centered', description: 'Bold headline centered, no image needed' },
		{ id: 'split', label: 'Split', description: 'Text left, image right' },
		{ id: 'minimal', label: 'Minimal', description: 'Large typography, maximum whitespace' },
		{ id: 'stacked', label: 'Stacked', description: 'Headline above, full-width image below' }
	];

	onMount(async () => {
		await Promise.all([loadProfile(), loadSiteSettings(), loadSMTPSettings()]);
	});

	async function loadProfile() {
		try {
			const records = await collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				profile = records.items[0] as unknown as Profile;
				selectedAccentColor = (profile.accent_color as AccentColor) || DEFAULT_ACCENT_COLOR;
				customHexColor = (profile as unknown as { custom_hex_color?: string }).custom_hex_color || '';
				selectedFontPack = (profile.font_pack as FontPack) || DEFAULT_FONT_PACK;
				selectedHeroLayout = (profile as any).hero_layout || 'standard';
			}
		} catch (err) {
			console.error('Failed to load profile:', err);
		}
	}

	async function saveAccentColor(color: AccentColor) {
		if (!profile) return;

		savingAppearance = true;
		try {
			// Clearing custom hex when a curated swatch is chosen keeps storage in sync.
			await collection('profile').update(profile.id, {
				accent_color: color,
				custom_hex_color: ''
			});
			selectedAccentColor = color;
			customHexColor = '';
			profile.accent_color = color;
			(profile as unknown as { custom_hex_color?: string }).custom_hex_color = '';
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

	async function saveCustomHexColor(hex: string) {
		if (!profile) return;
		savingAppearance = true;
		try {
			await collection('profile').update(profile.id, {
				custom_hex_color: hex
			});
			customHexColor = hex;
			(profile as unknown as { custom_hex_color?: string }).custom_hex_color = hex;
			toasts.add('success', $t('admin.settings_page.appearance.accent_color_updated'));
			// Notify layout to re-apply
			window.dispatchEvent(new CustomEvent('accent-color-changed', { detail: hex }));
		} catch (err) {
			console.error('Failed to save custom hex color:', err);
			toasts.add('error', $t('admin.settings_page.appearance.accent_color_error'));
		} finally {
			savingAppearance = false;
		}
	}

	async function saveFontPack(pack: FontPack) {
		if (!profile) return;
		if (pack === selectedFontPack) return;

		savingFontPack = true;
		try {
			await collection('profile').update(profile.id, {
				font_pack: pack
			});
			selectedFontPack = pack;
			profile.font_pack = pack;
			toasts.add('success', 'Typography updated');
			window.dispatchEvent(new CustomEvent('font-pack-changed', { detail: pack }));
		} catch (err) {
			console.error('Failed to save font pack:', err);
			toasts.add('error', 'Failed to update typography');
		} finally {
			savingFontPack = false;
		}
	}

	async function saveHeroLayout(layout: string) {
		if (!profile) return;
		if (layout === selectedHeroLayout) return;

		savingHeroLayout = true;
		try {
			await collection('profile').update(profile.id, {
				hero_layout: layout
			});
			selectedHeroLayout = layout;
			(profile as any).hero_layout = layout;
			toasts.add('success', 'Hero layout updated');
		} catch (err) {
			console.error('Failed to save hero layout:', err);
			toasts.add('error', 'Failed to update hero layout');
		} finally {
			savingHeroLayout = false;
		}
	}

	async function loadSiteSettings() {
		try {
			// Add cache buster to prevent browser from caching stale settings
			const response = await fetch(`/api/site-settings?_=${Date.now()}`);
			if (response.ok) {
				const data = await response.json();
				customCSS = data.custom_css || '';
				gaMeasurementId = data.ga_measurement_id || '';
				faviconUrl = data.favicon ? `${data.favicon}?v=${Date.now()}` : null;
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
				toasts.add('success', $t('admin.settings_page.appearance.favicon_updated'));
			} else {
				// Backend returned success but no favicon URL - something went wrong
				console.error('Favicon upload succeeded but no URL returned:', result);
				toasts.add('error', $t('admin.settings_page.appearance.favicon_error'));
			}
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

	async function loadSMTPSettings() {
		try {
			const response = await fetch('/api/smtp-settings', {
				headers: { Authorization: pb.authStore.token }
			});
			if (response.ok) {
				const data = await response.json();
				smtpSettings = {
					enabled: data.enabled || false,
					host: data.host || '',
					port: data.port || 587,
					username: data.username || '',
					password: '',
					password_set: data.password_set || false,
					auth_method: data.auth_method || 'PLAIN',
					tls: data.tls !== undefined ? data.tls : true,
					sender_name: data.sender_name || '',
					sender_address: data.sender_address || ''
				};
			}
		} catch (err) {
			console.error('Failed to load SMTP settings:', err);
		} finally {
			smtpLoading = false;
		}
	}

	async function saveSMTPSettings() {
		smtpSaving = true;
		try {
			const payload: Record<string, unknown> = {
				enabled: smtpSettings.enabled,
				host: smtpSettings.host,
				port: smtpSettings.port,
				username: smtpSettings.username,
				auth_method: smtpSettings.auth_method,
				tls: smtpSettings.tls,
				sender_name: smtpSettings.sender_name,
				sender_address: smtpSettings.sender_address
			};
			// Only send password if the user typed something
			if (smtpSettings.password) {
				payload.password = smtpSettings.password;
			}

			const response = await fetch('/api/smtp-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify(payload)
			});

			const result = await response.json();
			if (!response.ok) {
				toasts.add('error', result.error || $t('admin.settings_page.email.save_error_toast'));
				return;
			}

			smtpSettings.password = '';
			smtpSettings.password_set = result.password_set;
			toasts.add('success', $t('admin.settings_page.email.saved_toast'));
		} catch (err) {
			console.error('Failed to save SMTP settings:', err);
			toasts.add('error', $t('admin.settings_page.email.save_error_toast'));
		} finally {
			smtpSaving = false;
		}
	}

	async function sendTestEmail() {
		smtpTesting = true;
		try {
			const response = await fetch('/api/smtp-settings/test', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({ recipient: smtpTestRecipient })
			});

			const result = await response.json();
			if (result.success) {
				toasts.add('success', $t('admin.settings_page.email.test_success_toast'));
			} else {
				toasts.add('error', $t('admin.settings_page.email.test_error_toast', { values: { error: result.error } }));
			}
		} catch (err) {
			console.error('Failed to send test email:', err);
			toasts.add('error', $t('admin.settings_page.email.test_error_toast', { values: { error: 'Network error' } }));
		} finally {
			smtpTesting = false;
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
	<title>{$t('admin.settings_page.title')} - {$t('admin.settings_page.page_title_suffix')}</title>
	<!-- Load all font pack fonts for typography preview -->
	{#each FONT_PACK_LIST as pack}
		{@const packInfo = FONT_PACKS[pack]}
		<link href={packInfo.googleFontsUrl} rel="stylesheet" />
	{/each}
</svelte:head>

<div class="max-w-4xl mx-auto">
	<!-- Appearance section -->
	<div id="appearance" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.appearance.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.appearance.section_description')}</p>
		</div>

		<!-- Accent Color (collapsible) -->
		<AccordionSection
			title={$t('admin.settings_page.appearance.accent_color_title')}
			description={$t('admin.settings_page.appearance.accent_color_description')}
			iconPath="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"
		>
			{#if profile}
				<AccentPicker
					value={selectedAccentColor}
					customHex={customHexColor}
					onchange={(color) => saveAccentColor(color)}
					onhexchange={(hex) => saveCustomHexColor(hex)}
				/>
			{:else}
				<div class="text-gray-500 dark:text-gray-400 text-center py-4">
					<p>{$t('admin.settings_page.appearance.no_profile_message')}</p>
					<a href="/admin/homepage" class="text-primary-600 dark:text-primary-400 hover:underline mt-2 inline-block">
						{$t('admin.settings_page.appearance.go_to_homepage')}
					</a>
				</div>
			{/if}
		</AccordionSection>

		<!-- Typography -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Typography</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				Choose a font combination for your profile. Each pack includes fonts for headings, body text, and code blocks.
			</p>

		{#if profile}
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				{#each FONT_PACK_LIST as pack}
					{@const packInfo = FONT_PACKS[pack]}
					{@const isSelected = selectedFontPack === pack}
					<button
						type="button"
						class="relative text-left p-4 rounded-xl border-2 transition-all duration-200
							{isSelected
							? 'border-primary-500 bg-primary-50 dark:bg-primary-950/20'
							: 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
						onclick={() => saveFontPack(pack)}
						disabled={savingFontPack}
					>
						{#if isSelected}
							<div class="absolute top-3 right-3">
								<svg class="w-5 h-5 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
								</svg>
							</div>
						{/if}
						<span class="block text-sm font-semibold text-gray-900 dark:text-white mb-1">
							{packInfo.label}
						</span>
						<span class="block text-xs text-gray-500 dark:text-gray-400 mb-3">
							{packInfo.description}
						</span>
						<div class="space-y-1 border-t border-gray-100 dark:border-gray-700 pt-3">
							<p class="text-base font-semibold text-gray-800 dark:text-gray-200" style="font-family: '{packInfo.heading}', serif">
								Heading Text
							</p>
							<p class="text-sm text-gray-600 dark:text-gray-400" style="font-family: '{packInfo.body}', sans-serif">
								Body text looks like this sentence.
							</p>
							<p class="text-xs text-gray-500 dark:text-gray-300" style="font-family: '{packInfo.code}', monospace">
								const code = "example";
							</p>
						</div>
					</button>
				{/each}
			</div>

			{#if selectedFontPack !== DEFAULT_FONT_PACK}
				<button
					type="button"
					class="mt-3 text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
					onclick={() => saveFontPack(DEFAULT_FONT_PACK)}
					disabled={savingFontPack}
				>
					Reset to default
				</button>
			{/if}
		{:else}
			<div class="text-gray-500 dark:text-gray-400 text-center py-4">
				<p>Set up your profile first to customize typography.</p>
			</div>
		{/if}
		</div>

		<!-- Hero Layout -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Hero Layout</h2>
			<p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
				Choose the default hero section layout for your profile. Views can override this with their own layout.
			</p>

		{#if profile}
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				{#each HERO_LAYOUTS as layout}
					{@const isSelected = selectedHeroLayout === layout.id}
					<button
						type="button"
						class="relative text-left p-4 rounded-xl border-2 transition-all duration-200
							{isSelected
							? 'border-primary-500 bg-primary-50 dark:bg-primary-950/20'
							: 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
						onclick={() => saveHeroLayout(layout.id)}
						disabled={savingHeroLayout}
					>
						{#if isSelected}
							<div class="absolute top-3 right-3">
								<svg class="w-5 h-5 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
								</svg>
							</div>
						{/if}
						<span class="block text-sm font-semibold text-gray-900 dark:text-white mb-1">
							{layout.label}
						</span>
						<span class="block text-xs text-gray-500 dark:text-gray-400">
							{layout.description}
						</span>
					</button>
				{/each}
			</div>
		{:else}
			<div class="text-gray-500 dark:text-gray-400 text-center py-4">
				<p>Set up your profile first to customize the hero layout.</p>
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
			<p class="text-gray-500 dark:text-gray-300 text-xs mt-3">
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

	<!-- Email / SMTP section -->
	<div id="email" class="space-y-4 mb-8">
		<div>
			<p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{$t('admin.settings_page.email.section_title')}</p>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.settings_page.email.section_description')}</p>
		</div>

		<div class="card p-6">
			{#if smtpLoading}
				<div class="animate-pulse text-center py-4 text-gray-500 dark:text-gray-400">{$t('admin.settings_page.email.loading')}</div>
			{:else}
				<div class="space-y-4">
					<!-- Enable toggle -->
					<label class="flex items-center gap-3">
						<input type="checkbox" bind:checked={smtpSettings.enabled} class="w-4 h-4" disabled={smtpSaving} />
						<span class="font-medium text-gray-900 dark:text-white">{$t('admin.settings_page.email.enable_label')}</span>
					</label>

					{#if smtpSettings.enabled}
						<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
							<div>
								<label for="smtp-host" class="label">{$t('admin.settings_page.email.host_label')}</label>
								<input
									type="text"
									id="smtp-host"
									bind:value={smtpSettings.host}
									class="input"
									placeholder={$t('admin.settings_page.email.host_placeholder')}
									disabled={smtpSaving}
								/>
							</div>
							<div>
								<label for="smtp-port" class="label">{$t('admin.settings_page.email.port_label')}</label>
								<input
									type="number"
									id="smtp-port"
									bind:value={smtpSettings.port}
									class="input"
									placeholder="587"
									min="1"
									max="65535"
									disabled={smtpSaving}
								/>
							</div>
						</div>

						<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
							<div>
								<label for="smtp-username" class="label">{$t('admin.settings_page.email.username_label')}</label>
								<input
									type="text"
									id="smtp-username"
									bind:value={smtpSettings.username}
									class="input"
									placeholder={$t('admin.settings_page.email.username_placeholder')}
									disabled={smtpSaving}
								/>
							</div>
							<div>
								<label for="smtp-password" class="label">{$t('admin.settings_page.email.password_label')}</label>
								<input
									type="password"
									id="smtp-password"
									bind:value={smtpSettings.password}
									class="input"
									placeholder={smtpSettings.password_set ? $t('admin.settings_page.email.password_placeholder_set') : $t('admin.settings_page.email.password_placeholder')}
									disabled={smtpSaving}
								/>
							</div>
						</div>

						<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
							<div>
								<label for="smtp-auth" class="label">{$t('admin.settings_page.email.auth_method_label')}</label>
								<select id="smtp-auth" bind:value={smtpSettings.auth_method} class="input" disabled={smtpSaving}>
									<option value="PLAIN">PLAIN</option>
									<option value="LOGIN">LOGIN</option>
								</select>
							</div>
							<div class="flex items-end pb-1">
								<label class="flex items-center gap-2">
									<input type="checkbox" bind:checked={smtpSettings.tls} class="w-4 h-4" disabled={smtpSaving} />
									<span>{$t('admin.settings_page.email.tls_label')}</span>
								</label>
							</div>
						</div>

						<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
							<div>
								<label for="smtp-sender-name" class="label">{$t('admin.settings_page.email.sender_name_label')}</label>
								<input
									type="text"
									id="smtp-sender-name"
									bind:value={smtpSettings.sender_name}
									class="input"
									placeholder={$t('admin.settings_page.email.sender_name_placeholder')}
									disabled={smtpSaving}
								/>
							</div>
							<div>
								<label for="smtp-sender-address" class="label">{$t('admin.settings_page.email.sender_address_label')}</label>
								<input
									type="email"
									id="smtp-sender-address"
									bind:value={smtpSettings.sender_address}
									class="input"
									placeholder={$t('admin.settings_page.email.sender_address_placeholder')}
									disabled={smtpSaving}
								/>
							</div>
						</div>

						<!-- Save + Test -->
						<div class="flex flex-wrap items-end gap-4 pt-2 border-t border-gray-200 dark:border-gray-700">
							<button class="btn btn-primary" onclick={saveSMTPSettings} disabled={smtpSaving}>
								{smtpSaving ? $t('admin.settings_page.email.saving_button') : $t('admin.settings_page.email.save_button')}
							</button>

							<div class="flex items-end gap-2">
								<div>
									<label for="smtp-test-recipient" class="label text-xs">{$t('admin.settings_page.email.test_recipient_label')}</label>
									<input
										type="email"
										id="smtp-test-recipient"
										bind:value={smtpTestRecipient}
										class="input text-sm"
										placeholder={$t('admin.settings_page.email.test_recipient_placeholder')}
										disabled={smtpTesting}
									/>
								</div>
								<button
									class="btn btn-secondary"
									onclick={sendTestEmail}
									disabled={smtpTesting || !smtpSettings.enabled}
								>
									{#if smtpTesting}
										<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
											<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
											<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
										</svg>
										{$t('admin.settings_page.email.testing_button')}
									{:else}
										{$t('admin.settings_page.email.test_button')}
									{/if}
								</button>
							</div>
						</div>
					{:else}
						<p class="text-sm text-gray-500 dark:text-gray-400 mt-2">
							{$t('admin.settings_page.email.not_configured_warning')}
						</p>
						<button class="btn btn-primary mt-2" onclick={saveSMTPSettings} disabled={smtpSaving}>
							{smtpSaving ? $t('admin.settings_page.email.saving_button') : $t('admin.settings_page.email.save_button')}
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>

</div>

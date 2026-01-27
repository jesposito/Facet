<script lang="ts">
	import { t } from 'svelte-i18n';
	import { locale, setLocale, SUPPORTED_LOCALES, LOCALE_NAMES } from '$lib/i18n';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';

	let currentLocale = $derived($locale || 'en');
	let saving = $state(false);

	async function handleChange(event: Event) {
		const select = event.target as HTMLSelectElement;
		const newLocale = select.value;

		// Update UI immediately
		setLocale(newLocale);

		// Save to server
		saving = true;
		try {
			const response = await fetch('/api/site-settings', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify({
					default_locale: newLocale
				})
			});

			if (!response.ok) {
				throw new Error('Failed to save locale');
			}

			toasts.success($t('admin.settings_page.language.saved'));
		} catch (err) {
			console.error('Failed to save locale:', err);
			toasts.error($t('admin.settings_page.language.save_error'));
		} finally {
			saving = false;
		}
	}
</script>

<div class="space-y-2">
	<label for="language-select" class="label">
		{$t('admin.settings_page.language.label')}
	</label>
	<div class="flex items-center gap-2">
		<select
			id="language-select"
			class="input max-w-xs"
			value={currentLocale}
			onchange={handleChange}
			disabled={saving}
		>
			{#each SUPPORTED_LOCALES as loc}
				<option value={loc}>
					{LOCALE_NAMES[loc] || loc}
				</option>
			{/each}
		</select>
		{#if saving}
			<span class="text-sm text-gray-500">{$t('admin.settings_page.language.saving')}</span>
		{/if}
	</div>
	<p class="text-xs text-gray-500 dark:text-gray-400">
		{$t('admin.settings_page.language.help')}
	</p>
</div>

<script lang="ts">
	import { t } from 'svelte-i18n';
	import { locale, setLocale, SUPPORTED_LOCALES, LOCALE_NAMES } from '$lib/i18n';

	let currentLocale = $derived($locale || 'en');

	function handleChange(event: Event) {
		const select = event.target as HTMLSelectElement;
		setLocale(select.value);
	}
</script>

<div class="space-y-2">
	<label for="language-select" class="label">
		{$t('admin.settings_page.language.label')}
	</label>
	<select
		id="language-select"
		class="input max-w-xs"
		value={currentLocale}
		onchange={handleChange}
	>
		{#each SUPPORTED_LOCALES as loc}
			<option value={loc}>
				{LOCALE_NAMES[loc] || loc}
			</option>
		{/each}
	</select>
	<p class="text-xs text-gray-500 dark:text-gray-400">
		{$t('admin.settings_page.language.help')}
	</p>
</div>

<script lang="ts">
	import { setupWizard } from '$lib/stores/setupWizard';
	import { t } from 'svelte-i18n';

	function handleNameChange(e: Event) {
		const target = e.target as HTMLInputElement;
		setupWizard.updateProfile({ name: target.value });
	}

	function handleHeadlineChange(e: Event) {
		const target = e.target as HTMLInputElement;
		setupWizard.updateProfile({ headline: target.value });
	}

	function handleSummaryChange(e: Event) {
		const target = e.target as HTMLTextAreaElement;
		setupWizard.updateProfile({ summary: target.value });
	}
</script>

<div class="space-y-6">
	<div class="text-center">
		<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
			{$t('admin.wizard.step1_title')}
		</h2>
		<p class="text-gray-600 dark:text-gray-400">
			{$t('admin.wizard.step1_subtitle')}
		</p>
	</div>

	<div class="space-y-4">
		<div>
			<label for="wizard-name" class="label">{$t('admin.wizard.full_name')} {$t('admin.wizard.required_field')}</label>
			<input
				type="text"
				id="wizard-name"
				class="input"
				placeholder={$t('admin.wizard.full_name_placeholder')}
				value={$setupWizard.profile.name || ''}
				oninput={handleNameChange}
			/>
		</div>

		<div>
			<label for="wizard-headline" class="label">{$t('admin.wizard.professional_title')} {$t('admin.wizard.required_field')}</label>
			<input
				type="text"
				id="wizard-headline"
				class="input"
				placeholder={$t('admin.wizard.professional_title_placeholder')}
				value={$setupWizard.profile.headline || ''}
				oninput={handleHeadlineChange}
			/>
			<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
				{$t('admin.wizard.professional_title_help')}
			</p>
		</div>

		<div>
			<label for="wizard-summary" class="label">{$t('admin.wizard.brief_summary')}</label>
			<textarea
				id="wizard-summary"
				class="input min-h-[100px]"
				placeholder={$t('admin.wizard.brief_summary_placeholder')}
				value={$setupWizard.profile.summary || ''}
				oninput={handleSummaryChange}
				rows={3}
			></textarea>
			<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
				{$t('admin.wizard.brief_summary_help')}
			</p>
		</div>
	</div>
</div>

<script lang="ts">
	import { setupWizard, viewTemplates } from '$lib/stores/setupWizard';
	import { t } from 'svelte-i18n';

	const selectedTemplate = $derived(
		viewTemplates.find(t => t.id === $setupWizard.selectedTemplate)
	);
</script>

<div class="space-y-6">
	<div class="text-center">
		<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
			{$t('admin.wizard.step3_title')}
		</h2>
		<p class="text-gray-600 dark:text-gray-400">
			{$t('admin.wizard.step3_subtitle')}
		</p>
	</div>
	
	<div class="border-2 border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-900/20 rounded-lg p-6">
		<div class="flex items-start gap-4">
			<div class="bg-gradient-to-br from-primary-500 to-primary-600 w-14 h-14 rounded-full flex items-center justify-center text-white text-xl font-semibold flex-shrink-0">
				{$setupWizard.profile.name?.[0]?.toUpperCase() || '?'}
			</div>
			<div class="min-w-0 flex-1">
				<h3 class="font-semibold text-lg text-gray-900 dark:text-white">
					{$setupWizard.profile.name || $t('admin.wizard.your_name_placeholder')}
				</h3>
				<p class="text-gray-600 dark:text-gray-400">
					{$setupWizard.profile.headline || $t('admin.wizard.your_title_placeholder')}
				</p>
				{#if $setupWizard.profile.summary}
					<p class="text-sm text-gray-500 dark:text-gray-400 mt-2 line-clamp-2">
						{$setupWizard.profile.summary}
					</p>
				{/if}
			</div>
		</div>
		
		{#if selectedTemplate}
			<div class="mt-4 pt-4 border-t border-green-200 dark:border-green-700">
				<div class="flex items-center gap-3">
					<span class="text-2xl">{selectedTemplate.icon}</span>
					<div>
						<p class="font-medium text-gray-900 dark:text-white">
							{$setupWizard.newView.name}
						</p>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							/{$setupWizard.newView.slug} &middot; {$setupWizard.newView.visibility}
						</p>
					</div>
				</div>
			</div>
		{/if}
	</div>
	
	<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
		<h4 class="font-medium text-gray-900 dark:text-white mb-3">{$t('admin.wizard.next_steps_title')}</h4>
		<div class="space-y-2 text-sm text-gray-600 dark:text-gray-400">
			<div class="flex items-center gap-2">
				<span class="text-blue-500">📁</span>
				<span>{$t('admin.wizard.next_step_projects')}</span>
			</div>
			<div class="flex items-center gap-2">
				<span class="text-purple-500">🔗</span>
				<span>{$t('admin.wizard.next_step_share')}</span>
			</div>
			<div class="flex items-center gap-2">
				<span class="text-orange-500">⚡</span>
				<span>{$t('admin.wizard.next_step_import')}</span>
			</div>
		</div>
	</div>
	
	<div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
		<div class="flex items-start gap-3">
			<svg class="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<div class="text-sm">
				<p class="font-medium text-blue-900 dark:text-blue-100 mb-1">{$t('admin.wizard.ai_features_title')}</p>
				<p class="text-blue-700 dark:text-blue-300 mb-2">
					{@html $t('admin.wizard.ai_features_description', {
						values: {
							openai: '<a href="https://platform.openai.com/api-keys" target="_blank" rel="noopener noreferrer" class="underline hover:text-blue-900 dark:hover:text-blue-100">OpenAI</a>',
							anthropic: '<a href="https://console.anthropic.com/settings/keys" target="_blank" rel="noopener noreferrer" class="underline hover:text-blue-900 dark:hover:text-blue-100">Anthropic</a>',
							ollama: '<a href="https://ollama.com" target="_blank" rel="noopener noreferrer" class="underline hover:text-blue-900 dark:hover:text-blue-100">Ollama</a>'
						}
					})}
				</p>
				<p class="text-blue-600 dark:text-blue-400">
					{@html $t('admin.wizard.ai_features_settings', {
						values: {
							settings: '<a href="/admin/settings#integrations" class="underline hover:text-blue-900 dark:hover:text-blue-100"><strong>Settings</strong></a>'
						}
					})}
				</p>
			</div>
		</div>
	</div>
</div>

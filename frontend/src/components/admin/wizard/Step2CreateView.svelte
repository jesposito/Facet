<script lang="ts">
	import { setupWizard, viewTemplates, generateSlug } from '$lib/stores/setupWizard';

	function selectTemplate(templateId: string) {
		const template = viewTemplates.find(t => t.id === templateId);
		if (!template) return;
		
		setupWizard.selectTemplate(templateId);
		setupWizard.updateNewView({
			name: template.suggestedName,
			slug: generateSlug(template.suggestedName),
			visibility: template.visibility,
			description: template.description
		});
	}

	function handleNameChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const name = target.value;
		setupWizard.updateNewView({ 
			name,
			slug: generateSlug(name)
		});
	}
</script>

<div class="space-y-6">
	<div class="text-center">
		<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
			Create your first Facet
		</h2>
		<p class="text-gray-600 dark:text-gray-400">
			Different audiences see different sides of you
		</p>
	</div>
	
	<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
		{#each viewTemplates as template}
			<button 
				type="button"
				class="p-4 border-2 rounded-lg text-left transition-colors {
					$setupWizard.selectedTemplate === template.id
						? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
						: 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
				}"
				onclick={() => selectTemplate(template.id)}
			>
				<div class="flex items-start gap-3">
					<div class="text-2xl">{template.icon}</div>
					<div class="min-w-0">
						<h3 class="font-medium text-gray-900 dark:text-white">
							{template.name}
						</h3>
						<p class="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
							{template.description}
						</p>
					</div>
				</div>
			</button>
		{/each}
	</div>
	
	{#if $setupWizard.selectedTemplate}
		<div class="space-y-4 pt-4 border-t border-gray-200 dark:border-gray-700">
			<div>
				<label for="view-name" class="label">Facet Name</label>
				<input 
					type="text"
					id="view-name"
					class="input"
					placeholder="My Facet"
					value={$setupWizard.newView.name}
					oninput={handleNameChange}
				/>
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
					URL: <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded">/{$setupWizard.newView.slug || 'your-facet'}</code>
				</p>
			</div>
			
			<div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
				<div class="flex items-center gap-2 text-blue-800 dark:text-blue-200">
					<svg class="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span class="text-sm font-medium">
						{#if $setupWizard.newView.visibility === 'public'}
							This facet will be public - anyone with the link can view it
						{:else if $setupWizard.newView.visibility === 'unlisted'}
							This facet is unlisted - only people with the link can find it
						{:else}
							This facet is private - only you can see it
						{/if}
					</span>
				</div>
			</div>
		</div>
	{/if}
</div>

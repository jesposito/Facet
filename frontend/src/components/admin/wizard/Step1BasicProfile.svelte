<script lang="ts">
	import { setupWizard } from '$lib/stores/setupWizard';

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
			Let's start with the basics
		</h2>
		<p class="text-gray-600 dark:text-gray-400">
			This creates your foundation profile
		</p>
	</div>
	
	<div class="space-y-4">
		<div>
			<label for="wizard-name" class="label">Full Name *</label>
			<input
				type="text"
				id="wizard-name"
				class="input"
				placeholder="Jane Smith"
				value={$setupWizard.profile.name || ''}
				oninput={handleNameChange}
			/>
		</div>
		
		<div>
			<label for="wizard-headline" class="label">Professional Title *</label>
			<input
				type="text"
				id="wizard-headline"
				class="input"
				placeholder="Software Engineer"
				value={$setupWizard.profile.headline || ''}
				oninput={handleHeadlineChange}
			/>
			<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
				What do you do? This appears below your name.
			</p>
		</div>
		
		<div>
			<label for="wizard-summary" class="label">Brief Summary (optional)</label>
			<textarea
				id="wizard-summary"
				class="input min-h-[100px]"
				placeholder="I'm a passionate developer who loves building user-friendly applications..."
				value={$setupWizard.profile.summary || ''}
				oninput={handleSummaryChange}
				rows={3}
			></textarea>
			<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
				2-3 sentences about yourself. You can always edit this later.
			</p>
		</div>
	</div>
</div>

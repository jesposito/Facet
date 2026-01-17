<script lang="ts">
	import { setupWizard, canProceed, viewTemplates } from '$lib/stores/setupWizard';
	import { collection } from '$lib/stores/demo';
	import { toasts, triggerSidebarFacetsReload } from '$lib/stores';
	import Step1BasicProfile from './wizard/Step1BasicProfile.svelte';
	import Step2CreateView from './wizard/Step2CreateView.svelte';
	import Step3ReviewLaunch from './wizard/Step3ReviewLaunch.svelte';

	interface Props {
		onComplete?: () => void;
	}

	let { onComplete }: Props = $props();
	let saving = $state(false);

	const steps = [
		{ num: 1, label: 'Profile' },
		{ num: 2, label: 'First Facet' },
		{ num: 3, label: 'Launch' }
	];

	function handleNext() {
		if ($setupWizard.currentStep < 3) {
			setupWizard.nextStep();
		}
	}

	function handleBack() {
		if ($setupWizard.currentStep > 1) {
			setupWizard.previousStep();
		}
	}

	function handleSkip() {
		setupWizard.dismissPermanently();
	}

	async function handleComplete() {
		saving = true;
		try {
			const profileData = $setupWizard.profile;
			const viewData = $setupWizard.newView;
			const template = viewTemplates.find(t => t.id === $setupWizard.selectedTemplate);
			
			const profiles = await collection('profile').getList(1, 1);
			if (profiles.items.length > 0) {
				await collection('profile').update(profiles.items[0].id, {
					name: profileData.name,
					headline: profileData.headline,
					summary: profileData.summary || ''
				});
			} else {
				await collection('profile').create({
					name: profileData.name,
					headline: profileData.headline,
					summary: profileData.summary || '',
					visibility: 'public'
				});
			}
			
			if (template && viewData.name) {
				const sections = template.sections.map((section) => ({
					section,
					enabled: true,
					items: [],
					layout: 'default',
					width: 'full' as const
				}));
				
				await collection('views').create({
					name: viewData.name,
					slug: viewData.slug,
					visibility: viewData.visibility,
					description: viewData.description || '',
					sections,
					is_active: true,
					is_default: false
				});
			}
			
			toasts.add('success', 'Your profile is ready!');
			triggerSidebarFacetsReload();
			setupWizard.complete();
			onComplete?.();
		} catch (err) {
			console.error('Failed to save wizard data:', err);
			toasts.add('error', 'Failed to save. Please try again.');
		} finally {
			saving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && !saving) {
			handleSkip();
		}
	}

	function canGoNext(): boolean {
		const step = $setupWizard.currentStep;
		if (step === 1) return $canProceed.step1;
		if (step === 2) return $canProceed.step2;
		return true;
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $setupWizard.isOpen}
	<div 
		class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
		role="dialog"
		aria-labelledby="wizard-title"
		aria-modal="true"
	>
		<div class="card w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
			<header class="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
				<div class="flex items-center justify-between mb-4">
					<h1 id="wizard-title" class="text-lg font-semibold text-gray-900 dark:text-white">
						Quick Setup
					</h1>
					<button 
						type="button"
						class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
						onclick={handleSkip}
					>
						Skip setup
					</button>
				</div>
				
				<div class="flex items-center gap-2">
					{#each steps as step}
						<div class="flex items-center gap-2 flex-1">
							<div 
								class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors {
									$setupWizard.currentStep === step.num
										? 'bg-primary-600 text-white'
										: $setupWizard.currentStep > step.num
											? 'bg-green-500 text-white'
											: 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
								}"
							>
								{#if $setupWizard.currentStep > step.num}
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
								{:else}
									{step.num}
								{/if}
							</div>
							<span class="text-sm text-gray-600 dark:text-gray-400 hidden sm:inline">
								{step.label}
							</span>
							{#if step.num < 3}
								<div class="flex-1 h-0.5 bg-gray-200 dark:bg-gray-700 {
									$setupWizard.currentStep > step.num ? 'bg-green-500' : ''
								}"></div>
							{/if}
						</div>
					{/each}
				</div>
			</header>
			
			<main class="flex-1 overflow-y-auto p-6">
				{#if $setupWizard.currentStep === 1}
					<Step1BasicProfile />
				{:else if $setupWizard.currentStep === 2}
					<Step2CreateView />
				{:else}
					<Step3ReviewLaunch />
				{/if}
			</main>
			
			<footer class="border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-between">
				<button 
					type="button"
					class="btn btn-ghost"
					disabled={$setupWizard.currentStep === 1 || saving}
					onclick={handleBack}
				>
					Back
				</button>
				
				{#if $setupWizard.currentStep === 3}
					<button 
						type="button"
						class="btn btn-primary"
						disabled={saving}
						onclick={handleComplete}
					>
						{#if saving}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Saving...
						{:else}
							Launch Profile
						{/if}
					</button>
				{:else}
					<button 
						type="button"
						class="btn btn-primary"
						disabled={!canGoNext()}
						onclick={handleNext}
					>
						Continue
					</button>
				{/if}
			</footer>
		</div>
	</div>
{/if}

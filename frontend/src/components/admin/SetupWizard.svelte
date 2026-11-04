<script lang="ts">
	import { setupWizard, canProceed, viewTemplates } from '$lib/stores/setupWizard';
	import { collection } from '$lib/stores/demo';
	import { DraftReadError, findDraftRecord, mayHaveLanded } from '$lib/admin/create-record';
	import { toasts, triggerSidebarFacetsReload } from '$lib/stores';
	import { t } from 'svelte-i18n';
	import { get } from 'svelte/store';
	import Step1BasicProfile from './wizard/Step1BasicProfile.svelte';
	import Step2CreateView from './wizard/Step2CreateView.svelte';
	import Step3ReviewLaunch from './wizard/Step3ReviewLaunch.svelte';

	// Helper to get translation in non-reactive contexts
	function tr(key: string): string {
		const translate = get(t);
		return translate(key);
	}

	interface Props {
		/** A blocking modal sits on top, so this one hands over the keyboard and scroll lock */
		blocked?: boolean;
		onComplete?: () => void;
	}

	let { blocked = false, onComplete }: Props = $props();
	let saving = $state(false);
	let saveError = $state('');
	// The lookup failed, so nobody knows yet whether the Facet was created
	let viewUnknown = $state(false);
	// The stored Facet differs from this draft and the owner has been told, so their next press replaces it
	let replaceStoredView = $state(false);
	let dialogEl: HTMLDivElement | undefined = $state();
	let mainEl: HTMLElement | undefined = $state();

	// Derived so the effects below ignore keystrokes, which also update the store
	const active = $derived($setupWizard.isOpen && !blocked);
	const currentStep = $derived($setupWizard.currentStep);

	const steps = [
		{ num: 1, labelKey: 'admin.wizard.step_profile' },
		{ num: 2, labelKey: 'admin.wizard.step_first_facet' },
		{ num: 3, labelKey: 'admin.wizard.step_launch' }
	];

	const FOCUSABLE =
		'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

	// While open, hold the page still and remember where focus came from. The Resume
	// button is the fallback because an auto-open has no trigger to return to.
	$effect(() => {
		if (!active) return;
		const returnTo = document.activeElement;
		const bodyOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';

		return () => {
			document.body.style.overflow = bodyOverflow;
			const target =
				returnTo instanceof HTMLElement && returnTo !== document.body && returnTo.isConnected
					? returnTo
					: document.getElementById('resume-setup-button');
			target?.focus();
		};
	});

	// Each step starts at its first field, or at the step itself when it has none
	$effect(() => {
		void currentStep;
		if (!active || !mainEl) return;
		(mainEl.querySelector<HTMLElement>('input, textarea') ?? mainEl).focus();
	});

	function handleNext() {
		if ($setupWizard.currentStep < 3) {
			setupWizard.nextStep();
		}
	}

	function handleBack() {
		if ($setupWizard.currentStep > 1) {
			saveError = '';
			viewUnknown = false;
			replaceStoredView = false;
			setupWizard.previousStep();
		}
	}

	function handleSkip() {
		setupWizard.close();
	}

	function describeSaveError(err: unknown, profileSaved: boolean): string {
		if (err instanceof DraftReadError) return tr('admin.wizard.error_view_unknown');
		const { status, response } = err as {
			status?: number;
			response?: { data?: { slug?: unknown } };
		};
		const reason = !status
			? tr('admin.wizard.error_network')
			: response?.data?.slug
				? tr('admin.wizard.error_slug')
				: tr('admin.wizard.error_message');
		if (replaceStoredView) return `${tr('admin.wizard.error_view_differs')} ${reason}`;
		return profileSaved ? `${tr('admin.wizard.error_profile_saved')} ${reason}` : reason;
	}

	type ViewFields = {
		name: string;
		slug: string;
		visibility: string;
		description: string;
		sections: {
			section: string;
			enabled: boolean;
			items: string[];
			itemSelection?: string;
			layout?: string;
			width?: string;
		}[];
	};
	type StoredView = ViewFields & { id: string; is_active: boolean; is_default: boolean };

	function matchesDraft(stored: StoredView, view: ViewFields) {
		const sectionState = (fields: ViewFields) =>
			JSON.stringify(
				fields.sections.map(
					({
						section,
						enabled,
						items,
						itemSelection = 'selected',
						layout = 'default',
						width = 'full'
					}) => ({ section, enabled, items, itemSelection, layout, width })
				)
			);
		return (
			stored.is_active === true &&
			stored.is_default === false &&
			stored.name === view.name &&
			stored.slug === view.slug &&
			stored.visibility === view.visibility &&
			stored.description === view.description &&
			sectionState(stored) === sectionState(view)
		);
	}

	// The draft's own record ID decides whether its Facet exists, so a repeat launch can't make a
	// second one. Returns false when the owner has to decide what happens to a stored Facet.
	async function saveView(view: ViewFields): Promise<boolean> {
		const id = $setupWizard.viewId;
		const views = collection('views');
		let stored = await findDraftRecord<StoredView>(views, id);
		if (!stored) {
			try {
				await views.create({ id, ...view, is_active: true, is_default: false });
				return true;
			} catch (err) {
				if (!mayHaveLanded(err)) throw err;
				stored = await findDraftRecord<StoredView>(views, id);
				if (!stored) throw err;
			}
		}

		if (matchesDraft(stored, view)) return true;
		if (replaceStoredView) {
			await views.update(id, { ...view, is_active: true, is_default: false });
			return true;
		}
		replaceStoredView = true;
		saveError = tr('admin.wizard.error_view_differs');
		return false;
	}

	async function handleComplete() {
		saving = true;
		saveError = '';
		viewUnknown = false;
		let profileSaved = false;
		try {
			const profileData = $setupWizard.profile;
			const viewData = $setupWizard.newView;
			const template = viewTemplates.find((t) => t.id === $setupWizard.selectedTemplate);

			// Only send what the owner filled in, so blank fields can't erase a saved profile
			const profileFields: Record<string, string> = {};
			for (const field of ['name', 'headline', 'summary'] as const) {
				const value = profileData[field]?.trim();
				if (value) profileFields[field] = value;
			}

			const profiles = await collection('profile').getList(1, 1, {
				requestKey: 'setup-wizard-save-lookup'
			});
			if (profiles.items.length > 0) {
				await collection('profile').update(profiles.items[0].id, profileFields);
			} else {
				await collection('profile').create({
					summary: '',
					...profileFields,
					visibility: 'public'
				});
			}
			profileSaved = true;

			if (template && viewData.name) {
				const sections = template.sections.map((section) => ({
					section,
					enabled: true,
					items: [],
					itemSelection: 'all_public' as const,
					layout: 'default',
					width: 'full' as const
				}));

				const saved = await saveView({
					name: viewData.name,
					slug: viewData.slug,
					visibility: viewData.visibility,
					description: viewData.description || '',
					sections
				});
				if (!saved) return;
			}

			toasts.add('success', tr('admin.wizard.success_message'));
			triggerSidebarFacetsReload();
			setupWizard.complete();
			onComplete?.();
		} catch (err) {
			console.error('Failed to save wizard data:', err);
			viewUnknown = err instanceof DraftReadError;
			saveError = describeSaveError(err, profileSaved);
		} finally {
			saving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!active) return;

		if (e.key === 'Escape') {
			if (!saving && !e.isComposing) handleSkip();
			return;
		}
		if (e.key !== 'Tab') return;

		const focusable = dialogEl?.querySelectorAll<HTMLElement>(FOCUSABLE);
		if (!focusable?.length) return;

		const first = focusable[0];
		const last = focusable[focusable.length - 1];
		const current = document.activeElement;

		if (!dialogEl?.contains(current)) {
			e.preventDefault();
			first.focus();
		} else if (e.shiftKey && current === first) {
			e.preventDefault();
			last.focus();
		} else if (!e.shiftKey && current === last) {
			e.preventDefault();
			first.focus();
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
		bind:this={dialogEl}
		class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
		role="dialog"
		aria-labelledby="wizard-title"
		aria-modal="true"
	>
		<div class="card w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
			<header class="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
				<div class="flex items-center justify-between mb-4">
					<h1 id="wizard-title" class="text-lg font-semibold text-gray-900 dark:text-white">
						{$t('admin.wizard.title')}
					</h1>
					<button
						type="button"
						class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
						disabled={saving}
						onclick={handleSkip}
					>
						{$t('admin.wizard.skip')}
					</button>
				</div>

				<div class="flex items-center gap-2">
					{#each steps as step}
						<div class="flex items-center gap-2 flex-1">
							<div
								class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors {$setupWizard.currentStep ===
								step.num
									? 'bg-primary-600 text-white'
									: $setupWizard.currentStep > step.num
										? 'bg-green-500 text-white'
										: 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'}"
							>
								{#if $setupWizard.currentStep > step.num}
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M5 13l4 4L19 7"
										/>
									</svg>
								{:else}
									{step.num}
								{/if}
							</div>
							<span class="text-sm text-gray-600 dark:text-gray-400 hidden sm:inline">
								{$t(step.labelKey)}
							</span>
							{#if step.num < 3}
								<div
									class="flex-1 h-0.5 bg-gray-200 dark:bg-gray-700 {$setupWizard.currentStep >
									step.num
										? 'bg-green-500'
										: ''}"
								></div>
							{/if}
						</div>
					{/each}
				</div>
			</header>

			<div bind:this={mainEl} tabindex="-1" class="flex-1 overflow-y-auto p-6 focus:outline-none">
				{#if $setupWizard.currentStep === 1}
					<Step1BasicProfile />
				{:else if $setupWizard.currentStep === 2}
					<Step2CreateView />
				{:else}
					<Step3ReviewLaunch />
				{/if}
			</div>

			{#if saveError}
				<div
					role="alert"
					class="mx-6 my-3 p-3 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm"
				>
					{saveError}
				</div>
			{/if}

			<footer class="border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-between">
				<button
					type="button"
					class="btn btn-ghost"
					disabled={$setupWizard.currentStep === 1 || saving}
					onclick={handleBack}
				>
					{$t('admin.wizard.back')}
				</button>

				{#if $setupWizard.currentStep === 3}
					<button type="button" class="btn btn-primary" disabled={saving} onclick={handleComplete}>
						{#if saving}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							{$t('admin.wizard.saving')}
						{:else}
							{viewUnknown ? $t('admin.dashboard.retry') : $t('admin.wizard.launch_profile')}
						{/if}
					</button>
				{:else}
					<button
						type="button"
						class="btn btn-primary"
						disabled={!canGoNext()}
						onclick={handleNext}
					>
						{$t('admin.wizard.continue')}
					</button>
				{/if}
			</footer>
		</div>
	</div>
{/if}

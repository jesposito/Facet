import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/env';
import type { Profile, View } from '$lib/pocketbase';
import { isRecordId, newRecordId } from '$lib/admin/create-record';

const STORAGE_KEY = 'facet_setup_wizard';

export interface WizardState {
	isOpen: boolean;
	currentStep: number;
	profile: Partial<Profile>;
	newView: {
		name: string;
		slug: string;
		visibility: 'public' | 'unlisted' | 'private';
		description: string;
	};
	selectedTemplate: string | null;
	// The record ID this draft's Facet will get, so a lost response can be checked, not repeated
	viewId: string;
	// The owner closed the wizard: no auto-open, but Resume stays available
	dismissed: boolean;
	completedWizard: boolean;
}

const defaultState: WizardState = {
	isOpen: false,
	currentStep: 1,
	profile: {},
	newView: {
		name: '',
		slug: '',
		visibility: 'public',
		description: ''
	},
	selectedTemplate: null,
	viewId: '',
	dismissed: false,
	completedWizard: false
};

type WizardDraft = Pick<
	WizardState,
	'currentStep' | 'profile' | 'newView' | 'selectedTemplate' | 'viewId'
>;

function readRecord(value: unknown): Record<string, unknown> {
	return value && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: {};
}

// Stored drafts can be stale or hand-edited, so only known fields survive. The
// address is validated separately; visibility and description come from the chosen template.
function readDraft(stored: unknown): WizardDraft {
	const draft = readRecord(stored);
	const storedProfile = readRecord(draft.profile);
	const profile: WizardDraft['profile'] = {};
	for (const field of ['name', 'headline', 'summary'] as const) {
		const value = storedProfile[field];
		if (typeof value === 'string' && value) profile[field] = value;
	}

	const template = viewTemplates.find((t) => t.id === draft.selectedTemplate);
	const viewName = template && typeof draft.viewName === 'string' ? draft.viewName : '';
	const storedSlug = typeof draft.viewSlug === 'string' ? draft.viewSlug : generateSlug(viewName);
	const newView: WizardState['newView'] = template
		? {
				name: viewName,
				slug: isValidWizardSlug(storedSlug) ? storedSlug : generateSlug(viewName),
				visibility: template.visibility,
				description: template.description
			}
		: defaultState.newView;

	const step1Done = !!profile.name?.trim() && !!profile.headline?.trim();
	const step2Done = !!template && !!viewName.trim() && isValidWizardSlug(newView.slug);
	const furthestStep = !step1Done ? 1 : !step2Done ? 2 : 3;
	const step = typeof draft.currentStep === 'number' ? Math.trunc(draft.currentStep) : 1;

	return {
		currentStep: Math.min(Math.max(step, 1), furthestStep),
		profile,
		newView,
		selectedTemplate: template?.id ?? null,
		viewId: isRecordId(draft.viewId) ? draft.viewId : newRecordId()
	};
}

function loadPersistedState(): Partial<WizardState> {
	if (!browser) return {};
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			const parsed = readRecord(JSON.parse(stored));
			const completedWizard = parsed.completedWizard === true;
			return {
				// Older builds wrote dismissedPermanently on every close. Those owners
				// stay out of auto-open and can now use Resume.
				dismissed: parsed.dismissed === true || parsed.dismissedPermanently === true,
				completedWizard,
				...(completedWizard ? {} : readDraft(parsed.draft))
			};
		}
	} catch {
		/* localStorage may be unavailable */
	}
	return {};
}

function createSetupWizardStore() {
	const initial: WizardState = { ...defaultState, viewId: newRecordId(), ...loadPersistedState() };

	const { subscribe, set, update } = writable<WizardState>(initial);

	// Only these fields reach localStorage. isOpen belongs to one page load, and
	// passwords and tokens never pass through this store.
	function persistState(state: WizardState) {
		if (!browser) return;
		try {
			localStorage.setItem(
				STORAGE_KEY,
				JSON.stringify({
					dismissed: state.dismissed,
					completedWizard: state.completedWizard,
					draft: state.completedWizard
						? undefined
						: {
								currentStep: state.currentStep,
								profile: {
									name: state.profile.name,
									headline: state.profile.headline,
									summary: state.profile.summary
								},
								selectedTemplate: state.selectedTemplate,
								viewName: state.newView.name,
								viewSlug: state.newView.slug,
								viewId: state.viewId
							}
				})
			);
		} catch {
			/* localStorage may be unavailable */
		}
	}

	subscribe(persistState);

	return {
		subscribe,

		open: () =>
			update((s) => {
				// Never reopen if already dismissed or completed
				if (s.dismissed || s.completedWizard) return s;
				return { ...s, isOpen: true };
			}),

		// The owner asked for it, so an earlier dismissal doesn't stand in the way
		resume: () => update((s) => (s.completedWizard ? s : { ...s, isOpen: true, dismissed: false })),

		// Closing stops auto-open and keeps the draft for Resume
		close: () => update((s) => ({ ...s, isOpen: false, dismissed: true })),

		// Saved values fill gaps only, so a resumed draft keeps what the owner typed
		prefillProfile: (saved: Profile | null) =>
			update((s) => ({
				...s,
				profile: {
					...s.profile,
					name: s.profile.name || saved?.name,
					headline: s.profile.headline || saved?.headline,
					summary: s.profile.summary || saved?.summary
				}
			})),

		nextStep: () =>
			update((s) => ({
				...s,
				currentStep: Math.min(s.currentStep + 1, 3)
			})),

		previousStep: () =>
			update((s) => ({
				...s,
				currentStep: Math.max(s.currentStep - 1, 1)
			})),

		setStep: (step: number) =>
			update((s) => ({
				...s,
				currentStep: Math.min(Math.max(step, 1), 3)
			})),

		updateProfile: (profile: Partial<Profile>) =>
			update((s) => ({
				...s,
				profile: { ...s.profile, ...profile }
			})),

		updateNewView: (view: Partial<WizardState['newView']>) =>
			update((s) => ({
				...s,
				newView: { ...s.newView, ...view }
			})),

		selectTemplate: (templateId: string | null) =>
			update((s) => ({
				...s,
				selectedTemplate: templateId
			})),

		complete: () =>
			update((s) => ({
				...s,
				isOpen: false,
				completedWizard: true,
				currentStep: 1,
				profile: {},
				newView: defaultState.newView,
				selectedTemplate: null
			})),

		reset: () => {
			set({ ...defaultState, viewId: newRecordId() });
			if (browser) {
				try {
					localStorage.removeItem(STORAGE_KEY);
				} catch {
					/* localStorage may be unavailable */
				}
			}
		}
	};
}

export const viewTemplates = [
	{
		id: 'recruiter',
		name: 'For Recruiters',
		icon: '💼',
		description: 'Professional focus with experience, skills, and education',
		suggestedName: 'Recruiter',
		visibility: 'public' as const,
		sections: ['experience', 'skills', 'education', 'projects']
	},
	{
		id: 'portfolio',
		name: 'Portfolio',
		icon: '🎨',
		description: 'Showcase your projects and creative work',
		suggestedName: 'Portfolio',
		visibility: 'public' as const,
		sections: ['projects', 'skills', 'experience']
	},
	{
		id: 'consulting',
		name: 'For Clients',
		icon: '🤝',
		description: 'Highlight expertise and past work for potential clients',
		suggestedName: 'Consulting',
		visibility: 'unlisted' as const,
		sections: ['projects', 'experience', 'skills']
	},
	{
		id: 'speaker',
		name: 'Speaker',
		icon: '🎤',
		description: 'Conference organizers see your talks and expertise',
		suggestedName: 'Speaker',
		visibility: 'public' as const,
		sections: ['talks', 'experience', 'posts']
	}
];

// viewTemplates has to exist first: the store reads it while restoring a draft
export const setupWizard = createSetupWizardStore();

export const canProceed = derived(setupWizard, ($wizard) => ({
	step1: !!$wizard.profile.name?.trim() && !!$wizard.profile.headline?.trim(),
	step2:
		!!$wizard.selectedTemplate &&
		!!$wizard.newView.name?.trim() &&
		isValidWizardSlug($wizard.newView.slug),
	step3: true
}));

export function isSetupIncomplete(
	profile: Profile | null,
	views: View[],
	isDemoMode: boolean
): boolean {
	if (isDemoMode) return false;

	const hasBasicProfile = !!profile?.name && !!profile?.headline;
	const hasViews = views.length > 0;

	return !hasBasicProfile || !hasViews;
}

export function shouldShowWizard(
	profile: Profile | null,
	views: View[],
	isDemoMode: boolean
): boolean {
	const state = get(setupWizard);

	if (state.dismissed || state.completedWizard) return false;

	return isSetupIncomplete(profile, views, isDemoMode);
}

function isValidWizardSlug(slug: string): boolean {
	return /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,99}$/.test(slug);
}

export function generateSlug(name: string): string {
	return name
		.toLowerCase()
		.replace(/[^a-z0-9\s-]/g, '')
		.replace(/\s+/g, '-')
		.replace(/-+/g, '-')
		.slice(0, 50);
}

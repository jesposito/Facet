import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import type { Profile, View } from '$lib/pocketbase';

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
	dismissedPermanently: boolean;
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
	dismissedPermanently: false,
	completedWizard: false
};

function loadPersistedState(): Partial<WizardState> {
	if (!browser) return {};
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored);
		}
	} catch {
		/* localStorage may be unavailable */
	}
	return {};
}

function createSetupWizardStore() {
	const persisted = loadPersistedState();
	const initial: WizardState = {
		...defaultState,
		dismissedPermanently: persisted.dismissedPermanently || false,
		completedWizard: persisted.completedWizard || false
	};

	const { subscribe, set, update } = writable<WizardState>(initial);

	function persistFlags(state: WizardState) {
		if (!browser) return;
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify({
				dismissedPermanently: state.dismissedPermanently,
				completedWizard: state.completedWizard
			}));
		} catch {
			/* localStorage may be unavailable */
		}
	}

	return {
		subscribe,
		
		open: () => update(s => ({ ...s, isOpen: true })),
		
		close: () => update(s => ({ ...s, isOpen: false })),
		
		nextStep: () => update(s => ({ 
			...s, 
			currentStep: Math.min(s.currentStep + 1, 3) 
		})),
		
		previousStep: () => update(s => ({ 
			...s, 
			currentStep: Math.max(s.currentStep - 1, 1) 
		})),
		
		setStep: (step: number) => update(s => ({ 
			...s, 
			currentStep: Math.min(Math.max(step, 1), 3) 
		})),
		
		updateProfile: (profile: Partial<Profile>) => update(s => ({ 
			...s, 
			profile: { ...s.profile, ...profile } 
		})),
		
		updateNewView: (view: Partial<WizardState['newView']>) => update(s => ({ 
			...s, 
			newView: { ...s.newView, ...view } 
		})),
		
		selectTemplate: (templateId: string | null) => update(s => ({ 
			...s, 
			selectedTemplate: templateId 
		})),
		
		dismissPermanently: () => update(s => {
			const newState = { ...s, isOpen: false, dismissedPermanently: true };
			persistFlags(newState);
			return newState;
		}),
		
		complete: () => update(s => {
			const newState = { 
				...s, 
				isOpen: false, 
				completedWizard: true,
				currentStep: 1,
				profile: {},
				newView: defaultState.newView,
				selectedTemplate: null
			};
			persistFlags(newState);
			return newState;
		}),
		
		reset: () => {
			set(defaultState);
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

export const setupWizard = createSetupWizardStore();

export const canProceed = derived(
	setupWizard,
	($wizard) => ({
		step1: !!$wizard.profile.name?.trim() && !!$wizard.profile.headline?.trim(),
		step2: !!$wizard.selectedTemplate && !!$wizard.newView.name?.trim(),
		step3: true
	})
);

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

export function shouldShowWizard(
	profile: Profile | null,
	views: View[],
	isDemoMode: boolean
): boolean {
	const state = get(setupWizard);
	
	if (isDemoMode) return false;
	if (state.dismissedPermanently || state.completedWizard) return false;
	
	const hasBasicProfile = !!profile?.name && !!profile?.headline;
	const hasViews = views.length > 0;
	
	return !hasBasicProfile || !hasViews;
}

export function generateSlug(name: string): string {
	return name
		.toLowerCase()
		.replace(/[^a-z0-9\s-]/g, '')
		.replace(/\s+/g, '-')
		.replace(/-+/g, '-')
		.slice(0, 50);
}

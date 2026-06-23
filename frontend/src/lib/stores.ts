import { writable, derived } from 'svelte/store';
import type { Profile, Experience, Project, Education, Skill } from './pocketbase';
import { fetchWithTimeout } from './utils';

// Theme store
//
// Theme resolution precedence (highest to lowest):
//   1. localStorage 'theme' — per-user override via ThemeToggle, persists across visits
//   2. serverDefault (admin setting `default_theme_mode`) — when 'light' or 'dark'
//   3. window prefers-color-scheme — when serverDefault is 'system' (or unset)
//   4. 'light' — final fallback
//
// The FOUC script in app.html runs before SvelteKit hydrates and only sees
// localStorage + prefers-color-scheme (no SSR data access). When admin's
// default_theme_mode is 'dark' but the user has no localStorage entry AND no
// system dark preference, the page paints light briefly, then this store
// flips it on hydration. Acceptable tradeoff for a static FOUC script.
function createThemeStore() {
	const { subscribe, set } = writable<'light' | 'dark'>('light');

	return {
		subscribe,
		initialize: (serverDefault?: string) => {
			if (typeof window === 'undefined') return;
			const saved = localStorage.getItem('theme');
			if (saved === 'dark' || saved === 'light') {
				set(saved);
				document.documentElement.classList.toggle('dark', saved === 'dark');
				return;
			}
			// No per-user override — honor admin default if explicit, else system pref
			if (serverDefault === 'dark' || serverDefault === 'light') {
				set(serverDefault);
				document.documentElement.classList.toggle('dark', serverDefault === 'dark');
				return;
			}
			if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
				set('dark');
				document.documentElement.classList.add('dark');
			}
		},
		toggle: () => {
			if (typeof window === 'undefined') return;
			const current = document.documentElement.classList.contains('dark') ? 'dark' : 'light';
			const next = current === 'dark' ? 'light' : 'dark';
			set(next);
			localStorage.setItem('theme', next);
			document.documentElement.classList.toggle('dark', next === 'dark');
		}
	};
}

export const theme = createThemeStore();

// Toast notifications
export interface Toast {
	id: string;
	type: 'success' | 'error' | 'info' | 'warning';
	message: string;
	duration?: number;
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);

	const add = (type: Toast['type'], message: string, duration?: number) => {
		const id = crypto.randomUUID?.() ?? Math.random().toString(36).slice(2);
		// Errors and warnings persist until dismissed — auto-dismissing a message
		// the user may need to read or act on is a WCAG 2.2.1 (Timing Adjustable)
		// hazard. Success/info still auto-dismiss after 5s. Callers can override.
		const effectiveDuration = duration ?? (type === 'error' || type === 'warning' ? 0 : 5000);
		update((toasts) => [...toasts, { id, type, message, duration: effectiveDuration }]);
		if (effectiveDuration > 0) {
			setTimeout(() => {
				update((toasts) => toasts.filter((t) => t.id !== id));
			}, effectiveDuration);
		}
		return id;
	};

	const remove = (id: string) => {
		update((toasts) => toasts.filter((t) => t.id !== id));
	};

	return {
		subscribe,
		add,
		remove,
		success: (message: string, duration?: number) => add('success', message, duration),
		error: (message: string, duration?: number) => add('error', message, duration),
		info: (message: string, duration?: number) => add('info', message, duration),
		warning: (message: string, duration?: number) => add('warning', message, duration)
	};
}

export const toasts = createToastStore();

// Loading state
export const isLoading = writable(false);

// Profile data store
export const profile = writable<Profile | null>(null);
export const experience = writable<Experience[]>([]);
export const projects = writable<Project[]>([]);
export const education = writable<Education[]>([]);
export const skills = writable<Skill[]>([]);

// Grouped skills derived store
export const groupedSkills = derived(skills, ($skills) => {
	const grouped: Record<string, Skill[]> = {};
	for (const skill of $skills) {
		const category = skill.category || 'Other';
		if (!grouped[category]) {
			grouped[category] = [];
		}
		grouped[category].push(skill);
	}
	return grouped;
});

// Featured projects
export const featuredProjects = derived(projects, ($projects) =>
	$projects.filter((p) => p.is_featured).slice(0, 3)
);

// Admin state
export const isAdmin = writable(false);
export const adminSidebarOpen = writable(true);

export const sidebarFacetsVersion = writable(0);
export function triggerSidebarFacetsReload() {
	sidebarFacetsVersion.update(n => n + 1);
}

// Sidebar section collapse states with localStorage persistence
export type SidebarSectionStates = Record<string, boolean>;

function createSidebarSectionStatesStore() {
	const STORAGE_KEY = 'sidebarSectionStates';
	const { subscribe, set, update } = writable<SidebarSectionStates>({});

	return {
		subscribe,
		initialize: (allSectionIds?: string[], defaultExpanded?: string | string[]) => {
			if (typeof window === 'undefined') return;
			try {
				const saved = localStorage.getItem(STORAGE_KEY);
				if (saved) {
					const parsed = JSON.parse(saved);
					if (typeof parsed === 'object' && parsed !== null) {
						set(parsed);
						return;
					}
				}
			} catch {
				// Invalid JSON, ignore and use defaults
			}
			// No saved state - set defaults with specified sections open
			if (allSectionIds && defaultExpanded) {
				const expandedIds = Array.isArray(defaultExpanded) ? defaultExpanded : [defaultExpanded];
				const defaultState: SidebarSectionStates = {};
				for (const id of allSectionIds) {
					defaultState[id] = expandedIds.includes(id);
				}
				set(defaultState);
				localStorage.setItem(STORAGE_KEY, JSON.stringify(defaultState));
			}
		},
		toggle: (sectionId: string, allSectionIds?: string[]) => {
			update((states) => {
				// Default to closed if not explicitly set
				const currentState = states[sectionId] ?? false;
				const willOpen = !currentState;

				// Accordion behavior: close all other sections when opening one
				const newStates: SidebarSectionStates = {};

				// Get all section IDs to manage
				const sectionsToManage = allSectionIds ?? Object.keys(states);

				if (willOpen) {
					// Close all sections, then open only this one
					for (const id of sectionsToManage) {
						newStates[id] = false;
					}
					newStates[sectionId] = true;
				} else {
					// Just close this section (keep others as they are)
					Object.assign(newStates, states);
					newStates[sectionId] = false;
				}

				if (typeof window !== 'undefined') {
					localStorage.setItem(STORAGE_KEY, JSON.stringify(newStates));
				}
				return newStates;
			});
		},
		setExpanded: (sectionId: string, expanded: boolean) => {
			update((states) => {
				const newStates = { ...states, [sectionId]: expanded };
				if (typeof window !== 'undefined') {
					localStorage.setItem(STORAGE_KEY, JSON.stringify(newStates));
				}
				return newStates;
			});
		},
		isExpanded: (states: SidebarSectionStates, sectionId: string, defaultExpanded = true): boolean => {
			return states[sectionId] ?? defaultExpanded;
		},
		expandAll: (sectionIds: string[]) => {
			update((states) => {
				const newStates = { ...states };
				for (const id of sectionIds) {
					newStates[id] = true;
				}
				if (typeof window !== 'undefined') {
					localStorage.setItem(STORAGE_KEY, JSON.stringify(newStates));
				}
				return newStates;
			});
		},
		collapseAll: (sectionIds: string[]) => {
			update((states) => {
				const newStates = { ...states };
				for (const id of sectionIds) {
					newStates[id] = false;
				}
				if (typeof window !== 'undefined') {
					localStorage.setItem(STORAGE_KEY, JSON.stringify(newStates));
				}
				return newStates;
			});
		}
	};
}

export const sidebarSectionStates = createSidebarSectionStatesStore();

// Current view context (for view pages)
export interface ViewContext {
	id: string;
	slug: string;
	name: string;
	heroHeadline?: string;
	heroSummary?: string;
	ctaText?: string;
	ctaUrl?: string;
	sections?: Record<string, unknown[]>;
}

export const currentView = writable<ViewContext | null>(null);

// Modal state
export const modalOpen = writable(false);
export const modalContent = writable<{ title: string; component: unknown; props?: Record<string, unknown> } | null>(null);

export function openModal(title: string, component: unknown, props?: Record<string, unknown>) {
	modalContent.set({ title, component, props });
	modalOpen.set(true);
}

export function closeModal() {
	modalOpen.set(false);
	modalContent.set(null);
}

// Confirm dialog system
export interface ConfirmOptions {
	title: string;
	message: string;
	confirmText?: string;
	cancelText?: string;
	danger?: boolean;
}

interface ConfirmState {
	open: boolean;
	options: ConfirmOptions | null;
	resolve: ((value: boolean) => void) | null;
	/** Keyed remount counter so pre-empted dialogs re-announce to AT. */
	contentKey: number;
}

function createConfirmStore() {
	const initial: ConfirmState = {
		open: false,
		options: null,
		resolve: null,
		contentKey: 0
	};
	const { subscribe, set, update } = writable<ConfirmState>(initial);

	// Snapshot of current state for synchronous read inside confirm().
	// Required so a new confirm() can resolve a prior unresolved promise
	// as false before setting new state — without this the prior caller
	// hangs forever (memory leak + silent wrong-row action when user
	// rapidly fires shortcuts on admin lists).
	let current: ConfirmState = initial;
	subscribe((s) => {
		current = s;
	});

	return {
		subscribe,
		confirm: (options: ConfirmOptions): Promise<boolean> => {
			return new Promise((resolve) => {
				// Reject-previous: if a prior confirm is unresolved, resolve it
				// false first. Caller treats it as "user cancelled by opening a
				// new prompt". Single source of truth is the new dialog the user
				// is about to see.
				const priorResolve = current.resolve;
				if (priorResolve) {
					try {
						priorResolve(false);
					} catch (err) {
						console.error('[confirmDialog] prior resolver threw', err);
					}
				}
				set({
					open: true,
					options,
					resolve,
					contentKey: current.contentKey + 1
				});
			});
		},
		respond: (value: boolean) => {
			update((state) => {
				// Dedupe: capture and null the resolver before invoking so a
				// re-entrant call can't double-resolve the same Promise.
				const resolver = state.resolve;
				const next: ConfirmState = {
					open: false,
					options: null,
					resolve: null,
					contentKey: state.contentKey
				};
				if (resolver) {
					try {
						resolver(value);
					} catch (err) {
						console.error('[confirmDialog] resolver threw', err);
					}
				}
				return next;
			});
		},
		close: () => {
			update((state) => {
				const resolver = state.resolve;
				const next: ConfirmState = {
					open: false,
					options: null,
					resolve: null,
					contentKey: state.contentKey
				};
				if (resolver) {
					try {
						resolver(false);
					} catch (err) {
						console.error('[confirmDialog] resolver threw on close', err);
					}
				}
				return next;
			});
		}
	};
}

export const confirmDialog = createConfirmStore();

// Convenience function for common confirm patterns
export async function confirm(options: ConfirmOptions | string): Promise<boolean> {
	if (typeof options === 'string') {
		return confirmDialog.confirm({
			title: 'Confirm',
			message: options
		});
	}
	return confirmDialog.confirm(options);
}

// Site settings store
export interface SiteSettings {
	hideDemoToggle: boolean;
	hideLoginButton: boolean;
	customCSS: string;
	gaMeasurementId: string;
	favicon?: string;
	defaultLocale: string;
}

function createSiteSettingsStore() {
	const { subscribe, set, update } = writable<SiteSettings>({
		hideDemoToggle: false,
		hideLoginButton: false,
		customCSS: '',
		gaMeasurementId: '',
		favicon: undefined,
		defaultLocale: 'en'
	});

	return {
		subscribe,
		loadSettings: async () => {
			if (typeof window === 'undefined') return;
			try {
				const response = await fetchWithTimeout('/api/site-settings');
				if (response.ok) {
					const data = await response.json();
					set({
						hideDemoToggle: data.hide_demo_toggle || false,
						hideLoginButton: data.hide_login_button || false,
						customCSS: data.custom_css || '',
						gaMeasurementId: data.ga_measurement_id || '',
						favicon: data.favicon || undefined,
						defaultLocale: data.default_locale || 'en'
					});
				}
			} catch {
			}
		},
		updateSetting: (key: keyof SiteSettings, value: any) => {
			update((settings) => ({
				...settings,
				[key]: value
			}));
		}
	};
}

export const siteSettings = createSiteSettingsStore();

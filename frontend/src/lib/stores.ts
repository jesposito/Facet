import { writable, derived } from 'svelte/store';
import type { Profile, Experience, Project, Education, Skill } from './pocketbase';

// Theme store
function createThemeStore() {
	const { subscribe, set } = writable<'light' | 'dark'>('light');

	return {
		subscribe,
		initialize: () => {
			if (typeof window === 'undefined') return;
			const saved = localStorage.getItem('theme');
			if (saved === 'dark' || saved === 'light') {
				set(saved);
				document.documentElement.classList.toggle('dark', saved === 'dark');
			} else if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
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

	return {
		subscribe,
		add: (type: Toast['type'], message: string, duration = 5000) => {
			const id = crypto.randomUUID();
			update((toasts) => [...toasts, { id, type, message, duration }]);
			if (duration > 0) {
				setTimeout(() => {
					update((toasts) => toasts.filter((t) => t.id !== id));
				}, duration);
			}
			return id;
		},
		remove: (id: string) => {
			update((toasts) => toasts.filter((t) => t.id !== id));
		},
		success: (message: string) => createToastStore().add('success', message),
		error: (message: string) => createToastStore().add('error', message),
		info: (message: string) => createToastStore().add('info', message),
		warning: (message: string) => createToastStore().add('warning', message)
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
}

function createConfirmStore() {
	const { subscribe, set, update } = writable<ConfirmState>({
		open: false,
		options: null,
		resolve: null
	});

	return {
		subscribe,
		confirm: (options: ConfirmOptions): Promise<boolean> => {
			return new Promise((resolve) => {
				set({
					open: true,
					options,
					resolve
				});
			});
		},
		respond: (value: boolean) => {
			update((state) => {
				if (state.resolve) {
					state.resolve(value);
				}
				return {
					open: false,
					options: null,
					resolve: null
				};
			});
		},
		close: () => {
			update((state) => {
				if (state.resolve) {
					state.resolve(false);
				}
				return {
					open: false,
					options: null,
					resolve: null
				};
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

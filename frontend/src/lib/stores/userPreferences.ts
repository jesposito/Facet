/**
 * User preferences store.
 *
 * Per-user toggles that should persist across sessions. Backed by
 * localStorage under a single key (`facet-user-prefs`). Defaults are
 * intentionally conservative — every new pref ships OFF until the user
 * explicitly opts in, so existing installs see zero behavior change.
 *
 * Client-only: on the server `load()` returns DEFAULTS and `persist()` is
 * a no-op.
 */
import { browser } from '$app/environment';
import { writable } from 'svelte/store';

const STORAGE_KEY = 'facet-user-prefs';

export interface UserPreferences {
	/** When true, admin layout renders a right-column iframe of the public
	 *  profile as a live preview. Default OFF. */
	livePreviewEnabled: boolean;
}

const DEFAULTS: UserPreferences = {
	livePreviewEnabled: false
};

function load(): UserPreferences {
	if (!browser) return DEFAULTS;
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return DEFAULTS;
		const parsed = JSON.parse(raw) as Partial<UserPreferences>;
		return { ...DEFAULTS, ...parsed };
	} catch {
		return DEFAULTS;
	}
}

function persist(value: UserPreferences): void {
	if (!browser) return;
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
	} catch {
		/* localStorage quota or private-mode — ignore, in-memory only */
	}
}

function createUserPreferencesStore() {
	const { subscribe, set, update } = writable<UserPreferences>(load());

	return {
		subscribe,
		set: (value: UserPreferences) => {
			persist(value);
			set(value);
		},
		/** Atomic update — passes current value, expects new value back. */
		update: (updater: (current: UserPreferences) => UserPreferences) =>
			update((current) => {
				const next = updater(current);
				persist(next);
				return next;
			}),
		/** Convenience: set a single pref by key without mangling the rest. */
		setPref: <K extends keyof UserPreferences>(key: K, value: UserPreferences[K]) =>
			update((current) => {
				const next = { ...current, [key]: value };
				persist(next);
				return next;
			})
	};
}

export const userPreferences = createUserPreferencesStore();

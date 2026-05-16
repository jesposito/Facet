/**
 * Settings dirty-state coordination store.
 *
 * Purpose: aggregate dirty state across independent settings sections so a
 * single sticky save bar can summarize and trigger a coordinated save/discard.
 * Use when an admin page hosts multiple independently-editable sections and
 * you want one shared "save all / discard all" affordance.
 *
 * Lifecycle: sections call `settingsDirty.set(id, dirty)` on mount/change and
 * `set(id, false)` (or `reset()`) on dismount or after a successful save.
 *
 * Events dispatched on `window`:
 *   - SETTINGS_SAVE_EVENT    ("facet:settings-save-all")
 *   - SETTINGS_DISCARD_EVENT ("facet:settings-discard-all")
 * Each section adds its own listener and runs its own save/revert logic; we
 * unify the UX affordance, not the payloads.
 */

import { writable, derived, type Readable } from 'svelte/store';

export const SETTINGS_SAVE_EVENT = 'facet:settings-save-all';
export const SETTINGS_DISCARD_EVENT = 'facet:settings-discard-all';

/**
 * Known section identifiers shipped with self-hosted Facet. Callers may pass
 * arbitrary string ids — the store does not enforce membership in this union,
 * it exists only to give the default `SECTION_LABELS` registry a typed shape
 * and to make refactors of well-known sections discoverable.
 */
export type SettingsSectionId =
	| 'profile'
	| 'branding'
	| 'integrations'
	| 'about'
	| 'danger';

type DirtyMap = Record<string, boolean>;

function createSettingsDirtyStore() {
	const internal = writable<DirtyMap>({});

	return {
		subscribe: internal.subscribe,

		/** Mark a section dirty (true) or clean (false). */
		set(section: string, dirty: boolean) {
			internal.update((map) => {
				// Avoid unnecessary re-emits when the value is unchanged.
				if (!!map[section] === dirty) return map;
				return { ...map, [section]: dirty };
			});
		},

		/** Clear all dirty flags (called after a successful save-all). */
		reset() {
			internal.set({});
		},

		/** Dispatch the save-all event; each section listens and runs its own save. */
		triggerSaveAll() {
			if (typeof window === 'undefined') return;
			window.dispatchEvent(new CustomEvent(SETTINGS_SAVE_EVENT));
		},

		/** Dispatch the discard-all event; each section listens and reverts its local state. */
		triggerDiscardAll() {
			if (typeof window === 'undefined') return;
			window.dispatchEvent(new CustomEvent(SETTINGS_DISCARD_EVENT));
		}
	};
}

export const settingsDirty = createSettingsDirtyStore();

/** Ordered list of dirty section ids (for the save-bar chip list). */
export const dirtySections: Readable<string[]> = derived(settingsDirty, ($map) =>
	Object.keys($map).filter((k) => $map[k] === true)
);

/** Any dirty section? */
export const isAnyDirty: Readable<boolean> = derived(
	dirtySections,
	($list) => $list.length > 0
);

/**
 * Human-readable labels for well-known section ids. Callers using ad-hoc ids
 * should provide their own labels at the call site.
 */
export const SECTION_LABELS: Record<SettingsSectionId, string> = {
	profile: 'Profile',
	branding: 'Branding',
	integrations: 'Integrations',
	about: 'About',
	danger: 'Danger zone'
};

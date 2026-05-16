/**
 * Form choreography helpers — validation timing policy + dirty-state tracking.
 *
 * Backported from cloud as infrastructure-only. No existing forms are migrated
 * in this PR; new forms should adopt these helpers and legacy forms can be
 * converted opportunistically. Pure TS, no framework imports, no SaaS deps.
 */

export type ValidationTiming = 'blur' | 'submit';
export type ValidationTrigger = 'input' | 'blur' | 'submit';

/**
 * Decide whether to run validation for the given trigger under the chosen
 * timing policy.
 *
 * - 'blur' timing: validate on blur + submit. Input-time validation is the
 *   caller's responsibility (typically: only after first blur).
 * - 'submit' timing: validate only on submit — input + blur are silent.
 */
export function shouldValidate(
	timing: ValidationTiming,
	trigger: ValidationTrigger
): boolean {
	if (trigger === 'submit') return true;
	if (timing === 'blur' && trigger === 'blur') return true;
	return false;
}

/**
 * Track dirty state for a form value. Returns a small controller that answers
 * "did the user change anything from the loaded state?".
 *
 * Usage:
 *   const form = useDirty({ name: '', bio: '' });
 *   <input bind:value={form.current.name} oninput={() => form.update(form.current)} />
 *   <button disabled={!form.dirty}>Save</button>
 */
export function useDirty<T>(initial: T) {
	// Structured clone snapshots the baseline so nested-object edits register
	// as dirty (shallow equality would miss them).
	let baseline: T = structuredClone(initial);
	let current: T = structuredClone(initial);
	let dirty = false;

	function equal(a: unknown, b: unknown): boolean {
		return JSON.stringify(a) === JSON.stringify(b);
	}

	return {
		get current() {
			return current;
		},
		get dirty() {
			return dirty;
		},
		/** Replace the current value and recompute dirty against baseline. */
		update(next: T) {
			current = next;
			dirty = !equal(current, baseline);
		},
		/** Reset back to baseline (e.g. after a successful save). */
		reset() {
			current = structuredClone(baseline);
			dirty = false;
		},
		/** Commit the current value as the new baseline (post-save). */
		commit() {
			baseline = structuredClone(current);
			dirty = false;
		}
	};
}

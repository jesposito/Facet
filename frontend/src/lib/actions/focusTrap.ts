/**
 * Svelte action that makes a modal/lightbox container keyboard-accessible
 * per WCAG 2.1.2 (No Keyboard Trap), 2.4.3 (Focus Order), and 4.1.2.
 *
 * Several modals in this codebase (ConfirmDialog, PasswordChangeModal,
 * ReportModal, TwoFactorModal) each hand-rolled the same focus-management
 * block: capture `document.activeElement`, move focus inside, wrap Tab /
 * Shift+Tab around the focusable children, lock body scroll, and restore
 * focus + scroll on teardown. Diverging fixes across those copies was a
 * standing review hazard, so this is the single source of truth.
 *
 * On mount it:
 *  - captures the currently-focused element as the return target,
 *  - moves focus to the first focusable child (or the container itself),
 *  - traps Tab / Shift+Tab within the container's focusable elements,
 *  - calls `onEscape` when Escape is pressed (the host owns closing),
 *  - locks body scroll (`document.body.style.overflow = 'hidden'`).
 *
 * On destroy it restores body scroll and returns focus to the captured
 * trigger (only if that element is still in the document and focusable).
 *
 * Because the action only mounts when its node is rendered, hosts gate it
 * behind their existing `{#if open}` so capture/restore line up with the
 * open/close lifecycle. The host keeps ownership of `open` state — Escape
 * is delegated via `onEscape` rather than toggling anything here, matching
 * the AIResumeModal/controller pattern where the page is authoritative.
 */

export interface FocusTrapParams {
	/** Called when Escape is pressed inside the trap. Host closes the modal. */
	onEscape?: () => void;
	/**
	 * Optional explicit return target. When omitted, the element that had
	 * focus at mount time is used. Pass this when the trigger may unmount in
	 * the same tick the modal opens (see WCAG 2.4.3 note in the controller).
	 */
	returnFocusTo?: HTMLElement | null;
	/**
	 * Set false to skip moving focus into the container on mount — useful
	 * when the host already focuses a specific field itself. Defaults to true.
	 */
	autoFocus?: boolean;
}

// Matches the selector the pre-existing hand-rolled modals used, so behavior
// is identical to what shipped before — just shared.
const FOCUSABLE_SELECTOR =
	'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function getFocusable(container: HTMLElement): HTMLElement[] {
	return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
		// offsetParent is null for display:none; keeps hidden controls out of
		// the tab cycle so Shift+Tab from the first visible element wraps right.
		(el) => el.offsetParent !== null || el === document.activeElement
	);
}

export function focusTrap(node: HTMLElement, params: FocusTrapParams = {}) {
	let current = params;
	// Capture BEFORE we move focus, so the return target is the trigger, not
	// whatever inside the modal we focus next.
	const previouslyFocused =
		current.returnFocusTo ?? (document.activeElement as HTMLElement | null);

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			current.onEscape?.();
			return;
		}
		if (event.key !== 'Tab') return;

		const focusable = getFocusable(node);
		if (focusable.length === 0) {
			// Nothing focusable inside — keep focus on the container so the user
			// can't tab out into the page behind the modal (No Keyboard Trap is
			// about being able to LEAVE via the modal's own affordances, not
			// about Tab escaping to the background).
			event.preventDefault();
			node.focus();
			return;
		}

		const first = focusable[0];
		const last = focusable[focusable.length - 1];
		const active = document.activeElement;

		if (event.shiftKey) {
			if (active === first || active === node) {
				event.preventDefault();
				last.focus();
			}
		} else if (active === last || active === node) {
			event.preventDefault();
			first.focus();
		}
	}

	// Listen on the node (capture phase) so Tab handling works even when focus
	// is on a child, and so multiple stacked traps don't fight over document.
	node.addEventListener('keydown', handleKeydown);

	const previousBodyOverflow = document.body.style.overflow;
	document.body.style.overflow = 'hidden';

	if (current.autoFocus !== false) {
		// Ensure the container can receive focus as a fallback target.
		if (!node.hasAttribute('tabindex')) node.setAttribute('tabindex', '-1');
		const focusable = getFocusable(node);
		(focusable[0] ?? node).focus();
	}

	return {
		update(next: FocusTrapParams) {
			current = next;
		},
		destroy() {
			node.removeEventListener('keydown', handleKeydown);
			document.body.style.overflow = previousBodyOverflow;
			if (
				previouslyFocused &&
				typeof previouslyFocused.focus === 'function' &&
				document.contains(previouslyFocused)
			) {
				previouslyFocused.focus();
			}
		}
	};
}

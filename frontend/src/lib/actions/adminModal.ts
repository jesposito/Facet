/**
 * Modal behaviour for admin overlays: a fixed overlay wrapping one `role="dialog"`
 * panel, rendered under the host's `{#if}`. Builds on `focusTrap`.
 *
 * This is deliberately not a native <dialog>. showModal() would lift the panel
 * above the root layout's ConfirmDialog and toast region and make both inert,
 * yet the resume modal asks for confirmation before deleting an export, and
 * these modals report errors through toasts. So the overlay moves to just after
 * `.admin-shell`, the shell goes inert, and ConfirmDialog and toasts, which render
 * later at the same z-index, stay on top and usable.
 *
 * Scroll is locked on <html> as well as focusTrap's lock on <body>, because
 * ConfirmDialog clears the body lock when it closes.
 */
import { get } from 'svelte/store';
import { confirmDialog } from '$lib/stores';
import { focusTrap } from './focusTrap';

export interface AdminModalParams {
	/** The host closes the modal. Never called while `busy`. */
	onClose: () => void;
	/** Work is in flight, so Escape leaves the modal open. */
	busy?: boolean;
}

export function adminModal(overlay: HTMLElement, params: AdminModalParams) {
	let current = params;
	const panel = overlay.querySelector<HTMLElement>('[role="dialog"]')!;
	const heading = panel.querySelector<HTMLElement>('h2, h3')!;
	const shell = document.querySelector<HTMLElement>('.admin-shell');
	const root = document.documentElement;
	const rootOverflow = root.style.overflow;

	function requestClose() {
		if (!current.busy && !get(confirmDialog).open) current.onClose();
	}

	// Focus parked on the heading sits outside focusTrap's list, so Shift+Tab would leave.
	function handleHeadingTab(event: KeyboardEvent) {
		if (event.key === 'Tab' && event.shiftKey && event.target === heading) {
			event.preventDefault();
			panel.focus();
		}
	}

	// A removed or disabled control drops focus to <body>, out of focusTrap's reach.
	function recoverFocus(event: KeyboardEvent) {
		if (document.activeElement !== document.body || get(confirmDialog).open) return;
		if (event.key === 'Escape') {
			requestClose();
		} else if (event.key === 'Tab') {
			event.preventDefault();
			panel.focus();
		}
	}

	// Runs first: focusTrap records the opener, which the inert shell would blur.
	const trap = focusTrap(panel, { autoFocus: false, onEscape: requestClose });

	// Carries the admin-only type rules out of the shell with the overlay.
	overlay.classList.add('admin-shell');
	shell?.after(overlay);
	if (shell) shell.inert = true;
	root.style.overflow = 'hidden';
	panel.tabIndex = -1;
	heading.tabIndex = -1;
	panel.addEventListener('keydown', handleHeadingTab);
	document.addEventListener('keydown', recoverFocus);

	const firstField = panel.querySelector<HTMLElement>(
		'input:not([disabled]), select:not([disabled]), textarea:not([disabled])'
	);
	(firstField ?? heading).focus();

	return {
		update(next: AdminModalParams) {
			current = next;
		},
		destroy() {
			document.removeEventListener('keydown', recoverFocus);
			root.style.overflow = rootOverflow;
			if (shell) shell.inert = false;
			// After the shell wakes, or focus can't return to the opener.
			trap.destroy();
			// Svelte may not remove a moved node: a parent block only walks its own range.
			overlay.remove();
		}
	};
}

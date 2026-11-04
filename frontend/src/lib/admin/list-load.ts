import { onDestroy, tick } from 'svelte';
import { get, writable } from 'svelte/store';
import { t } from 'svelte-i18n';
import { pb } from '$lib/pocketbase';
import { confirmDialog, toasts } from '$lib/stores';

const LIST_LOAD_TIMEOUT_MS = 15_000;

type ViewMemberships = Record<string, { id: string; name: string; slug: string }[]>;
type ListRead<T> = (requestKey: (part?: string) => string, signal: AbortSignal) => Promise<T>;

function isViewMemberships(value: unknown): value is ViewMemberships {
	if (typeof value !== 'object' || value === null || Array.isArray(value)) return false;
	return Object.values(value).every(
		(views) =>
			Array.isArray(views) &&
			views.every((view) => typeof view?.id === 'string' && typeof view?.name === 'string')
	);
}

export type ListLoadState = {
	loading: boolean;
	loaded: boolean;
	failed: boolean;
	mutating: boolean;
	membershipsKnown: boolean;
	ancillaryFailed: boolean;
};
export type ListLoader = ReturnType<typeof createListLoader>;

// Call during component init: the loader cancels its own reads on destroy.
export function createListLoader(resource: string) {
	const state = writable<ListLoadState>({
		loading: true,
		loaded: false,
		failed: false,
		mutating: false,
		membershipsKnown: false,
		ancillaryFailed: false
	});
	const sequences = new Map<string, number>();
	const cancels = new Map<string, () => void>();
	const failedAncillary = new Map<string, () => Promise<boolean>>();
	let disposed = false;
	let writing = false;
	let onLoaded = () => {};

	async function read<T>(
		name: string,
		request: ListRead<T>,
		apply: (value: T) => void,
		fail: (error: unknown) => void
	): Promise<boolean> {
		if (disposed) return false;
		cancels.get(name)?.();
		const sequence = (sequences.get(name) ?? 0) + 1;
		sequences.set(name, sequence);
		const isLatest = () => !disposed && sequences.get(name) === sequence;

		const controller = new AbortController();
		const keys = new Set<string>();
		const requestKey = (part = 'list') => {
			const key = `admin-list:${resource}:${name}:${part}`;
			keys.add(key);
			return key;
		};
		const cancel = () => {
			controller.abort();
			keys.forEach((key) => pb.cancelRequest(key));
		};
		cancels.set(name, cancel);
		const timeout = setTimeout(cancel, LIST_LOAD_TIMEOUT_MS);

		try {
			const value = await request(requestKey, controller.signal);
			if (!isLatest()) return false;
			apply(value);
			return true;
		} catch (error) {
			if (isLatest()) fail(error);
			return false;
		} finally {
			clearTimeout(timeout);
			if (cancels.get(name) === cancel) cancels.delete(name);
		}
	}

	function load<T>(request: ListRead<T>, apply: (value: T) => void) {
		state.update((current) => ({ ...current, loading: true }));
		return read(
			'primary',
			request,
			(value) => {
				apply(value);
				const recovered = get(state).failed;
				state.update((current) => ({ ...current, loading: false, loaded: true, failed: false }));
				onLoaded();
				if (recovered) restoreFocus(null);
			},
			(error) => {
				console.error(`Failed to load ${resource}:`, error);
				state.update((current) => ({ ...current, loading: false, failed: true }));
			}
		);
	}

	function loadAncillary<T>(
		name: string,
		request: ListRead<T>,
		apply: (value: T) => void
	): Promise<boolean> {
		return read(
			name,
			request,
			(value) => {
				apply(value);
				failedAncillary.delete(name);
				state.update((current) => ({ ...current, ancillaryFailed: failedAncillary.size > 0 }));
			},
			(error) => {
				console.error(`Failed to load ${resource} ${name}:`, error);
				failedAncillary.set(name, () => loadAncillary(name, request, apply));
				state.update((current) => ({ ...current, ancillaryFailed: true }));
			}
		);
	}

	function loadMemberships(collectionName: string, apply: (value: ViewMemberships) => void) {
		return loadAncillary(
			'memberships',
			async (_requestKey, signal) => {
				const response = await fetch(`/api/admin/view-memberships?collection=${collectionName}`, {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
					signal
				});
				if (!response.ok) throw new Error(`View memberships request failed (${response.status})`);
				const data = await response.json();
				if (!isViewMemberships(data?.memberships)) {
					throw new Error('Invalid view memberships response');
				}
				return data.memberships;
			},
			(value) => {
				apply(value);
				state.update((current) => ({ ...current, membershipsKnown: true }));
			}
		);
	}

	function canMutate() {
		const { loaded, failed, mutating, loading } = get(state);
		if (!disposed && loaded && !failed && !mutating && !loading) return true;
		toasts.add('error', get(t)('admin.list_status.changes_paused'));
		return false;
	}

	function setMutating(mutating: boolean) {
		state.update((current) => ({ ...current, mutating }));
	}

	function cancelRead(name: string) {
		cancels.get(name)?.();
		sequences.set(name, (sequences.get(name) ?? 0) + 1);
	}

	function retryAncillary() {
		return Promise.all([...failedAncillary.values()].map((retry) => retry()));
	}

	// Puts back focus a finished write dropped, unless the owner left this screen or a dialog is open.
	async function restoreFocus(opener: Element | null) {
		await tick();
		const active = document.activeElement;
		const focusIntact =
			active && active !== document.body && active.isConnected && !active.matches(':disabled');
		if (disposed || focusIntact || get(confirmDialog).open) return;
		const isUsable = (element: Element | null): element is HTMLElement =>
			element instanceof HTMLElement &&
			element !== document.body &&
			element.isConnected &&
			!element.matches(':disabled');
		const failedRow = document.querySelector(
			'main fieldset input[type="checkbox"]:checked:not(:disabled)'
		);
		const target = [opener, failedRow].find(isUsable);
		if (target) return target.focus();
		const heading = document.querySelector<HTMLElement>('main h1');
		heading?.setAttribute('tabindex', '-1');
		heading?.focus();
	}

	// One write at a time per page, from the first press through any confirm dialog to the
	// final reload. A repeat press while one is pending does nothing.
	function exclusive<A extends unknown[]>(run: (...args: A) => Promise<void>) {
		return async (...args: A) => {
			if (writing || !canMutate()) return;
			writing = true;
			const opener = document.activeElement;
			try {
				await run(...args);
			} finally {
				writing = false;
				await restoreFocus(opener);
			}
		};
	}

	onDestroy(() => {
		disposed = true;
		cancels.forEach((cancel) => cancel());
	});

	return {
		subscribe: state.subscribe,
		load,
		loadAncillary,
		loadMemberships,
		canMutate,
		setMutating,
		read,
		cancelRead,
		retryAncillary,
		restoreFocus,
		exclusive,
		afterLoad: (callback: () => void) => (onLoaded = callback)
	};
}

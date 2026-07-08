import { tick } from 'svelte';

export async function focusAfterRemove(opts: {
	deletedIndex: number;
	remainingIds: string[];
	selectRow: (id: string) => HTMLElement | null;
	fallbackEl?: HTMLElement | null;
}): Promise<void> {
	const { deletedIndex, remainingIds, selectRow, fallbackEl } = opts;

	if (remainingIds.length === 0) {
		await tick();
		fallbackEl?.focus();
		return;
	}

	const targetIndex = Math.min(Math.max(deletedIndex, 0), remainingIds.length - 1);
	const targetId = remainingIds[targetIndex];

	await tick();
	const targetEl = selectRow(targetId);
	if (targetEl) {
		targetEl.focus();
		return;
	}

	await tick();
	const retryEl = selectRow(targetId);
	if (retryEl) {
		retryEl.focus();
		return;
	}

	fallbackEl?.focus();
}

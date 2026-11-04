import { get } from 'svelte/store';
import { t } from 'svelte-i18n';
import type { ListLoader } from '$lib/admin/list-load';
import { pb } from '$lib/pocketbase';
import { confirm, toasts } from '$lib/stores';

const BULK_ITEM_TIMEOUT_MS = 30_000;

type Visibility = 'public' | 'unlisted' | 'private';
type BulkOutcome = { succeeded: string[]; failed: string[]; unknown: string[] };
type BulkRecords = {
	update(
		id: string,
		body: Record<string, unknown>,
		options: { requestKey: string }
	): Promise<unknown>;
	delete(id: string, options: { requestKey: string }): Promise<unknown>;
};
type BulkOptions = {
	records: () => BulkRecords;
	items: () => readonly { id: string; visibility?: string }[];
	reload: () => Promise<boolean>;
	finish: (remaining: Set<string>) => void;
};
type BulkRun = {
	prepare: () => Promise<boolean>;
	mutate: (id: string, requestKey: string) => Promise<unknown>;
	isApplied: (id: string) => boolean;
	summary: (succeeded: number, total: number) => string;
};

// A 4xx is a clear rejection. With no response or a 5xx, the write may still have landed.
function outcomeIsUnknown(error: unknown) {
	const status = (error as { status?: number }).status ?? 0;
	return status === 0 || status === 408 || status >= 500;
}

async function attemptEach(ids: string[], mutate: BulkRun['mutate']) {
	const outcome: BulkOutcome = { succeeded: [], failed: [], unknown: [] };
	for (const id of ids) {
		const requestKey = `admin-bulk:${id}`;
		const timeout = setTimeout(() => pb.cancelRequest(requestKey), BULK_ITEM_TIMEOUT_MS);
		try {
			await mutate(id, requestKey);
			outcome.succeeded.push(id);
		} catch (error) {
			console.error(`Bulk change failed for ${id}:`, error);
			(outcomeIsUnknown(error) ? outcome.unknown : outcome.failed).push(id);
		} finally {
			clearTimeout(timeout);
		}
	}
	return outcome;
}

// After an authoritative reload, each unknown request is either visibly applied or safe to retry.
function settleUnknown(outcome: BulkOutcome, isApplied: BulkRun['isApplied']): BulkOutcome {
	return {
		succeeded: [...outcome.succeeded, ...outcome.unknown.filter(isApplied)],
		failed: [...outcome.failed, ...outcome.unknown.filter((id) => !isApplied(id))],
		unknown: []
	};
}

function report(outcome: BulkOutcome, total: number, summary: BulkRun['summary']) {
	const format = get(t);
	const { succeeded, failed, unknown } = outcome;
	const parts = [summary(succeeded.length, total)];
	if (failed.length) {
		parts.push(format('admin.bulk_actions.result_failed', { values: { count: failed.length } }));
	}
	if (unknown.length) {
		parts.push(
			format('admin.bulk_actions.result_unconfirmed', { values: { count: unknown.length } })
		);
	}
	toasts.add(failed.length || unknown.length ? 'error' : 'success', parts.join(' '));
}

// Writes only the positions the last list read doesn't already show, then re-reads the list.
// False means some are still unsaved, so the caller keeps the owner's order for another save.
export async function saveSortOrders(
	records: () => Pick<BulkRecords, 'update'>,
	items: () => readonly { id: string; sort_order?: number }[],
	targets: { id: string; sort_order: number }[],
	reload: () => Promise<boolean>
) {
	const position = new Map(targets.map((target) => [target.id, target.sort_order]));
	const isSaved = (id: string) => {
		const item = items().find((candidate) => candidate.id === id);
		return !item || item.sort_order === position.get(id);
	};
	const pending = [...position.keys()].filter((id) => !isSaved(id));
	const { succeeded } = await attemptEach(pending, (id, requestKey) =>
		records().update(id, { sort_order: position.get(id) }, { requestKey })
	);
	const reloaded = await reload();
	const saved = reloaded ? pending.filter(isSaved).length : succeeded.length;
	if (saved === pending.length) return true;
	toasts.add(
		'error',
		get(t)(reloaded ? 'admin.list_status.order_partial' : 'admin.list_status.order_unconfirmed', {
			values: { saved, total: pending.length }
		})
	);
	return false;
}

export function createBulkActions(list: ListLoader, options: BulkOptions) {
	// Held when a batch ends unconfirmed, then settled by the next good list read.
	let unconfirmed: { outcome: BulkOutcome; total: number; batch: BulkRun } | null = null;

	list.afterLoad(() => {
		if (!unconfirmed) return;
		const { outcome, total, batch } = unconfirmed;
		unconfirmed = null;
		const settled = settleUnknown(outcome, batch.isApplied);
		options.finish(new Set(settled.failed));
		report(settled, total, batch.summary);
	});

	const run = list.exclusive(async (ids: string[], batch: BulkRun) => {
		if (ids.length === 0 || !(await batch.prepare())) return;
		list.setMutating(true);
		try {
			const attempted = await attemptEach(ids, batch.mutate);
			const reloaded = await options.reload();
			const outcome = reloaded ? settleUnknown(attempted, batch.isApplied) : attempted;
			options.finish(new Set([...outcome.failed, ...outcome.unknown]));
			report(outcome, ids.length, batch.summary);
			if (outcome.unknown.length) unconfirmed = { outcome, total: ids.length, batch };
		} finally {
			list.setMutating(false);
		}
	});

	function setVisibility(ids: string[], visibility: Visibility) {
		const format = get(t);
		const record = (id: string) => options.items().find((item) => item.id === id);
		const widened = ids.filter((id) => record(id)?.visibility !== 'public').length;
		return run(ids, {
			prepare: async () =>
				visibility !== 'public' ||
				widened === 0 ||
				confirm({
					title: format('admin.bulk_actions.confirm_public_title'),
					message: format('admin.bulk_actions.confirm_public_message', {
						values: { count: widened }
					}),
					confirmText: format('admin.bulk_actions.confirm_public_button')
				}),
			mutate: (id, requestKey) => options.records().update(id, { visibility }, { requestKey }),
			isApplied: (id) => record(id)?.visibility === visibility,
			summary: (succeeded, total) =>
				format('admin.bulk_actions.result_updated', {
					values: {
						succeeded,
						total,
						visibility: format(`admin.content.common.visibility_${visibility}`)
					}
				})
		});
	}

	function deleteItems(ids: string[], prepare: () => Promise<boolean>) {
		const format = get(t);
		return run(ids, {
			prepare,
			mutate: (id, requestKey) => options.records().delete(id, { requestKey }),
			isApplied: (id) => !options.items().some((item) => item.id === id),
			summary: (succeeded, total) =>
				format('admin.bulk_actions.result_deleted', { values: { succeeded, total } })
		});
	}

	return { setVisibility, delete: deleteItems };
}

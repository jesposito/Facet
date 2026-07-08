import { expect, test } from '@playwright/test';
import { focusAfterRemove } from '../src/lib/a11y/focusAfterRemove';

function focusable(id: string, focused: string[]): HTMLElement {
	return {
		focus: () => {
			focused.push(id);
		}
	} as unknown as HTMLElement;
}

test.describe('focusAfterRemove', () => {
	test('focuses the next row after a middle item is removed', async () => {
		const focused: string[] = [];
		const rows = new Map([
			['a', focusable('a', focused)],
			['c', focusable('c', focused)]
		]);

		await focusAfterRemove({
			deletedIndex: 1,
			remainingIds: ['a', 'c'],
			selectRow: (id) => rows.get(id) ?? null
		});

		expect(focused).toEqual(['c']);
	});

	test('focuses the previous row after the last item is removed', async () => {
		const focused: string[] = [];
		const rows = new Map([['a', focusable('a', focused)]]);

		await focusAfterRemove({
			deletedIndex: 1,
			remainingIds: ['a'],
			selectRow: (id) => rows.get(id) ?? null
		});

		expect(focused).toEqual(['a']);
	});

	test('falls back when no rows remain', async () => {
		const focused: string[] = [];

		await focusAfterRemove({
			deletedIndex: 0,
			remainingIds: [],
			selectRow: () => null,
			fallbackEl: focusable('fallback', focused)
		});

		expect(focused).toEqual(['fallback']);
	});
});

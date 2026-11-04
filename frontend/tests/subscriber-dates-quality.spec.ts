import { expect, test } from '@playwright/test';
import { adminEmail, adminPassword } from './config';
import { seedAdminSession } from './helpers';

const KNOWN_ID = 'subdateknown001';
const UNKNOWN_ID = 'subdateblank001';

test.describe('Subscriber dates', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	test('unknown dates show a dash and the sidebar badge loads', async ({ page, request }) => {
		await seedAdminSession(page, request);
		// Dummy rows only. The API can't create a blank created value, and this
		// test never touches the real subscriber list.
		await page.route(
			(url) => url.pathname === '/api/admin/subscribers',
			(route) =>
				route.fulfill({
					json: {
						data: [
							{
								id: KNOWN_ID,
								email: 'known@example.test',
								name: 'Known',
								source: 'test',
								status: 'active',
								created: '2026-06-15 12:00:00.000Z'
							},
							{
								id: UNKNOWN_ID,
								email: 'unknown@example.test',
								name: 'Unknown',
								source: 'test',
								status: 'active',
								created: ''
							}
						],
						totalPages: 1,
						totalItems: 2
					}
				})
		);
		const badge = page.waitForResponse((response) => {
			const url = new URL(response.url());
			return (
				url.pathname === '/api/collections/subscribers/records' &&
				url.searchParams.get('filter')?.startsWith('created >=') === true
			);
		});

		await page.goto('/admin/subscribers');

		expect((await badge).status()).toBe(200);
		const dateCell = (id: string) => page.locator(`tr[data-row-id="${id}"] td`).nth(5);
		await expect(dateCell(KNOWN_ID)).toContainText('2026');
		await expect(dateCell(UNKNOWN_ID)).toHaveText('—');
		await expect(page.getByRole('main')).not.toContainText('Invalid Date');
	});
});

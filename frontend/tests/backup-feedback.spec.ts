import { test, expect, type Page, type Route } from '@playwright/test';
import { seedAdminSession } from './helpers';

// Manual backup feedback on Settings → About. Every backup endpoint is mocked
// with page.route, so these tests never create, restore or delete a real
// archive. Only the admin session comes from the fixture server.

const EXISTING = {
	filename: 'facet_daily_20261102_030000.zip',
	size: 2048,
	created: '2026-11-02T03:00:00Z'
};

const STATUS_OK = {
	last_backup: EXISTING,
	backups: [EXISTING],
	total_count: 1,
	backup_dir: '/data/backups'
};

const INTERNAL_DETAIL =
	'Backup failed: open /data/secret/pb_data: permission denied at stack.go:42';

type Calls = { status: number; backup: number; restore: number; del: number };

async function mockBackupAPI(
	page: Page,
	opts: {
		status?: (call: number) => { code: number; body: unknown };
		backup?: (route: Route) => Promise<void>;
	}
) {
	const calls: Calls = { status: 0, backup: 0, restore: 0, del: 0 };
	// Catch-all first: Playwright tries the most recently added route first, so
	// the specific handlers below win and anything else under /backup is refused.
	await page.route('**/api/admin/backup/**', async (route) => {
		const method = route.request().method();
		if (method === 'DELETE') calls.del++;
		if (route.request().url().includes('/restore')) calls.restore++;
		await route.fulfill({ status: 418, json: { error: 'unexpected call in test' } });
	});
	await page.route('**/api/admin/backup/status', async (route) => {
		calls.status++;
		const res = opts.status ? opts.status(calls.status) : { code: 200, body: STATUS_OK };
		await route.fulfill({ status: res.code, json: res.body });
	});
	await page.route('**/api/admin/backup', async (route) => {
		if (route.request().method() !== 'POST') return route.fallback();
		calls.backup++;
		if (opts.backup) return opts.backup(route);
		await route.fulfill({ status: 500, json: { error: INTERNAL_DETAIL } });
	});
	return calls;
}

async function openAbout(page: Page) {
	await page.goto('/admin/settings/about');
	await expect(page.getByRole('button', { name: 'Backup Now' })).toBeVisible();
}

test.describe('Backup feedback (About page)', () => {
	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
	});

	test('failed backup keeps the list, says retry, hides server detail', async ({ page }) => {
		const calls = await mockBackupAPI(page, {});
		await openAbout(page);
		await expect(page.getByText(EXISTING.filename)).toBeVisible();

		await page.getByRole('button', { name: 'Backup Now' }).click();

		const alert = page.getByRole('alert').filter({ hasText: "The backup didn't finish" });
		await expect(alert).toContainText('try again');
		await expect(page.getByText(EXISTING.filename)).toBeVisible();
		await expect(page.locator('body')).not.toContainText('permission denied');
		await expect(page.locator('body')).not.toContainText('stack.go');
		expect(calls.backup).toBe(1);
		// No status reload after a failure: the existing list stays as it was.
		expect(calls.status).toBe(1);
		expect(calls.restore + calls.del).toBe(0);
	});

	test('busy backup (409) says another operation is running', async ({ page }) => {
		const calls = await mockBackupAPI(page, {
			backup: (route) =>
				route.fulfill({
					status: 409,
					json: { error: 'Another backup or restore operation is already in progress.' }
				})
		});
		await openAbout(page);

		await page.getByRole('button', { name: 'Backup Now' }).click();

		await expect(
			page.getByRole('alert').filter({ hasText: 'Another backup or restore is already running' })
		).toContainText('try again');
		await expect(page.getByText(EXISTING.filename)).toBeVisible();
		expect(calls.restore + calls.del).toBe(0);
	});

	test('network failure shows the retry message', async ({ page }) => {
		await mockBackupAPI(page, { backup: (route) => route.abort('failed') });
		await openAbout(page);

		await page.getByRole('button', { name: 'Backup Now' }).click();

		await expect(
			page.getByRole('alert').filter({ hasText: "The backup didn't finish" })
		).toBeVisible();
		await expect(page.getByText(EXISTING.filename)).toBeVisible();
	});

	test('successful backup reloads status and confirms', async ({ page }) => {
		const created = {
			filename: 'facet_manual_20261103_101500.zip',
			size: 4096,
			created: '2026-11-03T10:15:00Z'
		};
		const calls = await mockBackupAPI(page, {
			status: (n) =>
				n === 1
					? { code: 200, body: STATUS_OK }
					: {
							code: 200,
							body: {
								...STATUS_OK,
								last_backup: created,
								backups: [created, EXISTING],
								total_count: 2
							}
						},
			backup: (route) =>
				route.fulfill({
					status: 200,
					json: { filename: created.filename, size: 4096, duration: 12 }
				})
		});
		await openAbout(page);

		await page.getByRole('button', { name: 'Backup Now' }).click();

		await expect(page.getByRole('status').filter({ hasText: 'Backup created.' })).toBeVisible();
		await expect(page.getByText(created.filename).first()).toBeVisible();
		expect(calls.status).toBe(2);
	});

	test('retention warning reports success plus cleanup failure and links alerts', async ({
		page
	}) => {
		await mockBackupAPI(page, {
			backup: (route) =>
				route.fulfill({
					status: 200,
					json: {
						filename: 'facet_manual_20261103_101500.zip',
						size: 4096,
						duration: 12,
						retention_warning:
							'Backup created, but old backups could not be pruned. See System Alerts.'
					}
				})
		});
		await openAbout(page);

		await page.getByRole('button', { name: 'Backup Now' }).click();

		const warning = page.getByRole('status').filter({ hasText: 'removing old backups failed' });
		await expect(warning).toContainText('Backup created');
		await expect(warning.getByRole('link', { name: 'View System Alerts' })).toHaveAttribute(
			'href',
			'/admin/alerts'
		);
	});

	test('double click sends one backup request', async ({ page }) => {
		let release: () => void = () => {};
		const gate = new Promise<void>((resolve) => (release = resolve));
		const calls = await mockBackupAPI(page, {
			backup: async (route) => {
				await gate;
				await route.fulfill({ status: 200, json: { filename: 'x.zip', size: 1, duration: 1 } });
			}
		});
		await openAbout(page);

		const button = page.getByRole('button', { name: 'Backup Now' });
		await button.dblclick();
		await expect(page.getByRole('button', { name: 'Backing up...' })).toBeDisabled();
		release();
		await expect(page.getByRole('status').filter({ hasText: 'Backup created.' })).toBeVisible();
		expect(calls.backup).toBe(1);
	});

	test('failed status load offers retry instead of showing zero or never', async ({ page }) => {
		const calls = await mockBackupAPI(page, {
			status: (n) =>
				n === 1
					? { code: 500, body: { error: 'database is locked' } }
					: { code: 200, body: STATUS_OK }
		});
		await page.goto('/admin/settings/about');

		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load backup status" });
		await expect(alert).toBeVisible();
		await expect(page.getByText('Never', { exact: true })).toHaveCount(0);
		await expect(page.locator('body')).not.toContainText('database is locked');

		await alert.getByRole('button', { name: 'Retry' }).click();

		await expect(alert).toHaveCount(0);
		await expect(page.getByText(EXISTING.filename)).toBeVisible();
		expect(calls.status).toBe(2);
		expect(calls.restore + calls.del).toBe(0);
	});
});

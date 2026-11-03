import { test, expect, type Page } from '@playwright/test';
import { seedAdminSession } from './helpers';

// Exercises the ported live-site preview: a header toggle (role=switch) that
// opens a side column with a sandboxed iframe of the public homepage, with the
// state persisted in localStorage. Desktop only (lg+).
const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';

const toggle = (page: Page) => page.getByRole('switch', { name: 'Toggle live preview' });
const previewFrame = (page: Page) =>
	page.locator('iframe[title="Live preview of your public site"]');

test.describe('Live site preview', () => {
	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
		await page.goto(`${BASE_URL}/admin`);
	});

	test.afterEach(async ({ page }) => {
		// Leave the preference off for the next test/file.
		const t = toggle(page);
		if ((await t.count()) > 0 && (await t.getAttribute('aria-checked')) === 'true') {
			await t.click();
		}
	});

	test('toggle is an accessible switch, off by default', async ({ page }) => {
		const t = toggle(page);
		await expect(t).toBeVisible();
		await expect(t).toHaveAttribute('aria-checked', 'false');
		// No preview iframe until enabled.
		await expect(previewFrame(page)).toHaveCount(0);
	});

	test('enabling shows the sandboxed live-site iframe', async ({ page }) => {
		await toggle(page).click();
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'true');
		const frame = previewFrame(page);
		await expect(frame).toBeVisible();
		await expect(frame).toHaveAttribute('src', /\/\?preview=1/);
		// Sealed from tab order (read-only pane).
		await expect(frame).toHaveAttribute('tabindex', '-1');
	});

	test('preference persists across a reload (localStorage)', async ({ page }) => {
		await toggle(page).click();
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'true');
		await page.reload();
		await page.waitForURL(/\/admin/);
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'true');
		await expect(previewFrame(page)).toBeVisible();
	});

	test('disabling hides the preview column', async ({ page }) => {
		await toggle(page).click();
		await expect(previewFrame(page)).toBeVisible();
		await toggle(page).click();
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'false');
		await expect(previewFrame(page)).toHaveCount(0);
	});
});

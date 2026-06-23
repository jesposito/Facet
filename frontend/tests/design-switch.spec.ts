import { test, expect, type Page } from '@playwright/test';

// Exercises the real /admin product: the opt-in Visual Design switch on the
// site settings page, and its effect on the rendered <html data-design> attribute.
const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';
// Defaults match the `SEED_DATA=dev` seed; override via env for other instances.
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

async function login(page: Page) {
	await page.goto(`${BASE_URL}/admin/login`);
	await page.waitForSelector('input[type="email"]', { timeout: 10_000 });
	await page.fill('input[type="email"]', LOGIN_EMAIL);
	await page.fill('input[type="password"]', LOGIN_PASSWORD);
	await page.click('button[type="submit"]');
	// Auth success reactively redirects to the /admin dashboard. Wait for the
	// login route to actually go away (not just any /admin* URL, which the login
	// page itself matches).
	await page.waitForURL((url) => /\/admin(\/|$)/.test(url.pathname) && !url.pathname.includes('/login'), {
		timeout: 15_000
	});
}

async function gotoSettings(page: Page) {
	await page.goto(`${BASE_URL}/admin/settings/site`);
	await page.waitForSelector('[aria-labelledby="design-style-heading"]', { timeout: 10_000 });
}

test.describe('Visual Design switch (/admin product)', () => {
	test.beforeEach(async ({ page }) => {
		await login(page);
		await gotoSettings(page);
	});

	test.afterEach(async ({ page }) => {
		// Always restore Classic AND wait for the server to persist it, so the
		// next spec file (sharing this instance) starts from a clean classic
		// state instead of racing an unsaved change.
		const classic = page.locator('[role="radio"]', { hasText: 'Classic' }).first();
		if ((await classic.getAttribute('aria-checked')) === 'false') {
			const saved = page.waitForResponse(
				(r) =>
					r.url().includes('/api/site-settings') &&
					r.request().method() === 'PUT' &&
					r.ok()
			);
			await classic.click();
			await saved;
		}
	});

	test('renders an accessible radiogroup with Classic selected by default', async ({ page }) => {
		const group = page.locator('[role="radiogroup"][aria-labelledby="design-style-heading"]');
		await expect(group).toBeVisible();
		const radios = group.locator('[role="radio"]');
		await expect(radios).toHaveCount(2);
		// Exactly one selected; it is Classic.
		await expect(group.locator('[role="radio"][aria-checked="true"]')).toHaveCount(1);
		const classic = radios.filter({ hasText: 'Classic' }).first();
		await expect(classic).toHaveAttribute('aria-checked', 'true');
		// Roving tabindex: the selected radio is the tab stop.
		await expect(classic).toHaveAttribute('tabindex', '0');
	});

	test('selecting Soft Premium adds data-design and persists', async ({ page }) => {
		const soft = page.locator('[role="radio"]', { hasText: 'Soft Premium' }).first();
		await soft.click();
		await expect(soft).toHaveAttribute('aria-checked', 'true');
		// Product effect: the html attribute the warm token layer keys off of.
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
		// Persists across a full reload (SSR re-injects it).
		await page.reload();
		await page.waitForSelector('[aria-labelledby="design-style-heading"]', { timeout: 10_000 });
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
	});

	test('Classic removes the data-design attribute entirely', async ({ page }) => {
		// Switch to soft-premium first, then back.
		await page.locator('[role="radio"]', { hasText: 'Soft Premium' }).first().click();
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
		await page.locator('[role="radio"]', { hasText: 'Classic' }).first().click();
		// Classic must emit NO attribute (byte-identical to pre-redesign output).
		await expect(page.locator('html')).not.toHaveAttribute('data-design', /.*/);
	});

	test('keyboard: arrow keys move selection and toggle the attribute', async ({ page }) => {
		const classic = page.locator('[role="radio"]', { hasText: 'Classic' }).first();
		const soft = page.locator('[role="radio"]', { hasText: 'Soft Premium' }).first();
		await classic.focus();
		await expect(classic).toBeFocused();
		// ArrowRight -> Soft Premium (selection follows focus).
		await page.keyboard.press('ArrowRight');
		await expect(soft).toBeFocused();
		await expect(soft).toHaveAttribute('aria-checked', 'true');
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
		// ArrowLeft -> back to Classic, attribute removed.
		await page.keyboard.press('ArrowLeft');
		await expect(classic).toBeFocused();
		await expect(page.locator('html')).not.toHaveAttribute('data-design', /.*/);
	});

	test('public site reflects the chosen design', async ({ page, context }) => {
		await page.locator('[role="radio"]', { hasText: 'Soft Premium' }).first().click();
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
		// A fresh visitor (new tab, no admin session) sees the SSR-injected attribute.
		const visitor = await context.newPage();
		await visitor.goto(`${BASE_URL}/`);
		await expect(visitor.locator('html')).toHaveAttribute('data-design', 'soft-premium');
		await visitor.close();
	});
});

import { test, expect, type Page } from '@playwright/test';

// B4: Soft Premium defaults the accent to warm terracotta when the operator
// hasn't set one; classic keeps the static sky default. (Assumes the seeded
// profile has no explicit accent — the dev seed does not.)
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';
const TERRACOTTA = '#c2410c';

async function login(page: Page) {
	await page.goto(`${BASE_URL}/admin/login`);
	await page.waitForSelector('input[type="email"]', { timeout: 10_000 });
	await page.fill('input[type="email"]', LOGIN_EMAIL);
	await page.fill('input[type="password"]', LOGIN_PASSWORD);
	await page.click('button[type="submit"]');
	await page.waitForURL((url) => /\/admin(\/|$)/.test(url.pathname) && !url.pathname.includes('/login'), {
		timeout: 15_000
	});
}
async function setDesign(page: Page, label: 'Classic' | 'Soft Premium') {
	await page.goto(`${BASE_URL}/admin/settings/site`);
	await page.waitForSelector('[aria-labelledby="design-style-heading"]', { timeout: 10_000 });
	const radio = page.locator('[role="radio"]', { hasText: label }).first();
	if ((await radio.getAttribute('aria-checked')) === 'false') {
		await radio.click();
		await expect(radio).toHaveAttribute('aria-checked', 'true');
	}
}
const accent500 = (page: Page) =>
	page.evaluate(() =>
		getComputedStyle(document.documentElement).getPropertyValue('--color-primary-500').trim().toLowerCase()
	);

test.describe('Soft Premium default accent', () => {
	test.beforeEach(async ({ page }) => login(page));
	test.afterEach(async ({ page }) => setDesign(page, 'Classic'));

	test('soft-premium defaults to terracotta when no operator accent', async ({ page }) => {
		await setDesign(page, 'Soft Premium');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		expect(await accent500(page)).toBe(TERRACOTTA);
	});

	test('classic keeps the sky default accent', async ({ page }) => {
		await setDesign(page, 'Classic');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		expect(await accent500(page)).not.toBe(TERRACOTTA);
		expect(await accent500(page)).toBe('#0ea5e9');
	});
});

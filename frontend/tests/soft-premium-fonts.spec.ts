import { test, expect, type Page } from '@playwright/test';

// B3: soft-premium swaps the type to Hanken Grotesk; classic keeps Plus Jakarta.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

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
const bodyFont = (page: Page) =>
	page.evaluate(() => getComputedStyle(document.body).fontFamily.toLowerCase());

test.describe('Soft Premium fonts', () => {
	test.beforeEach(async ({ page }) => login(page));
	test.afterEach(async ({ page }) => setDesign(page, 'Classic'));

	test('classic uses Plus Jakarta (no Hanken)', async ({ page }) => {
		await setDesign(page, 'Classic');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		expect(font).toContain('jakarta');
		expect(font).not.toContain('hanken');
	});

	test('soft-premium leads with Hanken Grotesk', async ({ page }) => {
		await setDesign(page, 'Soft Premium');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		expect(font).toContain('hanken');
		// Hanken is the primary (precedes the Jakarta fallback).
		expect(font.indexOf('hanken')).toBeLessThan(font.indexOf('jakarta'));
	});
});

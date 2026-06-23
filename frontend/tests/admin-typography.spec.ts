import { test, expect, type Page } from '@playwright/test';

// A12: admin headings use the sans body font, not the public editorial serif.
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

test('public headings stay serif; admin headings are sans', async ({ page }) => {
	await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
	const publicH1 = await page.evaluate(() => {
		const h = document.querySelector('h1');
		return h ? getComputedStyle(h).fontFamily : '';
	});
	// Public editorial serif (Lora) is preserved.
	expect(publicH1.toLowerCase()).toContain('lora');

	await login(page);
	await page.goto(`${BASE_URL}/admin/experience`, { waitUntil: 'domcontentloaded' });
	await page.waitForSelector('.admin-shell h1', { timeout: 10_000 });
	const adminH1 = await page.evaluate(() => {
		const h = document.querySelector('.admin-shell h1');
		return h ? getComputedStyle(h).fontFamily : '';
	});
	// Admin pinned to the sans body stack — not the serif.
	expect(adminH1.toLowerCase()).toContain('jakarta');
	expect(adminH1.toLowerCase()).not.toContain('lora');
});

import { test, expect, type Page } from '@playwright/test';

// Verifies the B1 token layer through the real product: classic renders the
// stock Tailwind gray ramp (byte-identical), soft-premium warms the whole ramp
// to the luminance-matched stone values, and a fresh public visitor sees it.
const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

// Classic = Tailwind v3.4 stock gray channels. Soft-premium = warm stone.
const STOCK = { '--gray-100-rgb': '243 244 246', '--gray-800-rgb': '31 41 55' };
const STONE = { '--gray-100-rgb': '246 244 237', '--gray-800-rgb': '42 40 40' };

async function readVars(page: Page) {
	return page.evaluate(() => ({
		'--gray-100-rgb': getComputedStyle(document.documentElement)
			.getPropertyValue('--gray-100-rgb')
			.trim(),
		'--gray-800-rgb': getComputedStyle(document.documentElement)
			.getPropertyValue('--gray-800-rgb')
			.trim()
	}));
}

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
		// Wait for the server PUT to persist before returning — a following
		// page.goto() reads design from the server (SSR) and would otherwise
		// race the not-yet-saved change, contaminating cross-file batch runs.
		const saved = page.waitForResponse(
			(r) =>
				r.url().includes('/api/site-settings') &&
				r.request().method() === 'PUT' &&
				r.ok()
		);
		await radio.click();
		await saved;
		await expect(radio).toHaveAttribute('aria-checked', 'true');
	}
}

test.describe('Soft Premium token layer', () => {
	test.beforeEach(async ({ page }) => {
		await login(page);
	});

	test.afterEach(async ({ page }) => {
		await setDesign(page, 'Classic');
	});

	test('classic renders the stock gray ramp', async ({ page }) => {
		await setDesign(page, 'Classic');
		await page.goto(`${BASE_URL}/`);
		expect(await readVars(page)).toEqual(STOCK);
	});

	test('soft-premium warms the whole gray ramp', async ({ page }) => {
		await setDesign(page, 'Soft Premium');
		await page.goto(`${BASE_URL}/`);
		expect(await readVars(page)).toEqual(STONE);
		// A real surface using a gray utility must resolve to the warm value.
		const probe = await page.evaluate(() => {
			const el = document.createElement('div');
			el.className = 'bg-gray-100';
			document.body.appendChild(el);
			const bg = getComputedStyle(el).backgroundColor;
			el.remove();
			return bg;
		});
		// rgb(246, 244, 237) — warm stone-100, not cool gray rgb(243, 244, 246).
		expect(probe.replace(/\s/g, '')).toBe('rgb(246,244,237)');
	});
});

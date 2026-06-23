import { test, expect, type Page } from '@playwright/test';

// Soft Premium is always on and defaults the accent to a warm terracotta when the
// operator hasn't set one (the dev seed has no explicit accent). An operator
// accent always wins; this just verifies the always-on default.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const TERRACOTTA = '#c2410c';

const accent500 = (page: Page) =>
	page.evaluate(() =>
		getComputedStyle(document.documentElement).getPropertyValue('--color-primary-500').trim().toLowerCase()
	);

test.describe('Soft Premium default accent (always on)', () => {
	test('defaults to terracotta when no operator accent', async ({ page }) => {
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		expect(await accent500(page)).toBe(TERRACOTTA);
	});
});

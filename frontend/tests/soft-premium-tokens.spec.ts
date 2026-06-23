import { test, expect, type Page } from '@playwright/test';

// Soft Premium is always on. The token layer warms the entire gray ramp to the
// luminance-matched stone values for every visitor — no opt-in, no classic mode.
const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';

// Warm stone channels (replace Tailwind's cool stock gray).
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

test.describe('Soft Premium token layer (always on)', () => {
	test('a fresh public page carries data-design and the warm stone ramp', async ({ page }) => {
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		// The master gate is always present.
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
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

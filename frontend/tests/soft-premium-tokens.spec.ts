import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { setSiteDesign } from './helpers';

// The operator can switch between Classic and Soft Premium. Classic restores the
// old cool gray/blue defaults; Soft Premium warms the neutral ramp.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';
const API_BASE = process.env.API_BASE_URL ?? process.env.POCKETBASE_URL ?? 'http://localhost:8090';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

// Warm stone channels (replace Tailwind's cool stock gray).
const STONE = { '--gray-100-rgb': '246 244 237', '--gray-800-rgb': '42 40 40' };
const CLASSIC = {
	'--gray-100-rgb': '243 244 246',
	'--gray-800-rgb': '31 41 55',
};

async function adminToken(request: APIRequestContext): Promise<string> {
	const res = await request.post(`${API_BASE}/api/collections/users/auth-with-password`, {
		data: { identity: LOGIN_EMAIL, password: LOGIN_PASSWORD },
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).token as string;
}

async function readVars(page: Page) {
	return page.evaluate(() => ({
		'--gray-100-rgb': getComputedStyle(document.documentElement).getPropertyValue('--gray-100-rgb').trim(),
		'--gray-800-rgb': getComputedStyle(document.documentElement).getPropertyValue('--gray-800-rgb').trim(),
	}));
}

test.describe('design token layer', () => {
	test.afterEach(async ({ request }) => {
		const token = await adminToken(request);
		await setSiteDesign(request, token, 'classic');
	});

	test('Soft Premium carries data-design and the warm stone ramp', async ({ page, request }) => {
		const token = await adminToken(request);
		await setSiteDesign(request, token, 'soft-premium');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		await expect(page.locator('html')).toHaveAttribute('data-design', 'soft-premium');
		expect(await readVars(page)).toEqual(STONE);

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

	test('Classic removes the design gate and restores the old cool gray ramp', async ({ page, request }) => {
		const token = await adminToken(request);
		await setSiteDesign(request, token, 'classic');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		expect(await page.locator('html').getAttribute('data-design')).toBeNull();
		expect(await readVars(page)).toEqual(CLASSIC);

		const probe = await page.evaluate(() => {
			const gray = document.createElement('div');
			const stone = document.createElement('div');
			gray.className = 'bg-gray-100';
			stone.className = 'bg-stone-100';
			document.body.append(gray, stone);
			const result = {
				gray: getComputedStyle(gray).backgroundColor,
				stone: getComputedStyle(stone).backgroundColor,
			};
			gray.remove();
			stone.remove();
			return result;
		});

		expect(probe.gray.replace(/\s/g, '')).toBe('rgb(243,244,246)');
		expect(probe.stone.replace(/\s/g, '')).toBe('rgb(243,244,246)');
	});
});

import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { patchFirstProfile, setSiteDesign } from './helpers';

// Soft Premium defaults the accent to a warm terracotta when the operator hasn't
// set one. Classic keeps the old sky accent.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const API_BASE = process.env.API_BASE_URL ?? process.env.POCKETBASE_URL ?? 'http://localhost:8090';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';
const TERRACOTTA = '#c2410c';
const SKY = '#0ea5e9';

async function adminToken(request: APIRequestContext): Promise<string> {
	const res = await request.post(`${API_BASE}/api/collections/users/auth-with-password`, {
		data: { identity: LOGIN_EMAIL, password: LOGIN_PASSWORD },
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).token as string;
}

const accent500 = (page: Page) =>
	page.evaluate(() =>
		getComputedStyle(document.documentElement).getPropertyValue('--color-primary-500').trim().toLowerCase(),
	);

test.describe('design default accent', () => {
	test.afterEach(async ({ request }) => {
		const token = await adminToken(request);
		await patchFirstProfile(request, token, {
			accent_color: '',
			custom_hex_color: '',
		});
		await setSiteDesign(request, token, 'classic');
	});

	test('Soft Premium defaults to terracotta when no operator accent', async ({ page, request }) => {
		const token = await adminToken(request);
		await patchFirstProfile(request, token, {
			accent_color: '',
			custom_hex_color: '',
		});
		await setSiteDesign(request, token, 'soft-premium');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		expect(await accent500(page)).toBe(TERRACOTTA);
	});

	test('Classic defaults to the old sky accent when no operator accent', async ({ page, request }) => {
		const token = await adminToken(request);
		await patchFirstProfile(request, token, {
			accent_color: '',
			custom_hex_color: '',
		});
		await setSiteDesign(request, token, 'classic');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		expect(await accent500(page)).toBe(SKY);
	});
});

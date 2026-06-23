import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

// Soft Premium is always on. The font pack picker fully governs the typeface:
// an unconfigured instance (no profile.font_pack) falls back to the Soft Premium
// default pack (Hanken Grotesk); selecting any other pack is honored verbatim.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const API_BASE = process.env.API_BASE_URL ?? process.env.POCKETBASE_URL ?? 'http://localhost:8090';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

async function adminToken(request: APIRequestContext): Promise<string> {
	const res = await request.post(`${API_BASE}/api/collections/users/auth-with-password`, {
		data: { identity: LOGIN_EMAIL, password: LOGIN_PASSWORD }
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).token as string;
}

async function setFontPack(request: APIRequestContext, token: string, pack: string) {
	const list = await request
		.get(`${API_BASE}/api/collections/profile/records?perPage=1`, { headers: { Authorization: token } })
		.then((r) => r.json());
	const id = list.items[0].id as string;
	const res = await request.patch(`${API_BASE}/api/collections/profile/records/${id}`, {
		headers: { Authorization: token },
		data: { font_pack: pack }
	});
	expect(res.ok()).toBeTruthy();
}

const bodyFont = (page: Page) =>
	page.evaluate(() => getComputedStyle(document.body).fontFamily.toLowerCase());

test.describe('Soft Premium fonts (font selector governs)', () => {
	// Always leave the instance in its default (unset) font_pack state.
	test.afterEach(async ({ request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, '');
	});

	test('default (no operator pack) leads with Hanken Grotesk', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, '');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		expect(font).toContain('hanken');
	});

	test('selecting a non-default pack (modern) is honored — body becomes Inter', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'modern');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		// modern pack body = Inter; Hanken must NOT be present.
		expect(font).toContain('inter');
		expect(font).not.toContain('hanken');
	});

	test('selecting editorial (Lora/Jakarta) is honored over the Soft Premium default', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'editorial');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		// editorial pack body = Plus Jakarta Sans; the default Soft Premium Hanken
		// must not leak through.
		expect(font).toContain('jakarta');
		expect(font).not.toContain('hanken');
	});

	test('resetting to the default pack restores Hanken', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'modern');
		await setFontPack(request, token, '');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		expect(font).toContain('hanken');
	});
});

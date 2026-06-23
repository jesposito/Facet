import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

// Admin headings use the sans body font, not the public heading face. We prove
// this with a pack whose heading ≠ body (editorial: Lora heading / Jakarta body):
// the public h1 follows the selected pack's heading, while admin headings stay
// pinned to the body stack via the .admin-shell rule.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const API_BASE = process.env.API_BASE_URL ?? process.env.POCKETBASE_URL ?? 'http://localhost:8090';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

async function adminAuth(request: APIRequestContext): Promise<string> {
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

test.describe('Admin typography', () => {
	// Restore the default (unset) font_pack so other suites see Soft Premium.
	test.afterEach(async ({ request }) => {
		const token = await adminAuth(request);
		await setFontPack(request, token, '');
	});

	test('public headings follow the selected pack; admin headings stay on the body stack', async ({ page, request }) => {
		const token = await adminAuth(request);
		// editorial: Lora headings, Plus Jakarta Sans body — heading ≠ body.
		await setFontPack(request, token, 'editorial');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		// Public heading face follows the selected pack. Read --font-heading
		// directly: the hero name uses the editorial --font-accent (Newsreader)
		// flourish, so a raw `h1` lookup would read the accent face, not the pack
		// heading. The resolved --font-heading is what the pack actually governs.
		const headingVar = await page.evaluate(() =>
			getComputedStyle(document.documentElement).getPropertyValue('--font-heading').trim().toLowerCase()
		);
		// editorial pack heading = Lora.
		expect(headingVar).toContain('lora');

		await login(page);
		await page.goto(`${BASE_URL}/admin/experience`, { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('.admin-shell h1', { timeout: 10_000 });
		const adminH1 = await page.evaluate(() => {
			const h = document.querySelector('.admin-shell h1');
			return h ? getComputedStyle(h).fontFamily : '';
		});
		// Admin pinned to the sans body stack — not the serif heading face.
		expect(adminH1.toLowerCase()).toContain('jakarta');
		expect(adminH1.toLowerCase()).not.toContain('lora');
	});

	test('under the default Soft Premium pack, admin headings use Hanken (the body stack)', async ({ page, request }) => {
		const token = await adminAuth(request);
		await setFontPack(request, token, '');

		await login(page);
		await page.goto(`${BASE_URL}/admin/experience`, { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('.admin-shell h1', { timeout: 10_000 });
		const adminH1 = await page.evaluate(() => {
			const h = document.querySelector('.admin-shell h1');
			return h ? getComputedStyle(h).fontFamily : '';
		});
		// Soft Premium default body = Hanken Grotesk.
		expect(adminH1.toLowerCase()).toContain('hanken');
	});
});

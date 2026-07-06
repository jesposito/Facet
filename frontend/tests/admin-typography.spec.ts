import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { setSiteDesign } from './helpers';

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
		data: { identity: LOGIN_EMAIL, password: LOGIN_PASSWORD },
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).token as string;
}

async function setFontPack(request: APIRequestContext, token: string, pack: string) {
	const list = await request
		.get(`${API_BASE}/api/collections/profile/records?perPage=1`, {
			headers: { Authorization: token },
		})
		.then((r) => r.json());
	const id = list.items[0].id as string;
	const res = await request.patch(`${API_BASE}/api/collections/profile/records/${id}`, {
		headers: { Authorization: token },
		data: { font_pack: pack },
	});
	expect(res.ok()).toBeTruthy();
}

const adminShellHeadingFont = (page: Page) =>
	page.evaluate(() => {
		const shell = document.createElement('div');
		shell.className = 'admin-shell';
		shell.innerHTML = '<h1>Admin heading probe</h1>';
		document.body.appendChild(shell);
		const font = getComputedStyle(shell.querySelector('h1')!).fontFamily;
		shell.remove();
		return font;
	});

test.describe('Admin typography', () => {
	test.beforeEach(async ({ request }) => {
		const token = await adminAuth(request);
		await setSiteDesign(request, token, 'soft-premium');
	});

	// Restore the default Classic design and unset font_pack for other suites.
	test.afterEach(async ({ request }) => {
		const token = await adminAuth(request);
		await setFontPack(request, token, '');
		await setSiteDesign(request, token, 'classic');
	});

	test('public headings follow the selected pack; admin headings stay on the body stack', async ({ page, request }) => {
		const token = await adminAuth(request);
		// editorial: Lora headings, Plus Jakarta Sans body — heading ≠ body.
		await setFontPack(request, token, 'editorial');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		// Public heading face follows the selected pack. Read --font-heading
		// directly: hero treatments may use --font-accent, while --font-heading is
		// what the pack actually governs.
		const headingVar = await page.evaluate(() =>
			getComputedStyle(document.documentElement).getPropertyValue('--font-heading').trim().toLowerCase(),
		);
		// editorial pack heading = Lora.
		expect(headingVar).toContain('lora');

		const adminH1 = await adminShellHeadingFont(page);
		// Admin pinned to the sans body stack — not the serif heading face.
		expect(adminH1.toLowerCase()).toContain('jakarta');
		expect(adminH1.toLowerCase()).not.toContain('lora');
	});

	test('under the default Soft Premium pack, admin headings use Hanken (the body stack)', async ({ page, request }) => {
		const token = await adminAuth(request);
		await setFontPack(request, token, '');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const adminH1 = await adminShellHeadingFont(page);
		// Soft Premium default body = Hanken Grotesk.
		expect(adminH1.toLowerCase()).toContain('hanken');
	});
});

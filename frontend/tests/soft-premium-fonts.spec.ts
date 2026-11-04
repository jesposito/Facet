import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { loginAsAdmin, setSiteDesign } from './helpers';

// Under Soft Premium, the font pack picker fully governs the typeface: an
// unconfigured instance falls back to Hanken/Newsreader, and selecting another
// pack is honored verbatim.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const API_BASE = process.env.API_BASE_URL ?? process.env.POCKETBASE_URL ?? 'http://localhost:8090';
async function adminToken(request: APIRequestContext): Promise<string> {
	return (await loginAsAdmin(request)).token;
}

async function setFontPack(request: APIRequestContext, token: string, pack: string) {
	const list = await request
		.get(`${API_BASE}/api/collections/profile/records?perPage=1`, {
			headers: { Authorization: token }
		})
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
	test.beforeEach(async ({ request }) => {
		const token = await adminToken(request);
		await setSiteDesign(request, token, 'soft-premium');
	});

	// Always leave the instance in Classic with an unset font_pack state.
	test.afterEach(async ({ request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, '');
		await setSiteDesign(request, token, 'classic');
	});

	test('default (no operator pack) leads with Hanken Grotesk', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, '');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		expect(font).toContain('hanken');
	});

	test('selecting a non-default pack (modern) is honored — body becomes Inter', async ({
		page,
		request
	}) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'modern');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		// modern pack body = Inter; Hanken must NOT be present.
		expect(font).toContain('inter');
		expect(font).not.toContain('hanken');
	});

	test('selecting a non-default pack also governs the accent/heading face', async ({
		page,
		request
	}) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'modern');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const vars = await page.evaluate(() => {
			const styles = getComputedStyle(document.documentElement);
			return {
				heading: styles.getPropertyValue('--font-heading').trim().toLowerCase(),
				accent: styles.getPropertyValue('--font-accent').trim().toLowerCase()
			};
		});

		expect(vars.heading).toContain('inter');
		expect(vars.accent).toContain('inter');
		expect(vars.accent).not.toContain('newsreader');
	});

	test('selecting editorial (Lora/Jakarta) is honored over the Soft Premium default', async ({
		page,
		request
	}) => {
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

	// Applying the default pack through live preview must load its bundled face,
	// including after client-side navigation.
	const applyDefaultPackOnClient = (page: Page) =>
		page.evaluate(
			() =>
				new Promise<void>((resolve) => {
					window.dispatchEvent(new CustomEvent('font-pack-changed', { detail: 'soft-premium' }));
					// let the layout's handler + Svelte effect flush
					requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
				})
		);

	const localFontState = (page: Page) =>
		page.evaluate(async () => {
			const faces = await document.fonts.load('400 16px "Hanken Grotesk"');
			return {
				hankenLoaded: faces.length > 0 && faces.every((face) => face.status === 'loaded'),
				externalFontLinks: document.querySelectorAll(
					'link[href*="fonts.googleapis.com"], link[href*="fonts.gstatic.com"]'
				).length,
				fontBody: getComputedStyle(document.body).fontFamily.toLowerCase(),
				sameDoc: (window as unknown as Record<string, boolean>).__spaMarker === true
			};
		});

	test('applying the default pack loads bundled Hanken through SPA navigation', async ({
		page,
		request
	}) => {
		const token = await adminToken(request);
		await setFontPack(request, token, '');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
		// Tag the live document so we can prove the next navigation is client-side.
		await page.evaluate(() => ((window as unknown as Record<string, boolean>).__spaMarker = true));

		// Drive the real admin live-preview event for the DEFAULT pack.
		await applyDefaultPackOnClient(page);

		const applied = await localFontState(page);
		expect(applied.hankenLoaded).toBe(true);
		expect(applied.externalFontLinks).toBe(0);
		expect(applied.fontBody).toContain('hanken');

		// Now navigate client-side; the SvelteKit font $effect re-runs applyFontPack
		// and must preserve the default pack's local face.
		const inAppHref = await page.evaluate(() => {
			const a = Array.from(document.querySelectorAll('a[href]')).find((el) => {
				const href = (el as HTMLAnchorElement).getAttribute('href') || '';
				return href.startsWith('/') && !href.startsWith('//') && !href.startsWith('/#');
			}) as HTMLAnchorElement | undefined;
			return a ? a.getAttribute('href') : null;
		});

		if (inAppHref) {
			await Promise.all([
				page.waitForURL((url) => url.pathname === inAppHref, {
					timeout: 10_000
				}),
				page.click(`a[href="${inAppHref}"]`)
			]);
			// Re-apply (as the picker would on the new page) and assert persistence.
			await applyDefaultPackOnClient(page);
			const afterNav = await localFontState(page);
			expect(afterNav.sameDoc, 'navigation should be client-side (SPA)').toBeTruthy();
			expect(afterNav.hankenLoaded).toBe(true);
			expect(afterNav.externalFontLinks).toBe(0);
			expect(afterNav.fontBody).toContain('hanken');
		}
	});
});

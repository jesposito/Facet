import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { setSiteDesign } from './helpers';

// Under Soft Premium, the font pack picker fully governs the typeface: an
// unconfigured instance falls back to Hanken/Newsreader, and selecting another
// pack is honored verbatim.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const API_BASE = process.env.API_BASE_URL ?? process.env.POCKETBASE_URL ?? 'http://localhost:8090';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

async function adminToken(request: APIRequestContext): Promise<string> {
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

const bodyFont = (page: Page) => page.evaluate(() => getComputedStyle(document.body).fontFamily.toLowerCase());

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

	test('selecting a non-default pack (modern) is honored — body becomes Inter', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'modern');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const font = await bodyFont(page);
		// modern pack body = Inter; Hanken must NOT be present.
		expect(font).toContain('inter');
		expect(font).not.toContain('hanken');
	});

	test('selecting a non-default pack also governs the accent/heading face', async ({ page, request }) => {
		const token = await adminToken(request);
		await setFontPack(request, token, 'modern');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const vars = await page.evaluate(() => {
			const styles = getComputedStyle(document.documentElement);
			return {
				heading: styles.getPropertyValue('--font-heading').trim().toLowerCase(),
				accent: styles.getPropertyValue('--font-accent').trim().toLowerCase(),
			};
		});

		expect(vars.heading).toContain('inter');
		expect(vars.accent).toContain('inter');
		expect(vars.accent).not.toContain('newsreader');
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

	// Regression guard: applying the DEFAULT pack (soft-premium) on the client must
	// KEEP the Google Fonts link. The live admin font picker drives applyFontPack via
	// the 'font-pack-changed' window event; the regression special-cased the default
	// pack and *removed* the dynamic link, so a creator who previews soft-premium (or
	// switches back to it) lost Hanken/Newsreader and the body fell back to Plus
	// Jakarta. We exercise the exact event the picker uses, then confirm a Hanken
	// dynamic link is present and the resolved --font-body is Hanken. This is the
	// discriminating assertion: the buggy code produced no soft-premium dynamic link.
	const applyDefaultPackOnClient = (page: Page) =>
		page.evaluate(
			() =>
				new Promise<void>((resolve) => {
					window.dispatchEvent(new CustomEvent('font-pack-changed', { detail: 'soft-premium' }));
					// let the layout's handler + Svelte effect flush
					requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
				}),
		);

	const dynamicLinkState = (page: Page) =>
		page.evaluate(() => {
			const link = document.querySelector('link#dynamic-google-fonts') as HTMLLinkElement | null;
			return {
				dynamicHasHanken: !!link && /Hanken/i.test(link.href),
				fontBody: getComputedStyle(document.body).fontFamily.toLowerCase(),
				sameDoc: (window as unknown as Record<string, boolean>).__spaMarker === true,
			};
		});

	test('applying the default pack on the client keeps the Hanken link, and it survives SPA nav', async ({
		page,
		request,
	}) => {
		const token = await adminToken(request);
		await setFontPack(request, token, '');

		await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
		// Tag the live document so we can prove the next navigation is client-side.
		await page.evaluate(() => ((window as unknown as Record<string, boolean>).__spaMarker = true));

		// Drive the real admin live-preview event for the DEFAULT pack.
		await applyDefaultPackOnClient(page);

		const applied = await dynamicLinkState(page);
		expect(
			applied.dynamicHasHanken,
			'applying soft-premium on the client must create/keep #dynamic-google-fonts pointing at Hanken (regression removed it)',
		).toBeTruthy();
		expect(applied.fontBody).toContain('hanken');

		// Now navigate client-side; the SvelteKit font $effect re-runs applyFontPack
		// and must not drop the default pack's link.
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
					timeout: 10_000,
				}),
				page.click(`a[href="${inAppHref}"]`),
			]);
			// Re-apply (as the picker would on the new page) and assert persistence.
			await applyDefaultPackOnClient(page);
			const afterNav = await dynamicLinkState(page);
			expect(afterNav.sameDoc, 'navigation should be client-side (SPA)').toBeTruthy();
			expect(
				afterNav.dynamicHasHanken,
				'dynamic Hanken link must remain after SPA nav with the default pack',
			).toBeTruthy();
			expect(afterNav.fontBody).toContain('hanken');
		}
	});
});

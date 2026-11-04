import { test, expect, type APIRequestContext, type Browser, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';
import { isValidMaxUses, localExpiryToISO } from '../src/lib/share-links';

// The released editors revoked a link, then sent its stored expiry back to
// /api/share/generate, which refused it: the old link died and no new one came.
// These tests drive both editors against a real backend with a future expiry.

const LINK_NAME = 'playwright-regenerate';
const TOKENS_REGENERATE = 'Regenerate token (revokes old, creates new)';
const EDITOR_REGENERATE = 'Regenerate link (revokes old, creates new)';

test('share link input checks', () => {
	expect(localExpiryToISO('')).toBe('');
	expect(localExpiryToISO('not-a-date')).toBeNull();
	expect(localExpiryToISO('2035-02-30T12:00')).toBeNull();
	expect(localExpiryToISO('2000-01-01T12:00')).toBeNull();
	for (const valid of [0, 1, 1_000_000, null]) expect(isValidMaxUses(valid)).toBe(true);
	for (const invalid of [-1, 1.5, 1_000_001, Number.NaN]) {
		expect(isValidMaxUses(invalid)).toBe(false);
	}
});

test.describe('share link regeneration', () => {
	const viewIds: string[] = [];
	let token: string;

	test.beforeEach(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
	});

	test.afterEach(async ({ request }) => {
		const headers = { Authorization: token };
		for (const viewId of viewIds.splice(0)) {
			const links = await request.get(`${apiBaseURL}/api/collections/share_tokens/records`, {
				headers,
				params: { filter: `view_id="${viewId}"`, perPage: 50 }
			});
			for (const { id } of (await links.json()).items) {
				await request.delete(`${apiBaseURL}/api/collections/share_tokens/records/${id}`, {
					headers
				});
			}
			await request.delete(`${apiBaseURL}/api/collections/views/records/${viewId}`, { headers });
		}
	});

	async function createView(request: APIRequestContext, visibility = 'unlisted') {
		const suffix = randomUUID().slice(0, 8);
		const slug = `share-regen-${suffix}`;
		const res = await request.post(`${apiBaseURL}/api/collections/views/records`, {
			headers: { Authorization: token },
			data: { name: `Share regeneration ${suffix}`, slug, visibility, is_active: true }
		});
		expect(res.ok(), await res.text()).toBe(true);
		const id = (await res.json()).id as string;
		viewIds.push(id);
		return { id, slug };
	}

	async function createLink(request: APIRequestContext, viewId: string) {
		const res = await request.post(`${apiBaseURL}/api/share/generate`, {
			headers: { Authorization: token },
			data: {
				view_id: viewId,
				name: LINK_NAME,
				expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
				max_uses: 5
			}
		});
		expect(res.ok(), await res.text()).toBe(true);
		const body = await res.json();
		return { id: body.id as string, raw: body.token as string };
	}

	async function storedLink(request: APIRequestContext, id: string) {
		const res = await request.get(`${apiBaseURL}/api/collections/share_tokens/records/${id}`, {
			headers: { Authorization: token }
		});
		expect(res.ok(), await res.text()).toBe(true);
		return res.json();
	}

	async function countLinks(request: APIRequestContext, filter: string) {
		const res = await request.get(`${apiBaseURL}/api/collections/share_tokens/records`, {
			headers: { Authorization: token },
			params: { filter, perPage: 1 }
		});
		expect(res.ok(), await res.text()).toBe(true);
		return (await res.json()).totalItems as number;
	}

	async function setViewActive(request: APIRequestContext, viewId: string, isActive: boolean) {
		const res = await request.patch(`${apiBaseURL}/api/collections/views/records/${viewId}`, {
			headers: { Authorization: token },
			data: { is_active: isActive }
		});
		expect(res.ok(), await res.text()).toBe(true);
	}

	// The /s/ route redirects to the Facet only when the link is good, and says
	// "invalid" only when the backend refused it. A 429 never reached the token,
	// so it alone may be repeated, after the wait the server asked for. Anything
	// else, such as a 5xx that may have counted a use, fails the test.
	async function opensForVisitor(browser: Browser, raw: string, slug: string) {
		for (let denied = 0; denied < 4; denied += 1) {
			const visitor = await browser.newContext();
			const page = await visitor.newPage();
			const response = await page.goto(`/s/${raw}`);
			expect(response?.status()).toBe(200);
			if (new URL(page.url()).pathname === `/${slug}`) {
				await visitor.close();
				return true;
			}

			const code = await page.locator('[data-code]').getAttribute('data-code');
			const retryAfter = Number(response?.headers()['retry-after']);
			await visitor.close();
			if (code === 'invalid') return false;
			if (code !== 'rate_limited' || !(retryAfter > 0)) {
				throw new Error(`share entry answered ${code}`);
			}

			test.info().setTimeout(test.info().timeout + retryAfter * 1000);
			await new Promise((resolve) => setTimeout(resolve, retryAfter * 1000));
		}
		throw new Error('share entry stayed rate limited after every allowed wait');
	}

	function regenerateResponse(page: Page, id: string) {
		return page.waitForResponse((r) => r.url().includes(`/api/share/regenerate/${id}`));
	}

	async function confirmRegeneration(page: Page) {
		await page
			.getByRole('alertdialog')
			.getByRole('button', { name: 'Regenerate', exact: true })
			.click();
	}

	function tokensCard(page: Page, viewId: string) {
		return page.locator('.card').filter({ has: page.locator(`a[href="/admin/views/${viewId}"]`) });
	}

	async function expectReplaced(
		browser: Browser,
		request: APIRequestContext,
		view: { slug: string },
		old: { id: string; raw: string; expires_at: string },
		next: { id: string; token: string }
	) {
		expect(await storedLink(request, next.id)).toMatchObject({
			name: LINK_NAME,
			expires_at: old.expires_at,
			max_uses: 5,
			use_count: 0,
			is_active: true
		});
		expect((await storedLink(request, old.id)).is_active).toBe(false);
		expect(await opensForVisitor(browser, next.token, view.slug)).toBe(true);
		expect(await opensForVisitor(browser, old.raw, view.slug)).toBe(false);
	}

	test('tokens page replaces a link that has a future expiry', async ({
		page,
		request,
		browser
	}) => {
		const view = await createView(request);
		const old = await createLink(request, view.id);
		const { expires_at } = await storedLink(request, old.id);
		await seedAdminSession(page, request);
		await page.goto('/admin/tokens');

		const replaced = regenerateResponse(page, old.id);
		await tokensCard(page, view.id).getByTitle(TOKENS_REGENERATE).click();
		await confirmRegeneration(page);
		const response = await replaced;
		expect(response.status()).toBe(200);
		const next = await response.json();

		await expect(page.locator('code', { hasText: `/s/${next.token}` })).toBeVisible();
		await expectReplaced(browser, request, view, { ...old, expires_at }, next);
	});

	test('Facet editor sends one replacement for a double click', async ({
		page,
		request,
		browser
	}) => {
		const view = await createView(request);
		const old = await createLink(request, view.id);
		const { expires_at } = await storedLink(request, old.id);
		await seedAdminSession(page, request);
		await page.goto(`/admin/views/${view.id}`);

		let requests = 0;
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route('**/api/share/regenerate/*', async (route) => {
			requests += 1;
			await held;
			await route.continue();
		});

		const button = page.getByTitle(EDITOR_REGENERATE);
		const replaced = regenerateResponse(page, old.id);
		await button.dblclick();
		await confirmRegeneration(page);
		await expect(button).toBeDisabled();
		release();
		const next = await (await replaced).json();

		await expect(page.locator('input[readonly].font-mono')).toHaveValue(
			new RegExp(`/s/${next.token}$`)
		);
		expect(requests).toBe(1);
		await expectReplaced(browser, request, view, { ...old, expires_at }, next);
	});

	for (const ui of ['tokens page', 'Facet editor'] as const) {
		test(`${ui} keeps the old link when replacement is refused`, async ({
			page,
			request,
			browser
		}) => {
			const view = await createView(request);
			const old = await createLink(request, view.id);
			await seedAdminSession(page, request);
			await page.goto(ui === 'tokens page' ? '/admin/tokens' : `/admin/views/${view.id}`);
			const button =
				ui === 'tokens page'
					? tokensCard(page, view.id).getByTitle(TOKENS_REGENERATE)
					: page.getByTitle(EDITOR_REGENERATE);
			await expect(button).toBeVisible();

			// A real refusal from the backend, after the page offered the action.
			await setViewActive(request, view.id, false);
			const refused = regenerateResponse(page, old.id);
			await button.click();
			await confirmRegeneration(page);
			expect((await refused).status()).toBe(409);

			await expect(page.getByText("inactive facets can't have share links").first()).toBeVisible();
			await expect(page.locator('code', { hasText: '/s/' })).toHaveCount(0);
			await expect(page.locator('input[readonly].font-mono')).toHaveCount(0);

			await setViewActive(request, view.id, true);
			expect(await countLinks(request, `view_id="${view.id}"`)).toBe(1);
			expect((await storedLink(request, old.id)).is_active).toBe(true);
			expect(await opensForVisitor(browser, old.raw, view.slug)).toBe(true);
		});
	}

	test('panel offers links only for a saved unlisted Facet', async ({ page, request }) => {
		const unlisted = await createView(request);
		await seedAdminSession(page, request);
		await page.goto(`/admin/views/${unlisted.id}`);
		const generate = page.getByRole('button', { name: 'Generate Share Link', exact: true });
		await expect(generate).toBeVisible();

		await page.locator('#visibility').selectOption('private');
		await expect(page.getByText(/Save this Facet to apply/)).toBeVisible();
		await expect(generate).toHaveCount(0);
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText(/only you can open this Facet/)).toBeVisible();
		await expect(generate).toHaveCount(0);

		await page.locator('#visibility').selectOption('public');
		await expect(page.getByText(/Save this Facet to apply/)).toBeVisible();
		await expect(generate).toHaveCount(0);

		const open = await createView(request, 'public');
		await page.goto(`/admin/views/${open.id}`);
		await expect(page.getByLabel('Public URL')).toHaveValue(new RegExp(`/${open.slug}$`));
		await expect(generate).toHaveCount(0);
		await expect(page.locator('#token_expires')).toHaveCount(0);

		const closed = await createView(request, 'private');
		const refused = await request.post(`${apiBaseURL}/api/share/generate`, {
			headers: { Authorization: token },
			data: { view_id: closed.id }
		});
		expect(refused.status()).toBe(409);
		expect(await countLinks(request, `view_id="${closed.id}"`)).toBe(0);
	});

	for (const { zone, instant } of [
		{ zone: 'America/New_York', instant: '2035-01-15T14:30:00.000Z' },
		{ zone: 'Pacific/Auckland', instant: '2035-01-14T20:30:00.000Z' }
	]) {
		test.describe(zone, () => {
			test.use({ timezoneId: zone });

			test('a local expiry is stored as the instant the owner meant', async ({ page, request }) => {
				const local = '2035-01-15T09:30';
				const view = await createView(request);
				await seedAdminSession(page, request);
				const generated = () =>
					page.waitForResponse((r) => r.url().endsWith('/api/share/generate'));
				const expectStored = async (response: Awaited<ReturnType<typeof generated>>) => {
					expect(response.status()).toBe(200);
					expect(response.request().postDataJSON().expires_at).toBe(instant);
					const stored = await storedLink(request, (await response.json()).id);
					expect(new Date(stored.expires_at.replace(' ', 'T')).toISOString()).toBe(instant);
				};

				await page.goto('/admin/tokens');
				await page.getByRole('button', { name: 'Generate Token', exact: true }).first().click();
				const form = page.locator('form:has(#view_id)');
				await form.locator('#view_id').selectOption(view.id);
				await form.locator('#expires_at').fill(local);
				const fromTokens = generated();
				await form.getByRole('button', { name: 'Generate Token', exact: true }).click();
				await expectStored(await fromTokens);

				await page.goto(`/admin/views/${view.id}`);
				await page.getByRole('button', { name: 'Generate Share Link', exact: true }).click();
				await page.locator('#token_expires').fill(local);
				const fromEditor = generated();
				await page.getByRole('button', { name: 'Generate Link', exact: true }).click();
				await expectStored(await fromEditor);
			});
		});
	}
});

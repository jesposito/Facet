import { test, expect, type Browser, type Response } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin } from './helpers';

// A single-use share link must let the visitor who opened it keep browsing the
// view (and the private projects it links) for that visit, without letting the
// raw link or token admit anyone else.
test.describe('single-use share link visit', () => {
	const suffix = randomUUID().slice(0, 8);
	const viewSlug = `share-visit-${suffix}`;
	const projectSlug = `share-visit-project-${suffix}`;
	const projectTitle = `Share visit private project ${suffix}`;
	const created: { collection: string; id: string }[] = [];
	let token: string;
	let shareToken: string;

	test.beforeAll(async ({ request }) => {
		// The owner token only authenticates these API calls; it is never added
		// to a browser context, so the visitor below stays anonymous.
		({ token } = await loginAsAdmin(request));
		const headers = { Authorization: token };

		const project = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
			headers,
			data: {
				title: projectTitle,
				slug: projectSlug,
				summary: 'Share visit fixture.',
				description: 'Share visit fixture.',
				visibility: 'private',
				is_draft: false
			}
		});
		expect(project.ok(), await project.text()).toBe(true);
		const projectId = (await project.json()).id as string;
		created.push({ collection: 'projects', id: projectId });

		const view = await request.post(`${apiBaseURL}/api/collections/views/records`, {
			headers,
			data: {
				name: `Share visit Facet ${suffix}`,
				slug: viewSlug,
				visibility: 'unlisted',
				is_active: true,
				sections: [{ section: 'projects', enabled: true, layout: 'grid-2', items: [projectId] }]
			}
		});
		expect(view.ok(), await view.text()).toBe(true);
		const viewId = (await view.json()).id as string;
		created.push({ collection: 'views', id: viewId });

		const shown = await request.patch(
			`${apiBaseURL}/api/collections/projects/records/${projectId}`,
			{ headers, data: { view_visibility: { [viewId]: true } } }
		);
		expect(shown.ok(), await shown.text()).toBe(true);

		const generated = await request.post(`${apiBaseURL}/api/share/generate`, {
			headers,
			data: { view_id: viewId, name: 'playwright-share-visit', max_uses: 1 }
		});
		expect(generated.ok(), await generated.text()).toBe(true);
		shareToken = (await generated.json()).token as string;
	});

	test.afterAll(async ({ request }) => {
		for (const { collection, id } of [...created].reverse()) {
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
		}
	});

	function expectPrivate(response: Response | null) {
		expect(response, 'navigation response').not.toBeNull();
		expect(response!.status()).toBe(200);
		expect(response!.headers()['cache-control']).toBe('private, no-store');
	}

	async function anonymousContext(browser: Browser) {
		const context = await browser.newContext();
		expect(await context.cookies()).toEqual([]);
		return context;
	}

	test('admits the visit that spent the use and no one else', async ({ browser, baseURL }) => {
		const visitor = await anonymousContext(browser);
		const page = await visitor.newPage();

		// Opening the link spends its only use and lands on the clean view URL.
		const redirectResponse = page.waitForResponse(
			(r) => new URL(r.url()).pathname === `/s/${shareToken}` && r.status() === 302
		);
		const opened = await page.goto(`/s/${shareToken}`);
		const entryHeaders = (await redirectResponse).headers();
		expect(entryHeaders['cache-control']).toBe('private, no-store');
		expect(entryHeaders['referrer-policy']).toBe('no-referrer');
		await expect(page).toHaveURL(new RegExp(`/${viewSlug}$`));
		expectPrivate(opened);
		const cookies = await visitor.cookies();
		expect(cookies.map((c) => c.name)).not.toContain('pb_auth');
		const visit = cookies.find((c) => c.name === 'me_share_visit');
		expect(visit, 'share visit proof cookie').toBeTruthy();
		expect(visit!.httpOnly).toBe(true);
		expect(visit!.value).not.toBe(shareToken);

		// The view links the private project; following it reuses the visit.
		const card = page.locator(`a[href^="/projects/${projectSlug}"]`).first();
		await expect(card).toBeVisible();
		const projectLoaded = page.waitForResponse(
			(r) => new URL(r.url()).pathname.startsWith(`/projects/${projectSlug}`) && r.ok()
		);
		await card.click();
		await expect(page.getByRole('heading', { level: 1, name: projectTitle })).toBeVisible();
		await projectLoaded;

		// A full SSR load of the project detail and a return to the view both
		// succeed on the visit proof even though the token has no uses left.
		expectPrivate(await page.reload());
		await expect(page.getByRole('heading', { level: 1, name: projectTitle })).toBeVisible();
		expectPrivate(await page.goto(`/${viewSlug}`));
		await expect(page.locator(`a[href^="/projects/${projectSlug}"]`).first()).toBeVisible();

		// The same visitor without the proof is back to the raw token alone.
		const proof = (await visitor.cookies()).find((c) => c.name === 'me_share_visit')!;
		await visitor.clearCookies({ name: 'me_share_visit' });
		expect((await page.goto(`/${viewSlug}`))!.status()).toBe(404);
		await visitor.close();

		// A fresh browser reusing the link is refused: the use is spent.
		const other = await anonymousContext(browser);
		const otherPage = await other.newPage();
		await otherPage.goto(`/s/${shareToken}`);
		await expect(otherPage).toHaveURL(new RegExp(`/s/${shareToken}$`));
		await expect(otherPage.locator(`a[href^="/projects/${projectSlug}"]`)).toHaveCount(0);
		expect((await other.cookies()).map((c) => c.name)).not.toContain('me_share_visit');

		// Holding the raw token as a cookie is not enough either.
		await other.addCookies([{ name: 'me_share_token', value: shareToken, url: baseURL! }]);
		expect((await otherPage.goto(`/${viewSlug}`))!.status()).toBe(404);
		expect((await otherPage.goto(`/projects/${projectSlug}?from=${viewSlug}`))!.status()).toBe(404);

		// Nor is a proof replayed into a context that never held the token.
		const stranger = await anonymousContext(browser);
		await stranger.addCookies([
			{ name: 'me_share_visit', value: proof.value, url: baseURL! },
			{ name: 'me_share_token', value: 'forged-share-token', url: baseURL! }
		]);
		const strangerPage = await stranger.newPage();
		expect((await strangerPage.goto(`/${viewSlug}`))!.status()).toBe(404);

		await other.close();
		await stranger.close();
	});
});

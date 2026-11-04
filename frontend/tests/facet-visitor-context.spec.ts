import { test, expect, type BrowserContext, type Route } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { createPublicProjectAndPost, deletePublicProjectAndPost, loginAsAdmin } from './helpers';

// A signed-out visitor enters a single-use link once. The private post's thread
// and files then work from the HttpOnly cookies that entry set, and nothing else.
test.describe('private post from a single-use share link', () => {
	const suffix = randomUUID().slice(0, 8);
	const viewSlug = `visitor-context-${suffix}`;
	const postSlug = `visitor-context-post-${suffix}`;
	const created: { collection: string; id: string }[] = [];
	let token: string;
	let shareToken: string;
	let retryToken: string;
	let leaveToken: string;
	let postId: string;
	let fileName: string;

	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		const headers = { Authorization: token };

		const post = await request.post(`${apiBaseURL}/api/collections/posts/records`, {
			headers,
			multipart: {
				title: `Visitor context ${suffix}`,
				slug: postSlug,
				content: 'Visitor context fixture.',
				visibility: 'private',
				is_draft: 'false',
				comments_enabled: 'true',
				downloadable_files: {
					name: 'guide.pdf',
					mimeType: 'application/pdf',
					buffer: Buffer.from('VISITOR FILE BYTES')
				}
			}
		});
		expect(post.ok(), await post.text()).toBe(true);
		const record = await post.json();
		postId = record.id as string;
		fileName = record.downloadable_files[0] as string;
		created.push({ collection: 'posts', id: postId });

		const view = await request.post(`${apiBaseURL}/api/collections/views/records`, {
			headers,
			data: {
				name: `Visitor context ${suffix}`,
				slug: viewSlug,
				visibility: 'unlisted',
				is_active: true,
				sections: [{ section: 'posts', enabled: true, layout: 'grid-3', items: [postId] }]
			}
		});
		expect(view.ok(), await view.text()).toBe(true);
		const viewId = (await view.json()).id as string;
		created.push({ collection: 'views', id: viewId });

		const grant = await request.post(`${apiBaseURL}/api/view/${viewId}/private-items`, { headers });
		expect(grant.ok(), await grant.text()).toBe(true);

		const share = async (name: string) => {
			const generated = await request.post(`${apiBaseURL}/api/share/generate`, {
				headers,
				data: { view_id: viewId, name, max_uses: 1 }
			});
			expect(generated.ok(), await generated.text()).toBe(true);
			return (await generated.json()).token as string;
		};
		shareToken = await share('playwright-visitor-context');
		retryToken = await share('playwright-visitor-retry');
		leaveToken = await share('playwright-visitor-leave');
	});

	test.afterAll(async ({ request }) => {
		for (const { collection, id } of [...created].reverse()) {
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
		}
	});

	const isFileRead = (url: URL) => url.pathname === `/api/content/posts/${postId}/download-urls`;

	// Enters a single-use link and opens the post, with respond answering reads
	// of the file list. Links the browser tries to redeem are collected.
	async function openPost(
		context: BrowserContext,
		link: string,
		respond: (route: Route) => Promise<void>
	) {
		const page = await context.newPage();
		const redeemed: string[] = [];
		page.on('request', (request) => {
			const { pathname } = new URL(request.url());
			if (pathname.startsWith('/api/downloads/')) redeemed.push(pathname);
		});
		await page.route(isFileRead, respond);
		await page.goto(`/s/${link}`);
		await expect(page).toHaveURL(new RegExp(`/${viewSlug}$`));
		await page.locator(`a[href*="/posts/${postSlug}"]`).first().click();
		await expect(page.getByText('Visitor context fixture.')).toBeVisible();
		return { page, redeemed };
	}

	test('cookies carry the visit only while its proof lasts', async ({ browser }) => {
		const context = await browser.newContext();
		const page = await context.newPage();
		await page.goto(`/s/${shareToken}`);
		await expect(page).toHaveURL(new RegExp(`/${viewSlug}$`));
		await page.locator(`a[href*="/posts/${postSlug}"]`).first().click();
		await expect(page.getByText('Visitor context fixture.')).toBeVisible();

		const link = page.locator('a[href^="/api/downloads/"]');
		await expect(link).toBeVisible();
		const file = (await link.getAttribute('href'))!;
		expect(file).not.toContain(shareToken);

		type Init = { method?: string; headers?: Record<string, string>; body?: string };
		const status = (url: string, init: Init = {}) =>
			page.evaluate(async (a) => (await fetch(a.url, a.init)).status, { url, init });
		const thread = `/api/comments/posts/${postId}?from=${viewSlug}`;
		expect(await status(thread)).toBe(200);
		const comment = JSON.stringify({ author_name: 'Ada', body: `Hello ${suffix}` });
		const post = { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: comment };
		expect(await status(thread, post)).toBe(201);
		expect(await status(file)).toBe(200);

		// The token has no uses left, so without the proof it admits nobody.
		await context.clearCookies({ name: 'me_share_visit' });
		expect(await status(thread)).toBe(404);
		expect(await status(file)).toBe(404);
		await context.close();
	});

	test('a failed read offers a retry and a click redeems only a fresh link', async ({
		browser
	}) => {
		test.setTimeout(45_000);
		const context = await browser.newContext();
		const expired = {
			filename: fileName,
			display_name: fileName,
			url: '/api/downloads/expired-link'
		};
		const replies: ((route: Route) => Promise<void>)[] = [
			// Left unanswered, so the component's own timeout has to end the read.
			async () => {},
			(route) => route.fulfill({ status: 503, json: { error: 'unavailable' } }),
			(route) =>
				route.fulfill({ json: { files: [{ ...expired, url: 'https://evil.example/guide.pdf' }] } }),
			(route) => route.fulfill({ json: { files: [expired] } }),
			(route) => route.abort('failed'),
			(route) => route.continue()
		];
		const { page, redeemed } = await openPost(context, retryToken, (route) =>
			replies.shift()!(route)
		);
		const downloads = page.getByRole('region', { name: 'Downloads' });
		const alert = downloads.getByRole('alert');
		const retry = downloads.getByRole('button', { name: 'Try again' });
		const retried = () =>
			Promise.all([page.waitForResponse((r) => isFileRead(new URL(r.url()))), retry.click()]);

		await expect(alert).toBeVisible({ timeout: 15_000 });
		await retried();
		await expect(retry).toBeEnabled();
		await expect(alert).toBeVisible();

		await retried();
		await expect(retry).toBeEnabled();
		await expect(alert).toBeVisible();
		await expect(downloads.getByRole('link')).toHaveCount(0);

		await retried();
		await expect(alert).toBeHidden();
		await expect(retry).toBeHidden();
		const link = downloads.getByRole('link', { name: fileName });
		await expect(link).toHaveAttribute('href', expired.url);

		// A failed refresh keeps the visitor on the page and redeems nothing.
		await link.click();
		await expect(alert).toBeVisible();
		expect(redeemed).toEqual([]);
		await expect(page).toHaveURL(new RegExp(`/posts/${postSlug}`));

		const [download] = await Promise.all([page.waitForEvent('download'), link.click()]);
		expect(await download.failure()).toBeNull();
		expect(redeemed).toHaveLength(1);
		expect(redeemed[0]).not.toBe(expired.url);
		expect(download.url()).toContain(redeemed[0]);
		await expect(link).toHaveAttribute('href', redeemed[0]);
		await expect(alert).toBeHidden();
		await context.close();
	});

	test('leaving the page abandons a pending link', async ({ browser }) => {
		const context = await browser.newContext();
		const held: Route[] = [];
		let reads = 0;
		const { page, redeemed } = await openPost(context, leaveToken, async (route) => {
			if (reads++ === 0) await route.continue();
			else held.push(route);
		});

		await page.getByRole('region', { name: 'Downloads' }).getByRole('link').click();
		await expect.poll(() => held.length).toBe(1);
		await page.locator(`a[href="/${viewSlug}"]`).first().click();
		await expect(page).toHaveURL(new RegExp(`/${viewSlug}$`));

		// The answer arrives too late, so the link it signs must go nowhere.
		await held[0].continue().catch(() => {});
		await page.waitForLoadState('networkidle');
		expect(redeemed).toEqual([]);
		await context.close();
	});

	test('a page without files asks for none', async ({ page, request }) => {
		const fixture = await createPublicProjectAndPost(request, token);
		try {
			const reads: string[] = [];
			page.on('request', (r) => {
				if (r.url().includes('/download-urls')) reads.push(r.url());
			});
			for (const path of [`/posts/${fixture.postSlug}`, `/projects/${fixture.projectSlug}`]) {
				await page.goto(path);
				await page.waitForLoadState('networkidle');
			}
			expect(reads).toEqual([]);
		} finally {
			await deletePublicProjectAndPost(request, token, fixture);
		}
	});
});

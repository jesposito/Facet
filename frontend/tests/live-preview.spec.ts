import { test, expect, type Page } from '@playwright/test';
import { seedAdminSession } from './helpers';
import { apiBaseURL } from './config';

// Exercises the ported live-site preview: a header toggle (role=switch) that
// opens a side column with a sandboxed iframe of the public homepage, with the
// state persisted in localStorage. Desktop only (lg+).
const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';

const toggle = (page: Page) => page.getByRole('switch', { name: 'Toggle live preview' });
const previewFrame = (page: Page) =>
	page.locator('iframe[title="Live preview of your public site"]');

const waitForSubscription = (page: Page, topic: string) =>
	page.waitForResponse((response) => {
		const req = response.request();
		return (
			new URL(req.url()).pathname === '/api/realtime' &&
			req.method() === 'POST' &&
			response.ok() &&
			req.postDataJSON()?.subscriptions?.includes(topic)
		);
	});

const waitForPreviewReload = (page: Page) =>
	page.waitForRequest((req) => {
		const url = new URL(req.url());
		return (
			url.origin === new URL(BASE_URL).origin &&
			url.pathname === '/' &&
			url.searchParams.get('preview') === '1' &&
			Number(url.searchParams.get('_')) > 0
		);
	});

test.describe('Live site preview', () => {
	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
		await page.goto(`${BASE_URL}/admin`);
	});

	test.afterEach(async ({ page }) => {
		// Leave the preference off for the next test/file.
		const t = toggle(page);
		if ((await t.count()) > 0 && (await t.getAttribute('aria-checked')) === 'true') {
			await t.click();
		}
	});

	test('toggle is an accessible switch, off by default', async ({ page }) => {
		const t = toggle(page);
		await expect(t).toBeVisible();
		await expect(t).toHaveAttribute('aria-checked', 'false');
		// No preview iframe until enabled.
		await expect(previewFrame(page)).toHaveCount(0);
	});

	test('enabling shows the sandboxed live-site iframe', async ({ page }) => {
		await toggle(page).click();
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'true');
		const frame = previewFrame(page);
		await expect(frame).toBeVisible();
		await expect(frame).toHaveAttribute('src', /\/\?preview=1/);
		// Sealed from tab order (read-only pane).
		await expect(frame).toHaveAttribute('tabindex', '-1');
	});

	test('preference persists across a reload (localStorage)', async ({ page }) => {
		await toggle(page).click();
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'true');
		await page.reload();
		await page.waitForURL(/\/admin/);
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'true');
		await expect(previewFrame(page)).toBeVisible();
	});

	test('disabling hides the preview column', async ({ page }) => {
		await toggle(page).click();
		await expect(previewFrame(page)).toBeVisible();
		await toggle(page).click();
		await expect(toggle(page)).toHaveAttribute('aria-checked', 'false');
		await expect(previewFrame(page)).toHaveCount(0);
	});

	test('reloads changed public content after a non-profile mutation', async ({ page, request }) => {
		const { token } = await seedAdminSession(page, request);
		const headers = { Authorization: token };
		const listResponse = await request.get(
			`${apiBaseURL}/api/collections/experience/records?perPage=100`,
			{ headers }
		);
		expect(listResponse.ok(), await listResponse.text()).toBeTruthy();
		const records = (await listResponse.json()).items as Array<Record<string, any>>;
		const experience = records.find((record) => record.visibility === 'public' && !record.is_draft);
		if (!experience) throw new Error('A public experience record is required for this test');
		const originalTitle = experience.title as string;
		const changedTitle = `Live preview refresh ${Date.now()}`;

		try {
			const subscribed = waitForSubscription(page, 'experience/*');
			await toggle(page).click();
			const frame = previewFrame(page);
			const body = page
				.frameLocator('iframe[title="Live preview of your public site"]')
				.locator('body');
			await expect(frame).toBeVisible();
			await expect(body).toContainText(originalTitle);
			await subscribed;

			const refreshedRequest = waitForPreviewReload(page);
			const updateResponse = await request.patch(
				`${apiBaseURL}/api/collections/experience/records/${experience.id}`,
				{ headers, data: { title: changedTitle } }
			);
			expect(updateResponse.ok(), await updateResponse.text()).toBeTruthy();
			await refreshedRequest;
			await expect(body).toContainText(changedTitle);
		} finally {
			const restoreResponse = await request.patch(
				`${apiBaseURL}/api/collections/experience/records/${experience.id}`,
				{ headers, data: { title: originalTitle } }
			);
			expect(restoreResponse.ok(), 'original experience title must be restored').toBeTruthy();
		}
	});

	test('reloads public site config after a settings change', async ({ page, request }) => {
		const { token } = await seedAdminSession(page, request);
		const headers = { Authorization: token };
		const settingsResponse = await request.get(`${apiBaseURL}/api/site-settings`, { headers });
		expect(settingsResponse.ok(), 'site settings must load').toBeTruthy();
		const originalCSS = ((await settingsResponse.json()).custom_css ?? '') as string;
		const changedColor = 'rgb(1, 2, 3)';

		try {
			const subscribed = waitForSubscription(page, 'public_site_config');
			await toggle(page).click();
			const body = page
				.frameLocator('iframe[title="Live preview of your public site"]')
				.locator('body');
			await expect(previewFrame(page)).toBeVisible();
			await expect(body).toBeVisible();
			await expect(body).not.toHaveCSS('outline-color', changedColor);
			await subscribed;

			const refreshedRequest = waitForPreviewReload(page);
			const updateResponse = await request.put(`${apiBaseURL}/api/site-settings`, {
				headers,
				data: { custom_css: `body { outline-color: ${changedColor}; }` }
			});
			expect(updateResponse.ok(), 'custom CSS must save').toBeTruthy();
			await refreshedRequest;
			await expect(body).toHaveCSS('outline-color', changedColor);
		} finally {
			const restoreResponse = await request.put(`${apiBaseURL}/api/site-settings`, {
				headers,
				data: { custom_css: originalCSS }
			});
			expect(restoreResponse.ok(), 'original custom CSS must be restored').toBeTruthy();
		}
	});
});

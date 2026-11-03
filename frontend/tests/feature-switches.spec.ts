import { test, expect, type APIRequestContext } from '@playwright/test';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import { fetchFirstView, loginAsAdmin, seedAdminSession } from './helpers';

// Settings → Features switches. Turning a feature off must close its public
// API and hide its components and admin navigation, while readers can still
// unsubscribe and the owner keeps the Features page and their subscriber data.

const shouldRun = Boolean(adminEmail && adminPassword);

async function setFeatures(
	request: APIRequestContext,
	token: string,
	features: Record<string, boolean>
) {
	const res = await request.put(`${apiBaseURL}/api/site-settings`, {
		headers: { Authorization: token },
		data: { enabled_features: features }
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).enabled_features as Record<string, boolean>;
}

test.describe('Feature switches', () => {
	test.skip(!shouldRun, 'ADMIN_EMAIL and ADMIN_PASSWORD are required');

	let original: Record<string, boolean> = {};

	test.beforeEach(async ({ request }) => {
		const res = await request.get(`${apiBaseURL}/api/site-settings`);
		original = (await res.json()).enabled_features ?? {};
	});

	test.afterEach(async ({ request }) => {
		const { token } = await loginAsAdmin(request);
		await setFeatures(request, token, {
			analytics: original.analytics !== false,
			courses: original.courses !== false,
			newsletter: original.newsletter !== false,
			discussions: original.discussions !== false
		});
	});

	test('reports every switch, on unless saved as off', async ({ request }) => {
		const { token } = await loginAsAdmin(request);
		const features = await setFeatures(request, token, {
			analytics: true,
			courses: true,
			newsletter: true,
			discussions: true
		});
		expect(features).toEqual({
			analytics: true,
			courses: true,
			newsletter: true,
			discussions: true
		});

		// A partial update leaves the other switches alone.
		const partial = await setFeatures(request, token, { newsletter: false });
		expect(partial).toEqual({
			analytics: true,
			courses: true,
			newsletter: false,
			discussions: true
		});
	});

	test('switching features off closes public routes but never unsubscribe', async ({ request }) => {
		const { token } = await loginAsAdmin(request);
		await setFeatures(request, token, { courses: false, newsletter: false, discussions: false });

		const view = await fetchFirstView(request, token);
		const slug = view?.slug ?? 'home';
		const subscribe = await request.post(`${apiBaseURL}/api/public/${slug}/subscribe`, {
			data: { email: 'feature-switch-e2e@example.com' }
		});
		expect(subscribe.status()).toBe(404);

		expect((await request.get(`${apiBaseURL}/api/courses`)).status()).toBe(404);
		expect((await request.get(`${apiBaseURL}/api/comments/posts/anything`)).status()).toBe(404);

		const unsubscribe = await request.post(`${apiBaseURL}/api/public/unsubscribe/by-email`, {
			data: { email: 'feature-switch-e2e@example.com' }
		});
		expect(unsubscribe.status()).toBe(200);
	});

	test('admin navigation follows the switches and keeps owner data reachable', async ({
		page,
		request
	}) => {
		const { token } = await loginAsAdmin(request);
		await setFeatures(request, token, {
			analytics: false,
			courses: false,
			newsletter: false,
			discussions: false
		});

		await seedAdminSession(page, request);
		await page.goto('/admin');

		// Nothing anywhere on the page links to a switched-off tool, whether in
		// the main sidebar or its section menus.
		await expect(page.locator('a[href="/admin/newsletter/compose"]')).toHaveCount(0);
		await expect(page.locator('a[href="/admin/courses"]')).toHaveCount(0);
		await expect(page.locator('a[href="/admin/comments"]')).toHaveCount(0);
		await expect(page.locator('a[href="/admin/analytics"]')).toHaveCount(0);

		// The main navigation reaches subscribers through the Audience hub.
		const mainNav = page.locator('aside nav').first();
		await mainNav.locator('a[href="/admin/audience"]').click();
		await expect(page).toHaveURL(/\/admin\/audience$/);
		// The hub renders its own <main> inside the admin layout's <main>.
		const hub = page.locator('main').last();
		await expect(hub.getByRole('heading', { level: 1 })).toBeVisible();
		await expect(hub.locator('a[href="/admin/subscribers"]')).toBeVisible();
		await expect(hub.locator('a[href="/admin/contacts"]')).toBeVisible();
		await expect(hub.locator('a[href="/admin/testimonials"]')).toBeVisible();
		await expect(hub.locator('a[href="/admin/newsletter"]')).toHaveCount(0);
		await expect(hub.locator('a[href="/admin/comments"]')).toHaveCount(0);
		await expect(hub.locator('a[href="/admin/analytics"]')).toHaveCount(0);

		await hub.locator('a[href="/admin/subscribers"]').click();
		await expect(page).toHaveURL(/\/admin\/subscribers$/);

		// The Content hub has no route to courses while they're off.
		await page.goto('/admin/content');
		await expect(page.locator('a[href="/admin/courses"]')).toHaveCount(0);

		// The way back is always there.
		await page.goto('/admin/settings/features');
		await expect(page.getByRole('switch').first()).toBeVisible();

		await setFeatures(request, token, {
			analytics: true,
			courses: true,
			newsletter: true,
			discussions: true
		});
		await page.goto('/admin/audience');
		const restored = page.locator('main').last();
		await expect(restored.locator('a[href="/admin/newsletter"]')).toBeVisible();
		await expect(restored.locator('a[href="/admin/comments"]')).toBeVisible();
		await expect(restored.locator('a[href="/admin/analytics"]')).toBeVisible();
		await expect(restored.locator('a[href="/admin/subscribers"]')).toBeVisible();
	});

	test('public newsletter signup disappears when the newsletter is off', async ({
		page,
		request
	}) => {
		const { token } = await loginAsAdmin(request);
		await setFeatures(request, token, { newsletter: false });

		await page.goto('/');
		await expect(page.locator('section#newsletter')).toHaveCount(0);
		await expect(page.locator('section.newsletter-subscribe')).toHaveCount(0);
	});
});

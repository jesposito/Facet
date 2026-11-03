import { test, expect } from '@playwright/test';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import { createPublicProjectAndPost, deletePublicProjectAndPost, loginAsAdmin } from './helpers';

const shouldRunAdmin = Boolean(adminEmail && adminPassword);

test.describe('Media management', () => {
	test.skip(!shouldRunAdmin, 'ADMIN_EMAIL and ADMIN_PASSWORD are required for media tests');

	test('can list media files', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const mediaRes = await request.get(`${apiBaseURL}/api/media`, {
			headers: { Authorization: token }
		});

		expect(mediaRes.ok()).toBeTruthy();
		const media = await mediaRes.json();
		expect(media).toHaveProperty('items');
		expect(Array.isArray(media.items)).toBeTruthy();
	});

	test('can detect orphaned media files', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const orphansRes = await request.get(`${apiBaseURL}/api/media?orphans=1`, {
			headers: { Authorization: token }
		});

		expect(orphansRes.ok()).toBeTruthy();
		const orphans = await orphansRes.json();
		expect(orphans).toHaveProperty('items');
		expect(Array.isArray(orphans.items)).toBeTruthy();
		expect(orphans.stats).toHaveProperty('orphanFiles');
	});

	test('bulk delete endpoint validates request', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		// Test with empty orphans array - should fail
		const emptyRes = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: {
				Authorization: token,
				'Content-Type': 'application/json'
			},
			data: {
				orphans: []
			}
		});

		expect(emptyRes.status()).toBe(400);
		const emptyBody = await emptyRes.json();
		expect(emptyBody.message).toMatch(/no orphans/i);
	});

	test('bulk delete endpoint enforces limit', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		// Create an array with 101 items (over the 100 limit)
		const tooMany = Array.from({ length: 101 }, (_, i) => `file${i}.jpg`);

		const limitRes = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: {
				Authorization: token,
				'Content-Type': 'application/json'
			},
			data: {
				orphans: tooMany
			}
		});

		expect(limitRes.status()).toBe(400);
		const limitBody = await limitRes.json();
		expect(limitBody.message).toContain('100');
	});

	test('bulk delete rejects invalid paths', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		// Attempt path traversal attack
		const maliciousRes = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: {
				Authorization: token,
				'Content-Type': 'application/json'
			},
			data: {
				orphans: ['../../../etc/passwd', '../../malicious.file']
			}
		});

		expect(maliciousRes.ok()).toBeTruthy(); // Returns 200 but with errors
		const maliciousBody = await maliciousRes.json();
		expect(maliciousBody).toHaveProperty('deleted');
		expect(maliciousBody).toHaveProperty('failed');
		expect(maliciousBody).toHaveProperty('errors');

		// All should have failed due to invalid paths
		expect(maliciousBody.failed).toBe(2);
		expect(maliciousBody.deleted).toBe(0);
		expect(maliciousBody.errors.length).toBe(2);
	});

	test('bulk delete requires authentication', async ({ request }) => {
		const noAuthRes = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			data: {
				orphans: ['test.jpg']
			}
		});

		// Should fail without auth
		expect(noAuthRes.status()).toBe(401);
	});

	test('external media collection is accessible', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const externalMediaRes = await request.get(
			`${apiBaseURL}/api/collections/external_media/records?page=1&perPage=10`,
			{
				headers: { Authorization: token }
			}
		);

		expect(externalMediaRes.ok()).toBeTruthy();
		const externalMedia = await externalMediaRes.json();
		expect(externalMedia).toHaveProperty('items');
		expect(Array.isArray(externalMedia.items)).toBeTruthy();
	});
});

test.describe('Media public rendering', () => {
	let token: string;
	let fixture: Awaited<ReturnType<typeof createPublicProjectAndPost>> | undefined;

	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		fixture = await createPublicProjectAndPost(request, token);
	});

	test.afterAll(async ({ request }) => {
		if (fixture) await deletePublicProjectAndPost(request, token, fixture);
	});

	test('project pages handle media_refs gracefully', async ({ page }) => {
		const response = await page.goto(`/projects/${fixture!.projectSlug}`);
		expect(response?.status()).toBe(200);
		await expect(page.locator('h1')).toBeVisible();
	});

	test('post pages handle media_refs gracefully', async ({ page }) => {
		const response = await page.goto(`/posts/${fixture!.postSlug}`);
		expect(response?.status()).toBe(200);
		await expect(page.locator('h1')).toBeVisible();
	});
});

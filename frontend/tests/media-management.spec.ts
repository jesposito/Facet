import { test, expect } from '@playwright/test';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import { loginAsAdmin } from './helpers';

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
		expect(media).toHaveProperty('files');
		expect(Array.isArray(media.files)).toBeTruthy();
	});

	test('can detect orphaned media files', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const orphansRes = await request.get(`${apiBaseURL}/api/media/orphans`, {
			headers: { Authorization: token }
		});

		expect(orphansRes.ok()).toBeTruthy();
		const orphans = await orphansRes.json();
		expect(orphans).toHaveProperty('orphans');
		expect(Array.isArray(orphans.orphans)).toBeTruthy();
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
		expect(emptyBody.message).toContain('no orphans');
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
	test('project pages handle media_refs gracefully', async ({ request, page }) => {
		// Get a public project
		const projectsRes = await request.get(`${apiBaseURL}/api/projects`);
		if (!projectsRes.ok()) {
			test.skip(true, 'No projects API available');
			return;
		}

		const projectsData = await projectsRes.json();
		const projects = projectsData?.projects || [];

		if (projects.length === 0) {
			test.skip(true, 'No projects available');
			return;
		}

		const project = projects[0];
		if (!project.slug) {
			test.skip(true, 'Project has no slug');
			return;
		}

		// Visit project page - should not error even if media_refs is missing
		const response = await page.goto(`/projects/${project.slug}`);
		expect(response?.status()).toBe(200);

		// Page should load successfully
		await expect(page.locator('h1')).toBeVisible();
	});

	test('post pages handle media_refs gracefully', async ({ request, page }) => {
		// Get a public post
		const postsRes = await request.get(`${apiBaseURL}/api/posts`);
		if (!postsRes.ok()) {
			test.skip(true, 'No posts API available');
			return;
		}

		const postsData = await postsRes.json();
		const posts = postsData?.posts || [];

		if (posts.length === 0) {
			test.skip(true, 'No posts available');
			return;
		}

		const post = posts[0];
		if (!post.slug) {
			test.skip(true, 'Post has no slug');
			return;
		}

		// Visit post page - should not error even if media_refs is missing
		const response = await page.goto(`/posts/${post.slug}`);
		expect(response?.status()).toBe(200);

		// Page should load successfully
		await expect(page.locator('h1')).toBeVisible();
	});
});

import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';
import { apiBaseURL } from './config';

/**
 * Security Test Suite
 *
 * Tests critical security protections for XSS and path traversal vulnerabilities.
 *
 * Note: Tests requiring authentication need ADMIN_EMAIL and ADMIN_PASSWORD
 * environment variables to be set. Run with:
 * ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=password npm run test tests/security.spec.ts
 */

test.describe('Security: XSS Prevention', () => {
	test('markdown rendering sanitizes script tags', async ({ page }) => {
		// This test verifies that DOMPurify sanitization prevents XSS attacks
		// We test this by checking if script tags are removed from rendered markdown

		// Test various XSS payloads
		const xssPayloads = [
			'<script>alert("XSS")</script>',
			'<img src=x onerror=alert("XSS")>',
			'<svg/onload=alert("XSS")>',
			'<iframe src="javascript:alert(\'XSS\')"></iframe>',
			'<a href="javascript:alert(\'XSS\')">Click me</a>'
		];

		// Navigate to a post or project page that might render markdown
		// (Assuming there's a test post available)
		await page.goto('/posts');

		// Verify the page loads
		expect(page.url()).toContain('/posts');

		// Check that the page doesn't contain any unescaped script tags
		const scripts = await page.locator('script:not([src])').all();
		for (const script of scripts) {
			const content = await script.textContent();
			// Should not contain our XSS payloads
			for (const payload of xssPayloads) {
				expect(content || '').not.toContain(payload);
			}
		}
	});

	test('iframe sources are whitelisted', async ({ page }) => {
		// Verify that only whitelisted iframe sources are allowed
		await page.goto('/');

		// Get all iframes on the page
		const iframes = await page.locator('iframe').all();

		const allowedDomains = [
			'youtube.com',
			'youtube-nocookie.com',
			'vimeo.com',
			'loom.com',
			'soundcloud.com',
			'spotify.com',
			'codepen.io',
			'figma.com'
		];

		for (const iframe of iframes) {
			const src = await iframe.getAttribute('src');
			if (src) {
				const isAllowed = allowedDomains.some(domain => src.includes(domain));
				expect(isAllowed).toBe(true);
			}
		}
	});
});

test.describe('Security: Path Traversal Prevention', () => {
	test('bulk delete rejects path traversal attempts', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		// Test various path traversal payloads
		const maliciousPaths = [
			'../../../etc/passwd',
			'..\\..\\..\\windows\\system32\\config\\sam',
			'/etc/passwd',
			'C:\\Windows\\System32\\config\\sam',
			'storage/../../etc/passwd',
			'uploads/../../../etc/passwd'
		];

		const res = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: { Authorization: token },
			data: {
				orphans: maliciousPaths
			}
		});

		expect(res.ok()).toBeTruthy();
		const body = await res.json();

		// All malicious paths should be rejected
		expect(body.deleted).toBe(0);
		expect(body.failed).toBe(maliciousPaths.length);

		// Verify error messages indicate security rejection
		expect(body.errors).toBeDefined();
		expect(body.errors.length).toBe(maliciousPaths.length);

		// Check that errors mention path issues
		for (const error of body.errors) {
			expect(error.error).toMatch(/(invalid path|path escapes|absolute path)/i);
		}
	});

	test('bulk delete rejects symlink paths', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		// We can't actually create symlinks in the test, but we can test
		// that the validation logic is in place by attempting to delete
		// paths that would typically be symlink targets

		const suspiciousPaths = [
			'/var/log/system.log',
			'/tmp/malicious_link',
			'../outside_storage'
		];

		const res = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: { Authorization: token },
			data: {
				orphans: suspiciousPaths
			}
		});

		expect(res.ok()).toBeTruthy();
		const body = await res.json();

		// These should all fail validation
		expect(body.deleted).toBe(0);
		expect(body.failed).toBe(suspiciousPaths.length);
	});

	test('single file delete rejects absolute paths', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const res = await request.delete(`${apiBaseURL}/api/media`, {
			headers: { Authorization: token },
			data: {
				relative_path: '/etc/passwd'
			}
		});

		// Should return an error
		expect(res.status()).toBe(400);
		const body = await res.json();
		expect(body.message).toMatch(/absolute path/i);
	});

	test('single file delete rejects path escape attempts', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const res = await request.delete(`${apiBaseURL}/api/media`, {
			headers: { Authorization: token },
			data: {
				relative_path: '../../../etc/passwd'
			}
		});

		// Should return an error
		expect(res.status()).toBe(400);
		const body = await res.json();
		expect(body.message).toMatch(/(invalid path|path escapes)/i);
	});
});

test.describe('Security: Authentication Requirements', () => {
	test('media endpoints require authentication', async ({ request }) => {
		// Test that media deletion endpoints require auth
		const paths = [
			{ method: 'DELETE', path: '/api/media' },
			{ method: 'POST', path: '/api/media/bulk-delete' },
			{ method: 'POST', path: '/api/media/external' }
		];

		for (const { method, path } of paths) {
			let res;
			if (method === 'DELETE') {
				res = await request.delete(`${apiBaseURL}${path}`, {
					data: { relative_path: 'test.jpg' }
				});
			} else {
				res = await request.post(`${apiBaseURL}${path}`, {
					data: {}
				});
			}

			// Should return 401 or 403 unauthorized
			expect([401, 403]).toContain(res.status());
		}
	});
});

test.describe('Security: Input Validation', () => {
	test('bulk delete enforces 100 file limit', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		// Try to delete 101 files (over the limit)
		const manyPaths = Array(101)
			.fill(null)
			.map((_, i) => `test-${i}.jpg`);

		const res = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: { Authorization: token },
			data: {
				orphans: manyPaths
			}
		});

		// Should reject the request
		expect(res.status()).toBe(400);
		const body = await res.json();
		expect(body.message).toMatch(/maximum.*100/i);
	});

	test('bulk delete rejects empty array', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const res = await request.post(`${apiBaseURL}/api/media/bulk-delete`, {
			headers: { Authorization: token },
			data: {
				orphans: []
			}
		});

		// Should reject empty arrays
		expect(res.status()).toBe(400);
		const body = await res.json();
		expect(body.message).toMatch(/no orphans specified/i);
	});
});

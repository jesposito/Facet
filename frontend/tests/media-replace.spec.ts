import { test, expect, type APIRequestContext } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin } from './helpers';

// #348 — media replace: uploading a replacement image must repoint every record
// that referenced the old library URL to the new file's URL in one step. This is
// the propagation that makes the feature a logo-fixer rather than a logo-breaker.
// API-level so it's deterministic under workers=1.

// A tiny valid 1x1 PNG, used as the replacement file.
const PNG_1x1 = Buffer.from(
	'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
	'base64'
);

// A stable old library URL the project record will reference before replacement.
const OLD_URL = `/api/files/uploads/e2e-media-replace-old/old-logo.png`;

async function createProject(request: APIRequestContext, token: string, libraryURL: string) {
	const res = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
		headers: { Authorization: token },
		data: {
			title: 'E2E Media Replace Project',
			slug: `e2e-media-replace-${Date.now()}`,
			cover_image_library_url: libraryURL
		}
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).id as string;
}

test.describe('Media replace propagation (#348)', () => {
	let token = '';
	let projectId = '';

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;
	});

	test.afterAll(async ({ request }) => {
		if (token && projectId) {
			await request.delete(
				`${apiBaseURL}/api/collections/projects/records/${projectId}`,
				{ headers: { Authorization: token } }
			);
		}
	});

	test('replacing a library image repoints every referencing record', async ({ request }) => {
		// A project references the old library URL via cover_image_library_url.
		projectId = await createProject(request, token, OLD_URL);

		// Sanity: the record points at the old URL before we replace.
		const before = await request.get(
			`${apiBaseURL}/api/collections/projects/records/${projectId}`,
			{ headers: { Authorization: token } }
		);
		expect(before.ok()).toBeTruthy();
		expect((await before.json()).cover_image_library_url).toBe(OLD_URL);

		// Replace: upload a new file + the old URL. The server stores the new file
		// and repoints every *_library_url reference.
		const res = await request.post(`${apiBaseURL}/api/media/replace`, {
			headers: { Authorization: token },
			multipart: {
				old_url: OLD_URL,
				file: {
					name: 'new-logo.png',
					mimeType: 'image/png',
					buffer: PNG_1x1
				}
			}
		});
		expect(res.ok()).toBeTruthy();
		const body = await res.json();
		expect(body.status).toBe('replaced');
		expect(typeof body.new_url).toBe('string');
		expect(body.new_url).not.toBe(OLD_URL);
		// PocketBase sanitizes the stored filename (e.g. new_logo_<hash>.png).
		expect(body.new_url).toMatch(/^\/api\/files\/[^/]+\/[^/]+\/new_logo.*\.png$/);
		// At least the project we created was updated.
		expect(body.updated_references).toBeGreaterThanOrEqual(1);

		const newURL = body.new_url as string;

		// The project record must now point at the new file's URL, not the old one.
		const after = await request.get(
			`${apiBaseURL}/api/collections/projects/records/${projectId}`,
			{ headers: { Authorization: token } }
		);
		expect(after.ok()).toBeTruthy();
		const rec = await after.json();
		expect(rec.cover_image_library_url).toBe(newURL);
		expect(rec.cover_image_library_url).not.toBe(OLD_URL);
	});
});

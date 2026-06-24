import { test, expect, type APIRequestContext } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin } from './helpers';

// Operator toggle for the Soft Premium footer CTA band (footer_cta_enabled).
// API-level so it's deterministic under workers=1: the /api/homepage response
// surfaces footer_cta_enabled, which the Footer component uses to gate the band.
// Default is true (zero regression for existing instances); setting it false
// hides the band; setting it back true shows it again.

async function homepageFooterCta(request: APIRequestContext): Promise<boolean | undefined> {
	const res = await request.get(`${apiBaseURL}/api/homepage`);
	expect(res.ok()).toBeTruthy();
	return (await res.json()).footer_cta_enabled as boolean | undefined;
}

test.describe('Footer CTA band operator toggle', () => {
	let token = '';
	let savedValue: boolean | undefined;

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;
		// Snapshot the current value so we can restore it afterwards.
		const cur = await request.get(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token }
		});
		if (cur.ok()) savedValue = (await cur.json()).footer_cta_enabled as boolean | undefined;
	});

	test.afterAll(async ({ request }) => {
		if (token) {
			await request.put(`${apiBaseURL}/api/site-settings`, {
				headers: { Authorization: token },
				// Restore to the snapshot, defaulting to true (the field default).
				data: { footer_cta_enabled: savedValue !== false }
			});
		}
	});

	test('default surfaces footer_cta_enabled as true (band shows)', async ({ request }) => {
		// Ensure the field is at its default-true state.
		const put = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data: { footer_cta_enabled: true }
		});
		expect(put.ok()).toBeTruthy();

		expect(await homepageFooterCta(request)).toBe(true);
	});

	test('setting footer_cta_enabled=false hides the band', async ({ request }) => {
		const put = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data: { footer_cta_enabled: false }
		});
		expect(put.ok()).toBeTruthy();
		// PUT response echoes the persisted value.
		expect((await put.json()).footer_cta_enabled).toBe(false);

		// Public homepage response reflects the disabled state.
		expect(await homepageFooterCta(request)).toBe(false);
	});

	test('setting footer_cta_enabled back to true shows the band again', async ({ request }) => {
		const put = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data: { footer_cta_enabled: true }
		});
		expect(put.ok()).toBeTruthy();
		expect((await put.json()).footer_cta_enabled).toBe(true);

		expect(await homepageFooterCta(request)).toBe(true);
	});
});

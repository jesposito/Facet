import { test, expect } from '@playwright/test';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import { fetchFirstView, loginAsAdmin } from './helpers';

const shouldRunAdmin = Boolean(adminEmail && adminPassword);

test.describe('Admin-secured flows', () => {
	test.skip(!shouldRunAdmin, 'ADMIN_EMAIL and ADMIN_PASSWORD are required for admin tests');

	test('can authenticate and list views', async ({ request }) => {
		const { token, record } = await loginAsAdmin(request);

		const viewsRes = await request.get(
			`${apiBaseURL}/api/collections/views/records?page=1&perPage=5`,
			{
				headers: { Authorization: token }
			}
		);
		expect(viewsRes.ok()).toBeTruthy();

		const views = await viewsRes.json();
		expect(Array.isArray(views.items)).toBeTruthy();
		expect(record).toBeTruthy();
	});

	test('share token lifecycle for a view', async ({ request }) => {
		const { token } = await loginAsAdmin(request);

		const view = await fetchFirstView(request, token);
		test.skip(!view, 'No views available to exercise share tokens');
		test.skip(!view?.slug, 'View is missing a slug');

		const generateRes = await request.post(`${apiBaseURL}/api/share/generate`, {
			headers: { Authorization: token },
			data: {
				view_id: view.id,
				name: 'playwright-share'
			}
		});
		expect(generateRes.ok()).toBeTruthy();
		const generated = await generateRes.json();
		expect(generated).toHaveProperty('token');
		expect(generated).toHaveProperty('id');

		const shareToken = generated.token as string;
		const shareId = generated.id as string;

		const validateRes = await request.post(`${apiBaseURL}/api/share/validate`, {
			data: { token: shareToken, view_id: view.id }
		});
		expect(validateRes.ok()).toBeTruthy();
		const validation = await validateRes.json();
		expect(validation.valid).toBe(true);
		expect(validation.view_slug || validation.view_id).toBeTruthy();

		const viewDataRes = await request.get(`${apiBaseURL}/api/view/${view.slug}/data`, {
			headers: { 'X-Share-Token': shareToken }
		});
		expect(viewDataRes.ok()).toBeTruthy();
		const viewData = await viewDataRes.json();
		expect(viewData.slug).toBe(view.slug);

		const revokeRes = await request.post(`${apiBaseURL}/api/share/revoke/${shareId}`, {
			headers: { Authorization: token }
		});
		expect(revokeRes.ok()).toBeTruthy();

		const validateAfter = await request.post(`${apiBaseURL}/api/share/validate`, {
			data: { token: shareToken, view_id: view.id }
		});
		expect(validateAfter.ok()).toBeTruthy();
		const validationAfter = await validateAfter.json();
		expect(validationAfter.valid).toBe(false);
	});
});

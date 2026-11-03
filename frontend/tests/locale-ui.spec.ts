import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';
import { apiBaseURL } from './config';

test('public unsubscribe form follows the configured locale', async ({ page, request }) => {
	const { token } = await loginAsAdmin(request);
	const headers = { Authorization: token };
	const before = await request.get(`${apiBaseURL}/api/site-settings`, { headers });
	expect(before.ok()).toBeTruthy();
	const original = (await before.json()).default_locale || 'en';
	try {
		const changed = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers,
			data: { default_locale: 'de' }
		});
		expect(changed.ok()).toBeTruthy();
		await page.goto('/unsubscribe');
		await expect(page.getByRole('heading', { level: 1 })).toHaveText('Vom Newsletter abmelden');
		await expect(page.getByLabel('E-Mail-Adresse', { exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Diese Adresse abmelden' })).toBeVisible();
	} finally {
		const restored = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers,
			data: { default_locale: original }
		});
		expect(restored.ok()).toBeTruthy();
	}
});

import { expect, test, type Page, type Route } from '@playwright/test';
import { seedAdminSession } from './helpers';

const siteSettings = {
	custom_css: 'body { letter-spacing: 0.01em; }',
	ga_measurement_id: 'G-RECOVERY',
	favicon: '',
	default_theme_mode: 'dark',
	design: 'classic',
	enabled_features: {
		analytics: false,
		courses: false,
		newsletter: true,
		discussions: false
	}
};

const smtpSettings = {
	enabled: true,
	host: 'smtp.invalid',
	port: 2525,
	username: 'recovery-fixture',
	password_set: true,
	auth_method: 'PLAIN',
	tls: true,
	sender_name: 'Recovery Fixture',
	sender_address: 'recovery@example.invalid'
};

const stripeStatus = {
	configured: true,
	secret_key_set: true,
	webhook_secret_set: true,
	source: 'database'
};

function json(route: Route, body: unknown, status = 200) {
	return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

async function mockProfile(page: Page) {
	await page.route(
		(url) => url.pathname === '/api/collections/profile/records',
		(route) =>
			json(route, {
				page: 1,
				perPage: 1,
				totalItems: 0,
				totalPages: 0,
				items: []
			})
	);
}

async function recordConfigurationWrites(page: Page) {
	const writes: string[] = [];
	await page.route(
		(url) => {
			const path = url.pathname;
			return (
				path === '/api/site-settings' ||
				path === '/api/smtp-settings' ||
				path === '/api/stripe-settings' ||
				path.startsWith('/api/collections/ai_providers/records')
			);
		},
		(route) => {
			if (route.request().method() === 'GET') return route.fallback();
			writes.push(`${route.request().method()} ${new URL(route.request().url()).pathname}`);
			return json(route, { error: 'Unexpected configuration write in recovery test' }, 500);
		}
	);
	return writes;
}

const failures = [
	{
		name: 'HTTP 503',
		respond: (route: Route) => json(route, { error: 'temporarily unavailable' }, 503)
	},
	{
		name: 'network failure',
		respond: (route: Route) => route.abort('failed')
	},
	{
		name: 'malformed JSON',
		respond: (route: Route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: '{not-json' })
	},
	{
		name: 'invalid response shape',
		respond: (route: Route) => json(route, { unexpected: true })
	}
];

function recoveryAlert(page: Page, resource: string) {
	return page.getByRole('alert').filter({ hasText: `Couldn't load ${resource}` });
}

test('pending appearance reads do not expose defaults or block a confirmed SMTP form', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	await mockProfile(page);
	const writes = await recordConfigurationWrites(page);
	let releaseSite!: () => void;
	const siteGate = new Promise<void>((resolve) => {
		releaseSite = resolve;
	});
	await page.route(
		(url) => url.pathname === '/api/site-settings',
		async (route) => {
			await siteGate;
			await json(route, siteSettings);
		}
	);
	await page.route(
		(url) => url.pathname === '/api/smtp-settings',
		(route) => json(route, smtpSettings)
	);

	await page.goto('/admin/settings/site');
	await expect(page.locator('#ga-id')).toHaveCount(0);
	await expect(page.locator('#smtp-host')).toBeVisible();
	await expect(page.locator('#smtp-host')).toBeEnabled();
	await expect(page.locator('#smtp-host')).toHaveValue('smtp.invalid');

	releaseSite();
	await expect(page.locator('#ga-id')).toBeVisible();
	await expect(page.locator('#ga-id')).toHaveValue('G-RECOVERY');
	await expect(page.locator('#ga-id')).toBeEnabled();
	expect(writes).toEqual([]);
});

for (const failure of failures) {
	test(`appearance ${failure.name} is unknown, retryable, and preserves SMTP drafts`, async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		await mockProfile(page);
		const writes = await recordConfigurationWrites(page);
		let attempts = 0;
		await page.route(
			(url) => url.pathname === '/api/site-settings',
			(route) => {
				attempts += 1;
				return attempts === 1 ? failure.respond(route) : json(route, siteSettings);
			}
		);
		await page.route(
			(url) => url.pathname === '/api/smtp-settings',
			(route) => json(route, smtpSettings)
		);

		await page.goto('/admin/settings/site');
		const alert = recoveryAlert(page, 'appearance settings');
		await expect(alert).toBeVisible();
		await expect(alert.getByRole('button', { name: 'Retry', exact: true })).toBeEnabled();
		await expect(page.locator('#ga-id')).toHaveCount(0);
		await expect(page.locator('#custom-css-title')).toHaveCount(0);

		await page.locator('#smtp-password').fill('dummy-draft-password');
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.locator('#ga-id')).toHaveValue('G-RECOVERY');
		await expect(page.locator('#smtp-password')).toHaveValue('dummy-draft-password');
		expect(attempts).toBe(2);
		expect(writes).toEqual([]);
	});
}

test('SMTP failure never enables a default save and leaves appearance independent', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	await mockProfile(page);
	const writes = await recordConfigurationWrites(page);
	await page.route(
		(url) => url.pathname === '/api/site-settings',
		(route) => json(route, siteSettings)
	);
	let smtpAttempts = 0;
	await page.route(
		(url) => url.pathname === '/api/smtp-settings',
		(route) => {
			smtpAttempts += 1;
			return smtpAttempts === 1
				? json(route, { error: 'temporarily unavailable' }, 503)
				: json(route, smtpSettings);
		}
	);

	await page.goto('/admin/settings/site');
	const alert = recoveryAlert(page, 'email settings');
	await expect(alert).toBeVisible();
	await expect(page.locator('#ga-id')).toBeEnabled();
	await expect(page.locator('#smtp-host')).toHaveCount(0);
	await expect(page.getByText('SMTP is not configured')).toHaveCount(0);

	await alert.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(alert).toHaveCount(0);
	await expect(page.locator('#smtp-host')).toHaveValue('smtp.invalid');
	await expect(page.locator('#smtp-host')).toBeEnabled();
	expect(writes).toEqual([]);
});

for (const failure of failures) {
	test(`features ${failure.name} cannot become all-true settings`, async ({ page, request }) => {
		await seedAdminSession(page, request);
		const writes = await recordConfigurationWrites(page);
		let attempts = 0;
		await page.route(
			(url) => url.pathname === '/api/site-settings',
			(route) => {
				attempts += 1;
				return attempts === 1 ? failure.respond(route) : json(route, siteSettings);
			}
		);

		await page.goto('/admin/settings/features');
		const alert = recoveryAlert(page, 'feature settings');
		await expect(alert).toBeVisible();
		await expect(page.getByRole('main').getByRole('switch')).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Save', exact: true })).toHaveCount(0);

		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByRole('switch', { name: 'Analytics', exact: true })).toHaveAttribute(
			'aria-checked',
			'false'
		);
		await expect(page.getByRole('switch', { name: 'Courses', exact: true })).toHaveAttribute(
			'aria-checked',
			'false'
		);
		await expect(page.getByRole('switch', { name: 'Newsletter', exact: true })).toHaveAttribute(
			'aria-checked',
			'true'
		);
		await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeEnabled();
		expect(writes).toEqual([]);
	});
}

for (const failure of failures) {
	test(`payment ${failure.name} is unavailable rather than missing configuration`, async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const writes = await recordConfigurationWrites(page);
		let attempts = 0;
		await page.route(
			(url) => url.pathname === '/api/stripe-settings',
			(route) => {
				attempts += 1;
				return attempts === 1 ? failure.respond(route) : json(route, stripeStatus);
			}
		);

		await page.goto('/admin/settings/payments');
		const alert = recoveryAlert(page, 'payment settings');
		await expect(alert).toBeVisible();
		await expect(page.getByText('Not Configured', { exact: true })).toHaveCount(0);
		await expect(page.locator('#stripe-secret-key')).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Test Connection', exact: true })).toHaveCount(0);

		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByText('Configured', { exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Test Connection', exact: true })).toBeEnabled();
		expect(writes).toEqual([]);
	});
}

for (const failure of failures) {
	test(`provider list ${failure.name} is unknown rather than empty`, async ({ page, request }) => {
		await seedAdminSession(page, request);
		const writes = await recordConfigurationWrites(page);
		let attempts = 0;
		await page.route(
			(url) => url.pathname === '/api/collections/ai_providers/records',
			(route) => {
				if (route.request().method() !== 'GET') return route.fallback();
				attempts += 1;
				return attempts === 1
					? failure.respond(route)
					: json(route, {
							page: 1,
							perPage: 50,
							totalItems: 0,
							totalPages: 0,
							items: []
						});
			}
		);

		await page.goto('/admin/settings/integrations');
		const alert = recoveryAlert(page, 'AI providers');
		await expect(alert).toBeVisible();
		await expect(page.getByText("Add one when you're ready", { exact: false })).toHaveCount(0);
		const addButton = page.getByRole('button', { name: '+ Add Provider', exact: true });
		await expect(addButton).toBeDisabled();
		await addButton.dispatchEvent('click');
		await expect(page.locator('#name')).toHaveCount(0);

		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(addButton).toBeEnabled();
		await expect(page.getByText("Add one when you're ready", { exact: false })).toBeVisible();
		expect(writes).toEqual([]);
	});
}

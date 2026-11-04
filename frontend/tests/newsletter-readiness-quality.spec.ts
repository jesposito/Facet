import { test, expect, type Page, type Route } from '@playwright/test';
import { seedAdminSession } from './helpers';

const SUBJECT = 'E2E readiness draft';
const BODY = 'This draft must survive every SMTP status check.';
const HISTORY_SUBJECT = 'E2E readiness history';

type SmtpHandler = (route: Route, attempt: number) => Promise<void>;

function json(route: Route, body: unknown, status = 200) {
	return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

async function mockReadinessApi(page: Page, smtpHandler: SmtpHandler) {
	let smtpChecks = 0;
	const writes: string[] = [];

	await page.route(
		(url) =>
			url.pathname.startsWith('/api/admin/newsletter') ||
			url.pathname === '/api/admin/subscribers/stats',
		async (route) => {
			const req = route.request();
			const path = new URL(req.url()).pathname;
			const method = req.method();

			if (method === 'GET' && path === '/api/admin/newsletter/smtp-status') {
				smtpChecks += 1;
				return smtpHandler(route, smtpChecks);
			}
			if (method === 'GET' && path === '/api/admin/newsletter/sends') {
				return json(route, {
					data: [
						{
							id: 'e2ereadiness001',
							subject: HISTORY_SUBJECT,
							status: 'partial',
							recipient_count: 2,
							list_id: '',
							list_name: '',
							sent_count: 1,
							failed_count: 1,
							pending_count: 0,
							uncertain_count: 0,
							skipped_count: 0,
							resumable_count: 1,
							open_count: 0,
							click_count: 0,
							sent_at: '2026-11-01 10:00:00.000Z',
							created: '2026-11-01 09:59:00.000Z'
						}
					]
				});
			}
			if (method === 'GET' && path === '/api/admin/newsletter/templates') return json(route, []);
			if (method === 'GET' && path === '/api/admin/newsletter-lists') {
				return json(route, { data: [] });
			}
			if (method === 'GET' && path === '/api/admin/subscribers/stats') {
				return json(route, { total_active: 0, total_unsubscribed: 0, total_bounced: 0 });
			}
			if (method === 'GET') return route.fallback();

			writes.push(`${method} ${path}`);
			return json(route, { error: 'refused by readiness test mock' }, 500);
		}
	);

	return {
		get smtpChecks() {
			return smtpChecks;
		},
		writes
	};
}

function sendTestButton(page: Page) {
	return page.getByRole('button', { name: 'Send Test', exact: true });
}

function sendAllButton(page: Page) {
	return page.getByRole('button', { name: 'Send to All', exact: true });
}

function resumeButton(page: Page) {
	return page.getByRole('button', { name: `Resume newsletter ${HISTORY_SUBJECT}`, exact: true });
}

function retryButton(page: Page) {
	return page.getByRole('button', { name: 'Retry SMTP check', exact: true });
}

function unavailableAlert(page: Page) {
	return page.getByRole('alert').filter({ hasText: 'SMTP status unavailable' });
}

async function fillDraft(page: Page) {
	await page.locator('#newsletter-subject').fill(SUBJECT);
	await page.locator('#newsletter-body').fill(BODY);
}

test('initial SMTP check keeps every sending action disabled until ready is confirmed', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	let releaseCheck!: () => void;
	const checkGate = new Promise<void>((resolve) => {
		releaseCheck = resolve;
	});
	await mockReadinessApi(page, async (route) => {
		await checkGate;
		await json(route, { configured: true, ready: true });
	});

	await page.goto('/admin/newsletter/compose');
	await fillDraft(page);

	await expect(sendTestButton(page)).toBeDisabled();
	await expect(sendAllButton(page)).toBeDisabled();
	await expect(resumeButton(page)).toBeDisabled();
	await expect(retryButton(page)).toBeDisabled();
	await expect(page.getByRole('alert').filter({ hasText: 'Checking SMTP status' })).toBeVisible();

	releaseCheck();
	await expect(sendTestButton(page)).toBeEnabled();
	await expect(sendAllButton(page)).toBeEnabled();
	await expect(resumeButton(page)).toBeEnabled();
});

const failures = [
	{
		name: 'HTTP 503',
		respond: (route: Route) => json(route, { error: 'status unavailable' }, 503)
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
		name: 'unexpected response shape',
		respond: (route: Route) => json(route, { ready: true })
	}
];

for (const failure of failures) {
	test(`${failure.name} reports unavailable SMTP and retries without losing the draft`, async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const mock = await mockReadinessApi(page, (route) => failure.respond(route));
		await page.goto('/admin/newsletter/compose');
		await fillDraft(page);

		await expect(unavailableAlert(page)).toBeVisible();
		await expect(unavailableAlert(page)).not.toContainText('SMTP not configured');
		await expect(sendTestButton(page)).toBeDisabled();
		await expect(sendAllButton(page)).toBeDisabled();
		await expect(resumeButton(page)).toBeDisabled();
		await expect(retryButton(page)).toBeEnabled();

		await retryButton(page).click();
		await expect(unavailableAlert(page)).toBeVisible();
		expect(mock.smtpChecks).toBe(2);
		await expect(page.locator('#newsletter-subject')).toHaveValue(SUBJECT);
		await expect(page.locator('#newsletter-body')).toHaveValue(BODY);
	});
}

test('a pending retry stays disabled and a ready response enables sending without clearing the draft', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	let releaseRetry!: () => void;
	const retryGate = new Promise<void>((resolve) => {
		releaseRetry = resolve;
	});
	await mockReadinessApi(page, async (route, attempt) => {
		if (attempt === 1) return json(route, { error: 'status unavailable' }, 503);
		await retryGate;
		await json(route, { configured: true, ready: true });
	});
	await page.goto('/admin/newsletter/compose');
	await fillDraft(page);
	await expect(unavailableAlert(page)).toBeVisible();

	await retryButton(page).click();
	await expect(retryButton(page)).toBeDisabled();
	await expect(sendTestButton(page)).toBeDisabled();
	await expect(sendAllButton(page)).toBeDisabled();
	await expect(resumeButton(page)).toBeDisabled();
	await expect(page.locator('#newsletter-subject')).toHaveValue(SUBJECT);
	await expect(page.locator('#newsletter-body')).toHaveValue(BODY);

	releaseRetry();
	await expect(unavailableAlert(page)).toHaveCount(0);
	await expect(sendTestButton(page)).toBeEnabled();
	await expect(sendAllButton(page)).toBeEnabled();
	await expect(resumeButton(page)).toBeEnabled();
	await expect(page.locator('#newsletter-subject')).toHaveValue(SUBJECT);
	await expect(page.locator('#newsletter-body')).toHaveValue(BODY);
});

for (const status of [
	{
		name: 'not configured',
		response: { configured: false, ready: false },
		heading: 'SMTP not configured'
	},
	{
		name: 'configured but not ready',
		response: { configured: true, ready: false },
		heading: 'SMTP not ready to send'
	}
]) {
	test(`a valid ${status.name} response keeps sending disabled with its distinct explanation`, async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		await mockReadinessApi(page, (route) => json(route, status.response));
		await page.goto('/admin/newsletter/compose');
		await fillDraft(page);

		const alert = page.getByRole('alert').filter({ hasText: status.heading });
		await expect(alert).toBeVisible();
		await expect(sendTestButton(page)).toBeDisabled();
		await expect(sendAllButton(page)).toBeDisabled();
		await expect(resumeButton(page)).toBeDisabled();
	});
}

test('unavailable SMTP guards send and resume handlers even for synthetic click events', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	const mock = await mockReadinessApi(page, (route) =>
		json(route, { error: 'status unavailable' }, 503)
	);
	await page.goto('/admin/newsletter/compose');
	await fillDraft(page);
	await expect(unavailableAlert(page)).toBeVisible();

	await sendTestButton(page).dispatchEvent('click');
	await sendAllButton(page).dispatchEvent('click');
	await resumeButton(page).dispatchEvent('click');

	await expect(page.getByRole('alertdialog')).toHaveCount(0);
	expect(mock.writes).toEqual([]);
});

test('the newsletter hub reports an unavailable status instead of missing configuration', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	await mockReadinessApi(page, (route) => json(route, { error: 'status unavailable' }, 503));
	await page.goto('/admin/newsletter');

	await expect(unavailableAlert(page)).toBeVisible();
	await expect(unavailableAlert(page)).not.toContainText('SMTP not configured');
	await expect(retryButton(page)).toBeEnabled();
});

test('the legacy compose URL uses the same readiness and retry behavior', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	await mockReadinessApi(page, (route, attempt) =>
		attempt === 1
			? json(route, { error: 'status unavailable' }, 503)
			: json(route, { configured: true, ready: true })
	);
	await page.goto('/admin/subscribers/compose');
	await fillDraft(page);
	await expect(unavailableAlert(page)).toBeVisible();
	await expect(sendAllButton(page)).toBeDisabled();
	await retryButton(page).click();
	await expect(sendAllButton(page)).toBeEnabled();
	await expect(page.locator('#newsletter-subject')).toHaveValue(SUBJECT);
	await expect(page.locator('#newsletter-body')).toHaveValue(BODY);
});

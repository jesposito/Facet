import { test, expect, type Page, type Request } from '@playwright/test';
import { seedAdminSession } from './helpers';

// Browser checks for the newsletter delivery history and SMTP readiness on both
// compose pages. Owner auth is real; the newsletter API is mocked per page so
// nothing is mailed and no backend SMTP or site settings change. Reads the mock
// doesn't know fall through to the backend; any other write is refused.

type MockSend = {
	id: string;
	subject: string;
	status: string;
	recipient_count: number;
	list_id: string;
	list_name: string;
	sent_count: number;
	failed_count: number;
	pending_count: number;
	uncertain_count: number;
	skipped_count: number;
	resumable_count: number;
	open_count: number;
	click_count: number;
	sent_at: string;
	created: string;
};

const PARTIAL = 'E2E delivery partial';
const INTERRUPTED = 'E2E delivery interrupted';
const FAILED = 'E2E delivery failed';
const LEGACY = 'E2E delivery legacy';

function sendRecord(fields: Partial<MockSend> & Pick<MockSend, 'id' | 'subject' | 'status'>) {
	return {
		recipient_count: 0,
		list_id: '',
		list_name: '',
		sent_count: 0,
		failed_count: 0,
		pending_count: 0,
		uncertain_count: 0,
		skipped_count: 0,
		resumable_count: 0,
		open_count: 0,
		click_count: 0,
		sent_at: '2026-11-01 10:00:00.000Z',
		created: '2026-11-01 09:59:00.000Z',
		...fields
	};
}

function history(): MockSend[] {
	return [
		// One accepted by SMTP, one rejected: Resume retries the rejected one.
		sendRecord({
			id: 'e2epartial00001',
			subject: PARTIAL,
			status: 'partial',
			recipient_count: 2,
			sent_count: 1,
			failed_count: 1,
			resumable_count: 1
		}),
		// Stopped mid-send with only unknown answers left: nothing safe to retry.
		sendRecord({
			id: 'e2einterrupt001',
			subject: INTERRUPTED,
			status: 'interrupted',
			recipient_count: 3,
			sent_count: 1,
			uncertain_count: 2
		}),
		// Resumable on its own, but must wait while another send is live.
		sendRecord({
			id: 'e2efailed000001',
			subject: FAILED,
			status: 'failed',
			recipient_count: 2,
			failed_count: 2,
			resumable_count: 2
		}),
		// Logged before per-recipient tracking: the failures are history, and
		// there is no record of who to retry.
		sendRecord({
			id: 'e2elegacy000001',
			subject: LEGACY,
			status: 'failed',
			recipient_count: 5,
			sent_count: 3,
			failed_count: 2
		})
	];
}

type Mock = { sends: MockSend[]; resumed: string[]; refusedWrites: string[] };

async function mockNewsletterApi(page: Page, smtp: { configured: boolean; ready: boolean }) {
	const mock: Mock = { sends: history(), resumed: [], refusedWrites: [] };
	await page.route(
		(url) => url.pathname.startsWith('/api/admin/newsletter'),
		async (route) => {
			const req = route.request();
			const path = new URL(req.url()).pathname;
			const method = req.method();
			const json = (body: unknown, status = 200) =>
				route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });

			if (method === 'GET' && path === '/api/admin/newsletter/smtp-status') {
				return json({ ...smtp, tested: false, delivery: 'unknown', host: 'smtp.dummy.invalid' });
			}
			if (method === 'GET' && path === '/api/admin/newsletter/sends') {
				return json({ data: mock.sends });
			}
			if (method === 'GET' && path === '/api/admin/newsletter/templates') return json([]);
			if (method === 'GET' && path === '/api/admin/newsletter-lists') return json({ data: [] });

			const resume = path.match(/^\/api\/admin\/newsletter\/sends\/([^/]+)\/resume$/);
			if (method === 'POST' && resume) {
				const id = resume[1];
				mock.resumed.push(id);
				mock.sends = mock.sends.map((s) =>
					s.id === id ? { ...s, status: 'sending', resumable_count: 0 } : s
				);
				return json({ status: 'sending', send_id: id });
			}

			if (method === 'GET') return route.fallback();
			mock.refusedWrites.push(`${method} ${path}`);
			return json({ error: 'refused by test mock' }, 500);
		}
	);
	return mock;
}

function row(page: Page, subject: string) {
	return page.locator('table tbody tr').filter({ hasText: subject });
}

function resumeButton(page: Page, subject: string) {
	return page.getByRole('button', { name: `Resume newsletter ${subject}`, exact: true });
}

function deleteButton(page: Page, subject: string) {
	return page.getByRole('button', { name: `Delete send record: ${subject}`, exact: true });
}

for (const route of ['/admin/newsletter/compose', '/admin/subscribers/compose']) {
	test.describe(`newsletter delivery history on ${route}`, () => {
		test('reports outcomes honestly and resumes only what is safe to retry', async ({
			page,
			request
		}) => {
			await seedAdminSession(page, request);
			const mock = await mockNewsletterApi(page, { configured: true, ready: true });
			await page.goto(route);

			const partial = row(page, PARTIAL);
			const interrupted = row(page, INTERRUPTED);
			await expect(partial).toHaveCount(1);
			await expect(page.getByText('SMTP not ready to send')).toHaveCount(0);

			// Accepted counts sit beside the total; only non-zero outcomes are listed.
			await expect(partial).toContainText('Partially completed');
			await expect(partial).toContainText('1/2');
			await expect(partial.getByTestId('send-outcomes').locator('li')).toHaveText(['1 failed']);
			await expect(interrupted).toContainText('Interrupted');
			await expect(interrupted).toContainText('1/3');
			await expect(interrupted.getByTestId('send-outcomes').locator('li')).toHaveText([
				'2 uncertain'
			]);
			await expect(page.getByTestId('uncertain-explainer')).toContainText(
				'Resume never sends to them again'
			);

			// Uncertain-only has nothing to retry; known failures do. A legacy
			// send shows its failures but has no ledger to resume from.
			await expect(resumeButton(page, INTERRUPTED)).toHaveCount(0);
			await expect(row(page, LEGACY).getByTestId('send-outcomes').locator('li')).toHaveText([
				'2 failed'
			]);
			await expect(resumeButton(page, LEGACY)).toHaveCount(0);
			await expect(resumeButton(page, PARTIAL)).toBeEnabled();
			await expect(resumeButton(page, FAILED)).toBeEnabled();
			await expect(deleteButton(page, PARTIAL)).toBeEnabled();

			await resumeButton(page, PARTIAL).click();
			const dialog = page.getByRole('alertdialog');
			await expect(dialog).toContainText('Resume newsletter');
			await expect(dialog).toContainText('Only pending and known failed recipients are retried');
			await expect(dialog).toContainText('Accepted and uncertain outcomes are never sent again');

			const resumeRequest = page.waitForRequest(
				(r: Request) => r.method() === 'POST' && /\/sends\/[^/]+\/resume$/.test(r.url())
			);
			await dialog.getByRole('button', { name: 'Resume', exact: true }).click();
			expect(new URL((await resumeRequest).url()).pathname).toBe(
				'/api/admin/newsletter/sends/e2epartial00001/resume'
			);

			// The reloaded history shows the live job; it can't be deleted or
			// resumed, and other resumable sends wait for it.
			await expect(partial).toContainText('Sending');
			await expect(deleteButton(page, PARTIAL)).toBeDisabled();
			await expect(resumeButton(page, PARTIAL)).toHaveCount(0);
			await expect(resumeButton(page, FAILED)).toBeDisabled();
			await expect(resumeButton(page, FAILED)).toHaveAttribute(
				'title',
				'A newsletter send is already in progress'
			);
			await expect(deleteButton(page, INTERRUPTED)).toBeEnabled();

			expect(mock.resumed).toEqual(['e2epartial00001']);
			expect(mock.refusedWrites).toEqual([]);
		});

		test('configured but not ready SMTP blocks sending and resuming', async ({ page, request }) => {
			await seedAdminSession(page, request);
			const mock = await mockNewsletterApi(page, { configured: true, ready: false });
			await page.goto(route);

			const banner = page.getByRole('alert').filter({ hasText: 'SMTP not ready to send' });
			await expect(banner).toBeVisible();
			await expect(banner).toContainText('SMTP is configured, but newsletters cannot go out yet');

			// A complete draft, so the buttons are off because of SMTP and not
			// because the form or audience is empty.
			await page.locator('#newsletter-subject').fill('E2E readiness draft');
			await page.locator('#newsletter-body').fill('Hello from the readiness check.');
			await expect(page.getByRole('button', { name: 'Send to All', exact: true })).toBeDisabled();
			await expect(page.getByRole('button', { name: 'Send Test', exact: true })).toBeDisabled();

			// The partial send is resumable in principle, but not without SMTP.
			await expect(row(page, PARTIAL)).toHaveCount(1);
			await expect(resumeButton(page, PARTIAL)).toBeDisabled();
			await expect(resumeButton(page, INTERRUPTED)).toHaveCount(0);
			await expect(resumeButton(page, LEGACY)).toHaveCount(0);

			expect(mock.resumed).toEqual([]);
			expect(mock.refusedWrites).toEqual([]);
		});
	});
}

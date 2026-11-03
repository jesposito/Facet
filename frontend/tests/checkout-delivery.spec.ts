/**
 * Checkout return page: confirming a Stripe session and having the access
 * link emailed.
 *
 * POST /api/content/checkout/confirm is mocked in the browser, so nothing
 * reaches Stripe or sends mail. The backend tests cover the real endpoint;
 * these check that the page follows its contract: every answer is shown with
 * the page's own wording, polling stays inside the endpoint's 10-a-minute
 * limit and stops, retries keep the same session, and nothing here grants
 * access or leaks the session ID.
 *
 * The item slug doesn't exist, so the server-side access check finds nothing
 * to unlock and the page goes on to confirm the session.
 */
import { test, expect, type Page, type Route } from '@playwright/test';

const run = Date.now().toString(36);
const slug = `delivery-check-${run}`;
const sessionId = `cs_test_a1B2c3D4e5F6g7H8i9J0_${run}`;
const returnPath = `/checkout/return?type=posts&slug=${slug}&session_id=${sessionId}`;
const confirmUrl = '**/api/content/checkout/confirm';

type ConfirmBody = { session_id: string; content_type: string; slug: string; resend: boolean };

interface Reply {
	status?: number;
	body: Record<string, unknown>;
	headers?: Record<string, string>;
}

/** Answers confirm requests in order, repeating the last reply. */
async function mockConfirm(page: Page, replies: Reply[]) {
	const calls: Array<{ method: string; body: ConfirmBody }> = [];
	await page.route(confirmUrl, async (route: Route) => {
		const request = route.request();
		calls.push({ method: request.method(), body: request.postDataJSON() });
		const reply = replies[Math.min(calls.length - 1, replies.length - 1)];
		await route.fulfill({
			status: reply.status ?? 200,
			contentType: 'application/json',
			headers: reply.headers,
			body: JSON.stringify(reply.body)
		});
	});
	return calls;
}

/**
 * Counts confirm requests as the page makes them. The button's click handler
 * calls fetch synchronously, so once a key press or dispatched click has
 * returned, this count already includes any request it started. That makes
 * "nothing was sent" checkable without waiting for a request that never comes.
 */
async function countConfirmFetches(page: Page) {
	await page.addInitScript(() => {
		const original = window.fetch.bind(window);
		const counted = window as unknown as { __confirmFetches: number };
		counted.__confirmFetches = 0;
		window.fetch = (input, init) => {
			const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
			if (new URL(url, location.href).pathname === '/api/content/checkout/confirm')
				counted.__confirmFetches++;
			return original(input, init);
		};
	});
	return () =>
		page.evaluate(() => (window as unknown as { __confirmFetches: number }).__confirmFetches);
}

const panel = (page: Page) => page.getByTestId('checkout-return');
const action = (page: Page) => page.getByTestId('confirm-action');
const liveStatus = (page: Page) => panel(page).getByRole('status');
const alert = (page: Page) => panel(page).getByRole('alert');

async function idle(page: Page) {
	await expect(panel(page)).toHaveAttribute('aria-busy', 'false');
}

test.describe('checkout return delivery', () => {
	test('a confirmed payment shows the masked email and grants nothing', async ({
		page,
		context
	}) => {
		const calls = await mockConfirm(page, [
			{ body: { status: 'confirmation_sent', email_hint: 'b***@example.com' } }
		]);
		const leaks: string[] = [];
		page.on('console', (message) => {
			if (message.text().includes(sessionId)) leaks.push(`console: ${message.text()}`);
		});
		page.on('request', (request) => {
			if (request.resourceType() !== 'document' && request.url().includes(sessionId))
				leaks.push(`url: ${request.url()}`);
		});

		const response = await page.goto(returnPath);
		expect(response!.headers()['cache-control']).toContain('no-store');
		expect(response!.headers()['x-robots-tag']).toContain('noindex');
		expect(await response!.text(), 'session ID copied into the HTML').not.toContain(sessionId);

		await expect(panel(page)).toHaveAttribute('data-status', 'sent');
		await expect(page.getByRole('heading', { level: 1 })).toHaveText('Check your email');
		await expect(liveStatus(page)).toContainText('b***@example.com');
		await expect(liveStatus(page)).toContainText('Open it on any device');

		expect(calls).toHaveLength(1);
		expect(calls[0]).toEqual({
			method: 'POST',
			body: { session_id: sessionId, content_type: 'posts', slug, resend: false }
		});
		expect((await context.cookies()).map((cookie) => cookie.name)).not.toContain('facet_purchases');
		expect(leaks).toEqual([]);
	});

	test('a pending payment is polled with backoff, then stops', async ({ page }) => {
		await page.clock.install();
		const calls = await mockConfirm(page, [{ body: { status: 'pending' } }]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'pending');
		await expect(liveStatus(page)).toContainText('checks again on its own');
		await idle(page);

		// The page's schedule: 10, 15, 20, 30, 45, 60, 60, 60 seconds.
		const delays = [10, 15, 20, 30, 45, 60, 60, 60];
		let elapsed = 0;
		for (const [i, delay] of delays.entries()) {
			await page.clock.runFor(delay * 1000);
			elapsed += delay;
			await expect.poll(() => calls.length).toBe(i + 2);
			await idle(page);
			// Four requests in the first minute, well inside 10 a minute.
			if (elapsed <= 60) expect(calls.length).toBeLessThanOrEqual(4);
		}

		await page.clock.runFor(120_000);
		expect(calls).toHaveLength(delays.length + 1);
		await expect(liveStatus(page)).toContainText('stopped checking');

		await action(page).focus();
		await page.keyboard.press('Enter');
		await expect.poll(() => calls.length).toBe(delays.length + 2);
		for (const call of calls) expect(call.body.session_id).toBe(sessionId);
	});

	test('leaving the page cancels polling', async ({ page }) => {
		await page.clock.install();
		const calls = await mockConfirm(page, [{ body: { status: 'pending' } }]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'pending');
		await idle(page);

		await page.getByRole('link', { name: /Back to/ }).click();
		await expect(page).toHaveURL(new RegExp(`/posts/${slug}$`));
		await page.clock.runFor(600_000);
		expect(calls).toHaveLength(1);
	});

	for (const [status, heading] of [
		['failed', 'The payment didn’t go through'],
		['refunded', 'This purchase was refunded']
	] as const) {
		test(`a ${status} session is explained and unlocks nothing`, async ({ page, context }) => {
			const calls = await mockConfirm(page, [{ body: { status } }]);
			await page.goto(returnPath);
			await expect(panel(page)).toHaveAttribute('data-status', status);
			await expect(page.getByRole('heading', { level: 1 })).toHaveText(heading);
			await expect(alert(page)).toBeVisible();
			await expect(action(page)).toHaveCount(0);
			expect(calls).toHaveLength(1);
			expect((await context.cookies()).map((cookie) => cookie.name)).not.toContain(
				'facet_purchases'
			);
		});
	}

	test('a failed email can be retried with the same session after a cooldown', async ({ page }) => {
		// Freeze time before the page loads so the 10-second cooldown can't run
		// out while the disabled button is pressed. Pausing at a fixed point a
		// minute after the install time never asks the clock to go backwards.
		const start = new Date('2026-01-01T09:00:00Z');
		await page.clock.install({ time: start });
		await page.clock.pauseAt(new Date(start.getTime() + 60_000));
		const pageCalls = await countConfirmFetches(page);
		const raw = `smtp 550 relay denied internal-${run}`;
		const calls = await mockConfirm(page, [
			{ status: 502, body: { status: 'delivery_failed', error: raw } },
			{ body: { status: 'confirmation_sent', email_hint: 'b***@example.com' } }
		]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'delivery_failed');
		await expect(alert(page)).toContainText('access email couldn’t be sent');
		await expect(panel(page)).not.toContainText(raw);

		await expect(action(page)).toHaveAttribute('aria-disabled', 'true');
		await expect(action(page)).toHaveText(/in 0:\d\d/);
		expect(await pageCalls()).toBe(1);

		// Playwright's click waits for an enabled element, so press the button
		// in ways that don't wait: the keyboard, and a dispatched click.
		await action(page).focus();
		await page.keyboard.press('Enter');
		await page.keyboard.press('Space');
		await action(page).dispatchEvent('click');
		expect(await pageCalls()).toBe(1);
		await expect(action(page)).toHaveAttribute('aria-disabled', 'true');
		await expect(panel(page)).toHaveAttribute('data-status', 'delivery_failed');
		expect(calls).toHaveLength(1);

		await page.clock.runFor(10_000);
		await expect(action(page)).toHaveAttribute('aria-disabled', 'false');
		await action(page).click();
		await expect(panel(page)).toHaveAttribute('data-status', 'sent');
		expect(calls.map((call) => call.body)).toEqual([
			{ session_id: sessionId, content_type: 'posts', slug, resend: false },
			{ session_id: sessionId, content_type: 'posts', slug, resend: false }
		]);
	});

	test('email delivery being unavailable is reported without the raw error', async ({ page }) => {
		const raw = `smtp not configured internal-${run}`;
		await mockConfirm(page, [
			{ status: 503, body: { status: 'delivery_unavailable', error: raw } }
		]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'delivery_unavailable');
		await expect(alert(page)).toContainText('can’t send email right now');
		await expect(panel(page)).not.toContainText(raw);
	});

	test('resending is a button press and honours the cooldown', async ({ page }) => {
		await page.clock.install();
		const pageCalls = await countConfirmFetches(page);
		const calls = await mockConfirm(page, [
			{ body: { status: 'confirmation_sent', email_hint: 'b***@example.com' } },
			{ body: { status: 'confirmation_sent', email_hint: 'b***@example.com', retry_after: 90 } }
		]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'sent');
		expect(calls).toHaveLength(1);
		expect(calls[0].body.resend).toBe(false);

		await action(page).focus();
		await page.keyboard.press('Enter');
		await expect(page.getByTestId('confirm-notice')).toContainText('went out a moment ago');
		await expect(action(page)).toHaveText(/in 1:30/);
		await expect(action(page)).toBeFocused();
		expect(calls[1]).toEqual({
			method: 'POST',
			body: { session_id: sessionId, content_type: 'posts', slug, resend: true }
		});

		await page.keyboard.press('Enter');
		expect(await pageCalls()).toBe(2);
		expect(calls).toHaveLength(2);

		await page.clock.runFor(91_000);
		await expect(action(page)).toHaveText('Send the email again');
		await page.keyboard.press('Enter');
		await expect.poll(() => calls.length).toBe(3);
		expect(calls[2].body).toEqual({
			session_id: sessionId,
			content_type: 'posts',
			slug,
			resend: true
		});
	});

	test('rate limiting waits for Retry-After and keeps the session', async ({ page }) => {
		await page.clock.install();
		const calls = await mockConfirm(page, [
			{ status: 429, headers: { 'Retry-After': '30' }, body: { error: 'Too many requests' } },
			{ body: { status: 'pending' } }
		]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'rate_limited');
		await expect(action(page)).toHaveText(/in 0:30/);
		await expect(panel(page)).not.toContainText('Too many requests');

		await page.clock.runFor(30_000);
		await action(page).click();
		await expect(panel(page)).toHaveAttribute('data-status', 'pending');
		expect(calls.map((call) => call.body.session_id)).toEqual([sessionId, sessionId]);
	});

	for (const [status, kind, heading] of [
		[404, 'not_found', 'We couldn’t find this checkout'],
		[400, 'not_found', 'We couldn’t find this checkout'],
		[409, 'mismatch', 'This checkout is for a different item'],
		[500, 'error', 'We couldn’t check your payment']
	] as const) {
		test(`a ${status} answer shows the ${kind} state`, async ({ page }) => {
			const raw = `raw-server-error-${run}`;
			await mockConfirm(page, [{ status, body: { error: raw } }]);
			await page.goto(returnPath);
			await expect(panel(page)).toHaveAttribute('data-status', kind);
			await expect(page.getByRole('heading', { level: 1 })).toHaveText(heading);
			await expect(alert(page)).toBeVisible();
			await expect(panel(page)).not.toContainText(raw);
		});
	}

	test('an unexpected success body is not trusted', async ({ page }) => {
		await mockConfirm(page, [
			{ body: { status: 'unlocked', access_token: `tok-${run}`, email_hint: '<b>x</b>' } }
		]);
		await page.goto(returnPath);
		await expect(panel(page)).toHaveAttribute('data-status', 'error');
		await expect(panel(page)).not.toContainText(`tok-${run}`);
	});

	test('a malformed session ID is never sent', async ({ page }) => {
		const calls = await mockConfirm(page, [{ body: { status: 'confirmation_sent' } }]);
		await page.goto(`/checkout/return?type=posts&slug=${slug}&session_id=not-a-session`);
		await expect(panel(page)).not.toHaveAttribute('data-status', /checking|sent/);
		await expect(page.getByRole('heading', { level: 1 })).not.toHaveText('Check your email');
		expect(calls).toHaveLength(0);
	});
});

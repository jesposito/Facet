import { test, expect } from '@playwright/test';
import { spawn, type ChildProcess } from 'node:child_process';
import { once } from 'node:events';
import http from 'node:http';
import { fileURLToPath } from 'node:url';

// The /s/ entry decides what a recipient sees during server rendering, which a
// browser route can't reach. These tests run the built frontend (npm run build)
// against a fake backend that fails on cue. No real backend or provider is used.

const buildEntry = fileURLToPath(new URL('../build/index.js', import.meta.url));
const SLUG = 'recovered-facet';
const STALE_SLUG = 'stale-facet';
const UPSTREAM_BODY = 'upstream-body-marker';

type Answer = (response: http.ServerResponse) => void;

function json(status: number, body: unknown, headers: Record<string, string> = {}): Answer {
	return (response) => {
		response.writeHead(status, { 'content-type': 'application/json', ...headers });
		response.end(JSON.stringify(body));
	};
}

function limited(retryAfter?: string): Answer {
	const headers: Record<string, string> = retryAfter ? { 'Retry-After': retryAfter } : {};
	return json(429, { error: 'too many requests' }, headers);
}

const invalid = json(200, { valid: false, error: 'invalid token' });
const admitted = json(
	200,
	{ valid: true, view_id: 'fixture-view', view_slug: SLUG },
	{
		'X-Share-Visit': 'fixture-proof',
		'X-Share-Visit-Expires': new Date(Date.now() + 60 * 60 * 1000).toISOString()
	}
);
const broken = json(503, { error: UPSTREAM_BODY });
const dropped: Answer = (response) => response.socket?.destroy();
const unreadable: Answer = (response) => {
	response.writeHead(200, { 'content-type': 'text/html' });
	response.end(`<html>${UPSTREAM_BODY}</html>`);
};

const outcomes: [string, Answer, string, string | null][] = [
	['an invalid link', invalid, 'invalid', null],
	['a refused request', json(400, { error: 'invalid request' }), 'invalid', null],
	['a denied admission', limited('30'), 'rate_limited', '30'],
	['a very long wait', limited('86400'), 'rate_limited', '300'],
	['no stated wait', limited(), 'rate_limited', null],
	['a zero wait', limited('0'), 'rate_limited', null],
	['a negative wait', limited('-5'), 'rate_limited', null],
	['a fractional wait', limited('1.5'), 'rate_limited', null],
	['a date wait', limited('Wed, 21 Oct 2026 07:28:00 GMT'), 'rate_limited', null],
	['a 5xx', broken, 'unavailable', null],
	['a dropped connection', dropped, 'unavailable', null],
	['an unreadable answer', unreadable, 'unavailable', null],
	['a valid answer with no Facet', json(200, { valid: true }), 'unavailable', null]
];

async function listen(server: http.Server) {
	await new Promise<void>((resolve, reject) => {
		server.once('error', reject);
		server.listen(0, '127.0.0.1', () => resolve());
	});
	const address = server.address();
	if (!address || typeof address === 'string') throw new Error('Expected a TCP server');
	return address.port;
}

async function close(server: http.Server) {
	server.closeAllConnections();
	await new Promise<void>((resolve) => server.close(() => resolve()));
}

test.describe('share entry recovery', () => {
	const answers: Answer[] = [];
	const tokens: string[] = [];
	let frontend: ChildProcess;
	let origin: string;
	let logs = '';

	const backend = http.createServer((request, response) => {
		if (request.method !== 'POST' || request.url !== '/api/share/validate') {
			response.writeHead(404).end();
			return;
		}
		let body = '';
		request.on('data', (chunk) => (body += chunk));
		request.on('end', () => {
			tokens.push(JSON.parse(body).token);
			(answers.shift() ?? invalid)(response);
		});
	});

	test.beforeAll(async () => {
		const backendPort = await listen(backend);
		const reserved = http.createServer();
		const port = await listen(reserved);
		await close(reserved);
		origin = `http://127.0.0.1:${port}`;
		frontend = spawn(process.execPath, [buildEntry], {
			env: {
				...process.env,
				// These tests call Node directly, with no proxy to set the header.
				ADDRESS_HEADER: '',
				POCKETBASE_URL: `http://127.0.0.1:${backendPort}`,
				HOST: '127.0.0.1',
				PORT: String(port),
				ORIGIN: origin
			},
			stdio: ['ignore', 'pipe', 'pipe']
		});
		frontend.stdout?.on('data', (chunk) => (logs += chunk));
		frontend.stderr?.on('data', (chunk) => (logs += chunk));
		await expect
			.poll(
				async () => {
					if (frontend.exitCode !== null) throw new Error(`Frontend exited: ${logs}`);
					return fetch(origin).then(
						() => true,
						() => false
					);
				},
				{ timeout: 10_000 }
			)
			.toBe(true);
	});

	test.afterAll(async () => {
		if (frontend?.exitCode === null) {
			const exited = once(frontend, 'exit');
			frontend.kill('SIGTERM');
			await exited;
		}
		await close(backend);
	});

	test.beforeEach(() => {
		answers.length = 0;
		tokens.length = 0;
	});

	function enter(token: string, answer: Answer) {
		answers.push(answer);
		return fetch(`${origin}/s/${token}`, { redirect: 'manual' });
	}

	for (const [name, answer, code, retryAfter] of outcomes) {
		test(`${name} answers ${code} without admitting anyone`, async () => {
			const response = await enter('entry-token', answer);
			const html = await response.text();

			expect(response.status).toBe(200);
			expect(html).toContain(`data-code="${code}"`);
			expect(response.headers.get('retry-after')).toBe(retryAfter);
			expect(response.headers.get('cache-control')).toBe('private, no-store');
			expect(response.headers.get('referrer-policy')).toBe('no-referrer');
			expect(response.headers.getSetCookie().join()).not.toContain('me_share');
			expect(tokens).toEqual(['entry-token']);
		});
	}

	test('a good link still lands on the clean URL with HttpOnly cookies', async () => {
		const response = await enter('good-token', admitted);
		const cookies = response.headers.getSetCookie();

		expect(response.status).toBe(302);
		expect(response.headers.get('location')).toBe(`/${SLUG}`);
		expect(response.headers.get('cache-control')).toBe('private, no-store');
		expect(response.headers.get('referrer-policy')).toBe('no-referrer');
		for (const name of ['me_share_token=good-token', 'me_share_visit=fixture-proof']) {
			const cookie = cookies.find((value) => value.startsWith(name));
			expect(cookie, name).toContain('HttpOnly');
		}
		expect(tokens).toEqual(['good-token']);
	});

	test('a denied admission waits for the cooldown and the recipient', async ({ page, context }) => {
		answers.push(limited('5'), admitted);
		const response = await page.goto(`${origin}/s/cooldown-token`);
		expect(response?.status()).toBe(200);
		expect(response?.headers()['retry-after']).toBe('5');
		await expect(page.getByRole('heading', { name: 'Too Many Requests Right Now' })).toBeVisible();
		await expect(page.getByText("This attempt didn't use it.")).toBeVisible();

		const retry = page.getByRole('button', { name: /^Try again/ });
		await expect(retry).toHaveText(/^Try again in \ds$/);
		await expect(retry).toBeDisabled();
		await expect(retry).toHaveText('Try again', { timeout: 10_000 });
		await expect(retry).toBeEnabled();
		expect(tokens).toEqual(['cooldown-token']);
		const names = (await context.cookies()).map((cookie) => cookie.name);
		expect(names).not.toContain('me_share_visit');

		await retry.click();
		await page.waitForURL(`${origin}/${SLUG}`);
		expect(tokens).toEqual(['cooldown-token', 'cooldown-token']);
		const visit = (await context.cookies()).find((cookie) => cookie.name === 'me_share_visit');
		expect(visit?.httpOnly).toBe(true);
	});

	for (const [name, answer] of [
		['a 5xx', broken],
		['a dropped connection', dropped]
	] as const) {
		test(`${name} warns before the recipient risks another use`, async ({ page }) => {
			answers.push(answer, invalid);
			const response = await page.goto(`${origin}/s/unknown-token`);
			expect(response?.status()).toBe(200);
			await expect(
				page.getByRole('heading', { name: "We Couldn't Check This Link" })
			).toBeVisible();
			await expect(page.getByText('trying again may count as another')).toBeVisible();
			await expect(page.getByText('Invalid or expired share link')).toHaveCount(0);
			await page.waitForLoadState('networkidle');
			expect(tokens).toEqual(['unknown-token']);

			await page.getByRole('button', { name: 'Try again', exact: true }).click();
			await expect(page.locator('[data-code="invalid"]')).toBeVisible();
			expect(tokens).toHaveLength(2);
		});
	}

	test('an invalid link offers no retry', async ({ page }) => {
		answers.push(invalid);
		await page.goto(`${origin}/s/invalid-token`);
		const card = page.locator('[data-code="invalid"]');
		await expect(card.getByRole('heading', { name: 'Link Not Valid' })).toBeVisible();
		await expect(card.getByText('Invalid or expired share link')).toBeVisible();
		await expect(card.getByRole('button')).toHaveCount(0);
		await expect(card.getByRole('link', { name: 'Go to Main Profile' })).toHaveAttribute(
			'href',
			'/'
		);
	});

	function legacyUrl(token: string) {
		return `${origin}/${STALE_SLUG}?t=${encodeURIComponent(token)}`;
	}

	test('a legacy link hands its token to /s/ before any check', async () => {
		const response = await fetch(legacyUrl('legacy-token'), { redirect: 'manual' });

		expect(response.status).toBe(302);
		expect(response.headers.get('location')).toBe('/s/legacy-token');
		expect(response.headers.get('cache-control')).toBe('private, no-store');
		expect(response.headers.get('referrer-policy')).toBe('no-referrer');
		expect(response.headers.getSetCookie().join()).not.toContain('me_share');
		expect(tokens).toEqual([]);
	});

	for (const [token, location] of [
		['a/b?c#d%e f', '/s/a%2Fb%3Fc%23d%25e%20f'],
		['//evil.example/x', '/s/%2F%2Fevil.example%2Fx'],
		['https://evil.example', '/s/https%3A%2F%2Fevil.example'],
		['\\\\evil.example', '/s/%5C%5Cevil.example']
	]) {
		test(`a legacy token of ${token} stays one path segment`, async () => {
			const response = await fetch(legacyUrl(token), { redirect: 'manual' });
			expect(response.headers.get('location')).toBe(location);

			await (await enter(location.slice('/s/'.length), invalid)).text();
			expect(tokens).toEqual([token]);
		});
	}

	for (const value of ['', '.', '..']) {
		test(`a legacy t of "${value}" stays on the bare route unchecked`, async () => {
			const response = await fetch(`${origin}/${STALE_SLUG}?t=${value}`, { redirect: 'manual' });

			expect(response.status).toBe(404);
			expect(response.headers.get('cache-control')).toBe('private, no-store');
			expect(response.headers.get('referrer-policy')).toBe('no-referrer');
			expect(tokens).toEqual([]);
		});
	}

	test('a denied legacy link waits for the cooldown and the recipient', async ({ page }) => {
		answers.push(limited('5'), admitted);
		const response = await page.goto(legacyUrl('legacy-cooldown'));
		expect(response?.status()).toBe(200);
		expect(response?.headers()['retry-after']).toBe('5');
		expect(page.url()).toBe(`${origin}/s/legacy-cooldown`);
		await expect(page.locator('[data-code="rate_limited"]')).toBeVisible();

		const retry = page.getByRole('button', { name: /^Try again/ });
		await expect(retry).toBeDisabled();
		await expect(retry).toHaveText('Try again', { timeout: 10_000 });
		expect(tokens).toEqual(['legacy-cooldown']);

		await retry.click();
		await page.waitForURL(`${origin}/${SLUG}`);
		expect(tokens).toEqual(['legacy-cooldown', 'legacy-cooldown']);
	});

	for (const [name, answer] of [
		['a 5xx', broken],
		['a dropped connection', dropped],
		['an unreadable answer', unreadable]
	] as const) {
		test(`${name} on a legacy link waits for a manual retry`, async ({ page }) => {
			answers.push(answer, admitted);
			const response = await page.goto(legacyUrl('legacy-retry'));
			expect(response?.status()).toBe(200);
			expect(page.url()).toBe(`${origin}/s/legacy-retry`);
			await expect(page.locator('[data-code="unavailable"]')).toBeVisible();
			await page.waitForLoadState('networkidle');
			expect(tokens).toEqual(['legacy-retry']);

			await page.getByRole('button', { name: 'Try again', exact: true }).click();
			await page.waitForURL(`${origin}/${SLUG}`);
			expect(tokens).toEqual(['legacy-retry', 'legacy-retry']);
		});
	}

	test('a good legacy link lands on the canonical Facet', async ({ page, context }) => {
		answers.push(admitted);
		await page.goto(legacyUrl('legacy-good'));
		await page.waitForURL(`${origin}/${SLUG}`);

		expect(tokens).toEqual(['legacy-good']);
		const cookies = await context.cookies();
		for (const [name, value] of [
			['me_share_token', 'legacy-good'],
			['me_share_visit', 'fixture-proof']
		]) {
			const cookie = cookies.find((candidate) => candidate.name === name);
			expect(cookie?.value, name).toBe(value);
			expect(cookie?.httpOnly, name).toBe(true);
		}
	});

	test('hovering a legacy link spends no check and clicking spends one', async ({ page }) => {
		await page.goto(`${origin}/s/invalid-token`);
		await page.waitForLoadState('networkidle');
		tokens.length = 0;
		await page.locator('[data-code="invalid"]').evaluate((card, href) => {
			const link = document.createElement('a');
			link.href = href;
			link.textContent = 'Legacy link';
			card.append(link);
		}, legacyUrl('legacy-hover'));
		const link = page.getByRole('link', { name: 'Legacy link' });

		const preload = page.waitForResponse((r) => /__data\.json.*legacy-hover/.test(r.url()));
		await link.hover();
		await preload;
		expect(tokens).toEqual([]);

		answers.push(admitted);
		await link.click();
		await page.waitForURL(`${origin}/${SLUG}`);
		expect(tokens).toEqual(['legacy-hover']);
	});

	test('keeps tokens, bodies and backend URLs out of the server log', async () => {
		logs = '';
		for (const answer of [broken, dropped, unreadable]) {
			await (await enter('log-secret-token', answer)).text();
		}
		await expect.poll(() => logs.match(/Share entry check failed/g)?.length).toBe(3);
		for (const secret of ['log-secret-token', UPSTREAM_BODY, '/api/share/validate']) {
			expect(logs).not.toContain(secret);
		}
	});
});

import { test, expect } from '@playwright/test';
import { spawn, type ChildProcess } from 'node:child_process';
import { once } from 'node:events';
import http from 'node:http';
import { fileURLToPath } from 'node:url';

// Share entry checks are limited per client address. The frontend makes them
// over loopback, so it has to tell the backend who asked, and the only address
// it may believe is the one its proxy wrote. Run through run-browser-smoke.sh
// after npm run build.

const buildEntry = fileURLToPath(new URL('../build/index.js', import.meta.url));
const ADDRESS_HEADER = 'x-facet-client-ip';
const SLUG = 'client-address-facet';
// TierModerate in backend/services/ratelimit.go.
const BURST = 5;

type Answer = { status: number; body: string; location?: string };

// localAddress picks the peer address the proxy sees. All of 127/8 is loopback on Linux.
function get(url: string, localAddress: string, headers: Record<string, string> = {}) {
	return new Promise<Answer>((resolve, reject) => {
		http
			.get(url, { agent: false, localAddress, headers }, (response) => {
				let body = '';
				response.setEncoding('utf8');
				response.on('data', (chunk) => (body += chunk));
				response.on('end', () =>
					resolve({ status: response.statusCode ?? 0, body, location: response.headers.location })
				);
			})
			.on('error', reject);
	});
}

function entryCode(answer: Answer) {
	return answer.body.match(/data-code="([a-z_]+)"/)?.[1];
}

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

test.describe('share check quota on the running stack', () => {
	// A retry needs addresses whose quota is still whole.
	const visitor = (host: number) => `127.77.${test.info().retry}.${host}`;

	test('forged headers buy no checks and another visitor is unaffected', async ({ baseURL }) => {
		const spender = visitor(1);
		const forged = (i: number) => ({
			[ADDRESS_HEADER]: `203.0.113.${i}`,
			'x-forwarded-for': `198.51.100.${i}`
		});

		for (let i = 0; i < BURST; i += 1) {
			const answer = await get(`${baseURL}/s/missing-${i}`, spender, forged(i));
			expect(entryCode(answer)).toBe('invalid');
		}
		const denied = await get(`${baseURL}/s/missing-last`, spender, forged(BURST));
		expect(entryCode(denied)).toBe('rate_limited');
		expect(entryCode(await get(`${baseURL}/s/missing-other`, visitor(2)))).toBe('invalid');
	});

	test("a legacy link spends its visitor's quota and nobody else's", async ({ baseURL }) => {
		const spender = visitor(3);

		for (let i = 0; i < BURST; i += 1) {
			const legacy = await get(`${baseURL}/${SLUG}?t=missing-${i}`, spender);
			expect(legacy.location).toBe(`/s/missing-${i}`);
			expect(entryCode(await get(`${baseURL}${legacy.location}`, spender))).toBe('invalid');
		}
		expect(entryCode(await get(`${baseURL}/s/missing-last`, spender))).toBe('rate_limited');
		expect(entryCode(await get(`${baseURL}/s/missing-other`, visitor(4)))).toBe('invalid');
	});
});

test.describe('share check address forwarding', () => {
	const posts: http.IncomingHttpHeaders[] = [];
	let frontend: ChildProcess;
	let origin: string;
	let logs = '';

	const backend = http.createServer((request, response) => {
		if (request.method !== 'POST' || request.url !== '/api/share/validate') {
			response.writeHead(404).end();
			return;
		}
		posts.push(request.headers);
		request.resume();
		request.on('end', () => {
			response.writeHead(200, {
				'content-type': 'application/json',
				'X-Share-Visit': 'fixture-proof',
				'X-Share-Visit-Expires': new Date(Date.now() + 60 * 60 * 1000).toISOString()
			});
			response.end(JSON.stringify({ valid: true, view_id: 'fixture-view', view_slug: SLUG }));
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
				ADDRESS_HEADER,
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
		posts.length = 0;
	});

	const proxied = {
		[ADDRESS_HEADER]: '203.0.113.10',
		'x-forwarded-for': '198.51.100.66',
		'x-internal': 'true',
		authorization: 'Bearer owner-token'
	};

	function expectOnlyProxyAddress() {
		expect(posts).toHaveLength(1);
		expect(posts[0]['x-forwarded-for']).toBe('203.0.113.10');
		expect(posts[0][ADDRESS_HEADER]).toBeUndefined();
		expect(posts[0]['x-internal']).toBeUndefined();
		expect(posts[0].authorization).toBeUndefined();
	}

	test('a share link forwards only the address its proxy wrote', async () => {
		const answer = await get(`${origin}/s/good-token`, '127.0.0.1', proxied);

		expect(answer.status).toBe(302);
		expectOnlyProxyAddress();
	});

	test('a legacy link forwards only the address its proxy wrote', async ({ page }) => {
		// A real browser follows the redirect, so the headers ride on every hop.
		await page.setExtraHTTPHeaders(proxied);
		await page.goto(`${origin}/${SLUG}?t=good-token`);

		await expect(page).toHaveURL(`${origin}/${SLUG}`);
		expectOnlyProxyAddress();
	});

	test('a request that skipped the proxy spends no check', async () => {
		logs = '';
		const share = await get(`${origin}/s/direct-token`, '127.0.0.1', {
			'x-forwarded-for': '198.51.100.66'
		});
		const legacy = await get(`${origin}/${SLUG}?t=direct-token`, '127.0.0.1');
		const redirected = await get(`${origin}${legacy.location}`, '127.0.0.1');

		expect(entryCode(share)).toBe('unavailable');
		expect(legacy.status).toBe(302);
		expect(entryCode(redirected)).toBe('unavailable');
		expect(posts).toHaveLength(0);
		await expect.poll(() => logs).toContain('Share entry check failed');
		expect(logs).not.toContain('direct-token');
	});
});

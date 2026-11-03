import { expect, test } from '@playwright/test';
import { buildRobotsTxt, publicOrigin, ROBOTS_DISALLOW } from '../src/lib/server/robots';
import {
	checkUnsubscribeToken,
	confirmUnsubscribe,
	isValidUnsubscribeToken,
	unsubscribeHeaders
} from '../src/lib/server/unsubscribe';
import { escapeXml, publicRecords, renderSitemap } from '../src/lib/server/sitemap';

// A token shaped like the backend's: 32 random bytes, unpadded base64url.
const TOKEN = 'AbCdEfGhIjKlMnOpQrStUvWxYz0123456789_-AbCd';

type Call = { url: string; method: string; body?: string };

function recordingFetch(respond: (call: Call) => Response) {
	const calls: Call[] = [];
	const fetcher = (async (input: RequestInfo | URL, init?: RequestInit) => {
		const call = {
			url: String(input),
			method: init?.method ?? 'GET',
			body: init?.body as string | undefined
		};
		calls.push(call);
		return respond(call);
	}) as typeof fetch;
	return { calls, fetcher };
}

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

/** Groups robots.txt into { agents, rules } blocks the way crawlers read it. */
function robotsGroups(text: string) {
	const groups: { agents: string[]; rules: string[] }[] = [];
	let current: { agents: string[]; rules: string[] } | null = null;
	for (const raw of text.split('\n')) {
		const line = raw.replace(/#.*/, '').trim();
		if (!line) continue;
		const [field, ...rest] = line.split(':');
		const value = rest.join(':').trim();
		if (field.toLowerCase() === 'user-agent') {
			if (!current || current.rules.length > 0) {
				current = { agents: [], rules: [] };
				groups.push(current);
			}
			current.agents.push(value);
		} else if (current && field.toLowerCase() !== 'sitemap') {
			current.rules.push(`${field}: ${value}`);
		}
	}
	return groups;
}

test.describe('robots.txt policy', () => {
	test('uses the configured origin, falling back to the request origin', () => {
		expect(publicOrigin({ PUBLIC_APP_URL: 'https://jed.example/' }, 'http://internal:3000')).toBe(
			'https://jed.example'
		);
		expect(publicOrigin({ APP_URL: 'https://app.example' }, 'http://internal:3000')).toBe(
			'https://app.example'
		);
		expect(
			publicOrigin({ PUBLIC_APP_URL: 'not a url', APP_URL: 'https://app.example' }, 'http://x')
		).toBe('https://app.example');
		expect(publicOrigin({ PUBLIC_APP_URL: 'javascript:alert(1)' }, 'http://req.example')).toBe(
			'http://req.example'
		);
		expect(publicOrigin({}, 'http://req.example')).toBe('http://req.example');
	});

	test('keeps search engines under the private-path disallows and blocks AI training crawlers', () => {
		const text = buildRobotsTxt('https://jed.example');
		expect(text).toContain('Sitemap: https://jed.example/sitemap.xml');

		const groups = robotsGroups(text);
		const star = groups.find((g) => g.agents.includes('*'));
		expect(star).toBeDefined();
		for (const path of ['/admin', '/api/', '/s/', '/v/', '/unsubscribe', '/testimonial/']) {
			expect(star!.rules).toContain(`Disallow: ${path}`);
		}

		// No crawler that's allowed in may have its own group, or it would skip
		// the disallows above.
		for (const agent of ['Googlebot', 'Bingbot', 'DuckDuckBot', 'Applebot']) {
			expect(groups.some((g) => g.agents.includes(agent))).toBe(false);
		}

		const ai = groups.find((g) => g.agents.includes('GPTBot'));
		expect(ai?.agents).toEqual(expect.arrayContaining(['ClaudeBot', 'CCBot', 'Google-Extended']));
		expect(ai?.rules).toEqual(['Disallow: /']);
	});

	test('the disallow list covers every token-bearing public route', () => {
		expect(ROBOTS_DISALLOW).toEqual(
			expect.arrayContaining(['/s/', '/v/', '/unsubscribe', '/testimonial/'])
		);
	});
});

test.describe('sitemap helpers', () => {
	test('lists only published public records', () => {
		const kept = publicRecords([
			{ slug: 'open', visibility: 'public' },
			{ slug: 'secret', visibility: 'private' },
			{ slug: 'by-link', visibility: 'unlisted' },
			{ slug: 'draft', visibility: 'public', is_draft: true },
			{ visibility: 'public' }
		]);
		expect(kept.map((r) => r.slug)).toEqual(['open']);
	});

	test('escapes every URL so the XML stays valid', () => {
		expect(escapeXml(`a&b<c>"d"'e`)).toBe('a&amp;b&lt;c&gt;&quot;d&quot;&apos;e');
		const xml = renderSitemap([
			{
				loc: 'https://jed.example/posts/a&b',
				lastmod: '2026-11-03',
				priority: '0.7',
				changefreq: 'monthly'
			}
		]);
		expect(xml).toContain('<loc>https://jed.example/posts/a&amp;b</loc>');
		expect(xml).not.toMatch(/&(?!amp;|lt;|gt;|quot;|apos;)/);
	});
});

test.describe('unsubscribe server helpers', () => {
	test('accepts backend-shaped tokens and rejects anything that could change the path', () => {
		expect(isValidUnsubscribeToken(TOKEN)).toBe(true);
		for (const bad of [
			undefined,
			'',
			'short',
			`${TOKEN}/..`,
			'../../admin',
			'a%2Fb'.repeat(5),
			`${TOKEN}=`,
			'a'.repeat(129)
		]) {
			expect(isValidUnsubscribeToken(bad)).toBe(false);
		}
	});

	test('checking a token only reads; it never unsubscribes', async () => {
		const { calls, fetcher } = recordingFetch(() =>
			json(200, { email: 'j***@example.com', status: 'active' })
		);
		await expect(checkUnsubscribeToken(fetcher, TOKEN)).resolves.toEqual({
			state: 'ready',
			maskedEmail: 'j***@example.com'
		});
		expect(calls).toHaveLength(1);
		expect(calls[0].method).toBe('GET');
		expect(calls[0].url).toMatch(new RegExp(`/api/public/unsubscribe/${TOKEN}$`));
	});

	test('reports an already-unsubscribed address as success', async () => {
		const { fetcher } = recordingFetch(() =>
			json(200, { email: 'j***@example.com', status: 'unsubscribed' })
		);
		await expect(checkUnsubscribeToken(fetcher, TOKEN)).resolves.toEqual({
			state: 'already',
			maskedEmail: 'j***@example.com'
		});
	});

	test('separates a bad link from a backend outage', async () => {
		const notFound = recordingFetch(() => json(404, { error: 'invalid or expired token' }));
		await expect(checkUnsubscribeToken(notFound.fetcher, TOKEN)).resolves.toEqual({
			state: 'invalid'
		});

		const down = recordingFetch(() => json(502, {}));
		await expect(checkUnsubscribeToken(down.fetcher, TOKEN)).resolves.toEqual({
			state: 'unavailable'
		});

		const refused = recordingFetch(() => {
			throw new TypeError('fetch failed');
		});
		await expect(checkUnsubscribeToken(refused.fetcher, TOKEN)).resolves.toEqual({
			state: 'unavailable'
		});
	});

	test('a malformed token never reaches the backend', async () => {
		const { calls, fetcher } = recordingFetch(() => json(200, {}));
		await expect(checkUnsubscribeToken(fetcher, '../admin')).resolves.toEqual({ state: 'invalid' });
		await expect(confirmUnsubscribe(fetcher, '../admin')).resolves.toBe('invalid');
		expect(calls).toHaveLength(0);
	});

	test('confirming posts the token in the body and maps every backend status', async () => {
		const { calls, fetcher } = recordingFetch(() => json(200, { status: 'unsubscribed' }));
		await expect(confirmUnsubscribe(fetcher, TOKEN)).resolves.toBe('unsubscribed');
		expect(calls[0]).toMatchObject({ method: 'POST', body: JSON.stringify({ token: TOKEN }) });
		expect(calls[0].url).toMatch(/\/api\/public\/unsubscribe$/);

		const cases: [Response, string][] = [
			[json(200, { status: 'already_unsubscribed' }), 'already_unsubscribed'],
			[json(404, { error: 'invalid or expired token' }), 'invalid'],
			[json(400, { error: 'token required' }), 'invalid'],
			[json(500, { error: 'failed to unsubscribe' }), 'unavailable'],
			[json(200, { status: 'something new' }), 'unavailable']
		];
		for (const [response, expected] of cases) {
			const stub = recordingFetch(() => response);
			await expect(confirmUnsubscribe(stub.fetcher, TOKEN)).resolves.toBe(expected);
		}
	});

	test('responses are private, uncached, unindexed and send no referrer', () => {
		expect(unsubscribeHeaders(false)).toEqual({
			'Cache-Control': 'private, no-store',
			'X-Robots-Tag': 'noindex, nofollow',
			'Referrer-Policy': 'no-referrer'
		});
		// When the layout has already set Cache-Control (signed-in visitor) it is
		// 'private, no-store' already and must not be set a second time.
		expect(unsubscribeHeaders(true)).not.toHaveProperty('Cache-Control');
	});
});

// These need the integrated stack (SvelteKit + PocketBase). They use the
// request fixture, so no browser window is opened.
test.describe('live responses', () => {
	test('GET /robots.txt is served by the dynamic route', async ({ request }) => {
		const response = await request.get('/robots.txt');
		expect(response.status()).toBe(200);
		expect(response.headers()['content-type']).toContain('text/plain');
		const text = await response.text();
		expect(text).toContain('Disallow: /s/');
		expect(text).toContain('Disallow: /unsubscribe');
		expect(text).toMatch(/^Sitemap: https?:\/\/[^\s]+\/sitemap\.xml$/m);
		expect(text).not.toContain('facet.example.com');
	});

	test('GET /unsubscribe/<unknown token> shows a recoverable page and changes nothing', async ({
		request
	}) => {
		const response = await request.get(`/unsubscribe/${TOKEN}`);
		expect(response.status()).toBe(200);
		const headers = response.headers();
		expect(headers['cache-control']).toContain('no-store');
		expect(headers['cache-control']).toContain('private');
		expect(headers['x-robots-tag']).toContain('noindex');
		expect(headers['referrer-policy']).toBe('no-referrer');
		const html = await response.text();
		expect(html).toContain('data-state="invalid"');
		// A bad link points back to the signed link in the email; an address
		// alone can never unsubscribe anyone.
		expect(html).toContain('data-testid="unsubscribe-signed-link-help"');
		expect(html).not.toContain('name="email"');
		expect(html).toMatch(/<meta name="robots" content="noindex, nofollow"/);
	});

	test('GET /unsubscribe with an encoded path segment is treated as invalid', async ({
		request
	}) => {
		const response = await request.get('/unsubscribe/..%2F..%2Fadmin');
		expect([200, 404]).toContain(response.status());
		if (response.status() === 200) {
			expect(await response.text()).toContain('data-state="invalid"');
		}
	});

	test('GET /unsubscribe without a token explains the signed link and has no email form', async ({
		request
	}) => {
		const response = await request.get('/unsubscribe');
		expect(response.status()).toBe(200);
		expect(response.headers()['x-robots-tag']).toContain('noindex');
		const html = await response.text();
		expect(html).toContain('data-state="no-token"');
		expect(html).toContain('data-testid="unsubscribe-signed-link-help"');
		expect(html).not.toContain('action="?/email"');
		expect(html).not.toContain('name="email"');
		expect(html).not.toContain('action="?/confirm"');
	});

	test('POST /unsubscribe?/email cannot unsubscribe by address', async ({ request, baseURL }) => {
		const response = await request.post('/unsubscribe?/email', {
			form: { email: 'reader@example.com' },
			headers: { Origin: new URL(baseURL ?? 'http://localhost').origin }
		});
		expect(response.status()).toBeGreaterThanOrEqual(400);
		expect(response.status()).toBeLessThan(500);
	});

	// An auth cookie makes the root layout set Cache-Control first. Setting it
	// again in the page used to fail with SvelteKit's header_already_set error.
	for (const [name, headers] of [
		['anonymous', {}],
		['with an auth cookie', { Cookie: 'pb_auth=not-a-real-session' }]
	] as const) {
		test(`GET /unsubscribe/<token> ${name} sets Cache-Control once`, async ({ request }) => {
			const response = await request.get(`/unsubscribe/${TOKEN}`, { headers });
			expect(response.status()).toBe(200);
			const cacheControl = response
				.headersArray()
				.filter((h) => h.name.toLowerCase() === 'cache-control');
			expect(cacheControl).toHaveLength(1);
			expect(cacheControl[0].value).toContain('no-store');
			expect(await response.text()).toContain('data-state="invalid"');
		});
	}

	test('GET /sitemap.xml uses the same origin as robots.txt and is valid XML', async ({
		request,
		baseURL
	}) => {
		const robots = await (await request.get('/robots.txt')).text();
		const sitemapLine = robots.match(/^Sitemap: (\S+)\/sitemap\.xml$/m);
		expect(sitemapLine).not.toBeNull();
		const origin = sitemapLine![1];

		const response = await request.get('/sitemap.xml');
		expect(response.status()).toBe(200);
		expect(response.headers()['content-type']).toContain('application/xml');
		const xml = await response.text();
		expect(xml.startsWith('<?xml')).toBe(true);
		expect(xml).not.toMatch(/&(?!amp;|lt;|gt;|quot;|apos;)/);

		const locs = [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
		expect(locs.length).toBeGreaterThan(0);
		for (const loc of locs) {
			expect(loc.startsWith(origin)).toBe(true);
			expect(loc).not.toMatch(/\/(admin|api|s|v|unsubscribe)\//);
		}
		// A configured URL wins. Without one the request origin is used, never a
		// guessed localhost port.
		const configured = process.env.PUBLIC_APP_URL || process.env.APP_URL;
		if (configured) {
			const url = new URL(configured.trim());
			expect(origin).toBe(url.origin + url.pathname.replace(/\/+$/, ''));
		} else if (baseURL) {
			expect(origin).toBe(new URL(baseURL).origin);
		}
	});
});

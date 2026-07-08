import { expect, test } from '@playwright/test';
import {
	fetchWith503Retry,
	isRetryableInternalApiRequest,
	parseRetryAfterMs
} from '../src/lib/server/internal-api-retry';

test.describe('internal API 503 retry', () => {
	test('retries safe internal API GETs and returns the first non-503 response', async () => {
		const statuses = [503, 503, 200];
		const delays: number[] = [];

		const response = await fetchWith503Retry(
			new Request('http://localhost:8090/api/homepage'),
			async () => new Response('', { status: statuses.shift() ?? 500 }),
			{
				apiOrigins: ['http://localhost:8090'],
				backoffsMs: [0, 0, 0],
				sleep: async (ms) => {
					delays.push(ms);
				}
			}
		);

		expect(response.status).toBe(200);
		expect(delays).toEqual([0, 0]);
	});

	test('does not retry unsafe methods or external API paths', async () => {
		for (const request of [
			new Request('http://localhost:8090/api/site-settings', { method: 'POST' }),
			new Request('https://example.com/api/status')
		]) {
			let calls = 0;
			const response = await fetchWith503Retry(
				request,
				async () => {
					calls += 1;
					return new Response('', { status: 503 });
				},
				{ apiOrigins: ['http://localhost:8090'], backoffsMs: [0, 0, 0], sleep: async () => {} }
			);

			expect(response.status).toBe(503);
			expect(calls).toBe(1);
		}
	});

	test('bounds Retry-After delays', () => {
		expect(parseRetryAfterMs('2')).toBe(2000);
		expect(parseRetryAfterMs('Wed, 08 Jul 2026 00:00:03 GMT', () => Date.parse('2026-07-08T00:00:00Z'))).toBe(3000);
		expect(isRetryableInternalApiRequest(new Request('http://localhost:8090/api/health', { method: 'HEAD' }), ['http://localhost:8090'])).toBe(true);
		expect(isRetryableInternalApiRequest(new Request('http://localhost:8090/status'), ['http://localhost:8090'])).toBe(false);
	});
});

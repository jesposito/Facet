import { expect, test } from '@playwright/test';
import { checkBackendHealth } from '../src/lib/server/health';

test.describe('health route helpers', () => {
	test('reports healthy when PocketBase health responds OK', async () => {
		let requestedUrl = '';

		const status = await checkBackendHealth(
			async (input, init) => {
				requestedUrl = String(input);
				expect(init?.headers).toEqual({ 'X-Internal': 'true' });
				return new Response('{}', { status: 200 });
			},
			'http://localhost:8090/'
		);

		expect(requestedUrl).toBe('http://localhost:8090/api/health');
		expect(status).toEqual({
			ok: true,
			frontend: 'ok',
			backend: 'ok',
			backendStatus: 200
		});
	});

	test('reports unhealthy when PocketBase health is unavailable', async () => {
		const status = await checkBackendHealth(
			async () => new Response('{}', { status: 503 }),
			'http://localhost:8090'
		);

		expect(status).toEqual({
			ok: false,
			frontend: 'ok',
			backend: 'error',
			backendStatus: 503
		});
	});

	test('reports request failures without leaking internal errors', async () => {
		const status = await checkBackendHealth(async () => {
			throw new Error('connect ECONNREFUSED 127.0.0.1:8090');
		}, 'http://localhost:8090');

		expect(status).toEqual({
			ok: false,
			frontend: 'ok',
			backend: 'error',
			error: 'request_failed'
		});
	});
});

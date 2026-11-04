import { test, expect } from '@playwright/test';
import { spawn, type ChildProcess } from 'node:child_process';
import http from 'node:http';
import { fileURLToPath } from 'node:url';

const proxyScript = fileURLToPath(new URL('../../scripts/ci/smoke-proxy.mjs', import.meta.url));

async function listen(server: http.Server, port = 0) {
	await new Promise<void>((resolve, reject) => {
		server.once('error', reject);
		server.listen(port, '127.0.0.1', () => resolve());
	});
	const address = server.address();
	if (!address || typeof address === 'string') throw new Error('Expected a TCP server');
	return address.port;
}

async function close(server: http.Server) {
	await new Promise<void>((resolve, reject) => {
		server.close((error) => (error ? reject(error) : resolve()));
	});
}

async function stop(child: ChildProcess | undefined) {
	if (!child || child.exitCode !== null || child.signalCode) return;
	const exited = new Promise<void>((resolve) => child.once('exit', () => resolve()));
	child.kill('SIGTERM');
	await exited;
}

async function proxyReady(child: ChildProcess, port: number) {
	let earlyExit = '';
	child.once('error', (error) => {
		earlyExit = error.message;
	});
	child.once('exit', (code) => {
		earlyExit = `Smoke proxy exited early: ${code}`;
	});
	await expect
		.poll(
			async () => {
				if (earlyExit) throw new Error(earlyExit);
				try {
					await fetch(`http://127.0.0.1:${port}/`);
					return true;
				} catch {
					return false;
				}
			},
			{ timeout: 5_000 }
		)
		.toBe(true);
}

test('smoke proxy uses configured ports and preserves the backend/frontend routing boundary', async () => {
	const backend = http.createServer((request, response) => {
		response.setHeader('x-test-upstream', 'backend');
		if (request.url === '/api/headers') {
			response.setHeader('content-type', 'application/json');
			response.end(JSON.stringify(request.headers));
			return;
		}
		response.end('backend response');
	});
	const frontend = http.createServer((request, response) => {
		response.setHeader('x-test-upstream', 'frontend');
		if (request.url === '/headers') {
			response.end(JSON.stringify(request.headers));
			return;
		}
		response.end('frontend response');
	});
	const collision = http.createServer();
	let proxy: ChildProcess | undefined;

	try {
		const backendPort = await listen(backend);
		const frontendPort = await listen(frontend);
		const proxyPort = await listen(collision);
		await close(collision);
		proxy = spawn(process.execPath, [proxyScript], {
			env: {
				...process.env,
				SMOKE_HOST: '127.0.0.1',
				SMOKE_PROXY_PORT: String(proxyPort),
				SMOKE_BACKEND_PORT: String(backendPort),
				SMOKE_FRONTEND_PORT: String(frontendPort)
			},
			stdio: 'ignore'
		});
		await proxyReady(proxy, proxyPort);

		const api = await fetch(`http://127.0.0.1:${proxyPort}/api/posts`);
		expect(api.status).toBe(200);
		expect(api.headers.get('x-test-upstream')).toBe('backend');
		expect(await api.text()).toBe('backend response');
		const forwarded = await fetch(`http://127.0.0.1:${proxyPort}/api/headers`, {
			headers: {
				'x-internal': 'true',
				'x-forwarded-proto': 'https',
				'x-forwarded-host': 'spoof.invalid',
				'x-forwarded-for': '203.0.113.99'
			}
		}).then((response) => response.json());
		expect(forwarded['x-internal']).toBeUndefined();
		expect(forwarded['x-forwarded-proto']).toBe('http');
		expect(forwarded['x-forwarded-host']).toBe(`127.0.0.1:${proxyPort}`);
		expect(forwarded['x-forwarded-for']).toBe('127.0.0.1');

		const page = await fetch(`http://127.0.0.1:${proxyPort}/projects/example`);
		expect(page.status).toBe(200);
		expect(page.headers.get('x-test-upstream')).toBe('frontend');
		expect(await page.text()).toBe('frontend response');

		const rendered = await fetch(`http://127.0.0.1:${proxyPort}/headers`, {
			headers: { 'x-facet-client-ip': '203.0.113.99' }
		}).then((response) => response.json());
		expect(rendered['x-facet-client-ip']).toBe('127.0.0.1');

		await close(backend);
		const unavailable = await fetch(`http://127.0.0.1:${proxyPort}/api/posts`);
		expect(unavailable.status).toBe(502);
	} finally {
		await stop(proxy);
		if (backend.listening) await close(backend);
		if (frontend.listening) await close(frontend);
		if (collision.listening) await close(collision);
	}
});

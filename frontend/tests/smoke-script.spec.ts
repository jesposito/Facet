import { test, expect } from '@playwright/test';
import { spawnSync } from 'node:child_process';
import { chmod, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import http from 'node:http';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const smokeScript = fileURLToPath(
	new URL('../../scripts/ci/run-browser-smoke.sh', import.meta.url)
);

async function executable(path: string, contents: string) {
	await writeFile(path, contents);
	await chmod(path, 0o755);
}

async function stubFixture() {
	const root = await mkdtemp(join(tmpdir(), 'facet-smoke-script-test-'));
	const bin = join(root, 'bin');
	const capture = join(root, 'capture');
	await mkdir(bin);
	await mkdir(capture);

	await executable(
		join(bin, 'go'),
		`#!/usr/bin/env bash
set -euo pipefail
out=
while [ "$#" -gt 0 ]; do
  if [ "$1" = -o ]; then out=$2; shift 2; else shift; fi
done
cat > "$out" <<'EOF'
#!/usr/bin/env bash
trap 'echo backend >> "$SMOKE_TEST_CAPTURE/cleaned"; exit 0' TERM INT
while true; do sleep 1; done
EOF
chmod +x "$out"
`
	);
	await executable(
		join(bin, 'node'),
		`#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = --input-type=module ]; then exit 0; fi
case "$1" in
  *frontend/build/index.js) name=frontend ;;
  *smoke-proxy.mjs) name=proxy ;;
  *) echo "Unexpected node command: $*" >&2; exit 2 ;;
esac
trap 'echo "$name" >> "$SMOKE_TEST_CAPTURE/cleaned"; exit 0' TERM INT
while true; do sleep 1; done
`
	);
	await executable(
		join(bin, 'npx'),
		`#!/usr/bin/env bash
set -euo pipefail
printf 'CALL\n' >> "$SMOKE_TEST_CAPTURE/calls"
printf '%s\n' "$@" >> "$SMOKE_TEST_CAPTURE/calls"
printf 'URL=%s\nAPI=%s\nPORT=%s\n' "$PLAYWRIGHT_BASE_URL" "$API_BASE_URL" "$PORT" >> "$SMOKE_TEST_CAPTURE/calls"
exit "$SMOKE_TEST_NPX_EXIT"
`
	);
	await executable(join(bin, 'curl'), '#!/usr/bin/env bash\nexit 0\n');

	return { root, bin, capture };
}

async function freePort() {
	const server = http.createServer();
	await new Promise<void>((resolve, reject) => {
		server.once('error', reject);
		server.listen(0, '127.0.0.1', () => resolve());
	});
	const address = server.address();
	if (!address || typeof address === 'string') throw new Error('Expected a TCP server');
	const port = address.port;
	await new Promise<void>((resolve, reject) => {
		server.close((error) => (error ? reject(error) : resolve()));
	});
	return port;
}

async function portSet() {
	const ports = [await freePort(), await freePort(), await freePort()];
	expect(new Set(ports).size).toBe(3);
	return ports;
}

function runSmoke(
	fixture: { bin: string; capture: string },
	ports: number[],
	args: string[] = [],
	npxExit = 0
) {
	return spawnSync('bash', [smokeScript, ...args], {
		encoding: 'utf8',
		timeout: 15_000,
		env: {
			...process.env,
			PATH: `${fixture.bin}:${process.env.PATH}`,
			SMOKE_TEST_CAPTURE: fixture.capture,
			SMOKE_TEST_NPX_EXIT: String(npxExit),
			SMOKE_PROXY_PORT: String(ports[0]),
			SMOKE_BACKEND_PORT: String(ports[1]),
			SMOKE_FRONTEND_PORT: String(ports[2])
		}
	});
}

test('smoke script forwards focused arguments and cleans up every process after a test failure', async () => {
	const fixture = await stubFixture();
	let logsRoot: string | undefined;
	try {
		const ports = await portSet();
		const result = runSmoke(fixture, ports, ['tests/example.spec.ts', '--grep', 'focused'], 23);
		expect(result.status, result.stdout + result.stderr).toBe(23);

		const calls = await readFile(join(fixture.capture, 'calls'), 'utf8');
		expect(calls).toContain('tests/example.spec.ts\n--grep\nfocused\n');
		expect(calls).toContain(`URL=http://127.0.0.1:${ports[0]}\n`);
		expect(calls).toContain(`API=http://127.0.0.1:${ports[1]}\n`);
		expect(calls).toContain(`PORT=${ports[2]}\n`);

		const cleaned = await readFile(join(fixture.capture, 'cleaned'), 'utf8');
		expect(cleaned.trim().split('\n').sort()).toEqual(['backend', 'frontend', 'proxy']);
		const logs = result.stdout.match(/Smoke-test logs: (.+)/)?.[1];
		expect(logs).toBeTruthy();
		logsRoot = dirname(logs!);
		for (const name of ['backend.log', 'frontend.log', 'proxy.log']) {
			expect(await readFile(join(logs!, name), 'utf8')).toBe('');
		}
	} finally {
		if (logsRoot) await rm(logsRoot, { recursive: true, force: true });
		await rm(fixture.root, { recursive: true, force: true });
	}
});

test('smoke script runs public APIs first, then source-unit and browser projects', async () => {
	const fixture = await stubFixture();
	let logsRoot: string | undefined;
	try {
		const result = runSmoke(fixture, await portSet());
		expect(result.status, result.stdout + result.stderr).toBe(0);
		const calls = (await readFile(join(fixture.capture, 'calls'), 'utf8'))
			.split('CALL\n')
			.filter(Boolean);
		expect(calls).toHaveLength(2);
		expect(calls[0]).toContain('--project=public-api\n');
		expect(calls[1]).toContain('--project=source-unit\n');
		expect(calls[1]).toContain('--project=chromium\n');
		const logs = result.stdout.match(/Smoke-test logs: (.+)/)?.[1];
		expect(logs).toBeTruthy();
		logsRoot = dirname(logs!);
	} finally {
		if (logsRoot) await rm(logsRoot, { recursive: true, force: true });
		await rm(fixture.root, { recursive: true, force: true });
	}
});

test('smoke script rejects an occupied configured port before building or starting processes', async () => {
	const fixture = await stubFixture();
	const occupied = http.createServer();
	try {
		await new Promise<void>((resolve, reject) => {
			occupied.once('error', reject);
			occupied.listen(0, '127.0.0.1', () => resolve());
		});
		const address = occupied.address();
		if (!address || typeof address === 'string') throw new Error('Expected a TCP server');
		const ports = await portSet();
		ports[0] = address.port;
		const result = runSmoke(fixture, ports);
		expect(result.status).not.toBe(0);
		expect(result.stdout + result.stderr).toMatch(/port .* already in use/i);
		await expect(readFile(join(fixture.capture, 'calls'), 'utf8')).rejects.toMatchObject({
			code: 'ENOENT'
		});
	} finally {
		await new Promise<void>((resolve, reject) => {
			occupied.close((error) => (error ? reject(error) : resolve()));
		});
		await rm(fixture.root, { recursive: true, force: true });
	}
});

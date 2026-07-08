export type HealthStatus = {
	ok: boolean;
	frontend: 'ok';
	backend: 'ok' | 'error';
	backendStatus?: number;
	error?: string;
};

type HealthFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export function resolvePocketBaseUrl(env: NodeJS.ProcessEnv = process.env): string {
	return env.POCKETBASE_URL || 'http://localhost:8090';
}

export async function checkBackendHealth(
	fetchFn: HealthFetch,
	pbUrl: string,
	timeoutMs = 1500
): Promise<HealthStatus> {
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), timeoutMs);
	const healthUrl = `${pbUrl.replace(/\/+$/, '')}/api/health`;

	try {
		const response = await fetchFn(healthUrl, {
			headers: { 'X-Internal': 'true' },
			signal: controller.signal
		});

		return {
			ok: response.ok,
			frontend: 'ok',
			backend: response.ok ? 'ok' : 'error',
			backendStatus: response.status
		};
	} catch (error) {
		return {
			ok: false,
			frontend: 'ok',
			backend: 'error',
			error: error instanceof Error && error.name === 'AbortError' ? 'timeout' : 'request_failed'
		};
	} finally {
		clearTimeout(timeout);
	}
}

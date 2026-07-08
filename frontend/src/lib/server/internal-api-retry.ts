type FetchRequest = (request: Request) => Promise<Response>;

type RetryOptions = {
	apiOrigins: readonly string[];
	backoffsMs?: readonly number[];
	sleep?: (ms: number) => Promise<void>;
	now?: () => number;
};

const DEFAULT_BACKOFFS_MS = [500, 1000, 2000] as const;

export async function fetchWith503Retry(
	request: Request,
	fetchFn: FetchRequest,
	options: RetryOptions
): Promise<Response> {
	const backoffsMs = options.backoffsMs ?? DEFAULT_BACKOFFS_MS;
	const sleep = options.sleep ?? ((ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms)));

	for (let attempt = 0; attempt <= backoffsMs.length; attempt += 1) {
		const response = await fetchFn(request.clone());
		if (response.status !== 503 || attempt === backoffsMs.length) {
			return response;
		}

		if (!isRetryableInternalApiRequest(request, options.apiOrigins)) {
			return response;
		}

		const retryAfter = parseRetryAfterMs(response.headers.get('retry-after'), options.now);
		await sleep(Math.min(retryAfter ?? backoffsMs[attempt], 3000));
	}

	return fetchFn(request);
}

export function isRetryableInternalApiRequest(request: Request, apiOrigins: readonly string[]): boolean {
	const method = request.method.toUpperCase();
	if (method !== 'GET' && method !== 'HEAD') {
		return false;
	}

	const target = new URL(request.url);
	if (!target.pathname.startsWith('/api/')) {
		return false;
	}

	return apiOrigins.some((origin) => {
		try {
			return target.origin === new URL(origin).origin;
		} catch {
			return false;
		}
	});
}

export function parseRetryAfterMs(value: string | null, now: () => number = Date.now): number | null {
	if (!value) {
		return null;
	}

	const seconds = Number(value);
	if (Number.isFinite(seconds) && seconds >= 0) {
		return seconds * 1000;
	}

	const dateMs = Date.parse(value);
	if (!Number.isNaN(dateMs)) {
		return Math.max(0, dateMs - now());
	}

	return null;
}

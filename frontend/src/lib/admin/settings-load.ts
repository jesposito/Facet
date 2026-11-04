const SETTINGS_LOAD_TIMEOUT_MS = 15_000;

export function isSettingsObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export async function fetchSettingsJson(
	input: string,
	init: RequestInit = {}
): Promise<Record<string, unknown>> {
	const controller = new AbortController();
	const abort = () => controller.abort();
	init.signal?.addEventListener('abort', abort, { once: true });
	if (init.signal?.aborted) controller.abort();
	const timeout = setTimeout(() => controller.abort(), SETTINGS_LOAD_TIMEOUT_MS);

	try {
		const response = await fetch(input, { ...init, signal: controller.signal });
		if (!response.ok) throw new Error(`Settings request failed (${response.status})`);
		const data: unknown = await response.json();
		if (!isSettingsObject(data)) throw new Error('Settings response must be an object');
		return data;
	} finally {
		clearTimeout(timeout);
		init.signal?.removeEventListener('abort', abort);
	}
}

export function settingsRequestTimeout(cancel: () => void): () => void {
	const timeout = setTimeout(cancel, SETTINGS_LOAD_TIMEOUT_MS);
	return () => clearTimeout(timeout);
}

import { pb } from '$lib/pocketbase';

// Read the response before SDK defaults can turn malformed JSON into an empty list.
export async function readAllRecords<T extends { id: string }>(
	collection: string,
	query: Record<string, string>,
	signal: AbortSignal
): Promise<T[]> {
	const records: T[] = [];
	const perPage = Number(query.perPage ?? '100');
	for (let page = 1, pages = 1; page <= pages; page++) {
		const url = new URL(pb.buildURL(`/api/collections/${encodeURIComponent(collection)}/records`));
		url.search = new URLSearchParams({
			...query,
			page: String(page),
			perPage: String(perPage)
		}).toString();
		const response = await fetch(url, {
			signal,
			headers: pb.authStore.isValid ? { Authorization: pb.authStore.token } : {}
		});
		if (!response.ok) throw new Error(`Record list request failed (${response.status})`);
		const data: unknown = await response.json();
		if (!data || typeof data !== 'object') throw new Error('Invalid record list response');
		const result = data as Record<string, unknown>;
		if (
			!Array.isArray(result.items) ||
			!result.items.every((item) => item && typeof item.id === 'string') ||
			!Number.isInteger(result.totalPages) ||
			(result.totalPages as number) < 0 ||
			result.page !== page ||
			result.perPage !== perPage
		) {
			throw new Error('Invalid record list response');
		}
		pages = result.totalPages as number;
		records.push(...(result.items as T[]));
	}
	return records;
}

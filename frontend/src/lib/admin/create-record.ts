import type { RecordService } from 'pocketbase';

const ID_ALPHABET = 'abcdefghijklmnopqrstuvwxyz0123456789';
const LOOKUP_TIMEOUT_MS = 15_000;

// The read that should settle a create failed, so the record may or may not exist.
export class DraftReadError extends Error {}

// PocketBase takes a client-chosen ID of 15 lowercase letters and digits. It names a record
// and grants nothing, so the slight modulo bias doesn't matter.
export function newRecordId(): string {
	const bytes = crypto.getRandomValues(new Uint8Array(15));
	return Array.from(bytes, (byte) => ID_ALPHABET[byte % ID_ALPHABET.length]).join('');
}

export function isRecordId(value: unknown): value is string {
	return typeof value === 'string' && /^[a-z0-9]{15}$/.test(value);
}

// True when a failed create may still have been stored. PocketBase reports a repeated ID as a
// 400 with a unique-field error, which can name another unique field (a Facet's slug) instead.
export function mayHaveLanded(err: unknown): boolean {
	const { status = 0, response } = err as {
		status?: number;
		response?: { data?: Record<string, { code?: string }> };
	};
	if (status === 0 || status === 408 || status === 409 || status >= 500) return true;
	return Object.values(response?.data ?? {}).some(
		(field) => field?.code === 'validation_not_unique'
	);
}

// Reads a draft's record by its exact ID. Null means the server has no such record. Any other
// failure leaves the answer unknown, so it's a DraftReadError rather than an empty result.
export async function findDraftRecord<T extends { id: string }>(
	records: RecordService,
	id: string
): Promise<T | null> {
	let stored: T;
	try {
		stored = await records.getOne<T>(id, {
			requestKey: null,
			signal: AbortSignal.timeout(LOOKUP_TIMEOUT_MS)
		});
	} catch (err) {
		if ((err as { status?: number }).status === 404) return null;
		console.error('Failed to read draft record:', err);
		throw new DraftReadError(`Couldn't read draft record ${id}`);
	}
	if (stored?.id !== id) throw new DraftReadError(`Invalid draft record ${id}`);
	return stored;
}

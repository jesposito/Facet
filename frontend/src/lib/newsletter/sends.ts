// Newsletter send history as the backend reports it. sent_count only means the
// operator's SMTP server accepted the message; nothing here tracks inbox
// delivery, bounces or complaints.

export type NewsletterSend = {
	id: string;
	subject: string;
	status: string;
	recipient_count: number;
	sent_count: number;
	failed_count: number;
	pending_count?: number;
	uncertain_count?: number;
	skipped_count?: number;
	/** Ledger recipients Resume would retry: pending and refused ones. */
	resumable_count?: number;
	open_count: number;
	click_count: number;
	sent_at: string;
	created: string;
};

export type SmtpReadiness = 'ready' | 'not_ready' | 'not_configured';

export type ResumeState = 'hidden' | 'blocked' | 'available';

export type OutcomeKey = 'failed_count' | 'pending_count' | 'uncertain_count' | 'skipped_count';

const RESUMABLE_STATUSES = new Set(['partial', 'failed', 'interrupted']);

/** Reads GET /api/admin/newsletter/smtp-status. Sending needs `ready`. */
export function smtpReadiness(data: unknown): SmtpReadiness {
	const status = (data && typeof data === 'object' ? data : {}) as {
		configured?: unknown;
		ready?: unknown;
	};
	if (status.ready === true) return 'ready';
	return status.configured === true ? 'not_ready' : 'not_configured';
}

/**
 * Whether a send can be resumed. Resume retries only pending and known
 * rejected recipients; accepted and uncertain ones are never sent again, so a
 * send with nothing else left has nothing to resume. Only the backend's
 * resumable_count says what is left: sends logged before per-recipient tracking
 * keep their historical failed counts but have nobody to resume, and a missing
 * count means resuming isn't known to be safe. Only one campaign runs at a
 * time, so resume waits while any send is still going.
 */
export function resumeState(send: NewsletterSend, sends: NewsletterSend[]): ResumeState {
	if (!RESUMABLE_STATUSES.has(send.status)) return 'hidden';
	if (typeof send.resumable_count !== 'number' || send.resumable_count <= 0) return 'hidden';
	return sends.some((s) => s.status === 'sending') ? 'blocked' : 'available';
}

/** The non-zero outcome counts besides SMTP-accepted, in display order. */
export function outcomeCounts(send: NewsletterSend): { key: OutcomeKey; count: number }[] {
	const keys: OutcomeKey[] = ['failed_count', 'pending_count', 'uncertain_count', 'skipped_count'];
	return keys.map((key) => ({ key, count: send[key] ?? 0 })).filter(({ count }) => count > 0);
}

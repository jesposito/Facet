/**
 * Helpers for the newsletter compose page. Kept apart from the Svelte file so
 * the rules the page follows are tested directly.
 */

/** The "All active subscribers" audience choice. */
export const ALL_SUBSCRIBERS = '';

export type SendInput = {
	subject: string;
	bodyMarkdown: string;
	template?: string;
	/** The chosen list id, or ALL_SUBSCRIBERS. Always sent as chosen. */
	listId: string;
	personalize: boolean;
	requestKey?: string;
};

/**
 * The request body for preview, test and send. list_id is always present,
 * so the server never has to guess the audience: '' means every active
 * subscriber, anything else is that list.
 */
export function buildSendPayload(input: SendInput): Record<string, unknown> {
	const payload: Record<string, unknown> = {
		subject: input.subject.trim(),
		body_markdown: input.bodyMarkdown,
		list_id: input.listId,
		personalize: input.personalize
	};
	if (input.template) payload.template = input.template;
	if (input.requestKey) payload.request_key = input.requestKey;
	return payload;
}

/** A fresh key for one Send click. */
export function newRequestKey(): string {
	if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
		return crypto.randomUUID();
	}
	const bytes = new Uint8Array(16);
	crypto.getRandomValues(bytes);
	return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}

const PLACEHOLDER =
	/\{\{\s*(subscriber\.name|subscriber\.email|list\.name|site\.name|site\.url)\s*\}\}/;
const SUBSCRIBER_PLACEHOLDER = /\{\{\s*subscriber\.(name|email)\s*\}\}/;
// Markdown link and image targets, and raw href/src attributes.
const LINK_TARGETS = /\]\(\s*<?([^)\s>]*)|(?:href|src)\s*=\s*["']([^"']*)["']/gi;

export const PER_READER_LINK_MESSAGE =
	"Links can't use {{ subscriber.name }} or {{ subscriber.email }}. Each link is tracked and checked against the newsletter as sent, so per-reader addresses can't be followed safely. Put the placeholder in the text instead.";
export const LINK_NEEDS_PERSONALIZE_MESSAGE =
	"A link uses a placeholder such as {{ site.url }}. Turn on personalization so it's filled in, or write the address out.";

function decodeBraces(value: string): string {
	return value.replace(/%7b/gi, '{').replace(/%7d/gi, '}').replace(/%20/g, ' ');
}

/**
 * The same check the server makes on links, so the author sees the problem
 * while writing. Returns a message, or '' when the links are fine.
 */
export function findLinkPlaceholderProblem(markdown: string, personalize: boolean): string {
	let needsPersonalize = false;
	for (const match of markdown.matchAll(LINK_TARGETS)) {
		const target = decodeBraces(match[1] ?? match[2] ?? '');
		if (SUBSCRIBER_PLACEHOLDER.test(target)) return PER_READER_LINK_MESSAGE;
		if (PLACEHOLDER.test(target) && !personalize) needsPersonalize = true;
	}
	return needsPersonalize ? LINK_NEEDS_PERSONALIZE_MESSAGE : '';
}

/** The message to show for a failed send, preview or test. */
export function sendErrorMessage(data: unknown, fallback: string): string {
	if (data && typeof data === 'object' && typeof (data as { error?: unknown }).error === 'string') {
		return (data as { error: string }).error;
	}
	return fallback;
}

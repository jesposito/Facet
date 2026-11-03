import { expect, test } from '@playwright/test';
import {
	ALL_SUBSCRIBERS,
	LINK_NEEDS_PERSONALIZE_MESSAGE,
	PER_READER_LINK_MESSAGE,
	buildSendPayload,
	findLinkPlaceholderProblem,
	newRequestKey,
	sendErrorMessage
} from '../src/lib/newsletter/compose';
import {
	buildSignupBody,
	loadPublicNewsletterLists,
	readSignupOutcome
} from '../src/lib/newsletter/signup';
import {
	outcomeCounts,
	resumeState,
	smtpReadiness,
	type NewsletterSend
} from '../src/lib/newsletter/sends';

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

test.describe('newsletter compose', () => {
	test('the chosen list is always sent, including "all"', () => {
		const toList = buildSendPayload({
			subject: ' Hi ',
			bodyMarkdown: 'Body',
			listId: 'list123',
			personalize: false
		});
		expect(toList.list_id).toBe('list123');
		expect(toList.subject).toBe('Hi');
		const toAll = buildSendPayload({
			subject: 'Hi',
			bodyMarkdown: 'Body',
			listId: ALL_SUBSCRIBERS,
			personalize: true
		});
		expect(toAll, 'list_id must be present for "all"').toHaveProperty('list_id', '');
		expect(toAll.personalize).toBe(true);
	});

	test('request keys are carried and unique per click', () => {
		const a = newRequestKey();
		const b = newRequestKey();
		expect(a).not.toBe(b);
		expect(a).toMatch(/^[A-Za-z0-9_-]{8,100}$/);
		const payload = buildSendPayload({
			subject: 'Hi',
			bodyMarkdown: 'B',
			listId: '',
			personalize: false,
			requestKey: a
		});
		expect(payload.request_key).toBe(a);
		expect(
			buildSendPayload({ subject: 'Hi', bodyMarkdown: 'B', listId: '', personalize: false })
		).not.toHaveProperty('request_key');
	});

	test('per-reader placeholders in links are flagged', () => {
		for (const md of [
			'[me](https://shop.example/?u={{subscriber.email}})',
			'[me](https://shop.example/{{subscriber.name}})',
			'<a href="https://x.example/%7B%7Bsubscriber.email%7D%7D">x</a>',
			'![pic](https://img.example/{{subscriber.name}}.png)'
		]) {
			expect(findLinkPlaceholderProblem(md, true), md).toBe(PER_READER_LINK_MESSAGE);
			expect(findLinkPlaceholderProblem(md, false), md).toBe(PER_READER_LINK_MESSAGE);
		}
	});

	test('shared placeholders in links need personalization on', () => {
		const md = 'See [the blog]({{site.url}}/blog)';
		expect(findLinkPlaceholderProblem(md, false)).toBe(LINK_NEEDS_PERSONALIZE_MESSAGE);
		expect(findLinkPlaceholderProblem(md, true)).toBe('');
	});

	test('placeholders in text are fine', () => {
		expect(
			findLinkPlaceholderProblem('Hi {{ subscriber.name }}, [home](https://facet.example)', true)
		).toBe('');
		expect(findLinkPlaceholderProblem('Hi {{ subscriber.name }}', false)).toBe('');
	});

	test('server errors are shown as written', () => {
		expect(sendErrorMessage({ error: 'newsletter list is inactive' }, 'x')).toBe(
			'newsletter list is inactive'
		);
		expect(sendErrorMessage({}, 'fallback')).toBe('fallback');
		expect(sendErrorMessage(null, 'fallback')).toBe('fallback');
	});
});

test.describe('newsletter signup', () => {
	test('a named list is sent; no list means the default', () => {
		expect(
			buildSignupBody({ email: ' Reader@Example.com ', name: ' Jo ', list: 'greenhouse' })
		).toEqual({
			email: 'reader@example.com',
			name: 'Jo',
			list: 'greenhouse'
		});
		expect(buildSignupBody({ email: 'r@example.com', list: '  ' })).not.toHaveProperty('list');
	});

	test('a refused list is reported, never shown as success', async () => {
		expect(await readSignupOutcome(jsonResponse(200, { status: 'confirmation_required' }))).toBe(
			'confirmation_required'
		);
		expect(
			await readSignupOutcome(
				jsonResponse(400, {
					error: "this newsletter list isn't open for signup",
					code: 'list_unavailable'
				})
			)
		).toBe('list_unavailable');
		expect(await readSignupOutcome(jsonResponse(429, {}))).toBe('rate_limited');
		expect(await readSignupOutcome(jsonResponse(500, {}))).toBe('error');
	});

	test('public lists load, and failures give none', async () => {
		const lists = [
			{ id: 'a', slug: 'default', name: 'Newsletter', description: '', is_default: true }
		];
		const ok = (async () => jsonResponse(200, { data: lists })) as unknown as typeof fetch;
		expect(await loadPublicNewsletterLists(ok)).toEqual(lists);
		const down = (async () => jsonResponse(404, {})) as unknown as typeof fetch;
		expect(await loadPublicNewsletterLists(down)).toEqual([]);
		const offline = (async () => {
			throw new Error('offline');
		}) as unknown as typeof fetch;
		expect(await loadPublicNewsletterLists(offline)).toEqual([]);
	});
});

function send(overrides: Partial<NewsletterSend>): NewsletterSend {
	return {
		id: 'send1',
		subject: 'Hello',
		status: 'sent',
		recipient_count: 10,
		sent_count: 10,
		failed_count: 0,
		pending_count: 0,
		uncertain_count: 0,
		skipped_count: 0,
		resumable_count: 0,
		open_count: 0,
		click_count: 0,
		sent_at: '',
		created: '',
		...overrides
	};
}

test.describe('newsletter send outcomes', () => {
	test('sending needs SMTP to be ready, not just configured', () => {
		expect(smtpReadiness({ configured: true, ready: true, host: 'smtp.example' })).toBe('ready');
		expect(smtpReadiness({ configured: true, ready: false })).toBe('not_ready');
		expect(smtpReadiness({ configured: false, ready: false })).toBe('not_configured');
		// The pre-readiness response shape is not enough to send.
		expect(smtpReadiness({ configured: true })).toBe('not_ready');
		expect(smtpReadiness(null)).toBe('not_configured');
	});

	test('a fully accepted send offers nothing to resume', () => {
		const done = send({});
		expect(resumeState(done, [done])).toBe('hidden');
		expect(outcomeCounts(done)).toEqual([]);
	});

	test('a partial send with pending or rejected recipients can be resumed', () => {
		const partial = send({
			status: 'partial',
			sent_count: 6,
			failed_count: 1,
			pending_count: 3,
			resumable_count: 4
		});
		expect(resumeState(partial, [partial])).toBe('available');
		expect(outcomeCounts(partial)).toEqual([
			{ key: 'failed_count', count: 1 },
			{ key: 'pending_count', count: 3 }
		]);

		const interrupted = send({
			status: 'interrupted',
			sent_count: 4,
			pending_count: 6,
			resumable_count: 6
		});
		expect(resumeState(interrupted, [interrupted])).toBe('available');
	});

	test('uncertain recipients are reported but never make a send resumable', () => {
		const uncertainOnly = send({ status: 'partial', sent_count: 8, uncertain_count: 2 });
		expect(resumeState(uncertainOnly, [uncertainOnly])).toBe('hidden');
		expect(outcomeCounts(uncertainOnly)).toEqual([{ key: 'uncertain_count', count: 2 }]);
	});

	test('skipped recipients alone are not resumable', () => {
		const skipped = send({ status: 'partial', sent_count: 9, skipped_count: 1 });
		expect(resumeState(skipped, [skipped])).toBe('hidden');
	});

	test('only one campaign runs at a time, so resume waits for an active send', () => {
		const failed = send({
			id: 'a',
			status: 'failed',
			sent_count: 0,
			failed_count: 10,
			resumable_count: 10
		});
		const active = send({
			id: 'b',
			status: 'sending',
			sent_count: 2,
			pending_count: 8,
			resumable_count: 8
		});
		expect(resumeState(failed, [failed, active])).toBe('blocked');
		expect(resumeState(active, [failed, active]), 'a running send is not resumed').toBe('hidden');
	});

	test('resume follows the ledger count, not historical counters', () => {
		// Logged before per-recipient tracking: failures are history only.
		const legacy = send({ status: 'failed', sent_count: 3, failed_count: 2, resumable_count: 0 });
		expect(resumeState(legacy, [legacy])).toBe('hidden');
		expect(outcomeCounts(legacy)).toEqual([{ key: 'failed_count', count: 2 }]);

		const interrupted = send({
			status: 'interrupted',
			sent_count: 3,
			failed_count: 2,
			pending_count: 0
		});
		delete interrupted.resumable_count;
		expect(resumeState(interrupted, [interrupted]), 'no count means not known to be safe').toBe(
			'hidden'
		);
	});

	test('older send records without the new counts still render', () => {
		const legacy = send({ status: 'failed', failed_count: 2 });
		delete legacy.pending_count;
		delete legacy.uncertain_count;
		delete legacy.skipped_count;
		delete legacy.resumable_count;
		expect(outcomeCounts(legacy)).toEqual([{ key: 'failed_count', count: 2 }]);
		expect(resumeState(legacy, [legacy])).toBe('hidden');
	});
});

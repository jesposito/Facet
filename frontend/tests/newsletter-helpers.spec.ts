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

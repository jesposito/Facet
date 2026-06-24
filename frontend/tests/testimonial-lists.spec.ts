import { test, expect, type APIRequestContext } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin } from './helpers';

// #283 — testimonial lists: a request carries a free-text list; the submitted
// testimonial inherits it; a facet's testimonials section pointed at that list
// auto-publishes approved testimonials from it (and hides off-list ones), with
// the empty-list path unchanged. API-level so it's deterministic under workers=1.

const LIST = 'e2e-clients';
const OTHER = 'e2e-other';

async function createRequest(request: APIRequestContext, token: string, list: string) {
	const res = await request.post(`${apiBaseURL}/api/testimonials/requests`, {
		headers: { Authorization: token },
		data: { label: `e2e ${list}`, list }
	});
	expect(res.ok()).toBeTruthy();
	return (await res.json()).token as string;
}

async function submitAndApprove(
	request: APIRequestContext,
	token: string,
	reqToken: string,
	author: string
) {
	const sub = await request.post(`${apiBaseURL}/api/testimonials/submit`, {
		data: { request_token: reqToken, content: `${author} says great.`, author_name: author }
	});
	expect(sub.ok()).toBeTruthy();
	const id = (await sub.json()).id as string;
	const ok = await request.post(`${apiBaseURL}/api/testimonials/${id}/approve`, {
		headers: { Authorization: token }
	});
	expect(ok.ok()).toBeTruthy();
	return id;
}

async function homepageAuthors(request: APIRequestContext): Promise<string[]> {
	const res = await request.get(`${apiBaseURL}/api/homepage`);
	expect(res.ok()).toBeTruthy();
	const ts = (await res.json()).testimonials ?? [];
	return ts.map((t: any) => t.author_name as string);
}

test.describe('Testimonial lists (#283)', () => {
	let token = '';
	let savedSections: unknown = null;

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;
		// Snapshot site-settings so we can restore the section config afterwards.
		const cur = await request.get(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token }
		});
		if (cur.ok()) savedSections = (await cur.json()).homepage_sections ?? null;
	});

	test.afterAll(async ({ request }) => {
		if (token) {
			await request.put(`${apiBaseURL}/api/site-settings`, {
				headers: { Authorization: token },
				data: { homepage_sections: savedSections ?? {} }
			});
		}
	});

	test('submitted testimonial inherits the request list', async ({ request }) => {
		const reqToken = await createRequest(request, token, LIST);
		const sub = await request.post(`${apiBaseURL}/api/testimonials/submit`, {
			data: { request_token: reqToken, content: 'Inherit check.', author_name: 'Inherit Tester' }
		});
		expect(sub.ok()).toBeTruthy();
		const id = (await sub.json()).id as string;
		const rec = await request.get(`${apiBaseURL}/api/collections/testimonials/records/${id}`, {
			headers: { Authorization: token }
		});
		expect(rec.ok()).toBeTruthy();
		expect((await rec.json()).list).toBe(LIST);
	});

	test('a facet list section auto-publishes approved list testimonials and hides off-list', async ({
		request
	}) => {
		const r1 = await createRequest(request, token, LIST);
		await submitAndApprove(request, token, r1, 'Ada Listmember');
		const r2 = await createRequest(request, token, OTHER);
		await submitAndApprove(request, token, r2, 'Bob Offlist');

		// Point the homepage testimonials section at LIST (no explicit picks).
		const put = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data: {
				homepage_enabled: true,
				homepage_section_order: ['testimonials'],
				homepage_sections: { testimonials: { enabled: true, items: [], list: LIST } }
			}
		});
		expect(put.ok()).toBeTruthy();

		const authors = await homepageAuthors(request);
		expect(authors).toContain('Ada Listmember'); // auto-shown
		expect(authors).not.toContain('Bob Offlist'); // off-list hidden
	});

	test('/api/testimonials/lists returns the distinct lists in use', async ({ request }) => {
		const res = await request.get(`${apiBaseURL}/api/testimonials/lists`, {
			headers: { Authorization: token }
		});
		expect(res.ok()).toBeTruthy();
		const lists = (await res.json()).lists as string[];
		expect(lists).toContain(LIST);
		expect(lists).toContain(OTHER);
	});
});

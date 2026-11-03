import { test, expect } from '@playwright/test';
import { apiBaseURL } from './config';
import { getWithRateLimitRetry } from './helpers';

test.describe('Public APIs and feeds', () => {
	test('default view or homepage is reachable', async ({ request }) => {
		const defaultViewRes = await getWithRateLimitRetry(request, `${apiBaseURL}/api/default-view`);
		expect(defaultViewRes.ok()).toBeTruthy();
		const defaultView = await defaultViewRes.json();

		if (defaultView.has_default && defaultView.slug) {
			const viewRes = await getWithRateLimitRetry(
				request,
				`${apiBaseURL}/api/view/${defaultView.slug}/data`
			);
			expect(viewRes.ok()).toBeTruthy();
			const view = await viewRes.json();
			expect(view.slug).toBe(defaultView.slug);
			expect(view.sections).toBeDefined();
		} else {
			const homeRes = await getWithRateLimitRetry(request, `${apiBaseURL}/api/homepage`);
			expect(homeRes.ok()).toBeTruthy();
			const home = await homeRes.json();
			expect(home).toHaveProperty('homepage_enabled');
		}
	});

	test('AI and AI Print capability endpoints respond', async ({ request }) => {
		const aiStatusRes = await getWithRateLimitRetry(request, `${apiBaseURL}/api/ai/status`);
		expect(aiStatusRes.ok()).toBeTruthy();
		const aiStatus = await aiStatusRes.json();
		expect(aiStatus).toHaveProperty('available');

		const printStatusRes = await getWithRateLimitRetry(
			request,
			`${apiBaseURL}/api/ai-print/status`
		);
		expect(printStatusRes.ok()).toBeTruthy();
		const printStatus = await printStatusRes.json();
		expect(printStatus).toHaveProperty('available');
		expect(printStatus).toHaveProperty('supported_formats');
	});

	test('Posts and talks listings return success (even if empty)', async ({ request }) => {
		const postsRes = await getWithRateLimitRetry(request, `${apiBaseURL}/api/posts`);
		expect(postsRes.ok(), await postsRes.text()).toBeTruthy();
		const posts = await postsRes.json();
		expect(posts).toHaveProperty('posts');

		const talksRes = await getWithRateLimitRetry(request, `${apiBaseURL}/api/talks`);
		expect(talksRes.ok(), await talksRes.text()).toBeTruthy();
		const talks = await talksRes.json();
		expect(talks).toHaveProperty('talks');
	});

	test('RSS and ICS feeds are available', async ({ request }) => {
		const rssRes = await getWithRateLimitRetry(request, `${apiBaseURL}/rss.xml`);
		expect(rssRes.status(), await rssRes.text()).toBe(200);
		const rssText = await rssRes.text();
		expect(rssText.toLowerCase()).toContain('<rss');

		const icsRes = await getWithRateLimitRetry(request, `${apiBaseURL}/talks.ics`);
		expect(icsRes.status(), await icsRes.text()).toBe(200);
		const icsText = await icsRes.text();
		expect(icsText).toContain('BEGIN:VCALENDAR');
	});
});

import { test, expect } from '@playwright/test';
import { apiBaseURL } from './config';

test.describe('Public APIs and feeds', () => {
	test('default view or homepage is reachable', async ({ request }) => {
		const defaultViewRes = await request.get(`${apiBaseURL}/api/default-view`);
		expect(defaultViewRes.ok()).toBeTruthy();
		const defaultView = await defaultViewRes.json();

		if (defaultView.has_default && defaultView.slug) {
			const viewRes = await request.get(`${apiBaseURL}/api/view/${defaultView.slug}/data`);
			expect(viewRes.ok()).toBeTruthy();
			const view = await viewRes.json();
			expect(view.slug).toBe(defaultView.slug);
			expect(view.sections).toBeDefined();
		} else {
			const homeRes = await request.get(`${apiBaseURL}/api/homepage`);
			expect(homeRes.ok()).toBeTruthy();
			const home = await homeRes.json();
			expect(home).toHaveProperty('homepage_enabled');
		}
	});

	test('AI and AI Print capability endpoints respond', async ({ request }) => {
		const aiStatusRes = await request.get(`${apiBaseURL}/api/ai/status`);
		expect(aiStatusRes.ok()).toBeTruthy();
		const aiStatus = await aiStatusRes.json();
		expect(aiStatus).toHaveProperty('available');

		const printStatusRes = await request.get(`${apiBaseURL}/api/ai-print/status`);
		expect(printStatusRes.ok()).toBeTruthy();
		const printStatus = await printStatusRes.json();
		expect(printStatus).toHaveProperty('available');
		expect(printStatus).toHaveProperty('supported_formats');
	});

	test('Posts and talks listings return success (even if empty)', async ({ request }) => {
		const postsRes = await request.get(`${apiBaseURL}/api/posts`);
		expect(postsRes.ok()).toBeTruthy();
		const posts = await postsRes.json();
		expect(posts).toHaveProperty('posts');

		const talksRes = await request.get(`${apiBaseURL}/api/talks`);
		expect(talksRes.ok()).toBeTruthy();
		const talks = await talksRes.json();
		expect(talks).toHaveProperty('talks');
	});

	test('RSS and ICS feeds are available', async ({ request }) => {
		const rssRes = await request.get(`${apiBaseURL}/rss.xml`);
		expect(rssRes.status()).toBe(200);
		const rssText = await rssRes.text();
		expect(rssText.toLowerCase()).toContain('<rss');

		const icsRes = await request.get(`${apiBaseURL}/talks.ics`);
		expect(icsRes.status()).toBe(200);
		const icsText = await icsRes.text();
		expect(icsText).toContain('BEGIN:VCALENDAR');
	});
});

import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';
import { apiBaseURL } from './config';

const richText = `<p>Embedded content stays within the page.</p><iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" width="1200" height="675" title="Video fixture"></iframe><p>${'LongLink'.repeat(60)}</p><pre><code>${'long_code_'.repeat(80)}</code></pre><iframe src="https://calendly.com/example/demo" width="1200" height="700" title="Booking fixture"></iframe>`;

test.describe('portfolio portability', () => {
	const created: { collection: string; id: string }[] = [];
	let token: string;
	let originalCSS: string;
	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		originalCSS =
			(await (await request.get(`${apiBaseURL}/api/site-settings`)).json()).custom_css || '';
		for (const [collection, data] of [
			[
				'projects',
				{
					title: 'Responsive project',
					slug: 'responsive-project',
					description: richText,
					source_id: 'private-project-source',
					field_locks: { description: true }
				}
			],
			['talks', { title: 'Responsive talk', slug: 'responsive-talk', description: richText }],
			['custom_content', { title: 'Responsive custom content', content: richText }],
			[
				'testimonials',
				{
					content: 'Public testimonial',
					author_name: 'Fixture author',
					status: 'approved',
					verification_identifier: 'private-fixture@example.test',
					verification_data: { secret: 'private-verification-data' },
					rejection_reason: 'private-reason'
				}
			]
		] as const) {
			const result = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
				headers: { Authorization: token },
				data: { ...data, visibility: 'public', is_draft: false }
			});
			expect(result.ok(), await result.text()).toBe(true);
			created.push({ collection, id: (await result.json()).id });
		}
		const sections = created.map(({ collection, id }) => ({
			section: collection === 'custom_content' ? `custom:${id}` : collection,
			enabled: true,
			layout: 'default',
			items: [id]
		}));
		for (const layout of ['default', 'card', 'hero']) {
			const response = await request.post(`${apiBaseURL}/api/collections/views/records`, {
				headers: { Authorization: token },
				data: {
					name: `Responsive ${layout}`,
					slug: `responsive-${layout}`,
					visibility: 'public',
					is_active: true,
					sections: sections.map((s) => ({
						...s,
						layout: s.section.startsWith('custom:') ? layout : 'default'
					}))
				}
			});
			expect(response.ok(), await response.text()).toBe(true);
			created.push({ collection: 'views', id: (await response.json()).id });
		}
	});
	test.afterAll(async ({ request }) => {
		await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data: { custom_css: originalCSS }
		});
		for (const { collection, id } of [...created].reverse())
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
	});
	test('anonymous responses exclude internal fields', async ({ request }) => {
		for (const path of [
			'/api/homepage',
			'/api/view/responsive-default/data',
			'/api/project/responsive-project'
		]) {
			const response = await request.get(`${apiBaseURL}${path}`);
			expect(response.ok(), await response.text()).toBe(true);
			const text = await response.text();
			expect(text).toContain('Responsive project');
			for (const secret of [
				'private-project-source',
				'private-fixture@example.test',
				'private-verification-data',
				'private-reason',
				'"field_locks"',
				'"admin_tags"',
				'"verification_identifier"'
			])
				expect(text).not.toContain(secret);
		}
		for (const collection of ['projects', 'testimonials']) {
			const id = created.find((x) => x.collection === collection)!.id;
			expect(
				(await request.get(`${apiBaseURL}/api/collections/${collection}/records/${id}`)).status()
			).toBe(404);
		}
	});
	test('CSS API rejects oversized input without changing saved text', async ({ request }) => {
		const headers = { Authorization: token },
			baseline = '/* saved baseline */';
		expect(
			(
				await request.put(`${apiBaseURL}/api/site-settings`, {
					headers,
					data: { custom_css: baseline }
				})
			).ok()
		).toBe(true);
		expect(
			(
				await request.put(`${apiBaseURL}/api/site-settings`, {
					headers,
					data: { custom_css: 'x'.repeat(20001) }
				})
			).status()
		).toBe(400);
		expect((await (await request.get(`${apiBaseURL}/api/site-settings`)).json()).custom_css).toBe(
			baseline
		);
		const unicode = '/*' + '🌱'.repeat(19996) + '*/';
		expect(
			(
				await request.put(`${apiBaseURL}/api/site-settings`, {
					headers,
					data: { custom_css: unicode }
				})
			).ok()
		).toBe(true);
		expect((await (await request.get(`${apiBaseURL}/api/site-settings`)).json()).custom_css).toBe(
			unicode
		);
		await request.put(`${apiBaseURL}/api/site-settings`, {
			headers,
			data: { custom_css: originalCSS }
		});
	});
	test('CSS editor preserves long pastes and saves Unicode', async ({
		page,
		request
	}, testInfo) => {
		const { token, record: model } = await loginAsAdmin(request);
		await page
			.context()
			.addCookies([
				{
					name: 'pb_auth',
					value: JSON.stringify({ token, model }),
					url: process.env.PLAYWRIGHT_BASE_URL!
				}
			]);
		await page.addInitScript(
			({ token, model }) => {
				localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
				localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
			},
			{ token, model }
		);
		await page.setViewportSize({ width: 390, height: 900 });
		await page.goto('/admin/settings/site');
		const editor = page.getByRole('textbox', { name: 'Custom CSS', exact: true });
		await expect(editor).toBeEnabled();
		const oversized = '/*' + '🌱'.repeat(20000) + '*/';
		await editor.fill(oversized);
		await expect(editor).toHaveValue(oversized);
		await expect(editor).toHaveAttribute('aria-invalid', 'true');
		await page.locator('#appearance').getByRole('button', { name: 'Save', exact: true }).click();
		await expect(editor).toBeFocused();
		await expect(page.locator('#custom-css-limit-error')).toBeVisible();
		await expect(page.getByText('20004/20000', { exact: true })).toBeVisible();
		await page.screenshot({ path: testInfo.outputPath('css-error-mobile.png') });
		expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
			true
		);
		const unicode = '/*' + '🌱'.repeat(19996) + '*/';
		await editor.fill(unicode);
		await expect(page.locator('#custom-css-limit-error')).toHaveCount(0);
		const saved = page.waitForResponse(
			(r) => r.url().endsWith('/api/site-settings') && r.request().method() === 'PUT'
		);
		await page.locator('#appearance').getByRole('button', { name: 'Save', exact: true }).click();
		expect((await saved).ok()).toBe(true);
		await page.reload();
		await expect(editor).toHaveValue(unicode);
	});
	for (const width of [390, 1440])
		for (const theme of ['light', 'dark']) {
			test(`rich embeds fit ${width}px in ${theme} mode`, async ({ page }, testInfo) => {
				await page.setViewportSize({ width, height: 1000 });
				await page.route(/https:\/\/(www\.youtube\.com|calendly\.com)\//, (route) =>
					route.fulfill({
						contentType: 'text/html',
						body: '<html><body style="margin:0;background:#dbeafe;color:#172554;font:20px sans-serif;padding:24px">Embedded content fixture</body></html>'
					})
				);
				for (const path of [
					'/projects/responsive-project',
					'/talks/responsive-talk',
					'/responsive-default',
					'/responsive-card',
					'/responsive-hero'
				]) {
					await page.goto(path);
					await page.evaluate(
						(theme) => document.documentElement.classList.toggle('dark', theme === 'dark'),
						theme
					);
					const video = page.locator('.rich-content-safe iframe[title="Video fixture"]').first();
					await expect(video).toBeVisible();
					const bounds = await video.boundingBox();
					expect(bounds!.width / bounds!.height).toBeCloseTo(16 / 9, 1);
					expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width);
					expect(
						(await page
							.locator('.rich-content-safe iframe[title="Booking fixture"]')
							.first()
							.boundingBox())!.height
					).toBeCloseTo(700, 1);
					expect(
						await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)
					).toBe(true);
					if (path === '/projects/responsive-project')
						await page.screenshot({
							path: testInfo.outputPath(`project-${width}-${theme}.png`),
							fullPage: true
						});
				}
			});
		}
});

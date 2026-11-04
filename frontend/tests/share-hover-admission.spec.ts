import { randomUUID } from 'node:crypto';
import { expect, test } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin } from './helpers';

test.describe('Share admission requires navigation', () => {
	let ownerToken: string;
	const created: { collection: string; id: string }[] = [];

	test.beforeEach(async ({ request }) => {
		({ token: ownerToken } = await loginAsAdmin(request));
	});

	test.afterEach(async ({ request }) => {
		for (const record of created.splice(0).reverse()) {
			const response = await request.delete(
				`${apiBaseURL}/api/collections/${record.collection}/records/${record.id}`,
				{ headers: { Authorization: ownerToken } }
			);
			expect(response.ok()).toBe(true);
		}
	});

	for (const entry of ['canonical', 'legacy'] as const) {
		test(`${entry} plain HTML link keeps its single use until clicked`, async ({
			page,
			request
		}) => {
			const headers = { Authorization: ownerToken };
			const slug = `share-hover-${randomUUID()}`;
			const facet = await request.post(`${apiBaseURL}/api/collections/views/records`, {
				headers,
				data: {
					name: 'Disposable share hover check',
					slug,
					visibility: 'unlisted',
					is_active: true,
					sections: []
				}
			});
			expect(facet.ok()).toBe(true);
			const { id: facetId } = await facet.json();
			created.push({ collection: 'views', id: facetId });
			const generated = await request.post(`${apiBaseURL}/api/share/generate`, {
				headers,
				data: { view_id: facetId, name: 'Disposable share hover check', max_uses: 1 }
			});
			expect(generated.ok()).toBe(true);
			const share = await generated.json();
			created.push({ collection: 'share_tokens', id: share.id });
			const useCount = async () => {
				const response = await request.get(
					`${apiBaseURL}/api/collections/share_tokens/records/${share.id}`,
					{ headers }
				);
				expect(response.ok()).toBe(true);
				return (await response.json()).use_count as number;
			};

			// API setup authenticates only the request fixture. The browser stays anonymous.
			expect(await page.context().cookies()).toEqual([]);
			await page.goto('/');
			await page.waitForLoadState('networkidle');
			const entryPath = entry === 'canonical' ? `/s/${share.token}` : `/${slug}`;
			const href = entry === 'canonical' ? entryPath : `${entryPath}?t=${share.token}`;
			let entryLoads = 0;
			page.on('request', (sent) => {
				const url = new URL(sent.url());
				if (
					url.pathname === `${entryPath}/__data.json` &&
					(entry === 'canonical' || url.searchParams.get('t') === share.token)
				)
					entryLoads += 1;
			});
			// Raw HTML links inherit the document policy, without a component adding attributes.
			await page.evaluate((href) => {
				const link = document.createElement('a');
				link.id = 'share-hover-check';
				link.href = href;
				link.textContent = 'Open the shared Facet';
				Object.assign(link.style, {
					position: 'fixed',
					top: '100px',
					left: '100px',
					zIndex: '9999',
					padding: '20px',
					background: 'white',
					color: 'black'
				});
				document.body.append(link);
			}, href);
			expect(await useCount()).toBe(0);
			await page.locator('#share-hover-check').hover();
			// SvelteKit waits 20 ms before hover preloading; allow the full admission to finish.
			await page.waitForTimeout(700);
			expect(entryLoads, 'hover must not call the share entry server loader').toBe(0);
			expect(await useCount(), 'hover must leave the single-use link available').toBe(0);

			await page.locator('#share-hover-check').click();
			await expect(page).toHaveURL(new RegExp(`/${slug}$`));
			await expect.poll(useCount).toBe(1);
			// A data request proves the router was active, rather than an unhydrated hard navigation.
			expect(entryLoads).toBe(1);
			expect((await page.context().cookies()).some((cookie) => cookie.name === 'pb_auth')).toBe(
				false
			);
		});
	}
});

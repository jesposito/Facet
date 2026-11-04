import { expect, test, type Page, type Route } from '@playwright/test';
import { adminEmail, adminPassword } from './config';
import { seedAdminSession } from './helpers';

const tokensApi = /\/api\/collections\/share_tokens\/records(?:\?|$)/;
const viewsApi = /\/api\/collections\/views\/records(?:\?|$)/;

function facet(id = 'facet1', visibility = 'unlisted', isActive = true) {
	return { id, name: `Facet ${id}`, slug: id, visibility, is_active: isActive };
}

function link(id: string, name: string, expandedFacet = facet()) {
	return {
		id,
		view_id: expandedFacet.id,
		name,
		use_count: 0,
		max_uses: 0,
		is_active: true,
		created: '2026-01-01 00:00:00.000Z',
		expand: { view_id: expandedFacet }
	};
}

async function fulfillPage(route: Route, records: unknown[]) {
	const url = new URL(route.request().url());
	const page = Number(url.searchParams.get('page') ?? 1);
	const perPage = Number(url.searchParams.get('perPage') ?? 30);
	await route.fulfill({
		json: {
			page,
			perPage,
			totalItems: records.length,
			totalPages: Math.ceil(records.length / perPage),
			items: records.slice((page - 1) * perPage, page * perPage)
		}
	});
}

async function mockViews(page: Page, records = [facet()]) {
	await page.route(viewsApi, (route) => fulfillPage(route, records));
}

test.describe('Share token list reliability', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
	});

	test('lists links beyond the first 100 records', async ({ page }) => {
		const records = Array.from({ length: 101 }, (_, index) =>
			link(`token${index + 1}`, `Paged link ${index + 1}`)
		);
		const requestedPages: number[] = [];
		await page.route(tokensApi, (route) => {
			requestedPages.push(Number(new URL(route.request().url()).searchParams.get('page') ?? 1));
			return fulfillPage(route, records);
		});
		await mockViews(page);

		await page.goto('/admin/tokens');
		await expect(page.getByText('Paged link 101', { exact: true })).toBeVisible();
		expect(requestedPages).toEqual([1, 2]);
	});

	test('a failed read shows Retry instead of an empty list', async ({ page }) => {
		let failReads = true;
		await page.route(tokensApi, (route) =>
			failReads
				? route.fulfill({ status: 500, json: { message: 'Injected read failure' } })
				: fulfillPage(route, [link('token1', 'Recovered link')])
		);
		await mockViews(page);

		await page.goto('/admin/tokens');
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load share links" });
		await expect(alert).toBeVisible();
		await expect(page.getByText('No share tokens yet.')).toHaveCount(0);

		failReads = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByText('Recovered link', { exact: true })).toBeVisible();
	});

	for (const malformed of ['not-json', { items: 'broken' }]) {
		test(`a malformed list response stays unknown (${typeof malformed})`, async ({ page }) => {
			let failReads = true;
			await page.route(tokensApi, (route) =>
				failReads
					? route.fulfill({
							contentType: 'application/json',
							body: typeof malformed === 'string' ? malformed : JSON.stringify(malformed)
						})
					: fulfillPage(route, [link('recovered', 'Recovered malformed response')])
			);
			await mockViews(page);
			await page.goto('/admin/tokens');
			const alert = page.getByRole('alert').filter({ hasText: "Couldn't load share links" });
			await expect(alert).toBeVisible();
			await expect(page.getByText('No share tokens yet.')).toHaveCount(0);
			failReads = false;
			await alert.getByRole('button', { name: 'Retry', exact: true }).click();
			await expect(page.getByText('Recovered malformed response', { exact: true })).toBeVisible();
		});
	}

	test('only an unlisted active Facet has an Active share link', async ({ page }) => {
		const records: unknown[] = [
			link('active', 'Usable link'),
			link('private', 'Private Facet link', facet('private', 'private')),
			link('inactive', 'Inactive Facet link', facet('inactive', 'unlisted', false)),
			link('public', 'Public Facet link', facet('public', 'public'))
		];
		const missing = link('missing', 'Missing Facet link');
		Reflect.deleteProperty(missing, 'expand');
		records.push(missing);
		await page.route(tokensApi, (route) => fulfillPage(route, records));
		await mockViews(page);

		await page.goto('/admin/tokens');
		const row = (name: string) => page.locator('.divide-y > div').filter({ hasText: name });
		await expect(row('Usable link').getByText('Active', { exact: true })).toBeVisible();
		for (const name of [
			'Private Facet link',
			'Inactive Facet link',
			'Public Facet link',
			'Missing Facet link'
		]) {
			await expect(row(name).getByText('Unavailable', { exact: true })).toBeVisible();
			await expect(row(name).getByText('Active', { exact: true })).toHaveCount(0);
		}
		await expect(page.getByText(/missing, inactive, or not unlisted/).first()).toBeVisible();
	});

	test('a pending create cannot be submitted twice', async ({ page }) => {
		await page.route(tokensApi, (route) => fulfillPage(route, []));
		await mockViews(page);
		let creates = 0;
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route('**/api/share/generate', async (route) => {
			creates += 1;
			await held;
			await route.fulfill({ json: { id: 'new-token', token: 'raw-created-token' } });
		});

		await page.goto('/admin/tokens');
		await page.getByRole('button', { name: 'Generate Token', exact: true }).first().click();
		const form = page.locator('form:has(#view_id)');
		await form.locator('#view_id').selectOption('facet1');
		const submit = form.locator('button[type=submit]');
		await submit.evaluate((button) => {
			(button as HTMLButtonElement).click();
			(button as HTMLButtonElement).click();
		});
		await expect(submit).toBeDisabled();
		expect(creates).toBe(1);
		release();
		await expect(page.locator('code', { hasText: '/s/raw-created-token' })).toBeVisible();
	});
});

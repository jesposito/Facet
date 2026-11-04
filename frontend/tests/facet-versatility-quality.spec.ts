import { randomUUID } from 'node:crypto';
import { test, expect, type APIRequestContext, type Page, type Route } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import {
	createPublicProjectAndPost,
	deletePublicProjectAndPost,
	loginAsAdmin,
	seedAdminSession
} from './helpers';

const VIEWPORTS = [
	{ width: 390, height: 844 },
	{ width: 768, height: 1024 },
	{ width: 1440, height: 900 }
];

const GUIDANCE_HEADING = 'One Facet, many uses';
const ERROR_TITLE = "Couldn't load all of your dashboard";
const WELCOME_TITLE = 'This is your space.';
const EMPTY_LIST = { page: 1, perPage: 100, totalItems: 0, totalPages: 0, items: [] };

type Fixture = { projectId: string; postId: string; viewId: string; name: string; slug: string };

function records(name: string) {
	return new RegExp(`/api/collections/(demo_)?${name}/records(\\?|$)`);
}

// In dev the SDK calls the backend on another port, so mocks need the CORS header.
function fulfillJson(route: Route, body: unknown, status = 200) {
	return route.fulfill({
		status,
		contentType: 'application/json',
		headers: { 'access-control-allow-origin': '*' },
		body: JSON.stringify(body)
	});
}

// The sidebar reads views too; only the dashboard asks for them by view count.
function isDashboardRead(route: Route, collection: string) {
	const request = route.request();
	if (request.method() !== 'GET') return false;
	if (collection !== 'views') return true;
	return new URL(request.url()).searchParams.get('sort') === '-view_count';
}

async function createFacet(request: APIRequestContext, token: string, name: string, slug: string) {
	const response = await request.post(`${apiBaseURL}/api/collections/views/records`, {
		headers: { Authorization: token },
		data: { name, slug, visibility: 'public', is_active: true }
	});
	expect(response.ok(), await response.text()).toBeTruthy();
	return (await response.json()).id as string;
}

async function deleteFacet(request: APIRequestContext, token: string, id: string) {
	await request.delete(`${apiBaseURL}/api/collections/views/records/${id}`, {
		headers: { Authorization: token }
	});
}

async function mockNoContent(page: Page) {
	for (const name of ['projects', 'experience', 'views']) {
		await page.route(records(name), (route) =>
			route.request().method() === 'GET' ? fulfillJson(route, EMPTY_LIST) : route.continue()
		);
	}
}

function guidance(page: Page) {
	return page.getByRole('region', { name: GUIDANCE_HEADING });
}

function metric(page: Page, label: string) {
	return page.locator('dl.stat-band > div').filter({ has: page.getByText(label, { exact: true }) });
}

// Scoped to main: the sidebar lists recent Facets too, and comes first in the DOM.
function facetLink(page: Page, name: string) {
	return page.getByRole('main').getByRole('link', { name }).first();
}

function createAction(page: Page) {
	return page.getByRole('main').getByRole('link', { name: 'Create a Facet' }).first();
}

async function expectGuidanceFits(page: Page, width: number) {
	const region = guidance(page);
	await expect(region).toBeVisible();
	await expect(region).toContainText(/Any Facet can be shared on its own|Share a Facet on its own/);
	const examples = region.locator('summary');
	if (await examples.count()) {
		expect((await region.boundingBox())!.height).toBeLessThan(240);
		await examples.click();
	}
	await expect(region).toContainText('Digital resume for a recruiter');
	await expect(region).toContainText('Client proposal');
	await expect(region).toContainText('Event or resource page');
	await expect(region).toContainText('Project page');
	await expect(region).toContainText('About page');
	await expect(region).toContainText('Writing page');

	// The admin shell clips overflow, so the page width alone would hide a bad layout.
	const box = await region.boundingBox();
	expect(box).not.toBeNull();
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(width);
	const overflowing = await region.evaluate(
		(element) =>
			[element, ...element.querySelectorAll('*')].filter(
				(node) => node.clientWidth > 0 && node.scrollWidth > node.clientWidth + 1
			).length
	);
	expect(overflowing).toBe(0);
	const pageWidth = await page.evaluate(() => document.documentElement.scrollWidth);
	expect(pageWidth).toBeLessThanOrEqual(width);
}

test.describe('Facet versatility guidance and dashboard resilience', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	let token: string;
	let fixture: Fixture;

	test.beforeEach(async ({ page, request }) => {
		({ token } = await loginAsAdmin(request));
		const suffix = randomUUID().slice(0, 8);
		const name = `Versatility ${suffix}`;
		const slug = `versatility-${suffix}`;
		const content = await createPublicProjectAndPost(request, token);
		const viewId = await createFacet(request, token, name, slug);
		fixture = { ...content, viewId, name, slug };
		await seedAdminSession(page, request);
	});

	test.afterEach(async ({ request }) => {
		await deleteFacet(request, token, fixture.viewId);
		await deletePublicProjectAndPost(request, token, fixture);
	});

	for (const viewport of VIEWPORTS) {
		test(`guidance fits with no Facets at ${viewport.width}px`, async ({ page }) => {
			await page.setViewportSize(viewport);
			await mockNoContent(page);

			await page.goto('/admin/views');
			await expect(
				page.getByRole('heading', { name: "You haven't created any facets yet." })
			).toBeVisible();
			await expectGuidanceFits(page, viewport.width);
			await expect(guidance(page)).toContainText('The experience and skills that fit one role.');

			await page.goto('/admin');
			await expect(page.getByRole('heading', { level: 1, name: WELCOME_TITLE })).toBeVisible();
			await expectGuidanceFits(page, viewport.width);
			await expect(guidance(page).getByRole('link', { name: 'Create a Facet' })).toBeVisible();
		});

		test(`guidance fits alongside Facets at ${viewport.width}px`, async ({ page }) => {
			await page.setViewportSize(viewport);

			await page.goto('/admin/views');
			await expect(facetLink(page, fixture.name)).toBeAttached();
			await expectGuidanceFits(page, viewport.width);

			await page.goto('/admin');
			await expect(facetLink(page, fixture.name)).toBeVisible();
			await expectGuidanceFits(page, viewport.width);
		});
	}

	test('mobile owner work sits above the fold, below the backup warning', async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.route('**/api/system/encryption-status', (route) =>
			fulfillJson(route, { source: 'auto' })
		);

		await page.goto('/admin');

		await expect(page.getByRole('heading', { name: /Backup Reminder/ })).toBeVisible();
		const firstFacet = page
			.getByRole('region', { name: 'Facets', exact: true })
			.getByRole('listitem')
			.first();
		await expect(createAction(page)).toBeInViewport({ ratio: 1 });
		await expect(firstFacet).toBeInViewport({ ratio: 1 });
	});

	test('metrics are a valid two-column description list on mobile', async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto('/admin');

		const groups = page.locator('dl.stat-band > div');
		await expect(groups).toHaveCount(4);
		const shapes = await groups.evaluateAll((elements) =>
			elements.map((element) => [...element.children].map((child) => child.tagName))
		);
		expect(shapes).toEqual(Array(4).fill(['DT', 'DD']));

		const [first, second, third] = await Promise.all(
			[0, 1, 2].map((index) => groups.nth(index).boundingBox())
		);
		expect(second!.y).toBe(first!.y);
		expect(second!.x).toBeGreaterThan(first!.x);
		expect(third!.y).toBeGreaterThan(first!.y);
	});

	test('recent activity follows update times, not record IDs', async ({ page }) => {
		const recent = (id: string, title: string, updated: string) => ({
			id,
			title,
			company: 'Fixture Co',
			updated
		});
		let sort: string | null = null;
		await page.route(records('projects'), (route) => {
			if (!isDashboardRead(route, 'projects')) return route.continue();
			sort = new URL(route.request().url()).searchParams.get('sort');
			return fulfillJson(route, {
				...EMPTY_LIST,
				totalItems: 2,
				items: [
					recent('zzzzzzzzzzzzzzz', 'Oldest project', '2024-01-01 09:00:00.000Z'),
					recent('aaaaaaaaaaaaaaa', 'Newest project', '2026-06-01 09:00:00.000Z')
				]
			});
		});
		await page.route(records('experience'), (route) =>
			isDashboardRead(route, 'experience')
				? fulfillJson(route, {
						...EMPTY_LIST,
						totalItems: 1,
						items: [recent('mmmmmmmmmmmmmmm', 'Middle role', '2025-01-01 09:00:00.000Z')]
					})
				: route.continue()
		);

		await page.goto('/admin');

		const activity = page
			.locator('div.card')
			.filter({ has: page.getByRole('heading', { name: 'Recent Activity' }) })
			.getByRole('listitem');
		await expect(activity).toHaveCount(3);
		await expect(activity.nth(0)).toContainText('Newest project');
		await expect(activity.nth(1)).toContainText('Middle role');
		await expect(activity.nth(2)).toContainText('Oldest project');
		expect(sort).toBe('-updated');
	});

	test('create, edit, search and delete stay available with no category gate', async ({
		page,
		request
	}) => {
		const disposableName = `Disposable ${randomUUID().slice(0, 8)}`;
		const disposableId = await createFacet(
			request,
			token,
			disposableName,
			disposableName.toLowerCase().replace(' ', '-')
		);

		try {
			await page.setViewportSize({ width: 1440, height: 900 });

			await page.goto('/admin');
			await createAction(page).click();
			await expect(page).toHaveURL(/\/admin\/views\/new$/);
			await expect(page.getByRole('heading', { name: 'Create Facet' })).toBeVisible();
			await expect(page.locator('#name')).toBeEditable();
			await expect(page.locator('#slug')).toBeEditable();

			await page.goto('/admin/views');
			await expect(
				page.getByRole('main').getByRole('link', { name: '+ New Facet' })
			).toHaveAttribute('href', '/admin/views/new');

			const search = page.locator('#views-search');
			await search.fill(fixture.slug);
			await expect(page.getByRole('status').filter({ hasText: /^1 of/ })).toBeVisible();
			const row = page.locator('tr', { hasText: fixture.name });
			await expect(row).toBeVisible();

			await search.fill(disposableName);
			const disposableRow = page.locator('tr', { hasText: disposableName });
			await disposableRow.getByRole('button', { name: 'Delete' }).click();
			await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();
			await expect(disposableRow).toHaveCount(0);

			await search.fill(fixture.slug);
			await row.getByRole('link', { name: 'Edit' }).click();
			await expect(page).toHaveURL(new RegExp(`/admin/views/${fixture.viewId}$`));
		} finally {
			await deleteFacet(request, token, disposableId);
		}
	});

	for (const failing of ['projects', 'views']) {
		test(`a ${failing} 503 shows Retry, keeps known data and recovers`, async ({ page }) => {
			let unavailable = true;
			let failedRequests = 0;
			await page.route(records(failing), (route) => {
				if (!unavailable || !isDashboardRead(route, failing)) return route.continue();
				failedRequests += 1;
				return fulfillJson(route, { status: 503, message: 'Unavailable' }, 503);
			});

			await page.goto('/admin');

			const alert = page.getByRole('alert').filter({ hasText: ERROR_TITLE });
			await expect(alert).toBeVisible();
			await expect(page.getByRole('heading', { name: WELCOME_TITLE })).toHaveCount(0);

			const unknownMetric = metric(page, failing === 'projects' ? 'Projects' : 'Total Visitors');
			await expect(unknownMetric.getByRole('definition')).toContainText('Not available');
			await expect(unknownMetric.getByRole('definition')).not.toContainText('0');

			// Exact, so this only matches once the button has left "Retrying...".
			const retry = alert.getByRole('button', { name: 'Retry', exact: true });

			// Still failing: what loaded the first time has to survive this retry.
			const failedAttempts = failedRequests;
			await retry.click();
			await expect.poll(() => failedRequests).toBeGreaterThan(failedAttempts);
			await expect(retry).toBeVisible();
			if (failing === 'projects') {
				await expect(facetLink(page, fixture.name)).toBeVisible();
			} else {
				await expect(metric(page, 'Projects').getByRole('definition')).not.toContainText(
					'Not available'
				);
			}

			unavailable = false;
			await retry.click();

			await expect(alert).toHaveCount(0);
			await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeFocused();
			await expect(facetLink(page, fixture.name)).toBeVisible();
			await expect(metric(page, 'Projects').getByRole('definition')).not.toContainText(
				'Not available'
			);
			await expect(metric(page, 'Total Visitors').getByRole('definition')).not.toContainText(
				'Not available'
			);
		});
	}

	test('a failed site settings read is an error, not zero visitors', async ({ page }) => {
		await page.route('**/api/site-settings', (route) =>
			route.request().method() === 'GET'
				? route.fulfill({ status: 503, body: 'Unavailable' })
				: route.continue()
		);

		await page.goto('/admin');

		await expect(page.getByRole('alert').filter({ hasText: ERROR_TITLE })).toBeVisible();
		await expect(metric(page, 'Total Visitors').getByRole('definition')).toContainText(
			'Not available'
		);
	});

	test('layout and dashboard reads do not cancel each other', async ({ page }) => {
		const cancelled: string[] = [];
		page.on('requestfailed', (request) => {
			const url = request.url();
			// perPage=4 is the sidebar, which debounces and cancels its own reloads.
			if (records('(projects|experience|views)').test(url) && !url.includes('perPage=4')) {
				cancelled.push(url);
			}
		});
		// Runs after the session seed, so the layout's wizard check reads views as well.
		await page.addInitScript(() => localStorage.removeItem('facet_setup_wizard'));

		await page.goto('/admin');

		await expect(facetLink(page, fixture.name)).toBeVisible();
		await expect(page.getByRole('alert').filter({ hasText: ERROR_TITLE })).toHaveCount(0);
		expect(cancelled).toEqual([]);
	});

	test('the Facet list shows a recoverable failure instead of an empty collection', async ({
		page
	}) => {
		let unavailable = true;
		await page.route(records('views'), (route) =>
			unavailable && route.request().method() === 'GET'
				? fulfillJson(route, { message: 'Unavailable' }, 503)
				: route.continue()
		);
		await page.goto('/admin/views');
		const alert = page.getByRole('alert').filter({ hasText: 'Failed to load facets' });
		await expect(alert).toBeVisible();
		await expect(
			page.getByRole('heading', { name: "You haven't created any facets yet." })
		).toHaveCount(0);
		await expect(page.getByRole('heading', { name: 'No matching facets' })).toHaveCount(0);
		unavailable = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(facetLink(page, fixture.name)).toBeVisible();
	});

	test('a stalled dashboard request times out into a retryable error', async ({ page }) => {
		await page.clock.install();
		let stalled = true;
		const held: Route[] = [];
		await page.route(records('projects'), (route) => {
			if (!stalled || !isDashboardRead(route, 'projects')) return route.continue();
			held.push(route);
		});
		const started = page.waitForRequest(records('projects'));
		await page.goto('/admin');
		await started;
		await page.clock.fastForward(21_000);
		const alert = page.getByRole('alert').filter({ hasText: ERROR_TITLE });
		await expect(alert).toBeVisible();
		stalled = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(facetLink(page, fixture.name)).toBeVisible();
		await Promise.all(held.map((route) => route.continue().catch(() => undefined)));
	});

	test('leaving during a load cancels it without an error', async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		const dashboardErrors: string[] = [];
		page.on('console', (message) => {
			if (message.type() === 'error' && message.text().includes('dashboard')) {
				dashboardErrors.push(message.text());
			}
		});
		const held: Route[] = [];
		await page.route(records('projects'), (route) => {
			if (!isDashboardRead(route, 'projects')) return route.continue();
			held.push(route);
		});

		const projectsRequested = page.waitForRequest(records('projects'));
		await page.goto('/admin');
		await projectsRequested;

		await page.locator('a[href="/admin/views"]').first().click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		await expect(page.getByRole('heading', { level: 1, name: 'Facets' })).toBeVisible();

		await Promise.all(held.map((route) => route.continue().catch(() => undefined)));
		await expect(page.getByRole('alert').filter({ hasText: ERROR_TITLE })).toHaveCount(0);
		expect(dashboardErrors).toEqual([]);
	});
});

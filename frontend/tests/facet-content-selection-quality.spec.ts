import {
	test,
	expect,
	type APIRequestContext,
	type Browser,
	type Locator,
	type Page
} from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { seedAdminSession } from './helpers';

// A Facet section shows its chosen items or all public content. These tests
// compare what the owner previews with what an anonymous visitor actually gets.
const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';

type SavedSection = { section: string; items?: string[]; itemSelection?: string };

test.describe('Facet content selection', () => {
	const suffix = randomUUID().slice(0, 8);
	const created: { collection: string; id: string }[] = [];
	let token = '';

	const title = (name: string) => `selection ${name} ${suffix}`;

	async function create(
		request: APIRequestContext,
		collection: string,
		data: Record<string, unknown>
	) {
		const res = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
			headers: { Authorization: token },
			data
		});
		expect(res.ok(), await res.text()).toBe(true);
		const { id } = (await res.json()) as { id: string };
		created.push({ collection, id });
		return id;
	}

	function createProject(
		request: APIRequestContext,
		name: string,
		fields: Record<string, unknown> = {}
	) {
		return create(request, 'projects', {
			title: title(name),
			slug: `selection-${name}-${suffix}`,
			summary: 'Content selection fixture.',
			description: 'Content selection fixture.',
			visibility: 'public',
			is_draft: false,
			...fields
		});
	}

	async function createFacet(
		request: APIRequestContext,
		name: string,
		section: Record<string, unknown>
	) {
		const slug = `selection-facet-${name}-${suffix}`;
		const id = await create(request, 'views', {
			name: `Selection ${name} ${suffix}`,
			slug,
			visibility: 'public',
			is_active: true,
			sections: [{ section: 'projects', enabled: true, layout: 'grid-2', ...section }]
		});
		return { id, slug };
	}

	async function getRecord<T>(request: APIRequestContext, collection: string, id: string) {
		const res = await request.get(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
			headers: { Authorization: token }
		});
		expect(res.ok(), await res.text()).toBe(true);
		return (await res.json()) as T;
	}

	async function savedSection(request: APIRequestContext, viewId: string) {
		const view = await getRecord<{ sections: SavedSection[] }>(request, 'views', viewId);
		return view.sections.find((section) => section.section === 'projects');
	}

	// The given titles found in the element's rendered text, in page order.
	async function shownTitles(scope: Locator, titles: string[]) {
		const text = (await scope.innerText()).toLowerCase();
		return titles
			.filter((name) => text.includes(name))
			.sort((a, b) => text.indexOf(a) - text.indexOf(b));
	}

	async function publishedTitles(browser: Browser, slug: string, titles: string[]) {
		const context = await browser.newContext();
		const visitor = await context.newPage();
		await visitor.goto(`${BASE_URL}/${slug}`);
		const shown = await shownTitles(visitor.locator('body'), titles);
		await context.close();
		return shown;
	}

	const preview = (page: Page) => page.locator('.preview-container');
	const contentControl = (page: Page) => page.getByRole('group', { name: 'Content for Projects' });

	async function save(page: Page) {
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Facet updated successfully')).toBeVisible();
	}

	test.beforeEach(async ({ page, request }) => {
		({ token } = await seedAdminSession(page, request));
	});

	test.afterEach(async ({ request }) => {
		for (const { collection, id } of created.splice(0).reverse()) {
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
		}
	});

	test('all public preview matches the published page', async ({ page, request, browser }) => {
		const facet = await createFacet(request, 'all-public', {
			items: [],
			itemSelection: 'all_public'
		});
		await createProject(request, 'public');
		await createProject(request, 'unlisted', { visibility: 'unlisted' });
		await createProject(request, 'private', {
			visibility: 'private',
			view_visibility: { [facet.id]: true }
		});
		await createProject(request, 'draft', { is_draft: true });
		const titles = ['public', 'unlisted', 'private', 'draft'].map(title);

		await page.goto(`${BASE_URL}/admin/views/${facet.id}`);
		await expect.poll(() => shownTitles(preview(page), titles)).toEqual([title('public')]);
		expect(await publishedTitles(browser, facet.slug, titles)).toEqual([title('public')]);
	});

	test('chosen items preview matches the published page', async ({ page, request, browser }) => {
		const chosen = await createProject(request, 'chosen');
		const draft = await createProject(request, 'chosen-draft', { is_draft: true });
		const optedIn = await createProject(request, 'opted-in', { visibility: 'private' });
		const facet = await createFacet(request, 'chosen', {
			items: [optedIn, draft, chosen],
			itemSelection: 'selected'
		});
		const optIn = await request.patch(`${apiBaseURL}/api/collections/projects/records/${optedIn}`, {
			headers: { Authorization: token },
			data: { view_visibility: { [facet.id]: true } }
		});
		expect(optIn.ok(), await optIn.text()).toBe(true);
		const titles = ['chosen', 'chosen-draft', 'opted-in'].map(title);
		const expected = [title('opted-in'), title('chosen')];

		await page.goto(`${BASE_URL}/admin/views/${facet.id}`);
		await expect.poll(() => shownTitles(preview(page), titles)).toEqual(expected);
		expect(await publishedTitles(browser, facet.slug, titles)).toEqual(expected);
	});

	test('all public shows a record created later', async ({ page, request, browser }) => {
		const facet = await createFacet(request, 'later', { itemSelection: 'all_public' });
		const titles = [title('later')];
		expect(await publishedTitles(browser, facet.slug, titles)).toEqual([]);

		await createProject(request, 'later');

		expect(await publishedTitles(browser, facet.slug, titles)).toEqual(titles);
		await page.goto(`${BASE_URL}/admin/views/${facet.id}`);
		await expect.poll(() => shownTitles(preview(page), titles)).toEqual(titles);
	});

	test('switching modes keeps picks without publishing them', async ({
		page,
		request,
		browser
	}) => {
		const pick = await createProject(request, 'pick', { visibility: 'private' });
		const facet = await createFacet(request, 'switch', {
			items: [pick],
			itemSelection: 'selected'
		});

		await page.goto(`${BASE_URL}/admin/views/${facet.id}`);
		await contentControl(page)
			.getByRole('radio', { name: /All public content/ })
			.check();
		await save(page);

		expect(await savedSection(request, facet.id)).toMatchObject({
			items: [pick],
			itemSelection: 'all_public'
		});
		const stored = await getRecord<{ view_visibility?: Record<string, boolean> | null }>(
			request,
			'projects',
			pick
		);
		expect(stored.view_visibility?.[facet.id]).toBeFalsy();
		expect(await publishedTitles(browser, facet.slug, [title('pick')])).toEqual([]);

		await page.reload();
		await contentControl(page).getByRole('radio', { name: 'Choose items' }).check();
		await page.getByRole('button', { name: 'Expand Projects section' }).click();
		await expect(page.getByRole('checkbox', { name: title('pick') })).toBeChecked();
	});

	test('a legacy empty section stays blank after the owner saves', async ({
		page,
		request,
		browser
	}) => {
		await createProject(request, 'legacy');
		const facet = await createFacet(request, 'legacy', { items: [] });
		const titles = [title('legacy')];

		await page.goto(`${BASE_URL}/admin/views/${facet.id}`);
		await expect(contentControl(page).getByRole('radio', { name: 'Choose items' })).toBeChecked();
		await expect(contentControl(page)).toContainText('Nothing appears');
		await save(page);

		expect(await savedSection(request, facet.id)).toMatchObject({
			items: [],
			itemSelection: 'selected'
		});
		expect(await shownTitles(preview(page), titles)).toEqual([]);
		expect(await publishedTitles(browser, facet.slug, titles)).toEqual([]);
	});

	test('a new Facet starts its sections on all public', async ({ page, request }) => {
		await page.goto(`${BASE_URL}/admin/views/new`);
		await page.getByLabel('Name *', { exact: true }).fill(`Selection new ${suffix}`);
		await page.getByLabel('URL Slug *', { exact: true }).fill(`selection-new-${suffix}`);
		const response = page.waitForResponse(
			(res) =>
				res.url().includes('/api/collections/views/records') && res.request().method() === 'POST'
		);
		await page.getByRole('button', { name: 'Create', exact: true }).click();
		const { id } = (await (await response).json()) as { id: string };
		created.push({ collection: 'views', id });

		expect(await savedSection(request, id)).toMatchObject({ itemSelection: 'all_public' });
	});

	test('contacts and testimonials preview matches the published page', async ({
		page,
		request,
		browser
	}) => {
		const author = title('author');
		const contact = title('contact');
		await create(request, 'testimonials', {
			content: 'Great to work with',
			author_name: author,
			relationship: 'colleague',
			verification_method: 'email',
			verification_identifier: `${suffix}@example.com`,
			status: 'approved',
			visibility: 'public'
		});
		await create(request, 'contact_methods', {
			type: 'website',
			value: 'https://example.com',
			label: contact,
			protection_level: 'none'
		});
		const slug = `selection-facet-people-${suffix}`;
		const id = await create(request, 'views', {
			name: `Selection people ${suffix}`,
			slug,
			visibility: 'public',
			is_active: true,
			sections: [
				{ section: 'testimonials', enabled: true, layout: 'wall', itemSelection: 'all_public' },
				{ section: 'contacts', enabled: true, layout: 'vertical', itemSelection: 'all_public' }
			]
		});
		const titles = [author, contact];

		await page.goto(`${BASE_URL}/admin/views/${id}`);
		await expect.poll(() => shownTitles(preview(page), titles)).toEqual(titles);
		await expect(preview(page)).not.toContainText('No sections enabled');
		expect(await publishedTitles(browser, slug, titles)).toEqual(titles);
	});

	test('empty contacts and testimonials show the empty preview', async ({ page, request }) => {
		const id = await create(request, 'views', {
			name: `Selection empty ${suffix}`,
			slug: `selection-facet-empty-${suffix}`,
			visibility: 'public',
			is_active: true,
			sections: [
				{ section: 'testimonials', enabled: true, layout: 'wall', items: [] },
				{ section: 'contacts', enabled: true, layout: 'vertical', items: [] }
			]
		});

		await page.goto(`${BASE_URL}/admin/views/${id}`);
		await expect(preview(page)).toContainText('No sections enabled');
	});

	async function startNewFacet(page: Page, name: string) {
		await page.goto(`${BASE_URL}/admin/views/new`);
		await page.getByLabel('Name *', { exact: true }).fill(`Selection ${name} ${suffix}`);
		await page.getByLabel('URL Slug *', { exact: true }).fill(`selection-${name}-${suffix}`);
		await contentControl(page).getByRole('radio', { name: 'Choose items' }).check();
		await page.getByRole('button', { name: 'Expand Projects section' }).click();
	}

	async function submitNewFacet(page: Page) {
		const response = page.waitForResponse(
			(res) =>
				res.url().includes('/api/collections/views/records') && res.request().method() === 'POST'
		);
		await page.getByRole('button', { name: 'Create', exact: true }).click();
		const { id } = (await (await response).json()) as { id: string };
		created.push({ collection: 'views', id });
		return id;
	}

	type Overridable = { view_visibility?: Record<string, boolean> | null };
	const grantURL = '**/api/view/*/private-items';

	test('a new Facet shows the private items chosen for it', async ({ page, request, browser }) => {
		const pick = await createProject(request, 'new-pick', { visibility: 'private' });
		await startNewFacet(page, 'new-pick');
		await page.getByRole('checkbox', { name: title('new-pick') }).check();
		const id = await submitNewFacet(page);
		await expect(page).toHaveURL(/\/admin\/views$/);

		const stored = await getRecord<Overridable>(request, 'projects', pick);
		expect(stored.view_visibility?.[id]).toBe(true);
		const titles = [title('new-pick')];
		expect(await publishedTitles(browser, `selection-new-pick-${suffix}`, titles)).toEqual(titles);
	});

	test('a failed private grant keeps the Facet and retries on Save', async ({ page, request }) => {
		const pick = await createProject(request, 'retry-pick', { visibility: 'private' });
		await page.route(grantURL, (route) =>
			route.fulfill({ status: 500, json: { message: 'grant failed' } })
		);
		let creates = 0;
		page.on('request', (req) => {
			if (req.method() === 'POST' && req.url().includes('/api/collections/views/records')) {
				creates += 1;
			}
		});

		await startNewFacet(page, 'retry-pick');
		await page.getByRole('checkbox', { name: title('retry-pick') }).check();
		const id = await submitNewFacet(page);

		await expect(page).toHaveURL(new RegExp(`/admin/views/${id}$`));
		await expect(page.getByText('Your Facet is saved')).toBeVisible();
		const before = await getRecord<Overridable>(request, 'projects', pick);
		expect(before.view_visibility?.[id]).toBeFalsy();

		await page.unroute(grantURL);
		await save(page);
		const after = await getRecord<Overridable>(request, 'projects', pick);
		expect(after.view_visibility?.[id]).toBe(true);
		expect(creates).toBe(1);
	});

	test('an unconfirmed grant after Save says the Facet saved', async ({ page, request }) => {
		const pick = await createProject(request, 'edit-pick', { visibility: 'private' });
		const facet = await createFacet(request, 'edit-pick', {
			items: [pick],
			itemSelection: 'selected'
		});
		await page.route(grantURL, (route) =>
			route.fulfill({ status: 500, json: { message: 'grant failed' } })
		);

		await page.goto(`${BASE_URL}/admin/views/${facet.id}`);
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Your Facet is saved')).toBeVisible();
		await expect(page.getByText('Failed to save facet')).toHaveCount(0);

		await page.unroute(grantURL);
		await save(page);
		const stored = await getRecord<Overridable>(request, 'projects', pick);
		expect(stored.view_visibility?.[facet.id]).toBe(true);
	});

	test('a private post opens from the Facet that shows it', async ({ request, browser }) => {
		const slug = `selection-private-post-${suffix}`;
		const post = await create(request, 'posts', {
			title: title('private-post'),
			slug,
			content: 'Private post fixture.',
			visibility: 'private',
			is_draft: false
		});
		const id = await create(request, 'views', {
			name: `Selection detail ${suffix}`,
			slug: `selection-detail-${suffix}`,
			visibility: 'public',
			is_active: true,
			sections: [{ section: 'posts', enabled: true, layout: 'grid-3', items: [post] }]
		});
		const grant = await request.post(`${apiBaseURL}/api/view/${id}/private-items`, {
			headers: { Authorization: token }
		});
		expect(grant.ok(), await grant.text()).toBe(true);

		const context = await browser.newContext();
		const visitor = await context.newPage();
		await visitor.goto(`${BASE_URL}/selection-detail-${suffix}`);
		await visitor.locator(`a[href*="/posts/${slug}"]`).first().click();
		await expect(visitor.getByText('Private post fixture.')).toBeVisible();
		const bare = await visitor.goto(`${BASE_URL}/posts/${slug}`);
		expect(bare?.status()).toBe(404);
		await context.close();
	});

	test('all public keeps category and Customize controls', async ({ page, request }) => {
		await createProject(request, 'shown');
		await createProject(request, 'hidden', { visibility: 'private' });
		const category = title('category');
		await create(request, 'skills', { name: title('skill'), category, visibility: 'public' });
		const id = await create(request, 'views', {
			name: `Selection tools ${suffix}`,
			slug: `selection-facet-tools-${suffix}`,
			visibility: 'public',
			is_active: true,
			sections: [
				{ section: 'projects', enabled: true, layout: 'grid-2', itemSelection: 'all_public' },
				{ section: 'skills', enabled: true, layout: 'grouped', itemSelection: 'all_public' }
			]
		});

		await page.goto(`${BASE_URL}/admin/views/${id}`);
		await page.getByRole('button', { name: 'Expand Projects section' }).click();
		const items = page.getByRole('list', { name: 'Content for Projects' });
		const row = items.getByRole('listitem').filter({ hasText: title('shown') });
		await expect(row.getByRole('button', { name: 'Customize' })).toBeVisible();
		await expect(items).not.toContainText(title('hidden'));

		await page.getByRole('button', { name: 'Expand Skills section' }).click();
		await expect(page.getByRole('button', { name: `Toggle ${category} category` })).toBeVisible();
		await page.getByRole('button', { name: `Expand ${category}`, exact: true }).click();
		const skillList = page.getByRole('listitem', { name: category, exact: true }).getByRole('list');
		await expect(skillList).toBeVisible();
		await expect(skillList.getByRole('checkbox')).toHaveCount(0);
		await expect(skillList.getByRole('button')).toHaveCount(0);
	});
});

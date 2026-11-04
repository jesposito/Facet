import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

type Section = Record<string, unknown> & { section: string };
type Facet = { id: string; name: string; slug: string; sections?: Section[] };

const failure = { status: 500, json: { message: 'Fixture failure' } };

const nameField = (page: Page) => page.locator('#name');
const saveButton = (page: Page) => page.getByRole('button', { name: 'Save', exact: true });
const createButton = (page: Page) => page.getByRole('button', { name: 'Create', exact: true });
const backLink = (page: Page) => page.getByRole('link', { name: 'Back to facets' }).first();

function failList(page: Page, collection: string, response: object = failure) {
	return page.route(`**/api/collections/${collection}/records?*`, (route) =>
		route.request().method() === 'GET' ? route.fulfill(response) : route.continue()
	);
}

function countWrites(page: Page, viewId: string) {
	const writes = { saves: 0, grants: 0 };
	page.on('request', (request) => {
		const path = new URL(request.url()).pathname;
		if (request.method() === 'PATCH' && path.endsWith(`/views/records/${viewId}`)) writes.saves++;
		if (path === `/api/view/${viewId}/private-items`) writes.grants++;
	});
	return writes;
}

// Answers every prompt the same way and counts them.
function answerPrompts(page: Page, accept: boolean) {
	const prompts = { count: 0, accept };
	page.on('dialog', async (dialog) => {
		prompts.count++;
		await (prompts.accept ? dialog.accept() : dialog.dismiss());
	});
	return prompts;
}

async function addLinks(page: Page, links: Record<string, string>) {
	await page.evaluate((entries) => {
		entries.forEach(([label, href], index) => {
			const link = document.createElement('a');
			link.style.cssText = `position:fixed;bottom:${20 + index * 60}px;right:20px;z-index:99999;background:white;padding:16px`;
			link.href = href;
			link.textContent = label;
			document.body.append(link);
		});
	}, Object.entries(links));
}

test.describe('Facet editor recovery', () => {
	let created: { collection: string; id: string }[] = [];
	let token: string;

	test.beforeEach(async ({ page, request }) => {
		({ token } = await loginAsAdmin(request));
		await seedAdminSession(page, request);
	});

	test.afterEach(async ({ request }) => {
		for (const { collection, id } of [...created].reverse()) {
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
		}
		created = [];
	});

	async function createRecord<T extends { id: string }>(
		request: APIRequestContext,
		collection: string,
		data: Record<string, unknown>
	) {
		const response = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
			headers: { Authorization: token },
			data
		});
		expect(response.ok(), await response.text()).toBe(true);
		const record = (await response.json()) as T;
		created.push({ collection, id: record.id });
		return record;
	}

	function createFacet(request: APIRequestContext, label: string, sections: Section[] = []) {
		const suffix = randomUUID().slice(0, 8);
		return createRecord<Facet>(request, 'views', {
			name: `Recovery ${label} ${suffix}`,
			slug: `recovery-${label}-${suffix}`,
			visibility: 'public',
			is_active: true,
			sections
		});
	}

	async function readFacet(request: APIRequestContext, id: string) {
		const response = await request.get(`${apiBaseURL}/api/collections/views/records/${id}`, {
			headers: { Authorization: token }
		});
		expect(response.ok(), await response.text()).toBe(true);
		return (await response.json()) as Facet;
	}

	// Also queues whatever a test's create landed for cleanup.
	async function facetsWithSlug(request: APIRequestContext, slug: string) {
		const response = await request.get(`${apiBaseURL}/api/collections/views/records`, {
			headers: { Authorization: token },
			params: { filter: `slug = "${slug}"` }
		});
		expect(response.ok(), await response.text()).toBe(true);
		const { items } = (await response.json()) as { items: Facet[] };
		for (const { id } of items) {
			if (!created.some((record) => record.id === id)) created.push({ collection: 'views', id });
		}
		return items;
	}

	async function openEditor(page: Page, facet: Facet) {
		await page.goto(`/admin/views/${facet.id}`);
		await expect(nameField(page)).toHaveValue(facet.name);
		await expect(saveButton(page)).toBeEnabled();
	}

	test('failed custom content pauses Save, and Retry keeps the draft and saved config', async ({
		page,
		request
	}) => {
		const custom = await createRecord(request, 'custom_content', {
			title: 'Recovery custom',
			content: 'Facet recovery fixture'
		});
		const saved = {
			section: `custom:${custom.id}`,
			enabled: true,
			items: [custom.id],
			layout: 'default',
			width: 'half',
			note: 'kept'
		};
		const facet = await createFacet(request, 'custom', [saved]);
		const writes = countWrites(page, facet.id);
		await failList(page, 'custom_content');

		await page.goto(`/admin/views/${facet.id}`);
		const alert = page.getByRole('alert').filter({ hasText: 'the preview are hidden' });
		await expect(alert).toBeVisible();
		await expect(saveButton(page)).toBeDisabled();
		await expect(page.getByText(/\d+ available|none selected/)).toHaveCount(0);
		await nameField(page).fill('Typed during the outage');
		await nameField(page).press('Enter');
		await expect(alert).toBeVisible();
		expect(writes).toEqual({ saves: 0, grants: 0 });

		await page.unroute('**/api/collections/custom_content/records?*');
		await alert.getByRole('button').click();
		await expect(alert).toHaveCount(0);
		await expect(nameField(page)).toHaveValue('Typed during the outage');
		await saveButton(page).click();
		await expect(page.getByText('Facet updated successfully')).toBeVisible();
		const stored = await readFacet(request, facet.id);
		expect(stored.name).toBe('Typed during the outage');
		expect(stored.sections?.find((s) => s.section === saved.section)).toMatchObject(saved);
	});

	for (const [label, response] of [
		['fails', failure],
		['comes back malformed', { status: 200, json: {} }]
	] as const) {
		test(`a selected private item can't be granted while its list ${label}`, async ({
			page,
			request
		}) => {
			const suffix = randomUUID().slice(0, 8);
			const item = await createRecord(request, 'projects', {
				title: `Recovery private ${suffix}`,
				slug: `recovery-private-${suffix}`,
				summary: 'Facet recovery fixture.',
				description: 'Facet recovery fixture.',
				visibility: 'private',
				is_draft: false
			});
			const facet = await createFacet(request, 'private', [
				{ section: 'projects', enabled: true, items: [item.id] }
			]);
			const writes = countWrites(page, facet.id);
			await failList(page, 'projects', response);

			await page.goto(`/admin/views/${facet.id}`);
			const alert = page.getByRole('alert').filter({ hasText: 'the preview are hidden' });
			await expect(alert).toBeVisible();
			await expect(saveButton(page)).toBeDisabled();
			expect(writes).toEqual({ saves: 0, grants: 0 });

			await page.unroute('**/api/collections/projects/records?*');
			await alert.getByRole('button').click();
			await expect(saveButton(page)).toBeEnabled();
			await expect(page.getByText(`Recovery private ${suffix}`).first()).toBeVisible();
		});
	}

	test('a failed Facet read offers Retry, never a default form', async ({ page, request }) => {
		const facet = await createFacet(request, 'lookup');
		const route = `**/api/collections/views/records/${facet.id}*`;
		await page.route(route, (r) => r.fulfill(failure));

		await page.goto(`/admin/views/${facet.id}`);
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load this Facet" });
		await expect(alert).toBeVisible();
		await expect(nameField(page)).toHaveCount(0);
		await expect(saveButton(page)).toHaveCount(0);

		await page.unroute(route);
		await alert.getByRole('button').click();
		await expect(nameField(page)).toHaveValue(facet.name);
		await expect(saveButton(page)).toBeEnabled();
	});

	test('a slow read of a Facet left behind cannot replace the open one', async ({
		page,
		request
	}) => {
		const [first, slow, last] = await Promise.all(
			['first', 'slow', 'last'].map((label) => createFacet(request, label))
		);
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route(`**/api/collections/views/records/${slow.id}*`, async (route) => {
			await held;
			await route.continue().catch(() => {});
		});

		await openEditor(page, first);
		await addLinks(page, { slow: `/admin/views/${slow.id}`, last: `/admin/views/${last.id}` });
		await page.getByRole('link', { name: 'slow', exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`${slow.id}$`));
		await page.getByRole('link', { name: 'last', exact: true }).click();
		await expect(nameField(page)).toHaveValue(last.name);
		await expect(saveButton(page)).toBeEnabled();
		release();

		await nameField(page).fill('Edited the open Facet');
		await saveButton(page).click();
		await expect(page.getByText('Facet updated successfully')).toBeVisible();
		await expect(nameField(page)).toHaveValue('Edited the open Facet');
		expect((await readFacet(request, last.id)).name).toBe('Edited the open Facet');
		expect((await readFacet(request, slow.id)).name).toBe(slow.name);
	});

	test('leaving an edited Facet in the app asks once and Cancel keeps the draft', async ({
		page,
		request
	}) => {
		const facet = await createFacet(request, 'leave');
		await openEditor(page, facet);
		await nameField(page).fill('Unsaved name');
		const prompts = answerPrompts(page, false);

		await backLink(page).click();
		await expect.poll(() => prompts.count).toBe(1);
		await expect(page).toHaveURL(new RegExp(`${facet.id}$`));
		await expect(nameField(page)).toHaveValue('Unsaved name');

		prompts.accept = true;
		await backLink(page).click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		expect(prompts.count).toBe(2);
	});

	test('closing an edited Facet uses one browser prompt', async ({ page, request }) => {
		const facet = await createFacet(request, 'unload');
		await openEditor(page, facet);
		await nameField(page).click();
		await nameField(page).pressSequentially(' edited');
		const prompts = answerPrompts(page, false);

		await page.close({ runBeforeUnload: true });
		await expect.poll(() => prompts.count).toBe(1);
		await expect(nameField(page)).toHaveValue(`${facet.name} edited`);

		prompts.accept = true;
		const closed = page.waitForEvent('close');
		await page.close({ runBeforeUnload: true });
		await closed;
		expect(prompts.count).toBe(2);
	});

	test('opening panels is not an edit, and a saved edit leaves freely', async ({
		page,
		request
	}) => {
		const facet = await createFacet(request, 'clean');
		await openEditor(page, facet);
		const prompts = answerPrompts(page, true);
		await page.getByRole('button', { name: 'Hide Preview' }).click();
		await backLink(page).click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		expect(prompts.count).toBe(0);

		await openEditor(page, facet);
		await nameField(page).fill('Saved name');
		await saveButton(page).click();
		await expect(page.getByText('Facet updated successfully')).toBeVisible();
		await backLink(page).click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		expect(prompts.count).toBe(0);
	});

	test('a failed save keeps the guard', async ({ page, request }) => {
		const facet = await createFacet(request, 'failed-save');
		await openEditor(page, facet);
		await page.route(`**/api/collections/views/records/${facet.id}*`, (route) =>
			route.request().method() === 'PATCH' ? route.fulfill(failure) : route.continue()
		);
		await nameField(page).fill('Not saved');
		await saveButton(page).click();
		await expect(page.getByText('Failed to save facet')).toBeVisible();

		const prompts = answerPrompts(page, false);
		await backLink(page).click();
		await expect.poll(() => prompts.count).toBe(1);
		await expect(nameField(page)).toHaveValue('Not saved');
	});

	test('a new Facet keeps its draft through a failed read and leaves only on Discard', async ({
		page
	}) => {
		await failList(page, 'custom_content');
		await page.goto('/admin/views/new');
		const alert = page.getByRole('alert').filter({ hasText: 'the preview are hidden' });
		await expect(alert).toBeVisible();
		await expect(createButton(page)).toBeDisabled();
		await nameField(page).fill('Draft Facet');

		await page.unroute('**/api/collections/custom_content/records?*');
		await alert.getByRole('button').click();
		await expect(createButton(page)).toBeEnabled();
		await expect(nameField(page)).toHaveValue('Draft Facet');

		const prompts = answerPrompts(page, false);
		await backLink(page).click();
		await expect.poll(() => prompts.count).toBe(1);
		await expect(nameField(page)).toHaveValue('Draft Facet');
		prompts.accept = true;
		await backLink(page).click();
		await expect(page).toHaveURL(/\/admin\/views$/);
	});

	async function loseCreateResponse(page: Page) {
		const posts = { count: 0 };
		await page.route('**/api/collections/views/records', async (route) => {
			if (route.request().method() !== 'POST') return route.continue();
			posts.count++;
			await route.fetch();
			await route.abort();
		});
		return posts;
	}

	async function fillNewFacet(page: Page, slug: string) {
		await page.goto('/admin/views/new');
		await expect(createButton(page)).toBeEnabled();
		await nameField(page).fill(`Recovery ${slug}`);
		await page.locator('#slug').fill(slug);
	}

	test('a lost create response is found by ID and leaves without a prompt', async ({
		page,
		request
	}) => {
		const slug = `recovery-lost-${randomUUID().slice(0, 8)}`;
		const posts = await loseCreateResponse(page);
		await fillNewFacet(page, slug);
		const prompts = answerPrompts(page, true);

		await createButton(page).click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		expect(posts.count).toBe(1);
		expect(prompts.count).toBe(0);
		expect(await facetsWithSlug(request, slug)).toHaveLength(1);
	});

	test('an unreadable landed create asks for another press and makes no second Facet', async ({
		page,
		request
	}) => {
		const slug = `recovery-unknown-${randomUUID().slice(0, 8)}`;
		const posts = await loseCreateResponse(page);
		const lookup = '**/api/collections/views/records/*';
		await page.route(lookup, (route) => route.fulfill(failure));
		await fillNewFacet(page, slug);

		await createButton(page).click();
		const alert = page.getByRole('alert').filter({ hasText: "couldn't confirm" });
		await expect(alert).toBeVisible();
		await expect(nameField(page)).toHaveValue(`Recovery ${slug}`);

		await page.unroute(lookup);
		await createButton(page).click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		expect(posts.count).toBe(1);
		expect(await facetsWithSlug(request, slug)).toHaveLength(1);
	});

	test("another Facet's slug stays a failed create with the guard on", async ({
		page,
		request
	}) => {
		const other = await createFacet(request, 'taken');
		await fillNewFacet(page, other.slug);
		const rejected = page.waitForResponse(
			(r) => r.request().method() === 'POST' && r.url().endsWith('/api/collections/views/records')
		);
		await createButton(page).click();
		expect((await rejected).status()).toBe(400);
		const prompts = answerPrompts(page, false);
		await backLink(page).click();
		await expect.poll(() => prompts.count).toBe(1);
		await expect(page).toHaveURL(/\/admin\/views\/new$/);
		expect(await facetsWithSlug(request, other.slug)).toHaveLength(1);
	});
});

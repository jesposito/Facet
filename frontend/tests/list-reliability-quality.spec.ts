import { randomUUID } from 'node:crypto';
import { expect, test, type APIRequestContext, type Page, type Route } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

const awardsApi = /\/api\/collections\/awards\/records(?:\/([^/?]+))?(?:\?|$)/;
const customApi = /\/api\/collections\/custom_content\/records(?:\/([^/?]+))?(?:\?|$)/;
const coursesApi = /\/api\/collections\/courses\/records(?:\?|$)/;
const modulesApi = /\/api\/collections\/modules\/records(?:\?|$)/;
const lessonsApi = /\/api\/collections\/lessons\/records(?:\?|$)/;

function paged(items: unknown[], perPage = 500) {
	return { page: 1, perPage, totalItems: items.length, totalPages: 1, items };
}

function courseRecord(id: string, title: string) {
	return {
		id,
		title,
		slug: id,
		access_tier: 'free',
		visibility: 'private',
		is_draft: true,
		sort_order: 0
	};
}

async function mockCourses(
	page: Page,
	courses: unknown[],
	curriculum: (filter: string, route: Route) => unknown
) {
	await page.route(coursesApi, (route) => route.fulfill({ json: paged(courses) }));
	await page.route(lessonsApi, (route) => route.fulfill({ json: paged([]) }));
	await page.route(modulesApi, (route) => {
		const filter = new URL(route.request().url()).searchParams.get('filter') ?? '';
		return filter ? curriculum(filter, route) : route.fulfill({ json: paged([]) });
	});
}

function bulkBar(page: Page) {
	return page.getByRole('region', { name: 'Bulk selection actions', exact: true });
}

function rowCheckbox(page: Page, title: string) {
	return page.getByRole('checkbox', { name: `Select ${title}`, exact: true });
}

function method(route: Route) {
	return route.request().method();
}

function recordId(route: Route) {
	return awardsApi.exec(route.request().url())?.[1] ?? '';
}

async function setVisibility(page: Page, name: string) {
	const bar = bulkBar(page);
	await bar.getByRole('button', { name: 'Change Visibility' }).click();
	await bar.getByRole('button', { name, exact: true }).click();
}

test.describe('Reliable admin content lists', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	let token: string;
	let created: string[] = [];

	async function createAward(request: APIRequestContext, title: string) {
		const response = await request.post(`${apiBaseURL}/api/collections/awards/records`, {
			headers: { Authorization: token },
			data: { title, issuer: 'List reliability fixture', visibility: 'private', is_draft: true }
		});
		expect(response.ok(), await response.text()).toBe(true);
		const { id } = (await response.json()) as { id: string };
		created.push(id);
		return id;
	}

	async function storedAward(request: APIRequestContext, id: string) {
		return request.get(`${apiBaseURL}/api/collections/awards/records/${id}`, {
			headers: { Authorization: token }
		});
	}

	async function storedVisibility(request: APIRequestContext, id: string) {
		const response = await storedAward(request, id);
		expect(response.ok(), await response.text()).toBe(true);
		return ((await response.json()) as { visibility: string }).visibility;
	}

	async function selectAwards(page: Page, run: string, titles: string[]) {
		await page.goto('/admin/awards');
		const main = page.getByRole('main');
		await main.getByRole('textbox', { name: /^Search / }).fill(run);
		await main.getByRole('button', { name: 'Select', exact: true }).click();
		for (const title of titles) await rowCheckbox(page, title).check();
	}

	test.beforeEach(async ({ page, request }) => {
		({ token } = await loginAsAdmin(request));
		await seedAdminSession(page, request);
	});

	test.afterEach(async ({ request }) => {
		for (const id of created) {
			await request.delete(`${apiBaseURL}/api/collections/awards/records/${id}`, {
				headers: { Authorization: token }
			});
		}
		created = [];
	});

	test('a failed first read shows retry instead of an empty list', async ({ page, request }) => {
		const title = `Load ${randomUUID()}`;
		await createAward(request, title);
		let failReads = true;
		await page.route(awardsApi, (route) =>
			failReads && method(route) === 'GET'
				? route.fulfill({ status: 500, json: { message: 'Injected read failure' } })
				: route.continue()
		);

		await page.goto('/admin/awards');
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load" });
		await expect(alert).toBeVisible();
		await expect(page.getByText('No awards yet')).toHaveCount(0);
		await expect(page.getByRole('main').getByRole('button', { name: /New/ })).toBeDisabled();

		failReads = false;
		await alert.getByRole('button', { name: 'Retry' }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByRole('heading', { name: title })).toBeVisible();
		await expect(page.getByRole('main').getByRole('button', { name: /New/ })).toBeEnabled();
	});

	test('a malformed successful awards read preserves the editing draft for a valid retry', async ({
		page,
		request
	}) => {
		const title = `Malformed ${randomUUID()}`;
		const editedTitle = `${title} edited`;
		const id = await createAward(request, title);
		let malformed = false;
		const writes: string[] = [];
		await page.route(awardsApi, (route) => {
			if (method(route) === 'GET') {
				return malformed ? route.fulfill({ json: {} }) : route.continue();
			}
			writes.push(`${method(route)} ${recordId(route)}`.trim());
			return route.continue();
		});

		await page.goto('/admin/awards');
		const main = page.getByRole('main');
		await main
			.locator('.card', { hasText: title })
			.getByRole('button', { name: 'Edit', exact: true })
			.click();
		await page.locator('#title').fill(editedTitle);
		await expect
			.poll(() =>
				page.evaluate(
					() => JSON.parse(localStorage.getItem('autosave-admin-awards') ?? '{}').editingId
				)
			)
			.toBe(id);

		malformed = true;
		await page.reload();
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load" });
		await expect(alert).toBeVisible();
		await expect(page.getByRole('button', { name: 'Restore Draft', exact: true })).toHaveCount(0);
		await expect(main.getByRole('button', { name: /New/ })).toBeDisabled();
		expect(
			await page.evaluate(
				() => JSON.parse(localStorage.getItem('autosave-admin-awards') ?? '{}').editingId
			)
		).toBe(id);
		expect(writes).toEqual([]);

		malformed = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
		await expect(page.locator('#title')).toHaveValue(editedTitle);
		await main.getByRole('button', { name: 'Update Award', exact: true }).click();
		await expect(page.getByText('Award updated successfully')).toBeVisible();
		expect(writes).toEqual([`PATCH ${id}`]);
		const stored = await storedAward(request, id);
		expect(stored.ok(), await stored.text()).toBe(true);
		expect(((await stored.json()) as { title: string }).title).toBe(editedTitle);
	});

	test('a malformed successful course read preserves the editing draft for a valid retry', async ({
		page
	}) => {
		const id = 'malformedcourse';
		const editedTitle = 'Recovered course edit';
		await mockCourses(page, [courseRecord(id, 'Course One')], (_filter, route) =>
			route.fulfill({ json: paged([]) })
		);
		let malformed = false;
		const writes: string[] = [];
		await page.route(/\/api\/collections\/courses\/records(?:\/[^/?]+)?(?:\?|$)/, (route) => {
			if (method(route) === 'GET') {
				return malformed ? route.fulfill({ json: {} }) : route.fallback();
			}
			const writeId = new URL(route.request().url()).pathname.split('/').at(-1) ?? '';
			writes.push(`${method(route)} ${writeId}`);
			return route.fulfill({
				status: method(route) === 'POST' ? 201 : 200,
				json: courseRecord(id, editedTitle)
			});
		});

		await page.goto('/admin/courses');
		const main = page.getByRole('main');
		await main.getByRole('button', { name: 'Edit', exact: true }).click();
		await page.locator('#title').fill(editedTitle);
		await expect
			.poll(() =>
				page.evaluate(
					() => JSON.parse(localStorage.getItem('autosave-admin-courses') ?? '{}').editingId
				)
			)
			.toBe(id);

		malformed = true;
		await page.reload();
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load" });
		await expect(alert).toBeVisible();
		await expect(page.getByRole('button', { name: 'Restore Draft', exact: true })).toHaveCount(0);
		await expect(main.getByRole('button', { name: 'New Course', exact: true })).toBeDisabled();
		expect(
			await page.evaluate(
				() => JSON.parse(localStorage.getItem('autosave-admin-courses') ?? '{}').editingId
			)
		).toBe(id);
		expect(writes).toEqual([]);

		malformed = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
		await expect(page.locator('#title')).toHaveValue(editedTitle);
		await main.getByRole('button', { name: 'Save Course', exact: true }).click();
		await expect(page.getByText('Course saved')).toBeVisible();
		expect(writes).toEqual([`PATCH ${id}`]);
	});

	test('a malformed successful curriculum read blocks module writes until Retry works', async ({
		page
	}) => {
		let malformed = true;
		const writes: string[] = [];
		page.on('request', (sent) => {
			if (sent.method() !== 'GET' && /\/collections\/(modules|lessons)\//.test(sent.url())) {
				writes.push(sent.url());
			}
		});
		const known = { id: 'moduleone000000', title: 'Known module' };
		await mockCourses(page, [courseRecord('curriculumone00', 'Course One')], (_filter, route) =>
			malformed ? route.fulfill({ json: {} }) : route.fulfill({ json: paged([known]) })
		);

		await page.goto('/admin/courses');
		const main = page.getByRole('main');
		await main.getByRole('button', { name: 'Edit', exact: true }).click();
		await page.locator('#title').fill('Typed draft title');
		await main.getByRole('button', { name: 'Curriculum' }).click();
		const alert = page.getByRole('alert').filter({ hasText: 'changes are paused' });
		await expect(alert).toBeVisible();
		await expect(page.getByText('No modules yet')).toHaveCount(0);
		await expect(main.getByRole('button', { name: /Add Module/ })).toBeDisabled();
		expect(writes).toEqual([]);

		malformed = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByText('Known module')).toBeVisible();
		await expect(main.getByRole('button', { name: /Add Module/ })).toBeEnabled();
		await main.getByRole('button', { name: 'Content', exact: true }).click();
		await expect(page.locator('#title')).toHaveValue('Typed draft title');
		expect(writes).toEqual([]);
	});

	test('a failed membership read leaves the list usable', async ({ page, request }) => {
		const title = `Membership ${randomUUID()}`;
		await createAward(request, title);
		await page.route('**/api/admin/view-memberships*', (route) =>
			route.fulfill({ status: 500, json: { message: 'Injected membership failure' } })
		);

		await page.goto('/admin/awards');
		await expect(page.getByRole('heading', { name: title })).toBeVisible();
		await expect(page.getByText("Some supporting details couldn't load")).toBeVisible();
		await expect(page.getByRole('alert').filter({ hasText: "Couldn't load" })).toHaveCount(0);
		await expect(page.getByText('Not in any view')).toHaveCount(0);
		await expect(page.getByRole('main').getByRole('button', { name: /New/ })).toBeEnabled();
	});

	test('the 101st record is listed', async ({ page }) => {
		const records = Array.from({ length: 101 }, (_, index) => ({
			id: `award${String(index + 1).padStart(10, '0')}`,
			title: `Paged award ${index + 1}`,
			visibility: 'private',
			is_draft: false,
			sort_order: 0
		}));
		await page.route(awardsApi, (route) => {
			const url = new URL(route.request().url());
			const pageNumber = Number(url.searchParams.get('page') ?? 1);
			const perPage = Number(url.searchParams.get('perPage') ?? 30);
			return route.fulfill({
				json: {
					page: pageNumber,
					perPage,
					totalItems: records.length,
					totalPages: Math.ceil(records.length / perPage),
					items: records.slice((pageNumber - 1) * perPage, pageNumber * perPage)
				}
			});
		});

		await page.goto('/admin/awards');
		await expect(page.getByRole('heading', { name: 'Paged award 101', exact: true })).toBeVisible();
	});

	test('a rejected item does not stop the batch or repeat successes', async ({ page, request }) => {
		const run = randomUUID();
		const titles = ['First', 'Second', 'Third'].map((label) => `${label} ${run}`);
		const ids: string[] = [];
		for (const title of titles) ids.push(await createAward(request, title));
		const patched: string[] = [];
		let rejected = ids[1];
		await page.route(awardsApi, (route) => {
			if (method(route) !== 'PATCH') return route.continue();
			patched.push(recordId(route));
			return recordId(route) === rejected
				? route.fulfill({ status: 400, json: { message: 'Injected rejection' } })
				: route.continue();
		});

		await selectAwards(page, run, titles);
		await setVisibility(page, 'Unlisted');
		await expect(page.getByText(/Updated 2 of 3 items.*1 item failed/)).toBeVisible();
		expect(patched).toEqual(ids);
		expect(await storedVisibility(request, ids[0])).toBe('unlisted');
		expect(await storedVisibility(request, ids[1])).toBe('private');
		expect(await storedVisibility(request, ids[2])).toBe('unlisted');
		await expect(rowCheckbox(page, titles[0])).not.toBeChecked();
		await expect(rowCheckbox(page, titles[1])).toBeChecked();
		await expect(rowCheckbox(page, titles[2])).not.toBeChecked();

		rejected = '';
		patched.length = 0;
		await setVisibility(page, 'Unlisted');
		await expect(page.getByText(/Updated 1 of 1 items/)).toBeVisible();
		expect(patched).toEqual([ids[1]]);
		expect(await storedVisibility(request, ids[1])).toBe('unlisted');
	});

	test('a lost response is reconciled from the reloaded list', async ({ page, request }) => {
		const run = randomUUID();
		const title = `Lost ${run}`;
		const id = await createAward(request, title);
		await page.route(awardsApi, async (route) => {
			if (method(route) === 'GET') return route.continue();
			await route.fetch();
			return route.abort('connectionfailed');
		});

		await selectAwards(page, run, [title]);
		await setVisibility(page, 'Unlisted');
		await expect(page.getByText(/Updated 1 of 1 items/)).toBeVisible();
		expect(await storedVisibility(request, id)).toBe('unlisted');
		await expect(bulkBar(page)).toHaveCount(0);

		await page.getByRole('main').getByRole('button', { name: 'Select', exact: true }).click();
		await rowCheckbox(page, title).check();
		await bulkBar(page).getByRole('button', { name: 'Delete', exact: true }).click();
		await page.getByRole('alertdialog').getByRole('button', { name: 'Delete All' }).click();
		await expect(page.getByText(/Deleted 1 of 1 items/)).toBeVisible();
		expect((await storedAward(request, id)).status()).toBe(404);
	});

	test('an unknown result stays unconfirmed until the list reloads', async ({ page, request }) => {
		const run = randomUUID();
		const title = `Unconfirmed ${run}`;
		const id = await createAward(request, title);
		let patches = 0;
		let failReads = false;
		await page.route(awardsApi, async (route) => {
			if (method(route) === 'GET') {
				return failReads
					? route.fulfill({ status: 500, json: { message: 'Injected read failure' } })
					: route.continue();
			}
			patches += 1;
			failReads = true;
			await route.fetch();
			return route.abort('connectionfailed');
		});

		await selectAwards(page, run, [title]);
		await setVisibility(page, 'Unlisted');
		await expect(page.getByText(/Could not confirm 1 item/)).toBeVisible();
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't refresh" });
		await expect(alert).toBeVisible();
		await expect(page.getByRole('heading', { name: title })).toBeVisible();
		await expect(bulkBar(page).getByRole('button', { name: 'Delete', exact: true })).toBeDisabled();

		failReads = false;
		await alert.getByRole('button', { name: 'Retry' }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByText(/Updated 1 of 1 items/)).toBeVisible();
		await expect(bulkBar(page)).toHaveCount(0);
		await expect(page.locator('main h1')).toBeFocused();
		expect(patches).toBe(1);
		expect(await storedVisibility(request, id)).toBe('unlisted');
	});

	test('a second click cannot start a second batch', async ({ page, request }) => {
		const run = randomUUID();
		const titles = ['One', 'Two'].map((label) => `${label} ${run}`);
		const ids: string[] = [];
		for (const title of titles) ids.push(await createAward(request, title));
		const patched: string[] = [];
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route(awardsApi, async (route) => {
			if (method(route) !== 'PATCH') return route.continue();
			patched.push(recordId(route));
			await held;
			return route.continue();
		});

		await selectAwards(page, run, titles);
		const bar = bulkBar(page);
		await bar.getByRole('button', { name: 'Change Visibility' }).click();
		await bar.getByRole('button', { name: 'Unlisted', exact: true }).evaluate((button) => {
			(button as HTMLButtonElement).click();
			(button as HTMLButtonElement).click();
		});
		await expect.poll(() => patched.length).toBe(1);
		await expect(bar.getByRole('button', { name: 'Delete', exact: true })).toBeDisabled();
		await expect(rowCheckbox(page, titles[0])).toBeDisabled();
		await expect(page.getByRole('status').filter({ hasText: 'Saving changes' })).toBeVisible();

		release();
		await expect(page.getByText(/Updated 2 of 2 items/)).toBeVisible();
		expect([...patched].sort()).toEqual([...ids].sort());
	});

	test('publishing private records asks first and names the count', async ({ page, request }) => {
		const run = randomUUID();
		const titles = ['Alpha', 'Beta'].map((label) => `${label} ${run}`);
		const ids: string[] = [];
		for (const title of titles) ids.push(await createAward(request, title));

		await selectAwards(page, run, titles);
		await setVisibility(page, 'Public');
		const dialog = page.getByRole('alertdialog');
		await expect(dialog).toContainText('2 private or unlisted items will become public');
		await dialog.getByRole('button', { name: 'Cancel' }).click();
		expect(await storedVisibility(request, ids[0])).toBe('private');
		await expect(rowCheckbox(page, titles[0])).toBeChecked();

		await setVisibility(page, 'Public');
		await dialog.getByRole('button', { name: 'Make public' }).click();
		await expect(page.getByText(/Updated 2 of 2 items/)).toBeVisible();
		for (const id of ids) expect(await storedVisibility(request, id)).toBe('public');
	});

	test('pressing Enter again during a save sends one create', async ({ page }) => {
		const title = `Enter ${randomUUID()}`;
		let posts = 0;
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route(awardsApi, async (route) => {
			if (method(route) !== 'POST') return route.continue();
			posts += 1;
			await held;
			const response = await route.fetch();
			created.push(((await response.json()) as { id: string }).id);
			return route.fulfill({ response });
		});

		await page.goto('/admin/awards');
		await page.getByRole('main').getByRole('button', { name: /New/ }).click();
		const field = page.locator('#title');
		await field.fill(title);
		await field.press('Enter');
		await field.press('Enter');
		await field.evaluate((input) => (input as HTMLInputElement).form?.requestSubmit());
		await expect.poll(() => posts).toBe(1);
		await expect(field).toHaveValue(title);

		release();
		await expect(page.getByText('Award created successfully')).toBeVisible();
		expect(posts).toBe(1);
	});

	test('focus moves to the heading when a finished batch removes its controls', async ({
		page,
		request
	}) => {
		const run = randomUUID();
		const title = `Focus ${run}`;
		await createAward(request, title);

		await selectAwards(page, run, [title]);
		await setVisibility(page, 'Unlisted');
		await expect(page.getByText(/Updated 1 of 1 items/)).toBeVisible();
		await expect(page.locator('main h1')).toBeFocused();
	});

	test('focus moves to a failed row when the batch keeps it selected', async ({
		page,
		request
	}) => {
		const run = randomUUID();
		const title = `Failed focus ${run}`;
		await createAward(request, title);
		await page.route(awardsApi, (route) =>
			method(route) === 'PATCH'
				? route.fulfill({ status: 400, json: { message: 'Injected rejection' } })
				: route.continue()
		);

		await selectAwards(page, run, [title]);
		await setVisibility(page, 'Unlisted');
		await expect(page.getByText(/1 item failed/)).toBeVisible();
		await expect(rowCheckbox(page, title)).toBeFocused();
	});

	test('a batch finishing after navigation leaves the new screen alone', async ({
		page,
		request
	}) => {
		const run = randomUUID();
		const title = `Departed ${run}`;
		await createAward(request, title);
		let patches = 0;
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route(awardsApi, async (route) => {
			if (method(route) !== 'PATCH') return route.continue();
			patches += 1;
			await held;
			return route.continue();
		});

		await selectAwards(page, run, [title]);
		await setVisibility(page, 'Unlisted');
		await expect.poll(() => patches).toBe(1);
		await page.getByRole('link', { name: 'Content', exact: true }).click();
		await expect(page).toHaveURL(/\/admin\/content/);
		await expect(page.getByRole('main').getByRole('heading', { level: 1 })).toBeVisible();
		await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());

		release();
		await expect(page.getByText(/Updated 1 of 1 items/)).toBeVisible();
		expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);
	});

	test('supporting data retries without touching an open draft', async ({ page, request }) => {
		const title = `Supporting ${randomUUID()}`;
		await createAward(request, title);
		let malformed = true;
		await page.route('**/api/admin/view-memberships*', (route) =>
			malformed
				? route.fulfill({ json: { memberships: { record: 'not a list' } } })
				: route.continue()
		);

		await page.goto('/admin/awards');
		await expect(page.getByRole('heading', { name: title })).toBeVisible();
		await expect(page.getByText("Some supporting details couldn't load")).toBeVisible();
		await expect(page.getByText('Not in any view')).toHaveCount(0);

		await page.getByRole('main').getByRole('button', { name: /New/ }).click();
		await page.locator('#title').fill('Draft kept during retry');
		malformed = false;
		await page.getByRole('button', { name: 'Retry supporting details' }).click();
		await expect(page.getByText("Some supporting details couldn't load")).toHaveCount(0);
		await expect(page.locator('#title')).toHaveValue('Draft kept during retry');
	});

	test('a partial reorder reports the result and retries only the rest', async ({ page }) => {
		const records = ['One', 'Two', 'Three'].map((title, index) => ({
			id: `customorder${index}000`,
			title,
			content: '',
			visibility: 'private',
			is_draft: false,
			sort_order: index
		}));
		const patched: string[] = [];
		let rejected = records[0].id;
		await page.route(customApi, (route) => {
			if (method(route) === 'GET') return route.fulfill({ json: paged(records, 100) });
			const id = customApi.exec(route.request().url())?.[1] ?? '';
			patched.push(id);
			if (id === rejected) {
				return route.fulfill({ status: 400, json: { message: 'Injected rejection' } });
			}
			const record = records.find((candidate) => candidate.id === id)!;
			record.sort_order = route.request().postDataJSON().sort_order;
			return route.fulfill({ json: record });
		});

		await page.goto('/admin/custom');
		const main = page.getByRole('main');
		await main.getByRole('button', { name: 'Reorder', exact: true }).click();
		const moveUp = main.getByRole('button', { name: 'Move Three up', exact: true });
		await moveUp.click();
		await moveUp.click();
		const save = main.getByRole('button', { name: 'Save Order', exact: true });
		await save.click();
		await expect(page.getByText(/Saved 2 of 3 positions/)).toBeVisible();
		expect(patched).toEqual([records[2].id, records[0].id, records[1].id]);
		await expect(save).toBeVisible();

		rejected = '';
		patched.length = 0;
		await page
			.getByRole('alert')
			.filter({ hasText: /Saved 2 of 3 positions/ })
			.getByRole('button', { name: 'Dismiss', exact: true })
			.click();
		await save.click();
		await expect(page.getByText('Order saved')).toBeVisible();
		expect(patched).toEqual([records[0].id]);
		expect(records.map((record) => record.sort_order)).toEqual([1, 2, 0]);
	});

	test('a failed curriculum read blocks module writes until Retry works', async ({ page }) => {
		let failRead = true;
		const writes: string[] = [];
		page.on('request', (sent) => {
			if (sent.method() !== 'GET' && /\/collections\/(modules|lessons)\//.test(sent.url())) {
				writes.push(sent.url());
			}
		});
		const known = { id: 'moduleone000000', title: 'Known module' };
		await mockCourses(page, [courseRecord('curriculumone00', 'Course One')], (_filter, route) =>
			failRead
				? route.fulfill({ status: 500, json: { message: 'Injected read failure' } })
				: route.fulfill({ json: paged([known]) })
		);

		await page.goto('/admin/courses');
		const main = page.getByRole('main');
		await main.getByRole('button', { name: 'Edit', exact: true }).click();
		await page.locator('#title').fill('Typed draft title');
		await main.getByRole('button', { name: 'Curriculum' }).click();
		const alert = page.getByRole('alert').filter({ hasText: 'changes are paused' });
		await expect(alert).toBeVisible();
		await expect(page.getByText('No modules yet')).toHaveCount(0);
		await expect(main.getByRole('button', { name: /Add Module/ })).toBeDisabled();

		failRead = false;
		await alert.getByRole('button', { name: 'Retry' }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByText('Known module')).toBeVisible();
		await expect(main.getByRole('button', { name: /Add Module/ })).toBeEnabled();
		await main.getByRole('button', { name: 'Content', exact: true }).click();
		await expect(page.locator('#title')).toHaveValue('Typed draft title');
		expect(writes).toEqual([]);
	});

	test('a late curriculum response cannot replace the open course', async ({ page }) => {
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		let late: Promise<unknown> = Promise.resolve();
		const courses = [
			courseRecord('curriculumaaa00', 'Course Alpha'),
			courseRecord('curriculumbbb00', 'Course Beta')
		];
		await mockCourses(page, courses, (filter, route) => {
			const alpha = filter.includes('curriculumaaa00');
			const title = alpha ? 'Alpha module' : 'Beta module';
			const respond = () => route.fulfill({ json: paged([{ id: 'modulerecord000', title }]) });
			if (!alpha) return respond();
			late = held.then(respond).catch(() => {});
			return late;
		});

		await page.goto('/admin/courses');
		const main = page.getByRole('main');
		const edit = (title: string) =>
			main.locator('.card', { hasText: title }).getByRole('button', { name: 'Edit', exact: true });
		await edit('Course Alpha').click();
		await main.getByRole('button', { name: 'Back to Courses' }).click();
		await edit('Course Beta').click();
		await main.getByRole('button', { name: 'Curriculum' }).click();
		await expect(page.getByText('Beta module')).toBeVisible();

		release();
		await late;
		await expect(page.getByText('Beta module')).toBeVisible();
		await expect(page.getByText('Alpha module')).toHaveCount(0);
	});
});

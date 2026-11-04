import { expect, test, type Page, type Route } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { randomUUID } from 'node:crypto';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

type Row = { id: string; title: string; [field: string]: unknown };

const courseA = 'courseaaa0000000';
const courseB = 'coursebbb0000000';
const moduleA = 'moduleaaa0000000';
const moduleB = 'modulebbb0000000';

function recordsApi(collection: string) {
	return new RegExp(`/api/collections/${collection}/records(?:/([^/?]+))?(?:\\?|$)`);
}

function paged(route: Route, items: unknown[]) {
	const url = new URL(route.request().url());
	const page = Number(url.searchParams.get('page') ?? '1');
	const perPage = Number(url.searchParams.get('perPage') ?? '100');
	return {
		page,
		perPage,
		totalItems: items.length,
		totalPages: Math.ceil(items.length / perPage),
		items: items.slice((page - 1) * perPage, page * perPage)
	};
}

function courseRow(id: string, title: string): Row {
	return {
		id,
		title,
		slug: id,
		description: '',
		access_tier: 'free',
		price: 0,
		visibility: 'private',
		is_draft: true,
		sort_order: 0,
		difficulty: 'beginner',
		sequential_access: false,
		enable_progress: true,
		show_progress_bar: true,
		require_enrollment: false,
		enable_certificates: true
	};
}

function moduleRow(id: string, title: string, course = courseA): Row {
	return { id, title, description: '', course, sort_order: 1 };
}

function method(route: Route) {
	return route.request().method();
}

async function formField(route: Route, field: string) {
	const body = route.request().postDataBuffer();
	if (!body) throw new Error('Expected a multipart course write');
	const form = await new Response(Uint8Array.from(body), {
		headers: { 'content-type': route.request().headers()['content-type'] }
	}).formData();
	return String(form.get(field) ?? '');
}

async function mockLists(page: Page, courses: Row[], modules: Row[]) {
	await page.route(recordsApi('courses'), (route) =>
		method(route) === 'GET'
			? route.fulfill({ json: paged(route, courses) })
			: route.fulfill({ json: courses[0] ?? courseRow('courseheld000000', 'Held course') })
	);
	await page.route(recordsApi('modules'), (route) =>
		method(route) === 'GET'
			? route.fulfill({ json: paged(route, modules) })
			: route.fulfill({ json: modules[0] ?? moduleRow('moduleheld000000', 'Held module') })
	);
	await page.route(recordsApi('lessons'), (route) => route.fulfill({ json: paged(route, []) }));
}

async function openCourses(page: Page) {
	await page.goto('/admin/courses');
	return page.getByRole('main');
}

async function openCourse(page: Page, title: string) {
	if (new URL(page.url()).pathname !== '/admin/courses') await openCourses(page);
	const main = page.getByRole('main');
	await main
		.locator('.card', { hasText: title })
		.getByRole('button', { name: 'Edit', exact: true })
		.click();
	await main.getByRole('button', { name: /^Curriculum(?: \d+)?$/ }).click();
	return main;
}

async function holdWrite(page: Page, collection: 'courses' | 'modules') {
	const held = { count: 0, release: () => {} };
	const released = new Promise<void>((resolve) => (held.release = resolve));
	await page.route(recordsApi(collection), async (route) => {
		if (method(route) === 'GET') return route.fallback();
		held.count += 1;
		await released;
		return route.fallback();
	});
	return held;
}

test.describe('Course and module form recovery', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
	});

	test('the open module editor has accessible controls and readable text', async ({ page }) => {
		await mockLists(page, [courseRow(courseA, 'Course A')], [moduleRow(moduleA, 'Module A')]);
		const main = await openCourse(page, 'Course A');
		await main.getByRole('button', { name: 'Edit Module', exact: true }).click();
		await expect(page.locator('#module_title')).toHaveValue('Module A');
		const { violations } = await new AxeBuilder({ page })
			.withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'])
			.analyze();
		expect(
			violations.map(({ id, nodes }) => ({ id, targets: nodes.map((node) => node.target) }))
		).toEqual([]);
	});

	test('a restored draft checks its existing course before attempting another create', async ({
		page,
		request
	}) => {
		const { token } = await loginAsAdmin(request);
		const headers = { Authorization: token };
		const slug = 'restored-' + randomUUID();
		const response = await request.post(apiBaseURL + '/api/collections/courses/records', {
			headers,
			data: {
				title: 'Restored course',
				slug,
				access_tier: 'free',
				visibility: 'private',
				is_draft: true,
				difficulty: 'beginner',
				enable_progress: true,
				show_progress_bar: true,
				enable_certificates: true
			}
		});
		expect(response.ok(), await response.text()).toBe(true);
		const { id } = await response.json();
		try {
			await page.addInitScript(
				(draft) => localStorage.setItem('autosave-admin-courses', JSON.stringify(draft)),
				{
					data: {
						draftId: id,
						title: 'Restored course',
						slug,
						visibility: 'private',
						isDraft: true
					},
					savedAt: Date.now(),
					isEditing: false
				}
			);
			let creates = 0;
			page.on('request', (req) => {
				if (
					req.method() === 'POST' &&
					new URL(req.url()).pathname === '/api/collections/courses/records'
				)
					creates++;
			});
			const main = await openCourses(page);
			await expect(main.locator('.card', { hasText: 'Restored course' })).toBeVisible();
			expect(
				await page.evaluate(
					() => JSON.parse(localStorage.getItem('autosave-admin-courses') || 'null')?.data?.draftId
				)
			).toBe(id);
			await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
			await expect(page.locator('#title')).toHaveValue('Restored course');
			await page.getByRole('button', { name: 'Save Course', exact: true }).click();
			await expect(page.getByRole('heading', { name: 'Edit Course', exact: true })).toBeVisible();
			expect(creates).toBe(0);
		} finally {
			await request.delete(apiBaseURL + '/api/collections/courses/records/' + id, { headers });
		}
	});

	test('a held module save cannot close or overwrite the next module draft', async ({ page }) => {
		const modules = [moduleRow(moduleA, 'Module A'), moduleRow(moduleB, 'Module B')];
		await mockLists(page, [courseRow(courseA, 'Course A')], modules);
		const writes: { id: string; title: string }[] = [];
		await page.route(recordsApi('modules'), async (route) => {
			if (method(route) === 'GET') return route.fallback();
			const id = recordsApi('modules').exec(route.request().url())?.[1] ?? '';
			const body = route.request().postDataJSON();
			writes.push({ id, title: body.title });
			return route.fulfill({ json: { ...modules.find((row) => row.id === id), ...body } });
		});
		const held = await holdWrite(page, 'modules');

		const main = await openCourse(page, 'Course A');
		await main.getByRole('button', { name: 'Edit Module', exact: true }).first().click();
		await page.locator('#module_title').fill('Module A saved');
		await main.getByRole('button', { name: 'Save Module', exact: true }).click();
		await expect.poll(() => held.count).toBe(1);

		await main.getByRole('button', { name: 'Cancel', exact: true }).last().click();
		await main.getByRole('button', { name: 'Edit Module', exact: true }).nth(1).click();
		await page.locator('#module_title').fill('Module B typed');
		held.release();

		await expect(page.getByText('Module updated')).toBeVisible();
		await expect(page.locator('#module_title')).toHaveValue('Module B typed');
		await expect(main.getByRole('button', { name: 'Save Module', exact: true })).toBeEnabled();
		expect(writes).toEqual([{ id: moduleA, title: 'Module A saved' }]);
	});

	test('a held course create cannot replace another course or clear its draft', async ({
		page
	}) => {
		await mockLists(page, [courseRow(courseB, 'Course B')], []);
		let created: Row | undefined;
		await page.route(recordsApi('courses'), async (route) => {
			if (method(route) === 'GET') return route.fallback();
			created = {
				...courseRow(await formField(route, 'id'), await formField(route, 'title')),
				slug: await formField(route, 'slug'),
				visibility: await formField(route, 'visibility')
			};
			return route.fulfill({ json: created });
		});
		const held = await holdWrite(page, 'courses');

		const main = await openCourses(page);
		await main.getByRole('button', { name: 'New Course', exact: true }).first().click();
		await page.locator('#title').fill('Created while away');
		await page.locator('#slug').fill('created-while-away');
		await main.getByRole('button', { name: 'Save Course', exact: true }).click();
		await expect.poll(() => held.count).toBe(1);

		await main.getByRole('button', { name: 'Back to Courses', exact: true }).click();
		await openCourse(page, 'Course B');
		await main.getByRole('button', { name: 'Content', exact: true }).click();
		await page.locator('#title').fill('Course B draft');
		held.release();

		await expect(page.locator('#title')).toHaveValue('Course B draft');
		await expect(page.locator('#slug')).toHaveValue(courseB);
		expect(created?.title).toBe('Created while away');
	});

	test('course and module fields lock only while their own save is pending', async ({ page }) => {
		await mockLists(page, [courseRow(courseA, 'Course A')], [moduleRow(moduleA, 'Module A')]);
		const moduleHeld = await holdWrite(page, 'modules');
		const main = await openCourse(page, 'Course A');
		await main.getByRole('button', { name: 'Edit Module', exact: true }).click();
		await main.getByRole('button', { name: 'Save Module', exact: true }).click();
		await expect.poll(() => moduleHeld.count).toBe(1);
		await expect(page.locator('#module_title')).toBeDisabled();
		await expect(main.getByRole('button', { name: 'Cancel', exact: true }).last()).toBeEnabled();
		moduleHeld.release();
		await expect(page.getByText('Module updated')).toBeVisible();

		const courseHeld = await holdWrite(page, 'courses');
		await main.getByRole('button', { name: 'Content', exact: true }).click();
		await main.getByRole('button', { name: 'Save Course', exact: true }).click();
		await expect.poll(() => courseHeld.count).toBe(1);
		await expect(page.locator('#title')).toBeDisabled();
		await expect(page.locator('#slug')).toBeDisabled();
		await expect(main.getByRole('button', { name: 'Back to Courses', exact: true })).toBeEnabled();
		courseHeld.release();
		await expect(page.locator('#title')).toBeEnabled();
	});

	test('a rejected write retains fields and unlocks the form', async ({ page }) => {
		await mockLists(page, [courseRow(courseA, 'Course A')], [moduleRow(moduleA, 'Module A')]);
		await page.route(recordsApi('modules'), (route) =>
			method(route) === 'GET'
				? route.fallback()
				: route.fulfill({ status: 400, json: { message: 'Injected rejection' } })
		);
		const main = await openCourse(page, 'Course A');
		await main.getByRole('button', { name: 'Edit Module', exact: true }).click();
		await page.locator('#module_title').fill('Still my module draft');
		await main.getByRole('button', { name: 'Save Module', exact: true }).click();
		await expect(page.getByText('Failed to save module')).toBeVisible();
		await expect(page.locator('#module_title')).toHaveValue('Still my module draft');
		await expect(page.locator('#module_title')).toBeEnabled();
	});

	for (const collection of ['courses', 'modules'] as const) {
		test(`a landed lost ${collection} create is read back without another POST`, async ({
			page
		}) => {
			const courses = collection === 'courses' ? [] : [courseRow(courseA, 'Course A')];
			const modules = collection === 'modules' ? [moduleRow(moduleA, 'Module A')] : [];
			await mockLists(page, courses, modules);
			let posts = 0;
			let stored: Row | undefined;
			await page.route(recordsApi(collection), async (route) => {
				const id = recordsApi(collection).exec(route.request().url())?.[1];
				if (method(route) === 'GET' && id) {
					return stored
						? route.fulfill({ json: stored })
						: route.fulfill({ status: 404, json: { status: 404, data: {} } });
				}
				if (method(route) !== 'POST') return route.fallback();
				posts += 1;
				const body =
					collection === 'courses'
						? {
								id: await formField(route, 'id'),
								title: await formField(route, 'title'),
								slug: await formField(route, 'slug'),
								visibility: await formField(route, 'visibility')
							}
						: route.request().postDataJSON();
				stored =
					collection === 'courses'
						? { ...courseRow(body.id, body.title), slug: body.slug, visibility: body.visibility }
						: moduleRow(body.id, body.title);
				return route.abort('connectionfailed');
			});

			const main =
				collection === 'courses' ? await openCourses(page) : await openCourse(page, 'Course A');
			if (collection === 'courses') {
				await main.getByRole('button', { name: 'New Course', exact: true }).first().click();
				await page.locator('#title').fill('Recovered course');
				await page.locator('#slug').fill('recovered-course');
				await main.getByRole('button', { name: 'Save Course', exact: true }).click();
			} else {
				await main.getByRole('button', { name: /Add Module/ }).click();
				await page.locator('#module_title').fill('Recovered module');
				await main.getByRole('button', { name: 'Save Module', exact: true }).click();
			}

			await expect.poll(() => posts).toBe(1);
			if (collection === 'courses') {
				await expect(page.getByRole('heading', { name: 'Edit Course', exact: true })).toBeVisible();
			} else {
				await expect(page.locator('#module_title')).toHaveCount(0);
			}
			expect(posts).toBe(1);
			expect(stored?.id).toMatch(/^[a-z0-9]{15}$/);
		});
	}
});

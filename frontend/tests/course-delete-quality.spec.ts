import { randomUUID } from 'node:crypto';
import { expect, test, type APIRequestContext, type Locator, type Page } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

const moduleDeletes = '**/api/admin/courses/*/modules/*';
const lessonDeletes = '**/api/admin/courses/*/modules/*/lessons/*';
const lessonRecords = /\/api\/collections\/lessons\/records(?:\/[^/?]+)?(?:\?|$)/;

// Run newest first after each test, so a fixture that fails halfway still leaves nothing behind.
let cleanups: (() => Promise<void>)[] = [];

// A 429 is refused before the operation starts. Honor the server's delay before
// repeating that fixture request; uncertain writes are never retried here.
async function postWithinQuota(request: APIRequestContext, url: string, data: object) {
	let response = await request.post(url, { data });
	if (response.status() === 429) {
		const seconds = Number(response.headers()['retry-after']);
		expect(Number.isInteger(seconds) && seconds > 0 && seconds <= 15).toBe(true);
		await new Promise((resolve) => setTimeout(resolve, seconds * 1000));
		response = await request.post(url, { data });
	}
	return response;
}

// A real course. It's unlisted and published because learner routes refuse drafts.
// Its cleanup is the course delete, which also clears any learner records.
async function ownedCourse(request: APIRequestContext) {
	const { token } = await loginAsAdmin(request);
	const headers = { Authorization: token };
	const run = randomUUID();
	const title = `Delete fixture ${run}`;
	const slug = `delete-fixture-${run}`;
	const created = await request.post(`${apiBaseURL}/api/collections/courses/records`, {
		headers,
		data: {
			title,
			slug,
			access_tier: 'free',
			visibility: 'unlisted',
			is_draft: false,
			enable_progress: true
		}
	});
	expect(created.ok(), await created.text()).toBe(true);
	const { id } = (await created.json()) as { id: string };
	cleanups.push(async () => {
		const removed = await request.delete(`${apiBaseURL}/api/collections/courses/records/${id}`, {
			headers
		});
		expect.soft(removed.ok(), await removed.text()).toBe(true);
	});

	const courseApi = `${apiBaseURL}/api/admin/courses/${id}`;
	async function add(path: string, data: object) {
		const response = await request.post(`${courseApi}/${path}`, { headers, data });
		expect(response.ok(), await response.text()).toBe(true);
		return ((await response.json()) as { id: string }).id;
	}

	return {
		title,
		slug,
		addModule: (moduleTitle: string, sort_order: number) =>
			add('modules', { title: moduleTitle, sort_order }),
		addLesson: (moduleId: string, lessonTitle: string, sort_order: number, content_type = 'text') =>
			add(`modules/${moduleId}/lessons`, {
				title: lessonTitle,
				content: '',
				content_type,
				sort_order
			}),
		async addQuiz(lessonId: string) {
			await add(`lessons/${lessonId}/quiz`, {
				title: 'Attempted quiz',
				questions: [
					{
						question_text: 'Pick blue',
						question_type: 'multiple_choice',
						options: [
							{ text: 'Blue', is_correct: true },
							{ text: 'Red', is_correct: false }
						],
						sort_order: 1
					}
				]
			});
			const quiz = await request.get(`${courseApi}/lessons/${lessonId}/quiz`, { headers });
			expect(quiz.ok(), await quiz.text()).toBe(true);
			return ((await quiz.json()) as { questions: { id: string }[] }).questions[0].id;
		},
		// The stored titles, in order.
		async stored(collection: 'modules' | 'lessons') {
			const filter = collection === 'modules' ? `course="${id}"` : `module.course="${id}"`;
			const response = await request.get(`${apiBaseURL}/api/collections/${collection}/records`, {
				headers,
				params: { filter, sort: 'sort_order' }
			});
			expect(response.ok(), await response.text()).toBe(true);
			return ((await response.json()).items as { title: string }[]).map((row) => row.title);
		}
	};
}

// Progress and attempts are written only by learner routes, and the owner can't reach those
// collections directly. A 100% coupon on a paid post is how a test gets a real learner cookie:
// it needs no Stripe call and no inbox. The cookie stays in this request context.
async function realLearner(request: APIRequestContext, baseURL: string, slug: string) {
	const { token } = await loginAsAdmin(request);
	const headers = { Authorization: token };
	const run = randomUUID();
	const post = await request.post(`${apiBaseURL}/api/collections/posts/records`, {
		headers,
		data: {
			title: `Learner fixture ${run}`,
			slug: `learner-fixture-${run}`,
			content: 'Learner fixture',
			visibility: 'private',
			access_tier: 'paid',
			price: 1299
		}
	});
	test.skip(!post.ok(), 'Paid content needs STRIPE_SECRET_KEY on the test backend');
	const { id: postId } = (await post.json()) as { id: string };
	cleanups.push(async () => {
		await request.delete(`${apiBaseURL}/api/collections/posts/records/${postId}`, { headers });
	});

	const code = `LEARNER-${run}`.toUpperCase();
	const coupon = await request.post(`${apiBaseURL}/api/admin/coupons`, {
		headers,
		data: {
			code,
			discount_type: 'percentage',
			discount_value: 100,
			applicable_type: 'posts',
			applicable_ids: [postId],
			max_uses: 1
		}
	});
	expect(coupon.status(), await coupon.text()).toBe(201);
	const { id: couponId } = (await coupon.json()) as { id: string };
	cleanups.push(async () => {
		await request.delete(`${apiBaseURL}/api/admin/coupons/${couponId}`, { headers });
	});
	cleanups.push(async () => {
		const purchases = await request.get(`${apiBaseURL}/api/collections/purchases/records`, {
			headers,
			params: { filter: `content_id = "${postId}"` }
		});
		for (const { id } of (await purchases.json()).items as { id: string }[]) {
			await request.delete(`${apiBaseURL}/api/collections/purchases/records/${id}`, { headers });
		}
	});

	const checkout = await postWithinQuota(request, `${apiBaseURL}/api/content/checkout`, {
		content_type: 'posts',
		content_id: postId,
		coupon_code: code,
		success_url: `${baseURL}/`,
		cancel_url: `${baseURL}/`
	});
	expect(checkout.ok(), await checkout.text()).toBe(true);
	const { checkout_url } = (await checkout.json()) as { checkout_url: string };
	const access = await request.get(`${apiBaseURL}${checkout_url}`, { maxRedirects: 0 });
	expect(access.status()).toBeLessThan(400);

	const learnerApi = `${apiBaseURL}/api/courses/${slug}`;
	return {
		async complete(lessonId: string) {
			const response = await request.post(`${learnerApi}/lessons/${lessonId}/complete`);
			expect(response.ok(), await response.text()).toBe(true);
		},
		async attempt(lessonId: string, questionId: string) {
			const response = await postWithinQuota(
				request,
				`${learnerApi}/lessons/${lessonId}/quiz/submit`,
				{
					answers: [{ question_id: questionId, selected_answer: 0 }]
				}
			);
			expect(response.ok(), await response.text()).toBe(true);
		},
		async attemptsUsed(lessonId: string) {
			const response = await request.get(`${learnerApi}/lessons/${lessonId}/quiz`);
			expect(response.ok(), await response.text()).toBe(true);
			return ((await response.json()) as { attempts_used: number }).attempts_used;
		},
		async completedLessons() {
			const response = await request.get(`${learnerApi}/progress`);
			expect(response.ok(), await response.text()).toBe(true);
			const { lesson_progress } = (await response.json()) as {
				lesson_progress: { lesson_id: string }[];
			};
			return lesson_progress.map((row) => row.lesson_id);
		}
	};
}

async function openCurriculum(page: Page, title: string) {
	await page.goto('/admin/courses');
	const main = page.getByRole('main');
	const card = main.locator('.card', { hasText: title });
	await card.getByRole('button', { name: 'Edit', exact: true }).click();
	await main.getByRole('button', { name: 'Curriculum' }).click();
	return main;
}

function editLesson(page: Page, index = 0) {
	return page.getByRole('button', { name: 'Edit Lesson', exact: true }).nth(index);
}

function deleteButton(main: Locator, name: 'Delete Module' | 'Delete Lesson') {
	return main.getByRole('button', { name, exact: true }).first();
}

async function confirmDelete(page: Page) {
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete', exact: true }).click();
}

// Presses delete and returns the status the server answered with.
async function deleteAndReadStatus(page: Page, button: Locator) {
	const reply = page.waitForResponse(
		(response) =>
			response.request().method() === 'DELETE' && response.url().includes('/api/admin/courses/')
	);
	await button.click();
	await confirmDelete(page);
	return (await reply).status();
}

// Holds lesson writes until release(), then sends them on or rejects them.
async function holdLessonSave(page: Page) {
	const save = { held: 0, reject: false, release: () => {} };
	const released = new Promise<void>((resolve) => (save.release = resolve));
	await page.route(lessonRecords, async (route) => {
		if (route.request().method() === 'GET') return route.fallback();
		save.held += 1;
		await released;
		return save.reject
			? route.fulfill({ status: 400, json: { status: 400, message: 'Injected', data: {} } })
			: route.fallback();
	});
	return save;
}

// Leaves Alpha's save held, its form cancelled, and a draft of Beta open.
async function leaveHeldSaveForAnotherLesson(page: Page, request: APIRequestContext) {
	const course = await ownedCourse(request);
	const moduleId = await course.addModule('Module', 1);
	await course.addLesson(moduleId, 'Alpha', 1);
	await course.addLesson(moduleId, 'Beta', 2);
	const save = await holdLessonSave(page);
	const title = page.locator('#lesson_title');

	const main = await openCurriculum(page, course.title);
	await editLesson(page, 0).click();
	await title.fill('Alpha as saved');
	await main.getByRole('button', { name: 'Save Lesson', exact: true }).click();
	await expect.poll(() => save.held).toBe(1);

	await main.getByRole('button', { name: 'Cancel', exact: true }).last().click();
	await expect(title).toHaveCount(0);
	await expect(editLesson(page, 0)).toBeDisabled();
	await expect(editLesson(page, 1)).toBeEnabled();

	await editLesson(page, 1).click();
	await expect(title).toHaveValue('Beta');
	await title.fill('Beta draft');
	await expect(editLesson(page, 0)).toBeDisabled();
	return { course, save, title };
}

test.describe('Curriculum deletes and held lesson saves', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
	});

	test.afterEach(async () => {
		for (const cleanup of cleanups.reverse()) await cleanup();
		cleanups = [];
	});

	test('a lesson and then a module that still has lessons delete from the page', async ({
		page,
		request
	}) => {
		const course = await ownedCourse(request);
		const doomed = await course.addModule('Doomed module', 1);
		const kept = await course.addModule('Kept module', 2);
		await course.addLesson(doomed, 'Doomed one', 1);
		await course.addLesson(doomed, 'Doomed two', 2);
		await course.addLesson(kept, 'Kept lesson', 1);

		const main = await openCurriculum(page, course.title);
		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Lesson'))).toBe(200);
		await expect(page.getByText('Lesson deleted')).toBeVisible();
		await expect(main.getByText('Doomed one', { exact: true })).toHaveCount(0);
		expect((await course.stored('lessons')).sort()).toEqual(['Doomed two', 'Kept lesson']);

		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Module'))).toBe(200);
		await expect(page.getByText('Module deleted')).toBeVisible();
		await expect(main.getByText('Doomed two', { exact: true })).toHaveCount(0);
		await expect(main.getByText('Kept lesson', { exact: true })).toBeVisible();
		expect(await course.stored('modules')).toEqual(['Kept module']);
		expect(await course.stored('lessons')).toEqual(['Kept lesson']);
	});

	test('a module delete takes its lessons and their learner progress with it', async ({
		page,
		request,
		baseURL
	}) => {
		const course = await ownedCourse(request);
		const doomed = await course.addModule('Doomed module', 1);
		const kept = await course.addModule('Kept module', 2);
		const finished = await course.addLesson(doomed, 'Finished lesson', 1);
		const keptLesson = await course.addLesson(kept, 'Kept lesson', 1);
		const learner = await realLearner(request, baseURL!, course.slug);
		await learner.complete(finished);
		await learner.complete(keptLesson);
		expect((await learner.completedLessons()).sort()).toEqual([finished, keptLesson].sort());

		const main = await openCurriculum(page, course.title);
		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Module'))).toBe(200);
		await expect(page.getByText('Module deleted')).toBeVisible();

		expect(await course.stored('modules')).toEqual(['Kept module']);
		expect(await course.stored('lessons')).toEqual(['Kept lesson']);
		expect(await learner.completedLessons()).toEqual([keptLesson]);
	});

	test('quiz attempts refuse both deletes and keep the records, list and draft', async ({
		page,
		request,
		baseURL
	}) => {
		const course = await ownedCourse(request);
		const moduleId = await course.addModule('Attempted module', 1);
		const lessonId = await course.addLesson(moduleId, 'Attempted lesson', 1, 'quiz');
		const questionId = await course.addQuiz(lessonId);
		const learner = await realLearner(request, baseURL!, course.slug);
		await learner.attempt(lessonId, questionId);

		const main = await openCurriculum(page, course.title);
		const title = page.locator('#lesson_title');
		await editLesson(page).click();
		await expect(page.locator('#quiz-question-0')).toHaveValue('Pick blue');
		await title.fill('Draft kept after refusal');

		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Lesson'))).toBe(409);
		await expect(page.getByText("This lesson can't be deleted")).toBeVisible();
		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Module'))).toBe(409);
		await expect(page.getByText("This module can't be deleted")).toBeVisible();

		await expect(page.getByText('Lesson deleted')).toHaveCount(0);
		await expect(page.getByText('Module deleted')).toHaveCount(0);
		await expect(title).toHaveValue('Draft kept after refusal');
		await expect(main.getByText('Attempted lesson', { exact: true })).toBeVisible();
		await expect(main.getByText('Attempted module', { exact: true })).toBeVisible();
		expect(await course.stored('modules')).toEqual(['Attempted module']);
		expect(await course.stored('lessons')).toEqual(['Attempted lesson']);
		expect(await learner.attemptsUsed(lessonId)).toBe(1);
	});

	test('a failed delete keeps the draft, and a later success closes it', async ({
		page,
		request
	}) => {
		const course = await ownedCourse(request);
		const moduleId = await course.addModule('Module', 1);
		await course.addLesson(moduleId, 'Kept row', 1);
		await page.route(moduleDeletes, (route) =>
			route.request().method() === 'DELETE'
				? route.fulfill({ status: 500, json: { error: 'failed to delete module' } })
				: route.fallback()
		);

		const main = await openCurriculum(page, course.title);
		const title = page.locator('#lesson_title');
		await editLesson(page).click();
		await title.fill('Unsaved draft');

		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Module'))).toBe(500);
		await expect(page.getByText('Failed to delete module')).toBeVisible();
		await expect(page.getByText('Module deleted')).toHaveCount(0);
		await expect(title).toHaveValue('Unsaved draft');
		await expect(main.getByText('Kept row', { exact: true })).toBeVisible();

		await page.unroute(moduleDeletes);
		expect(await deleteAndReadStatus(page, deleteButton(main, 'Delete Module'))).toBe(200);
		await expect(page.getByText('Module deleted')).toBeVisible();
		await expect(title).toHaveCount(0);
		expect(await course.stored('modules')).toEqual([]);
	});

	test('a delete whose reply is lost reads the list instead of claiming success', async ({
		page,
		request
	}) => {
		const course = await ownedCourse(request);
		const moduleId = await course.addModule('Module', 1);
		await course.addLesson(moduleId, 'Landed lesson', 1);
		await page.route(lessonDeletes, async (route) => {
			if (route.request().method() !== 'DELETE') return route.fallback();
			await route.fetch();
			return route.abort('connectionfailed');
		});

		const main = await openCurriculum(page, course.title);
		await deleteButton(main, 'Delete Lesson').click();
		await confirmDelete(page);
		await expect(page.getByText('Failed to delete lesson')).toBeVisible();
		await expect(page.getByText('Lesson deleted')).toHaveCount(0);
		await expect(main.getByText('Landed lesson', { exact: true })).toHaveCount(0);
		expect(await course.stored('lessons')).toEqual([]);
	});

	test('a held save locks only its own lesson and unlocks once it lands', async ({
		page,
		request
	}) => {
		const { course, save, title } = await leaveHeldSaveForAnotherLesson(page, request);

		save.release();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		await expect(editLesson(page, 0)).toBeEnabled();
		await expect(title).toHaveValue('Beta draft');
		expect(await course.stored('lessons')).toEqual(['Alpha as saved', 'Beta']);

		await editLesson(page, 0).click();
		await expect(title).toHaveValue('Alpha as saved');
	});

	test('a held save that fails unlocks its lesson on what is stored', async ({ page, request }) => {
		const { course, save, title } = await leaveHeldSaveForAnotherLesson(page, request);

		save.reject = true;
		save.release();
		await expect(page.getByText('Failed to save lesson')).toBeVisible();
		await expect(editLesson(page, 0)).toBeEnabled();
		await expect(title).toHaveValue('Beta draft');
		expect(await course.stored('lessons')).toEqual(['Alpha', 'Beta']);

		await editLesson(page, 0).click();
		await expect(title).toHaveValue('Alpha');
	});
});

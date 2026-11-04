import { randomUUID } from 'node:crypto';
import {
	expect,
	test,
	type APIRequestContext,
	type Locator,
	type Page,
	type Route
} from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

const quizApi = '**/api/admin/courses/*/lessons/*/quiz';
const studentsApi = '**/api/admin/courses/*/students';
const noQuiz = { status: 404, json: { error: 'no quiz found for this lesson' } };
const moduleId = 'moduleone000000';
const lessonId = 'lessonone000000';
const savedLessonId = 'lessonsaved0000';

type Row = { id: string; title: string; sort_order: number; [field: string]: unknown };

function recordsApi(collection: string) {
	return new RegExp(`/api/collections/${collection}/records(?:/([^/?]+))?(?:\\?|$)`);
}

function paged(items: unknown[]) {
	return { page: 1, perPage: 500, totalItems: items.length, totalPages: 1, items };
}

function courseRow(id: string, title: string): Row {
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

function moduleRow(id: string, title: string, sort_order: number): Row {
	return { id, title, description: '', course: 'courseone000000', sort_order };
}

function lessonRow(id: string, title: string, content_type = 'quiz'): Row {
	return {
		id,
		title,
		content_type,
		module: moduleId,
		video_url: '',
		content: '',
		estimated_minutes: 0,
		is_preview: false,
		sort_order: 1
	};
}

function storedQuiz(lesson: string, title: string, questionText: string) {
	return {
		id: `quiz${lesson.slice(0, 11)}`,
		lesson,
		title,
		passing_score: 80,
		randomize_questions: false,
		max_attempts: 2,
		questions: [
			{
				id: `question${lesson.slice(0, 7)}`,
				question_text: questionText,
				question_type: 'multiple_choice',
				options: [
					{ text: 'Right', is_correct: true },
					{ text: 'Wrong', is_correct: false }
				],
				explanation: '',
				points: 10,
				sort_order: 1
			}
		]
	};
}

// Serves the course screens from memory and returns each lesson write as "METHOD id".
async function mockCourses(
	page: Page,
	modules: Row[],
	lessons: Row[],
	courses = [courseRow('courseone000000', 'Course One')]
) {
	const lessonWrites: string[] = [];
	await page.route(recordsApi('courses'), (route) => route.fulfill({ json: paged(courses) }));
	await page.route(recordsApi('modules'), (route) =>
		route.fulfill({ json: paged([...modules].sort((a, b) => a.sort_order - b.sort_order)) })
	);
	await page.route(recordsApi('lessons'), (route) => {
		const method = route.request().method();
		if (method === 'GET') return route.fulfill({ json: paged(lessons) });
		const id = recordsApi('lessons').exec(route.request().url())?.[1] ?? '';
		lessonWrites.push(`${method} ${id}`.trim());
		const record = lessons.find((lesson) => lesson.id === id) ?? lessonRow(savedLessonId, 'Saved');
		if (!id) lessons.push(record);
		return route.fulfill({ json: record });
	});
	return lessonWrites;
}

async function openCurriculum(page: Page) {
	await page.goto('/admin/courses');
	const main = page.getByRole('main');
	await main.getByRole('button', { name: 'Edit', exact: true }).click();
	await main.getByRole('button', { name: 'Curriculum' }).click();
	return main;
}

function editLesson(page: Page, index = 0) {
	return page.getByRole('button', { name: 'Edit Lesson', exact: true }).nth(index);
}

function saveLesson(main: Locator) {
	return main.getByRole('button', { name: 'Save Lesson', exact: true });
}

async function draftNewQuizLesson(page: Page, main: Locator) {
	await main.getByRole('button', { name: /Add Lesson/ }).click();
	await page.locator('#lesson_title').fill('New quiz lesson');
	await page.locator('#lesson_content_type').selectOption('quiz');
	await main.getByRole('button', { name: /Add Question/ }).click();
	await page.locator('#quiz-question-0').fill('What colour is the sky?');
}

function method(route: Route) {
	return route.request().method();
}

// A real course and module, for tests that check what the server stored.
async function storedCourse(request: APIRequestContext) {
	const { token } = await loginAsAdmin(request);
	const headers = { Authorization: token };
	const run = randomUUID();
	const title = `Lost reply ${run}`;
	const courseResponse = await request.post(`${apiBaseURL}/api/collections/courses/records`, {
		headers,
		data: {
			title,
			slug: `lost-reply-${run}`,
			access_tier: 'free',
			visibility: 'private',
			is_draft: true
		}
	});
	expect(courseResponse.ok(), await courseResponse.text()).toBe(true);
	const { id } = (await courseResponse.json()) as { id: string };
	const moduleResponse = await request.post(`${apiBaseURL}/api/admin/courses/${id}/modules`, {
		headers,
		data: { title: 'Module', sort_order: 0 }
	});
	expect(moduleResponse.ok(), await moduleResponse.text()).toBe(true);
	return {
		title,
		async lessons() {
			const response = await request.get(`${apiBaseURL}/api/collections/lessons/records`, {
				headers,
				params: { filter: `module.course="${id}"` }
			});
			expect(response.ok(), await response.text()).toBe(true);
			return (await response.json()).items as Row[];
		},
		remove: () => request.delete(`${apiBaseURL}/api/collections/courses/records/${id}`, { headers })
	};
}

async function openStoredCurriculum(page: Page, title: string) {
	await page.goto('/admin/courses');
	const main = page.getByRole('main');
	const card = main.locator('.card', { hasText: title });
	await card.getByRole('button', { name: 'Edit', exact: true }).click();
	await main.getByRole('button', { name: 'Curriculum' }).click();
	return main;
}

// Sends the lesson create to the real server, then drops the reply.
async function loseLessonResponse(page: Page) {
	const lost = { posts: 0, failLookup: false };
	await page.route(recordsApi('lessons'), async (route) => {
		if (method(route) === 'POST') {
			lost.posts += 1;
			await route.fetch();
			return route.abort('connectionfailed');
		}
		const id = recordsApi('lessons').exec(route.request().url())?.[1];
		return id && lost.failLookup
			? route.fulfill({ status: 500, json: { status: 500, message: 'Injected', data: {} } })
			: route.continue();
	});
	return lost;
}

// Accepts the quiz write and returns the lessons it was sent for.
async function mockQuizSave(page: Page) {
	const quizLessons: string[] = [];
	await page.route(quizApi, (route) => {
		if (method(route) === 'GET') return route.fulfill(noQuiz);
		quizLessons.push(/lessons\/([^/]+)\/quiz/.exec(route.request().url())?.[1] ?? '');
		return route.fulfill({ status: 201, json: { id: 'quizsaved000000', title: 'New' } });
	});
	return quizLessons;
}

// Holds lesson writes until release(), counting the ones that arrive.
async function holdLessonWrites(page: Page) {
	const writes = { held: 0, release: () => {} };
	const released = new Promise<void>((resolve) => (writes.release = resolve));
	await page.route(recordsApi('lessons'), async (route) => {
		if (method(route) === 'GET') return route.fallback();
		writes.held += 1;
		await released;
		return route.fallback();
	});
	return writes;
}

// Records which attachments each lesson write carried, as the page built the form.
async function recordAttachments(page: Page) {
	await page.addInitScript(() => {
		const sent: string[][] = [];
		Object.assign(window, { sentAttachments: sent });
		const send = window.fetch.bind(window);
		window.fetch = (input, init) => {
			if (String(input).includes('/collections/lessons/') && init?.body instanceof FormData) {
				sent.push(init.body.getAll('attachments').map((file) => (file as File).name));
			}
			return send(input, init);
		};
	});
	return () =>
		page.evaluate(() => (window as unknown as { sentAttachments: string[][] }).sentAttachments);
}

test.describe('Course drafts survive failed reads and saves', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
	});

	test('a failed quiz read blocks saving until Retry loads the real quiz', async ({ page }) => {
		await mockCourses(
			page,
			[moduleRow(moduleId, 'Module One', 1)],
			[lessonRow(lessonId, 'Quiz lesson')]
		);
		const known = storedQuiz(lessonId, 'Known quiz', 'Known question');
		const writes: { method: string; body: { questions: { id?: string }[] } }[] = [];
		let failRead = true;
		await page.route(quizApi, (route) => {
			if (method(route) !== 'GET') {
				writes.push({ method: method(route), body: route.request().postDataJSON() });
				return route.fulfill({ json: { id: known.id, title: known.title } });
			}
			return failRead
				? route.fulfill({ status: 500, json: { error: 'Injected read failure' } })
				: route.fulfill({ json: known });
		});

		const main = await openCurriculum(page);
		await editLesson(page).click();
		await page.locator('#lesson_title').fill('Typed lesson title');
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't load this lesson's quiz" });
		await expect(alert).toBeVisible();
		await expect(page.locator('#quiz_title')).toHaveCount(0);
		await expect(saveLesson(main)).toBeDisabled();

		failRead = false;
		await alert.getByRole('button', { name: 'Retry' }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.locator('#lesson_title')).toHaveValue('Typed lesson title');
		await expect(page.locator('#quiz_title')).toHaveValue('Known quiz');
		await expect(page.locator('#quiz-question-0')).toHaveValue('Known question');

		await page.locator('#quiz-question-0').fill('Edited question');
		await saveLesson(main).click();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		expect(writes.map((write) => write.method)).toEqual(['PATCH']);
		expect(writes[0].body.questions[0].id).toBe(known.questions[0].id);
	});

	for (const [name, reply] of [
		['a malformed quiz', { json: { id: 'quizbad', lesson: lessonId, questions: 'nope' } }],
		['a quiz for another lesson', { json: storedQuiz('otherlesson0000', 'Other', 'Other') }],
		['a missing lesson', { status: 404, json: { error: 'lesson not found in this course' } }]
	] as const) {
		test(`${name} is a read failure, not an empty quiz`, async ({ page }) => {
			await mockCourses(
				page,
				[moduleRow(moduleId, 'Module One', 1)],
				[lessonRow(lessonId, 'Quiz lesson')]
			);
			await page.route(quizApi, (route) => route.fulfill(reply));

			const main = await openCurriculum(page);
			await editLesson(page).click();
			await expect(page.getByText("Couldn't load this lesson's quiz")).toBeVisible();
			await expect(page.locator('#quiz_title')).toHaveCount(0);
			await expect(saveLesson(main)).toBeDisabled();
		});
	}

	test('a confirmed "no quiz yet" opens an empty builder', async ({ page }) => {
		await mockCourses(
			page,
			[moduleRow(moduleId, 'Module One', 1)],
			[lessonRow(lessonId, 'Quiz lesson')]
		);
		await page.route(quizApi, (route) => route.fulfill(noQuiz));

		const main = await openCurriculum(page);
		await editLesson(page).click();
		await expect(page.locator('#quiz_title')).toHaveValue('');
		await expect(saveLesson(main)).toBeEnabled();
	});

	test('switching a lesson to quiz reads what is stored first', async ({ page }) => {
		await mockCourses(
			page,
			[moduleRow(moduleId, 'Module One', 1)],
			[lessonRow(lessonId, 'Text lesson', 'text')]
		);
		await page.route(quizApi, (route) =>
			route.fulfill({ json: storedQuiz(lessonId, 'Old quiz', 'Old question') })
		);

		await openCurriculum(page);
		await editLesson(page).click();
		await page.locator('#lesson_content_type').selectOption('quiz');
		await expect(page.locator('#quiz-question-0')).toHaveValue('Old question');
	});

	test('a failed quiz save keeps the draft and the next save updates the same lesson', async ({
		page
	}) => {
		const lessonWrites = await mockCourses(page, [moduleRow(moduleId, 'Module One', 1)], []);
		const quizPosts: { questions: { sort_order: number }[] }[] = [];
		await page.route(quizApi, (route) => {
			quizPosts.push(route.request().postDataJSON());
			return quizPosts.length === 1
				? route.fulfill({ status: 400, json: { error: 'Injected rejection' } })
				: route.fulfill({ status: 201, json: { id: 'quizsaved000000', title: 'New' } });
		});

		const main = await openCurriculum(page);
		await draftNewQuizLesson(page, main);
		await saveLesson(main).click();
		await expect(page.getByRole('alert').filter({ hasText: 'still here' })).toBeVisible();
		await expect(page.getByText('Lesson created')).toHaveCount(0);
		await expect(page.locator('#quiz-question-0')).toHaveValue('What colour is the sky?');
		await expect(page.locator('#lesson_title')).toHaveValue('New quiz lesson');
		expect(quizPosts[0].questions[0].sort_order).toBe(1);

		await saveLesson(main).click();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		await expect(page.locator('#lesson_title')).toHaveCount(0);
		expect(lessonWrites).toEqual(['POST', `PATCH ${savedLessonId}`]);
		expect(quizPosts).toHaveLength(2);
	});

	test('a quiz create that landed with a lost response is not sent again', async ({ page }) => {
		const lessonWrites = await mockCourses(page, [moduleRow(moduleId, 'Module One', 1)], []);
		let posts = 0;
		let stored: unknown = null;
		await page.route(quizApi, (route) => {
			if (method(route) === 'GET') {
				return stored ? route.fulfill({ json: stored }) : route.fulfill(noQuiz);
			}
			posts += 1;
			const payload = route.request().postDataJSON();
			stored = {
				...payload,
				id: 'quizlanded00000',
				lesson: savedLessonId,
				questions: payload.questions.map((question: object, index: number) => ({
					...question,
					id: `question${index}`
				}))
			};
			return route.abort('connectionfailed');
		});

		const main = await openCurriculum(page);
		await draftNewQuizLesson(page, main);
		await saveLesson(main).click();
		await expect(page.getByText('Lesson created')).toBeVisible();
		await expect(page.locator('#lesson_title')).toHaveCount(0);
		expect(posts).toBe(1);
		expect(lessonWrites).toEqual(['POST']);
	});

	test('a lesson create that landed with a lost reply is confirmed, not repeated', async ({
		page,
		request
	}) => {
		const course = await storedCourse(request);
		try {
			const lost = await loseLessonResponse(page);
			const quizLessons = await mockQuizSave(page);
			const main = await openStoredCurriculum(page, course.title);
			await draftNewQuizLesson(page, main);
			await saveLesson(main).click();
			await expect(page.getByText('Lesson created')).toBeVisible();

			const lessons = await course.lessons();
			expect(lost.posts).toBe(1);
			expect(lessons.map((lesson) => lesson.title)).toEqual(['New quiz lesson']);
			expect(lessons[0].id).toMatch(/^[a-z0-9]{15}$/);
			expect(quizLessons).toEqual([lessons[0].id]);
		} finally {
			await course.remove();
		}
	});

	test('a failed lesson check keeps the draft until Retry finds the lesson', async ({
		page,
		request
	}) => {
		const course = await storedCourse(request);
		try {
			const lost = await loseLessonResponse(page);
			lost.failLookup = true;
			const quizLessons = await mockQuizSave(page);
			const main = await openStoredCurriculum(page, course.title);
			await draftNewQuizLesson(page, main);
			await saveLesson(main).click();
			const alert = page.getByRole('alert').filter({ hasText: "couldn't confirm" });
			await expect(alert).toBeVisible();
			await expect(saveLesson(main)).toBeDisabled();
			await expect(page.locator('#lesson_title')).toHaveValue('New quiz lesson');
			await expect(page.locator('#quiz-question-0')).toHaveValue('What colour is the sky?');
			expect(quizLessons).toEqual([]);

			lost.failLookup = false;
			await alert.getByRole('button', { name: 'Retry' }).click();
			await expect(page.getByText('Lesson created')).toBeVisible();
			await expect(page.locator('#lesson_title')).toHaveCount(0);
			const lessons = await course.lessons();
			expect(lost.posts).toBe(1);
			expect(lessons).toHaveLength(1);
			expect(quizLessons).toEqual([lessons[0].id]);
		} finally {
			await course.remove();
		}
	});

	test('edits after a lost create wait for the next Save', async ({ page, request }) => {
		const course = await storedCourse(request);
		try {
			const lost = await loseLessonResponse(page);
			lost.failLookup = true;
			await mockQuizSave(page);
			const main = await openStoredCurriculum(page, course.title);
			await draftNewQuizLesson(page, main);
			await saveLesson(main).click();
			const alert = page.getByRole('alert').filter({ hasText: "couldn't confirm" });
			await expect(alert).toBeVisible();

			await page.locator('#lesson_title').fill('Edited after the lost reply');
			lost.failLookup = false;
			await alert.getByRole('button', { name: 'Retry' }).click();
			await expect(page.getByRole('alert').filter({ hasText: 'earlier version' })).toBeVisible();
			await expect(page.getByText('Lesson created')).toHaveCount(0);
			await expect(page.locator('#lesson_title')).toHaveValue('Edited after the lost reply');
			await expect(page.locator('#quiz-question-0')).toHaveValue('What colour is the sky?');
			let lessons = await course.lessons();
			expect(lessons.map((lesson) => lesson.title)).toEqual(['New quiz lesson']);

			await saveLesson(main).click();
			await expect(page.getByText('Lesson updated')).toBeVisible();
			lessons = await course.lessons();
			expect(lost.posts).toBe(1);
			expect(lessons.map((lesson) => lesson.title)).toEqual(['Edited after the lost reply']);
		} finally {
			await course.remove();
		}
	});

	test('a create that never landed resends its ID, and a reopened form gets its own', async ({
		page
	}) => {
		await mockCourses(page, [moduleRow(moduleId, 'Module One', 1)], []);
		const ids: string[] = [];
		await page.route(recordsApi('lessons'), (route) => {
			if (method(route) === 'POST') {
				ids.push(/name="id"\r\n\r\n(\w+)/.exec(route.request().postData() ?? '')?.[1] ?? '');
				return ids.length < 3 ? route.abort('connectionfailed') : route.fallback();
			}
			return recordsApi('lessons').exec(route.request().url())?.[1]
				? route.fulfill({ status: 404, json: { status: 404, message: 'Missing', data: {} } })
				: route.fallback();
		});

		const main = await openCurriculum(page);
		await main.getByRole('button', { name: /Add Lesson/ }).click();
		await page.locator('#lesson_title').fill('Lost lesson');
		await saveLesson(main).click();
		await expect.poll(() => ids.length).toBe(1);
		await expect(saveLesson(main)).toBeEnabled();
		await saveLesson(main).click();
		await expect.poll(() => ids.length).toBe(2);
		await expect(saveLesson(main)).toBeEnabled();
		await expect(page.locator('#lesson_title')).toHaveValue('Lost lesson');
		const cancel = main.getByRole('button', { name: 'Cancel', exact: true });
		await cancel.filter({ hasText: 'Cancel' }).click();

		await main.getByRole('button', { name: /Add Lesson/ }).click();
		await expect(page.locator('#lesson_title')).toHaveValue('');
		await page.locator('#lesson_title').fill('Fresh lesson');
		await saveLesson(main).click();
		await expect(page.getByText('Lesson created')).toBeVisible();
		expect(ids[0]).toMatch(/^[a-z0-9]{15}$/);
		expect(ids[1]).toBe(ids[0]);
		expect(ids[2]).not.toBe(ids[0]);
	});

	test('a lost create with attachments is confirmed without sending them again', async ({
		page
	}) => {
		await mockCourses(page, [moduleRow(moduleId, 'Module One', 1)], []);
		let posts = 0;
		await page.route(recordsApi('lessons'), (route) => {
			if (method(route) === 'POST') {
				posts += 1;
				return route.abort('connectionfailed');
			}
			const id = recordsApi('lessons').exec(route.request().url())?.[1];
			if (!id) return route.fallback();
			const stored = lessonRow(id, 'Lesson with notes', 'mixed');
			return route.fulfill({ json: { ...stored, attachments: ['notes.txt'] } });
		});

		const main = await openCurriculum(page);
		await main.getByRole('button', { name: /Add Lesson/ }).click();
		await page.locator('#lesson_title').fill('Lesson with notes');
		await page.locator('#lesson_attachments').setInputFiles({
			name: 'notes.txt',
			mimeType: 'text/plain',
			buffer: Buffer.from('Notes')
		});
		await saveLesson(main).click();
		await expect(page.getByText('Lesson created')).toBeVisible();
		expect(posts).toBe(1);
	});

	test('a held save writes the quiz it was pressed with and leaves the next form alone', async ({
		page
	}) => {
		const alpha = 'lessonaaa000000';
		const beta = 'lessonbbb000000';
		await mockCourses(
			page,
			[moduleRow(moduleId, 'Module One', 1)],
			[lessonRow(alpha, 'Lesson Alpha'), lessonRow(beta, 'Lesson Beta')]
		);
		const stored = {
			[alpha]: storedQuiz(alpha, 'Alpha quiz', 'Alpha question'),
			[beta]: storedQuiz(beta, 'Beta quiz', 'Beta question')
		};
		const quizWrites: {
			lesson: string;
			method: string;
			body: ReturnType<typeof storedQuiz>;
		}[] = [];
		await page.route(quizApi, (route) => {
			const lesson = route.request().url().includes(alpha) ? alpha : beta;
			if (method(route) === 'GET') return route.fulfill({ json: stored[lesson] });
			quizWrites.push({ lesson, method: method(route), body: route.request().postDataJSON() });
			return route.fulfill({ json: { id: stored[lesson].id, title: 'Saved' } });
		});
		const writes = await holdLessonWrites(page);
		const question = page.locator('#quiz-question-0');
		const answer = page.getByPlaceholder('Option 1');

		const main = await openCurriculum(page);
		await editLesson(page, 0).click();
		await expect(question).toHaveValue('Alpha question');
		await question.fill('Alpha as pressed');
		await answer.fill('Alpha answer');
		await saveLesson(main).click();
		await expect.poll(() => writes.held).toBe(1);

		await main.getByRole('button', { name: 'Cancel', exact: true }).last().click();
		await expect(page.locator('#lesson_title')).toHaveCount(0);
		await editLesson(page, 1).click();
		await expect(question).toHaveValue('Beta question');
		await page.locator('#quiz_title').fill('Beta draft');
		await question.fill('Beta typed while Alpha saves');
		await answer.fill('Beta answer');

		writes.release();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		expect(quizWrites).toHaveLength(1);
		const [write] = quizWrites;
		expect([write.lesson, write.method, write.body.title]).toEqual([alpha, 'PATCH', 'Alpha quiz']);
		expect(write.body.questions[0].id).toBe(stored[alpha].questions[0].id);
		expect(write.body.questions[0].question_text).toBe('Alpha as pressed');
		expect(write.body.questions[0].options[0].text).toBe('Alpha answer');

		await expect(page.locator('#lesson_title')).toHaveValue('Lesson Beta');
		await expect(page.locator('#quiz_title')).toHaveValue('Beta draft');
		await expect(question).toHaveValue('Beta typed while Alpha saves');
		await expect(answer).toHaveValue('Beta answer');
		await expect(saveLesson(main)).toBeEnabled();
		expect(quizWrites).toHaveLength(1);
	});

	test('the form locks while its save is pending and reopens when the save fails', async ({
		page
	}) => {
		const lessonWrites = await mockCourses(page, [moduleRow(moduleId, 'Module One', 1)], []);
		let quizPosts = 0;
		await page.route(quizApi, (route) => {
			quizPosts += 1;
			return quizPosts === 1
				? route.fulfill({ status: 400, json: { error: 'Injected rejection' } })
				: route.fulfill({ status: 201, json: { id: 'quizsaved000000', title: 'New' } });
		});
		const writes = await holdLessonWrites(page);

		const main = await openCurriculum(page);
		await draftNewQuizLesson(page, main);
		await saveLesson(main).click();
		await expect.poll(() => writes.held).toBe(1);
		for (const field of [
			'#lesson_title',
			'#lesson_content_type',
			'#lesson_is_preview',
			'#lesson_attachments',
			'#quiz_title',
			'#quiz-question-0'
		]) {
			await expect(page.locator(field)).toBeDisabled();
		}
		await expect(main.getByRole('button', { name: /Add Question/ })).toBeDisabled();

		writes.release();
		await expect(page.getByRole('alert').filter({ hasText: 'still here' })).toBeVisible();
		await expect(page.locator('#quiz-question-0')).toHaveValue('What colour is the sky?');
		await page.locator('#lesson_title').fill('Edited after the failure');
		await saveLesson(main).click();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		expect(lessonWrites).toEqual(['POST', `PATCH ${savedLessonId}`]);
		expect(quizPosts).toBe(2);
	});

	test('a retry skips files already stored and sends ones chosen since', async ({ page }) => {
		const attachments = await recordAttachments(page);
		await mockCourses(page, [moduleRow(moduleId, 'Module One', 1)], []);
		let rejections = 2;
		await page.route(quizApi, (route) => {
			if (rejections === 0) {
				return route.fulfill({ status: 201, json: { id: 'quizsaved000000', title: 'New' } });
			}
			rejections -= 1;
			return route.fulfill({ status: 400, json: { error: 'Injected rejection' } });
		});
		const file = (name: string) => ({
			name,
			mimeType: 'text/plain',
			buffer: Buffer.from('Notes')
		});

		const main = await openCurriculum(page);
		await draftNewQuizLesson(page, main);
		await page.locator('#lesson_attachments').setInputFiles(file('notes.txt'));
		await saveLesson(main).click();
		await expect(page.getByRole('alert').filter({ hasText: 'still here' })).toBeVisible();

		await saveLesson(main).click();
		await expect.poll(attachments).toEqual([['notes.txt'], []]);
		await expect(saveLesson(main)).toBeEnabled();

		await page.locator('#lesson_attachments').setInputFiles(file('newer.txt'));
		await saveLesson(main).click();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		expect(await attachments()).toEqual([['notes.txt'], [], ['newer.txt']]);
	});

	test('files chosen in one lesson form do not follow to the next', async ({ page }) => {
		const attachments = await recordAttachments(page);
		const beta = 'lessonbbb000000';
		const lessonWrites = await mockCourses(
			page,
			[moduleRow(moduleId, 'Module One', 1)],
			[lessonRow('lessonaaa000000', 'Lesson Alpha', 'text'), lessonRow(beta, 'Lesson Beta', 'text')]
		);

		const main = await openCurriculum(page);
		await editLesson(page, 0).click();
		await page.locator('#lesson_attachments').setInputFiles({
			name: 'alpha.txt',
			mimeType: 'text/plain',
			buffer: Buffer.from('Alpha')
		});
		await editLesson(page, 1).click();
		await expect(page.locator('#lesson_title')).toHaveValue('Lesson Beta');
		await saveLesson(main).click();
		await expect(page.getByText('Lesson updated')).toBeVisible();
		expect(lessonWrites).toEqual([`PATCH ${beta}`]);
		expect(await attachments()).toEqual([[]]);
	});

	test('a late quiz response cannot replace the open lesson', async ({ page }) => {
		await mockCourses(
			page,
			[moduleRow(moduleId, 'Module One', 1)],
			[lessonRow('lessonaaa000000', 'Lesson Alpha'), lessonRow('lessonbbb000000', 'Lesson Beta')]
		);
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		let late: Promise<unknown> = Promise.resolve();
		await page.route(quizApi, (route) => {
			const alpha = route.request().url().includes('lessonaaa000000');
			const lesson = alpha ? 'lessonaaa000000' : 'lessonbbb000000';
			const json = storedQuiz(lesson, 'Quiz', alpha ? 'Alpha question' : 'Beta question');
			if (!alpha) return route.fulfill({ json });
			late = held.then(() => route.fulfill({ json })).catch(() => {});
			return late;
		});

		await openCurriculum(page);
		await editLesson(page, 0).click();
		await editLesson(page, 1).click();
		await expect(page.locator('#quiz-question-0')).toHaveValue('Beta question');

		release();
		await late;
		await expect(page.locator('#quiz-question-0')).toHaveValue('Beta question');
	});

	test('a failed students read offers Retry instead of an empty list', async ({ page }) => {
		await mockCourses(page, [], []);
		let failRead = true;
		await page.route(studentsApi, (route) =>
			failRead
				? route.fulfill({ status: 500, json: { error: 'Injected read failure' } })
				: route.fulfill({ json: enrollment('known@example.test') })
		);

		await page.goto('/admin/courses');
		const main = page.getByRole('main');
		await main.getByRole('button', { name: 'Edit', exact: true }).click();
		await main.getByRole('button', { name: 'Enrolled Students', exact: true }).click();
		const alert = page.getByRole('alert').filter({ hasText: 'Failed to load enrollment data' });
		await expect(alert).toBeVisible();
		await expect(page.getByText('No students enrolled yet.')).toHaveCount(0);

		failRead = false;
		await alert.getByRole('button', { name: 'Retry' }).click();
		await expect(alert).toHaveCount(0);
		await expect(page.getByText('known@example.test')).toBeVisible();
	});

	test('a late students response cannot replace the open course', async ({ page }) => {
		await mockCourses(
			page,
			[],
			[],
			[courseRow('courseaaa000000', 'Course Alpha'), courseRow('coursebbb000000', 'Course Beta')]
		);
		let release = () => {};
		const held = new Promise<void>((resolve) => (release = resolve));
		let late: Promise<unknown> = Promise.resolve();
		await page.route(studentsApi, (route) => {
			const alpha = route.request().url().includes('courseaaa000000');
			const json = enrollment(alpha ? 'alpha@example.test' : 'beta@example.test');
			if (!alpha) return route.fulfill({ json });
			late = held.then(() => route.fulfill({ json })).catch(() => {});
			return late;
		});

		await page.goto('/admin/courses');
		const main = page.getByRole('main');
		const edit = (title: string) =>
			main.locator('.card', { hasText: title }).getByRole('button', { name: 'Edit', exact: true });
		const studentsTab = main.getByRole('button', { name: 'Enrolled Students', exact: true });
		await edit('Course Alpha').click();
		await studentsTab.click();
		await main.getByRole('button', { name: 'Back to Courses' }).click();
		await edit('Course Beta').click();
		await studentsTab.click();
		await expect(page.getByText('beta@example.test')).toBeVisible();

		release();
		await late;
		await expect(page.getByText('beta@example.test')).toBeVisible();
		await expect(page.getByText('alpha@example.test')).toHaveCount(0);
	});

	test('moving a module again finishes a partial reorder without repeating saved rows', async ({
		page
	}) => {
		const modules = [
			moduleRow('moduleaaa000000', 'Module A', 1),
			moduleRow('modulebbb000000', 'Module B', 2),
			moduleRow('moduleccc000000', 'Module C', 3)
		];
		await mockCourses(page, modules, []);
		const patched: string[] = [];
		let rejected = 'modulebbb000000';
		await page.route(recordsApi('modules'), (route) => {
			if (method(route) === 'GET') return route.fallback();
			const id = recordsApi('modules').exec(route.request().url())?.[1] ?? '';
			patched.push(id);
			if (id === rejected) {
				return route.fulfill({ status: 400, json: { message: 'Injected rejection' } });
			}
			const record = modules.find((candidate) => candidate.id === id)!;
			record.sort_order = route.request().postDataJSON().sort_order;
			return route.fulfill({ json: record });
		});

		const main = await openCurriculum(page);
		const moveCUp = main.getByRole('button', { name: 'Reorder Modules', exact: true }).nth(4);
		await moveCUp.click();
		await expect(page.getByText(/Saved 1 of 2 positions/)).toBeVisible();
		expect([...patched].sort()).toEqual(['modulebbb000000', 'moduleccc000000']);
		expect(modules.map((module) => module.sort_order)).toEqual([1, 2, 2]);

		rejected = '';
		patched.length = 0;
		await expect(moveCUp).toBeEnabled();
		await moveCUp.click();
		await expect.poll(() => modules.map((module) => module.sort_order)).toEqual([1, 3, 2]);
		expect(patched).toEqual(['modulebbb000000']);
	});

	test('a rejected question leaves the stored quiz untouched', async ({ request }) => {
		const { token } = await loginAsAdmin(request);
		const headers = { Authorization: token };
		const run = randomUUID();
		const courseResponse = await request.post(`${apiBaseURL}/api/collections/courses/records`, {
			headers,
			data: {
				title: `Atomic quiz ${run}`,
				slug: `atomic-quiz-${run}`,
				access_tier: 'free',
				visibility: 'private',
				is_draft: true
			}
		});
		expect(courseResponse.ok(), await courseResponse.text()).toBe(true);
		const { id: courseId } = (await courseResponse.json()) as { id: string };

		try {
			const courseApi = `${apiBaseURL}/api/admin/courses/${courseId}`;
			const moduleResponse = await request.post(`${courseApi}/modules`, {
				headers,
				data: { title: 'Module', sort_order: 0 }
			});
			expect(moduleResponse.ok(), await moduleResponse.text()).toBe(true);
			const { id: storedModule } = (await moduleResponse.json()) as { id: string };
			const lessonResponse = await request.post(`${courseApi}/modules/${storedModule}/lessons`, {
				headers,
				data: { title: 'Lesson', content: '', content_type: 'text', sort_order: 0 }
			});
			expect(lessonResponse.ok(), await lessonResponse.text()).toBe(true);
			const { id: storedLesson } = (await lessonResponse.json()) as { id: string };
			const quizUrl = `${courseApi}/lessons/${storedLesson}/quiz`;
			const valid = {
				question_text: 'Kept question',
				question_type: 'multiple_choice',
				options: [
					{ text: 'Right', is_correct: true },
					{ text: 'Wrong', is_correct: false }
				],
				sort_order: 1
			};
			const blank = { ...valid, question_text: '', sort_order: 2 };

			const halfCreate = await request.post(quizUrl, {
				headers,
				data: { title: 'Half', questions: [valid, blank] }
			});
			expect(halfCreate.ok()).toBe(false);
			expect((await request.get(quizUrl, { headers })).status()).toBe(404);

			const create = await request.post(quizUrl, {
				headers,
				data: { title: 'Kept', questions: [valid] }
			});
			expect(create.ok(), await create.text()).toBe(true);

			const halfUpdate = await request.patch(quizUrl, {
				headers,
				data: { title: 'Changed', questions: [blank] }
			});
			expect(halfUpdate.ok()).toBe(false);
			const stored = (await (await request.get(quizUrl, { headers })).json()) as {
				title: string;
				questions: { question_text: string }[];
			};
			expect(stored.title).toBe('Kept');
			expect(stored.questions.map((question) => question.question_text)).toEqual(['Kept question']);
		} finally {
			await request.delete(`${apiBaseURL}/api/collections/courses/records/${courseId}`, {
				headers
			});
		}
	});
});

function enrollment(email: string) {
	return {
		students: [
			{
				email,
				progress_percent: 50,
				lessons_completed: 1,
				total_lessons: 2,
				started_at: '',
				completed_at: '',
				last_accessed: '',
				has_certificate: false
			}
		],
		stats: { total_enrolled: 1, completed: 0, average_progress: 50, completion_rate: 0 }
	};
}

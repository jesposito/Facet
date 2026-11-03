/**
 * Public course and checkout flows.
 *
 * Courses and paid content are seeded through the admin API. Checkout is never sent to Stripe:
 * POST /api/content/checkout is intercepted in the browser and the Stripe
 * page is stubbed. Access is only ever granted by the backend, so these tests
 * check that a locked item stays locked whatever the browser is told.
 *
 * Needs ADMIN_EMAIL and ADMIN_PASSWORD. The backend refuses to save paid
 * content unless a Stripe secret key is set. The disposable test backend
 * sets a placeholder key and these tests never let a request reach Stripe,
 * so with COMMERCE_FIXTURE=1 the paid tests must run and fail loudly if the
 * paid course can't be created. Without it (an arbitrary developer
 * backend) they skip instead.
 *
 * Enrolment and quiz answers are mocked in the browser where the real flow
 * needs an emailed link or a learner cookie the test can't obtain.
 */
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import { loginAsAdmin } from './helpers';

const requirePaid = process.env.COMMERCE_FIXTURE === '1';

const run = Date.now().toString(36);
const SECRET = `paid-lesson-secret-${run}`;
const PREVIEW = `preview-lesson-text-${run}`;
const FREE_TEXT = `free-lesson-text-${run}`;
const EXPLANATION = `quiz-explanation-${run}`;

let token = '';
const created: string[] = [];
let freeSlug = '';
let enrolSlug = '';
let quizCourse: { slug: string; lessonId: string } | null = null;
let paid: { id: string; slug: string } | null = null;
const paidContent: Array<{
	type: 'posts' | 'projects' | 'talks';
	id: string;
	slug: string;
	secret: string;
}> = [];

/** What the learner quiz endpoint sends: option labels, no answer key. */
const quizFixture = {
	id: 'quiz1',
	title: 'Check your understanding',
	passing_score: 50,
	max_attempts: 2,
	attempts_used: 0,
	questions: [
		{
			id: 'q1',
			question_text: 'Pick blue',
			question_type: 'multiple_choice',
			points: 10,
			options: [{ text: 'Red' }, { text: 'Blue' }]
		},
		{
			id: 'q2',
			question_text: 'The sky is green',
			question_type: 'true_false',
			points: 10,
			options: [{ text: 'True' }, { text: 'False' }]
		}
	]
};

async function createCourse(request: APIRequestContext, data: Record<string, unknown>) {
	const res = await request.post(`${apiBaseURL}/api/collections/courses/records`, {
		headers: { Authorization: token },
		data: { visibility: 'public', is_draft: false, ...data }
	});
	if (!res.ok()) return null;
	const record = await res.json();
	created.push(record.id);
	return record as { id: string; slug: string };
}

async function addLesson(
	request: APIRequestContext,
	courseId: string,
	moduleId: string,
	data: Record<string, unknown>
) {
	const res = await request.post(
		`${apiBaseURL}/api/admin/courses/${courseId}/modules/${moduleId}/lessons`,
		{
			headers: { Authorization: token },
			data: { content_type: 'text', ...data }
		}
	);
	expect(res.ok(), 'create lesson').toBeTruthy();
	return (await res.json()).id as string;
}

async function openQuizLesson(page: Page) {
	await page.goto(`/courses/${quizCourse!.slug}/learn?lesson=${quizCourse!.lessonId}`);
}

async function addModule(request: APIRequestContext, courseId: string, title: string) {
	const res = await request.post(`${apiBaseURL}/api/admin/courses/${courseId}/modules`, {
		headers: { Authorization: token },
		data: { title, sort_order: 0 }
	});
	expect(res.ok(), 'create module').toBeTruthy();
	return (await res.json()).id as string;
}

test.describe('public courses and checkout', () => {
	test.skip(!adminEmail || !adminPassword, 'ADMIN_EMAIL and ADMIN_PASSWORD are required');

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;

		const free = await createCourse(request, {
			title: `Free course ${run}`,
			slug: `free-course-${run}`,
			access_tier: 'free',
			require_enrollment: false
		});
		expect(free, 'create free course').toBeTruthy();
		freeSlug = free!.slug;
		const freeModule = await addModule(request, free!.id, 'Getting started');
		await addLesson(request, free!.id, freeModule, {
			title: 'Welcome',
			content: FREE_TEXT,
			sort_order: 0
		});

		const enrol = await createCourse(request, {
			title: `Sign-up course ${run}`,
			slug: `signup-course-${run}`,
			access_tier: 'free',
			require_enrollment: true
		});
		expect(enrol, 'create sign-up course').toBeTruthy();
		enrolSlug = enrol!.slug;
		const enrolModule = await addModule(request, enrol!.id, 'Start here');
		await addLesson(request, enrol!.id, enrolModule, {
			title: 'Members lesson',
			content: `members-${run}`,
			sort_order: 0
		});

		const quiz = await createCourse(request, {
			title: `Quiz course ${run}`,
			slug: `quiz-course-${run}`,
			access_tier: 'free',
			require_enrollment: false,
			enable_progress: true
		});
		expect(quiz, 'create quiz course').toBeTruthy();
		const quizModule = await addModule(request, quiz!.id, 'Quiz module');
		const quizLesson = await addLesson(request, quiz!.id, quizModule, {
			title: 'Quiz lesson',
			content: '',
			sort_order: 0
		});
		const quizRes = await request.post(
			`${apiBaseURL}/api/admin/courses/${quiz!.id}/lessons/${quizLesson}/quiz`,
			{
				headers: { Authorization: token },
				data: {
					title: 'Real quiz',
					questions: [
						{
							question_text: 'Pick blue',
							question_type: 'multiple_choice',
							options: [
								{ text: 'Red', is_correct: false },
								{ text: 'Blue', is_correct: true }
							],
							explanation: EXPLANATION
						}
					]
				}
			}
		);
		expect(quizRes.ok(), 'create quiz').toBeTruthy();
		quizCourse = { slug: quiz!.slug, lessonId: quizLesson };

		paid = await createCourse(request, {
			title: `Paid course ${run}`,
			slug: `paid-course-${run}`,
			access_tier: 'paid',
			price: 1999
		});
		if (requirePaid) {
			expect(
				paid,
				'COMMERCE_FIXTURE=1 but the backend refused a paid course. Is STRIPE_SECRET_KEY set on the test backend?'
			).toBeTruthy();
		}
		if (paid) {
			const paidModule = await addModule(request, paid.id, 'Module one');
			await addLesson(request, paid.id, paidModule, {
				title: 'Preview lesson',
				content: PREVIEW,
				is_preview: true,
				sort_order: 0
			});
			await addLesson(request, paid.id, paidModule, {
				title: 'Paid lesson',
				content: SECRET,
				sort_order: 1
			});

			for (const [type, data] of [
				[
					'posts',
					{
						title: `Paid post ${run}`,
						content: `Public preview ${run}.\n\npaid-post-secret-${run}`,
						published_at: '2026-01-02 03:04:05.000Z'
					}
				],
				['projects', { title: `Paid project ${run}`, description: `paid-project-secret-${run}` }],
				[
					'talks',
					{
						title: `Paid talk ${run}`,
						description: `paid-talk-secret-${run}`,
						video_url: `https://video.example.test/${run}`,
						slides_url: `https://slides.example.test/${run}`
					}
				]
			] as const) {
				const response = await request.post(`${apiBaseURL}/api/collections/${type}/records`, {
					headers: { Authorization: token },
					data: {
						...data,
						slug: `paid-${type}-${run}`,
						visibility: 'public',
						is_draft: false,
						access_tier: 'paid',
						price: 1299
					}
				});
				expect(response.ok(), `create paid ${type}`).toBeTruthy();
				const record = await response.json();
				paidContent.push({
					type,
					id: record.id,
					slug: record.slug,
					secret: `paid-${type.slice(0, -1)}-secret-${run}`
				});
			}
		}
	});

	test.afterAll(async ({ request }) => {
		for (const id of created) {
			await request.delete(`${apiBaseURL}/api/collections/courses/records/${id}`, {
				headers: { Authorization: token }
			});
		}
		for (const item of paidContent) {
			await request.delete(`${apiBaseURL}/api/collections/${item.type}/records/${item.id}`, {
				headers: { Authorization: token }
			});
		}
	});

	test('free course lists lessons in order and opens them', async ({ page }) => {
		await page.goto(`/courses/${freeSlug}`);
		await expect(page.getByRole('heading', { level: 1 })).toHaveText(`Free course ${run}`);
		await expect(page.getByTestId('purchase-gate')).toHaveCount(0);

		const lesson = page.getByRole('link', { name: /1\.1 Welcome/ });
		await lesson.focus();
		await page.keyboard.press('Enter');
		await expect(page).toHaveURL(/\/learn\?lesson=/);
		await expect(page.getByRole('main')).toContainText(FREE_TEXT);
	});

	test.describe('paid course', () => {
		test.beforeEach(() => {
			// Under COMMERCE_FIXTURE=1 beforeAll has already failed if paid is null.
			test.skip(
				!paid,
				'Backend refused paid content (no Stripe key); set COMMERCE_FIXTURE=1 on a fixture backend to require these'
			);
		});

		test('stays locked and keeps the paid lesson out of the page', async ({ page }) => {
			const response = await page.goto(`/courses/${paid!.slug}`);
			const html = await response!.text();
			expect(html).not.toContain(SECRET);
			// The outline page never ships lesson bodies, even preview ones.
			expect(html).not.toContain(PREVIEW);

			await expect(page.getByTestId('purchase-gate')).toBeVisible();
			await expect(page.getByRole('button', { name: /Buy for .*19\.99/ })).toBeVisible();
			await expect(page.locator('[data-lesson-state="locked"]')).toContainText('Paid lesson');
			await expect(page.getByRole('link', { name: /Paid lesson/ })).toHaveCount(0);

			// The preview lesson is readable, the paid one sends you back.
			await page.getByRole('link', { name: /Preview lesson/ }).click();
			await expect(page.getByRole('main')).toContainText(PREVIEW);
			expect(await page.content()).not.toContain(SECRET);
		});

		test('opening a locked lesson directly redirects to the outline', async ({ page, request }) => {
			const detail = await request.get(`${apiBaseURL}/api/courses/${paid!.slug}`);
			const body = await detail.json();
			const locked = body.modules[0].lessons.find(
				(l: { title: string }) => l.title === 'Paid lesson'
			);
			const res = await page.goto(`/courses/${paid!.slug}/learn?lesson=${locked.id}`);
			await expect(page).toHaveURL(new RegExp(`/courses/${paid!.slug}$`));
			expect(await res!.text()).not.toContain(SECRET);
		});

		for (const type of ['posts', 'projects', 'talks'] as const) {
			test(`locked ${type} page hides paid content and offers keyboard checkout`, async ({
				page
			}) => {
				const item = paidContent.find((fixture) => fixture.type === type)!;
				expect(item, `create paid ${type} fixture`).toBeTruthy();
				let checkout: Record<string, string> | null = null;
				await page.route('**/api/content/checkout', async (route) => {
					checkout = route.request().postDataJSON();
					await route.fulfill({
						status: 503,
						contentType: 'application/json',
						body: JSON.stringify({ error: 'Stripe not configured' })
					});
				});

				const response = await page.goto(`/${item.type}/${item.slug}`);
				const html = await response!.text();
				expect(html).not.toContain(item.secret);
				if (item.type === 'talks') {
					expect(html).not.toContain('video.example.test');
					expect(html).not.toContain('slides.example.test');
				}
				if (item.type === 'posts') expect(html).toContain(`Public preview ${run}`);

				await expect(page.getByTestId('purchase-gate')).toBeVisible();
				const buy = page.getByRole('button', { name: /Buy for .*12\.99/ });
				await buy.focus();
				await page.keyboard.press('Enter');
				await expect(page.getByRole('alert')).toContainText("Payments aren't set up");
				// The slug travels in the return URLs' query, not as its own field.
				expect(checkout).toMatchObject({ content_type: item.type, content_id: item.id });
				expect(checkout).not.toHaveProperty('slug');
				const origin = new URL(page.url()).origin;
				const success = new URL(checkout!.success_url);
				const cancel = new URL(checkout!.cancel_url);
				for (const url of [success, cancel]) {
					expect(url.origin).toBe(origin);
					expect(url.pathname).toBe('/checkout/return');
					expect(url.searchParams.get('type')).toBe(item.type);
					expect(url.searchParams.get('slug')).toBe(item.slug);
					// The backend appends the Stripe session placeholder itself.
					expect(url.searchParams.has('session_id')).toBe(false);
				}
				expect(success.searchParams.has('cancelled')).toBe(false);
				expect(cancel.searchParams.get('cancelled')).toBe('1');
			});
		}

		test('shows the server error when checkout cannot start', async ({ page }) => {
			await page.route('**/api/content/checkout', (route) =>
				route.fulfill({
					status: 503,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'Stripe not configured' })
				})
			);
			await page.goto(`/courses/${paid!.slug}`);
			const buy = page.getByRole('button', { name: /Buy for/ });
			await buy.click();
			await expect(page.getByRole('alert')).toContainText("Payments aren't set up");
			await expect(buy).toBeEnabled();
			await expect(page.getByTestId('purchase-gate')).toBeVisible();
		});

		test('reports a rejected coupon', async ({ page }) => {
			await page.route('**/api/content/checkout', (route) =>
				route.fulfill({
					status: 400,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'invalid coupon: coupon has expired' })
				})
			);
			await page.goto(`/courses/${paid!.slug}`);
			await page.getByRole('button', { name: 'Have a coupon?' }).click();
			await page.getByLabel('Coupon code').fill('OLD');
			await page.getByRole('button', { name: /Buy for/ }).click();
			await expect(page.getByRole('alert')).toContainText('coupon has expired');
		});

		test('sends a real checkout request and follows the Stripe redirect', async ({
			page,
			baseURL
		}) => {
			let sent: Record<string, string> | null = null;
			await page.route('**/api/content/checkout', async (route) => {
				sent = route.request().postDataJSON();
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ checkout_url: 'https://checkout.stripe.com/c/pay/cs_test_mock' })
				});
			});
			await page.route('https://checkout.stripe.com/**', (route) =>
				route.fulfill({ status: 200, contentType: 'text/html', body: '<h1>Stripe test stub</h1>' })
			);

			await page.goto(`/courses/${paid!.slug}`);
			await page.getByRole('button', { name: /Buy for/ }).click();
			await expect(page).toHaveURL('https://checkout.stripe.com/c/pay/cs_test_mock');

			const origin = new URL(baseURL!).origin;
			expect(sent).toMatchObject({ content_type: 'courses', content_id: paid!.id });
			expect(sent!.success_url.startsWith(`${origin}/checkout/return?`)).toBeTruthy();
			expect(sent!.cancel_url).toContain('cancelled=1');
		});

		test('does not follow an unexpected checkout address', async ({ page }) => {
			await page.route('**/api/content/checkout', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ checkout_url: 'https://evil.example/pay' })
				})
			);
			await page.goto(`/courses/${paid!.slug}`);
			await page.getByRole('button', { name: /Buy for/ }).click();
			await expect(page.getByRole('alert')).toContainText("didn't follow it");
			await expect(page).toHaveURL(new RegExp(`/courses/${paid!.slug}`));
		});

		test('a forged success return does not unlock the course', async ({ page }) => {
			// The return page asks the backend about the session. Answer for it,
			// so the placeholder Stripe key is never sent to Stripe.
			await page.route('**/api/content/checkout/confirm', (route) =>
				route.fulfill({
					status: 404,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'checkout session not found' })
				})
			);
			await page.goto(
				`/checkout/return?type=courses&slug=${paid!.slug}&paid=1&session_id=cs_test_forged&is_purchased=true`
			);
			await expect(page.getByTestId('checkout-return')).toHaveAttribute('data-status', 'not_found');
			await expect(page.getByRole('heading', { level: 1 })).not.toHaveText(/all set/);

			const res = await page.goto(`/courses/${paid!.slug}`);
			expect(await res!.text()).not.toContain(SECRET);
			await expect(page.getByTestId('purchase-gate')).toBeVisible();
		});

		test('a visitor who already has access is shown it without confirming again', async ({
			page,
			request
		}) => {
			// Purchases can only be created by the server, so access comes from a
			// real redemption: a 100% coupon makes checkout grant the item directly,
			// without Stripe, and hand back the access link.
			const item = paidContent.find((fixture) => fixture.type === 'posts')!;
			const code = `FREE-${run}`.toUpperCase();
			const coupon = await request.post(`${apiBaseURL}/api/admin/coupons`, {
				headers: { Authorization: token },
				data: {
					code,
					discount_type: 'percentage',
					discount_value: 100,
					applicable_type: 'posts',
					applicable_ids: [item.id],
					max_uses: 1
				}
			});
			expect(coupon.status(), 'create coupon').toBe(201);
			const couponId = (await coupon.json()).id as string;
			let purchaseId = '';

			try {
				let confirmCalls = 0;
				await page.route('**/api/content/checkout/confirm', (route) => {
					confirmCalls++;
					return route.fulfill({ status: 500, contentType: 'application/json', body: '{}' });
				});

				await page.goto(`/posts/${item.slug}`);
				await expect(page.getByTestId('purchase-gate')).toBeVisible();
				await page.getByRole('button', { name: 'Have a coupon?' }).click();
				await page.getByLabel('Coupon code').fill(code);
				// The request goes to the real backend unchanged. Reading the answer
				// here keeps it available after the page navigates away.
				const redeemed: { status?: number; body?: { status?: string; checkout_url?: string } } = {};
				await page.route('**/api/content/checkout', async (route) => {
					const response = await route.fetch();
					redeemed.body = await response.json();
					redeemed.status = response.status();
					await route.fulfill({ response });
				});
				await page.getByRole('button', { name: /Buy for/ }).click();
				await expect.poll(() => redeemed.status, { message: 'redeem coupon' }).toBe(200);
				expect(redeemed.body?.status).toBe('authorized');
				expect(redeemed.body?.checkout_url).toMatch(/^\/api\/content\/access\/[^/]+$/);

				// The page follows the access link, which sets the purchase cookie
				// and sends the buyer back to the post.
				await expect(page).toHaveURL(new RegExp(`/posts/${item.slug}$`));
				const cookies = await page.context().cookies();
				const purchaseCookie = cookies.find((cookie) => cookie.name === 'facet_purchases');
				expect(purchaseCookie?.httpOnly).toBe(true);
				await expect(page.getByTestId('purchase-gate')).toHaveCount(0);
				await expect(page.getByRole('main')).toContainText(item.secret);

				const purchases = await request.get(`${apiBaseURL}/api/collections/purchases/records`, {
					headers: { Authorization: token },
					params: {
						filter: `content_id = "${item.id}" && stripe_checkout_session_id ~ "coupon_${code}_"`
					}
				});
				expect(purchases.ok(), 'find redeemed purchase').toBeTruthy();
				const redemption = (await purchases.json()).items as Array<{
					id: string;
					amount: number;
					status: string;
				}>;
				expect(redemption).toHaveLength(1);
				expect(redemption[0]).toMatchObject({ amount: 0, status: 'completed' });
				purchaseId = redemption[0].id;

				// Wait for the network to settle so a confirm request, if the page
				// made one after hydrating, would already have been counted.
				const response = await page.goto(
					`/checkout/return?type=posts&slug=${item.slug}&session_id=cs_test_return_${run}`,
					{
						waitUntil: 'networkidle'
					}
				);
				expect(response!.ok()).toBeTruthy();
				await expect(page.getByTestId('checkout-return')).toHaveAttribute(
					'data-status',
					'unlocked'
				);
				await expect(page.getByRole('heading', { level: 1 })).toHaveText("You're all set");
				expect(confirmCalls).toBe(0);
			} finally {
				if (purchaseId) {
					await request.delete(`${apiBaseURL}/api/collections/purchases/records/${purchaseId}`, {
						headers: { Authorization: token }
					});
				}
				const removed = await request.delete(`${apiBaseURL}/api/admin/coupons/${couponId}`, {
					headers: { Authorization: token }
				});
				expect(removed.ok(), 'delete coupon').toBeTruthy();
			}
		});

		test('a cancelled checkout explains what to check and links back', async ({ page }) => {
			await page.goto(`/checkout/return?type=courses&slug=${paid!.slug}&cancelled=1`);
			const panel = page.getByTestId('checkout-return');
			await expect(panel).toHaveAttribute('data-status', 'cancelled');
			await expect(page.getByRole('heading', { level: 1 })).toHaveText('Checkout was cancelled');
			// The cancel link is user-controlled, so it must not claim nothing was charged.
			await expect(panel).not.toContainText(/haven't been charged|nothing was charged/i);
			await expect(panel).toContainText('receipt from Stripe');
			await page.getByRole('link', { name: /Back to/ }).click();
			await expect(page).toHaveURL(new RegExp(`/courses/${paid!.slug}$`));
			await expect(page.getByTestId('purchase-gate')).toBeVisible();
		});
	});

	test.describe('free course sign-up', () => {
		test('shows a usable sign-up form instead of a paused notice', async ({ page }) => {
			await page.goto(`/courses/${enrolSlug}`);
			const form = page.getByTestId('enrollment-form');
			await expect(form).toBeVisible();
			await expect(page.getByText('Sign-up is paused')).toHaveCount(0);
			await expect(form.getByLabel('Email')).toBeVisible();
			await expect(page.locator('[data-lesson-state="locked"]')).toContainText('Members lesson');
		});

		test('a confirmation email is announced and grants nothing', async ({ page }) => {
			let sent: Record<string, string> | null = null;
			await page.route('**/api/courses/*/enroll', async (route) => {
				sent = route.request().postDataJSON();
				await route.fulfill({
					status: 202,
					contentType: 'application/json',
					body: JSON.stringify({ status: 'verification_sent' })
				});
			});
			await page.goto(`/courses/${enrolSlug}`);
			await page.getByLabel('Email').fill('learner@example.com');
			await page.getByLabel(/Name/).fill('Ada Learner');
			await page.getByLabel(/Name/).press('Enter');

			await expect(page.getByTestId('enrollment-sent')).toContainText('learner@example.com');
			await expect(page.getByRole('status').filter({ hasText: 'Check your inbox' })).toBeVisible();
			expect(sent).toEqual({ email: 'learner@example.com', name: 'Ada Learner' });

			// Still locked after a reload: only the emailed link enrols.
			await page.reload();
			await expect(page.getByTestId('enrollment-form')).toBeVisible();
			await expect(page.getByRole('link', { name: /Members lesson/ })).toHaveCount(0);
		});

		test('also accepts the confirmation_required status', async ({ page }) => {
			await page.route('**/api/courses/*/enroll', (route) =>
				route.fulfill({
					status: 202,
					contentType: 'application/json',
					body: JSON.stringify({ status: 'confirmation_required' })
				})
			);
			await page.goto(`/courses/${enrolSlug}`);
			await page.getByLabel('Email').fill('learner@example.com');
			await page.getByRole('button', { name: 'Send me the link' }).click();
			await expect(page.getByTestId('enrollment-sent')).toBeVisible();
		});

		test('a verified learner goes straight to the course', async ({ page }) => {
			await page.route('**/api/courses/*/enroll', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						status: 'already_enrolled',
						redirect_url: `/courses/${enrolSlug}/learn`
					})
				})
			);
			await page.goto(`/courses/${enrolSlug}`);
			await page.getByLabel('Email').fill('learner@example.com');
			const learn = page.waitForRequest((req) => req.url().includes(`/courses/${enrolSlug}/learn`));
			await page.getByRole('button', { name: 'Send me the link' }).click();
			await learn;
		});

		test('ignores a redirect to anywhere but the course', async ({ page }) => {
			await page.route('**/api/courses/*/enroll', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ status: 'enrolled', redirect_url: 'https://evil.example/' })
				})
			);
			await page.goto(`/courses/${enrolSlug}`);
			await page.getByLabel('Email').fill('learner@example.com');
			await page.getByRole('button', { name: 'Send me the link' }).click();
			await expect(page.getByRole('alert')).toContainText("didn't expect");
			await expect(page).toHaveURL(new RegExp(`/courses/${enrolSlug}$`));
		});

		for (const [status, error, headers, text] of [
			[
				503,
				"enrolment needs email confirmation, and this site can't send email yet",
				{},
				"can't send email yet"
			],
			[502, 'failed to send confirmation email', {}, "couldn't send the confirmation email"],
			[429, 'rate limit exceeded', { 'Retry-After': '120' }, 'Try again in 2 minutes'],
			[400, 'invalid email format', {}, "doesn't look right"]
		] as const) {
			test(`explains a ${status} from sign-up`, async ({ page }) => {
				await page.route('**/api/courses/*/enroll', (route) =>
					route.fulfill({
						status,
						headers: { ...headers },
						contentType: 'application/json',
						body: JSON.stringify({ error })
					})
				);
				await page.goto(`/courses/${enrolSlug}`);
				await page.getByLabel('Email').fill('learner@example.com');
				const button = page.getByRole('button', { name: 'Send me the link' });
				await button.click();
				await expect(page.getByRole('alert')).toContainText(text);
				await expect(button).toBeEnabled();
				await expect(page.getByTestId('enrollment-sent')).toHaveCount(0);
			});
		}
	});

	test.describe('quizzes', () => {
		test('the quiz answer key stays off the page and out of anonymous requests', async ({
			request
		}) => {
			const learner = await request.get(
				`${apiBaseURL}/api/courses/${quizCourse!.slug}/lessons/${quizCourse!.lessonId}/quiz`
			);
			expect(learner.status()).toBe(401);
			const body = await learner.text();
			expect(body).not.toContain('is_correct');
			expect(body).not.toContain(EXPLANATION);

			const page = await request.get(
				`/courses/${quizCourse!.slug}/learn?lesson=${quizCourse!.lessonId}`
			);
			const html = await page.text();
			expect(html).not.toContain('is_correct');
			expect(html).not.toContain(EXPLANATION);
		});

		test('without a learner cookie the quiz offers sign-up', async ({ page }) => {
			await openQuizLesson(page);
			const quiz = page.getByTestId('lesson-quiz');
			await expect(quiz).toHaveAttribute('data-phase', 'error');
			await expect(quiz.getByRole('alert')).toContainText('isn’t signed in');
			await expect(quiz.getByTestId('enrollment-form')).toBeVisible();
			await expect(page.getByText("Quizzes can't be taken")).toHaveCount(0);
		});

		test('answers by keyboard, submits each question once and shows server results', async ({
			page
		}) => {
			let submitted: { answers: Array<{ question_id: string; selected_answer: number }> } | null =
				null;
			// A stray answer key in the payload must not reach the page.
			const leaky = structuredClone(quizFixture) as Record<string, unknown>;
			(
				leaky.questions as Array<{ options: Array<Record<string, unknown>> }>
			)[0].options[1].is_correct = true;
			await page.route('**/api/courses/*/lessons/*/quiz', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(leaky) })
			);
			await page.route('**/api/courses/*/lessons/*/quiz/submit', async (route) => {
				submitted = route.request().postDataJSON();
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						score: 50,
						passed: true,
						total_points: 20,
						earned_points: 10,
						results: [
							{ question_id: 'q1', correct: true, explanation: 'Blue is blue.' },
							{ question_id: 'q2', correct: false, explanation: EXPLANATION }
						]
					})
				});
			});

			await openQuizLesson(page);
			const quiz = page.getByTestId('lesson-quiz');
			await expect(quiz.getByTestId('quiz-question')).toHaveCount(2);
			await expect(quiz).not.toContainText(EXPLANATION);
			await expect(quiz.getByTestId('quiz-outcome')).toHaveCount(0);
			expect(await page.content()).not.toContain('is_correct');

			// Submitting early asks for every answer and sends nothing.
			await quiz.getByRole('button', { name: 'Submit answers' }).click();
			await expect(quiz.getByRole('alert')).toContainText('Answer every question');
			expect(submitted).toBeNull();

			// Radios work from the keyboard: arrow keys move within a question.
			await quiz.getByLabel('Red').focus();
			await page.keyboard.press('ArrowDown');
			await expect(quiz.getByLabel('Blue')).toBeChecked();
			await quiz.getByLabel('False').check();
			await quiz.getByLabel('True').check();
			await quiz.getByRole('button', { name: 'Submit answers' }).click();

			await expect(quiz.getByTestId('quiz-result')).toHaveAttribute('data-passed', 'true');
			expect(submitted!.answers).toEqual([
				{ question_id: 'q1', selected_answer: 1 },
				{ question_id: 'q2', selected_answer: 0 }
			]);
			await expect(quiz.getByTestId('quiz-outcome')).toHaveCount(2);
			await expect(quiz).toContainText(EXPLANATION);
			await expect(quiz.getByLabel('Blue')).toBeDisabled();
		});

		test('reports used-up attempts from the server', async ({ page }) => {
			await page.route('**/api/courses/*/lessons/*/quiz', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify(quizFixture)
				})
			);
			await page.route('**/api/courses/*/lessons/*/quiz/submit', (route) =>
				route.fulfill({
					status: 403,
					contentType: 'application/json',
					body: JSON.stringify({ error: 'maximum attempts exceeded' })
				})
			);
			await openQuizLesson(page);
			const quiz = page.getByTestId('lesson-quiz');
			await quiz.getByLabel('Blue').check();
			await quiz.getByLabel('False').check();
			await quiz.getByRole('button', { name: 'Submit answers' }).click();
			await expect(quiz.getByTestId('quiz-no-attempts')).toBeVisible();
			await expect(quiz.getByTestId('quiz-result')).toHaveCount(0);
		});

		test('shows when no attempts are left before answering', async ({ page }) => {
			await page.route('**/api/courses/*/lessons/*/quiz', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						...quizFixture,
						attempts_used: 2,
						previous_attempt: { score: 0, passed: false }
					})
				})
			);
			await openQuizLesson(page);
			await expect(page.getByTestId('quiz-no-attempts')).toBeVisible();
			await expect(page.getByTestId('quiz-previous')).toContainText('0%');
			await expect(page.getByRole('button', { name: 'Submit answers' })).toHaveCount(0);
		});

		test('lets the learner retry after a load error', async ({ page }) => {
			let calls = 0;
			await page.route('**/api/courses/*/lessons/*/quiz', (route) => {
				calls += 1;
				return calls === 1
					? route.fulfill({ status: 500, contentType: 'application/json', body: '{}' })
					: route.fulfill({
							status: 200,
							contentType: 'application/json',
							body: JSON.stringify(quizFixture)
						});
			});
			await openQuizLesson(page);
			await page.getByRole('button', { name: 'Try again' }).click();
			await expect(page.getByTestId('quiz-question')).toHaveCount(2);
		});
	});

	test('an unknown certificate is a 404', async ({ page }) => {
		expect((await page.goto(`/certificates/missing${run}`))!.status()).toBe(404);
		expect((await page.goto('/certificates/..%2Fadmin'))!.status()).toBe(404);
	});

	test('rejects an incomplete return link', async ({ page }) => {
		await page.goto('/checkout/return?type=admin&slug=../x');
		await expect(page.getByTestId('checkout-return')).toHaveAttribute('data-status', 'invalid');
	});

	test('unknown course is a 404', async ({ page }) => {
		const res = await page.goto(`/courses/missing-${run}`);
		expect(res!.status()).toBe(404);
	});
});

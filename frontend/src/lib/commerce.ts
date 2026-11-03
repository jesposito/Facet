/**
 * Public commerce helpers: price formatting, course projections and the
 * checkout call shared by PurchaseGate and the course pages.
 *
 * Access is always decided by the backend. Nothing here unlocks content from
 * client state, query parameters or a checkout response.
 */

import { get } from 'svelte/store';
import { t } from 'svelte-i18n';

function tr(
	key: string,
	values?: Record<string, string | number | Date | null | undefined>
): string {
	return get(t)(`public.commerce.${key}`, { values });
}

/** Content types accepted by POST /api/content/checkout. */
export type PurchasableType = 'posts' | 'projects' | 'talks' | 'courses';

export const PURCHASABLE_TYPES: readonly PurchasableType[] = [
	'posts',
	'projects',
	'talks',
	'courses'
];

/** Name of the HttpOnly cookie the backend sets after a confirmed purchase. */
export const PURCHASE_COOKIE = 'facet_purchases';

/** Public page for each purchasable type, matching /api/content/access redirects. */
export function contentPath(type: PurchasableType, slug: string): string {
	return `/${type}/${encodeURIComponent(slug)}`;
}

export function isPurchasableType(value: unknown): value is PurchasableType {
	return typeof value === 'string' && (PURCHASABLE_TYPES as readonly string[]).includes(value);
}

/**
 * Formats a price held in minor units (cents). The backend stores integer
 * cents and currently charges in USD, so that is the default currency.
 */
export function formatPrice(amount: number, locale?: string | null, currency = 'usd'): string {
	const value = Number.isFinite(amount) ? amount / 100 : 0;
	try {
		return new Intl.NumberFormat(locale || undefined, {
			style: 'currency',
			currency: currency.toUpperCase()
		}).format(value);
	} catch {
		return `${currency.toUpperCase()} ${value.toFixed(2)}`;
	}
}

// ---------------------------------------------------------------------------
// Course projection (GET /api/courses/{slug})
// ---------------------------------------------------------------------------

export type LessonState = 'available' | 'preview' | 'locked' | 'drip_locked' | 'sequential_locked';

export interface CourseLessonSummary {
	id: string;
	title: string;
	content_type: string;
	estimated_minutes: number;
	is_preview: boolean;
	state: LessonState;
}

export interface CourseModuleSummary {
	id: string;
	title: string;
	description: string;
	drip_unlock_date: string | null;
	lessons: CourseLessonSummary[];
}

export interface CourseSummary {
	id: string;
	title: string;
	slug: string;
	description: string;
	instructor_name: string;
	instructor_bio: string;
	access_tier: 'free' | 'paid';
	price: number;
	difficulty: string;
	estimated_hours: number;
	tags: string[];
	learning_outcomes: string[];
	cover_image_url: string | null;
	visibility: string;
	is_draft: boolean;
	has_access: boolean;
	require_enrollment: boolean;
	sequential_access: boolean;
	enable_progress: boolean;
	show_progress_bar: boolean;
	enable_certificates: boolean;
	total_lessons: number;
	modules: CourseModuleSummary[];
}

export interface CourseLessonContent {
	id: string;
	content: string;
	video_url: string;
	attachments: Array<{ filename: string; url: string }>;
}

type Raw = Record<string, unknown>;

function asArray(value: unknown): Raw[] {
	return Array.isArray(value) ? (value.filter((v) => v && typeof v === 'object') as Raw[]) : [];
}

function asStrings(value: unknown): string[] {
	if (!Array.isArray(value)) return [];
	return value
		.map((v) =>
			typeof v === 'string'
				? v
				: v && typeof v === 'object'
					? String((v as Raw).text ?? (v as Raw).title ?? '')
					: ''
		)
		.filter((v) => v.trim() !== '');
}

function bySortOrder(a: Raw, b: Raw): number {
	return (Number(a.sort_order) || 0) - (Number(b.sort_order) || 0);
}

function lessonState(lesson: Raw, module: Raw): LessonState {
	// The backend only sends content when the request may read the lesson,
	// so the presence of the field is the signal we trust.
	const unlocked = typeof lesson.content === 'string' || typeof lesson.video_url === 'string';
	if (module.drip_locked && !lesson.is_preview) return 'drip_locked';
	if (lesson.sequential_locked) return 'sequential_locked';
	if (!unlocked) return 'locked';
	return lesson.is_preview ? 'preview' : 'available';
}

/**
 * Reduces the backend course response to metadata only. Lesson bodies,
 * video URLs and attachment links are dropped so the detail page cannot
 * serialise them into HTML or hydration data.
 */
export function toCourseSummary(raw: Raw): CourseSummary {
	const modules = asArray(raw.modules)
		.sort(bySortOrder)
		.map((module) => ({
			id: String(module.id ?? ''),
			title: String(module.title ?? ''),
			description: String(module.description ?? ''),
			drip_unlock_date:
				module.drip_locked && typeof module.drip_unlock_date === 'string'
					? module.drip_unlock_date
					: null,
			lessons: asArray(module.lessons)
				.sort(bySortOrder)
				.map((lesson) => ({
					id: String(lesson.id ?? ''),
					title: String(lesson.title ?? ''),
					content_type: String(lesson.content_type ?? 'text'),
					estimated_minutes: Number(lesson.estimated_minutes) || 0,
					is_preview: Boolean(lesson.is_preview),
					state: lessonState(lesson, module)
				}))
		}));

	return {
		id: String(raw.id ?? ''),
		title: String(raw.title ?? ''),
		slug: String(raw.slug ?? ''),
		description: String(raw.description ?? ''),
		instructor_name: String(raw.instructor_name ?? ''),
		instructor_bio: String(raw.instructor_bio ?? ''),
		access_tier: raw.access_tier === 'paid' ? 'paid' : 'free',
		price: Number(raw.price) || 0,
		difficulty: String(raw.difficulty ?? ''),
		estimated_hours: Number(raw.estimated_hours) || 0,
		tags: asStrings(raw.tags),
		learning_outcomes: asStrings(raw.learning_outcomes),
		cover_image_url:
			typeof raw.cover_image_url === 'string' && raw.cover_image_url ? raw.cover_image_url : null,
		visibility: String(raw.visibility ?? 'public'),
		is_draft: Boolean(raw.is_draft),
		has_access: raw.is_purchased === true,
		require_enrollment: Boolean(raw.require_enrollment),
		sequential_access: Boolean(raw.sequential_access),
		enable_progress: Boolean(raw.enable_progress),
		show_progress_bar: Boolean(raw.show_progress_bar),
		enable_certificates: Boolean(raw.enable_certificates),
		total_lessons: modules.reduce((n, m) => n + m.lessons.length, 0),
		modules
	};
}

/** Returns the readable content of one lesson, or null when the backend withheld it. */
export function findLessonContent(raw: Raw, lessonId: string): CourseLessonContent | null {
	for (const module of asArray(raw.modules)) {
		for (const lesson of asArray(module.lessons)) {
			if (lesson.id !== lessonId) continue;
			if (typeof lesson.content !== 'string' && typeof lesson.video_url !== 'string') return null;
			return {
				id: lessonId,
				content: typeof lesson.content === 'string' ? lesson.content : '',
				video_url: typeof lesson.video_url === 'string' ? lesson.video_url : '',
				attachments: asArray(lesson.attachments).map((a) => ({
					filename: String(a.filename ?? ''),
					url: String(a.url ?? '')
				}))
			};
		}
	}
	return null;
}

export function isReadable(state: LessonState): boolean {
	return state === 'available' || state === 'preview';
}

export function flattenLessons(course: CourseSummary): CourseLessonSummary[] {
	return course.modules.flatMap((m) => m.lessons);
}

// ---------------------------------------------------------------------------
// Checkout (POST /api/content/checkout)
// ---------------------------------------------------------------------------

export interface CheckoutRequest {
	contentType: PurchasableType;
	contentId: string;
	slug: string;
	couponCode?: string;
}

export type CheckoutResult = { ok: true; url: string } | { ok: false; message: string };

/**
 * Builds the return URLs. Both must be on this host or the backend rejects
 * them. The success page never trusts its query string; it only uses it to
 * know which item to re-check with the backend.
 */
export function checkoutReturnUrls(origin: string, type: PurchasableType, slug: string) {
	const params = new URLSearchParams({ type, slug });
	return {
		success_url: `${origin}/checkout/return?${params}`,
		cancel_url: `${origin}/checkout/return?${params}&cancelled=1`
	};
}

/**
 * Accepts only the two destinations the backend produces: a Stripe-hosted
 * checkout page, or the same-site access link issued when a coupon makes the
 * item free.
 */
export function isTrustedCheckoutUrl(value: unknown, origin: string): value is string {
	if (typeof value !== 'string' || !value) return false;
	let url: URL;
	try {
		url = new URL(value, origin);
	} catch {
		return false;
	}
	if (url.origin === origin) return /^\/api\/content\/access\/[^/]+$/.test(url.pathname);
	return (
		url.protocol === 'https:' &&
		(url.hostname === 'checkout.stripe.com' || url.hostname.endsWith('.stripe.com'))
	);
}

function checkoutErrorMessage(status: number, error: string): string {
	const e = error.toLowerCase();
	if (e.startsWith('invalid coupon'))
		return tr('errors.invalid_coupon', {
			reason:
				error.slice('invalid coupon: '.length) || get(t)('public.commerce.errors.coupon_not_valid')
		});
	if (status === 429) return tr('errors.checkout_rate_limited');
	if (status === 503 || e.includes('stripe not configured'))
		return tr('errors.payments_unavailable');
	if (status === 404) return tr('errors.item_not_found');
	if (e.includes('not gated')) return tr('errors.not_gated');
	if (e.includes('success_url') || e.includes('cancel_url')) return tr('errors.checkout_address');
	return tr('errors.checkout_start');
}

export async function startCheckout(
	request: CheckoutRequest,
	origin: string,
	fetchImpl: typeof fetch = fetch
): Promise<CheckoutResult> {
	let response: Response;
	try {
		response = await fetchImpl('/api/content/checkout', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'same-origin',
			body: JSON.stringify({
				content_type: request.contentType,
				content_id: request.contentId,
				coupon_code: request.couponCode?.trim() || '',
				...checkoutReturnUrls(origin, request.contentType, request.slug)
			})
		});
	} catch {
		return { ok: false, message: tr('errors.network') };
	}

	let body: Raw = {};
	try {
		body = (await response.json()) as Raw;
	} catch {
		// Fall through with an empty body.
	}

	if (!response.ok) {
		return { ok: false, message: checkoutErrorMessage(response.status, String(body.error ?? '')) };
	}
	if (!isTrustedCheckoutUrl(body.checkout_url, origin)) {
		return { ok: false, message: tr('errors.checkout_redirect') };
	}
	return { ok: true, url: body.checkout_url };
}

// ---------------------------------------------------------------------------
// Checkout return (POST /api/content/checkout/confirm)
// ---------------------------------------------------------------------------

/** Stripe checkout session IDs, matching the backend's own check. */
export function isCheckoutSessionId(value: unknown): value is string {
	return typeof value === 'string' && /^cs_[A-Za-z0-9_]{1,250}$/.test(value);
}

/**
 * What the confirm endpoint said about a checkout session. None of these
 * grant access: the access link only ever goes to the buyer's inbox.
 * Messages are written here, so the page never shows a raw server error.
 */
export type ConfirmResult =
	| { kind: 'sent'; emailHint: string; retryAfter: number }
	| { kind: 'pending' }
	| { kind: 'failed' }
	| { kind: 'refunded' }
	| { kind: 'delivery_failed' }
	| { kind: 'delivery_unavailable' }
	| { kind: 'not_found' }
	| { kind: 'mismatch' }
	| { kind: 'rate_limited'; retryAfter: number }
	| { kind: 'error' };

export interface ConfirmRequest {
	sessionId: string;
	contentType: PurchasableType;
	slug: string;
	/** Ask for the access email again. Only ever set by a button press. */
	resend?: boolean;
}

/** Whole seconds from a header or body value, or 0 when it isn't one. */
function seconds(value: unknown): number {
	const n = Math.ceil(Number(value));
	return Number.isFinite(n) && n > 0 ? Math.min(n, 3600) : 0;
}

/** Keeps a masked hint such as "j***@example.com" and drops anything else. */
function emailHint(value: unknown): string {
	return typeof value === 'string' && value.length <= 254 && /^[^\s@]*\*{3}@[^\s@]+$/.test(value)
		? value
		: '';
}

export async function confirmCheckout(
	request: ConfirmRequest,
	fetchImpl: typeof fetch = fetch,
	signal?: AbortSignal
): Promise<ConfirmResult> {
	let response: Response;
	try {
		response = await fetchImpl('/api/content/checkout/confirm', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'same-origin',
			cache: 'no-store',
			signal,
			body: JSON.stringify({
				session_id: request.sessionId,
				content_type: request.contentType,
				slug: request.slug,
				resend: request.resend === true
			})
		});
	} catch (err) {
		if (signal?.aborted) throw err;
		return { kind: 'error' };
	}

	let body: Raw = {};
	try {
		body = (await response.json()) as Raw;
	} catch {
		// Fall through with an empty body.
	}
	const status = String(body.status ?? '');

	if (response.status === 429)
		return { kind: 'rate_limited', retryAfter: seconds(response.headers.get('Retry-After')) || 60 };
	if (response.status === 400 || response.status === 404) return { kind: 'not_found' };
	if (response.status === 409) return { kind: 'mismatch' };
	if (status === 'delivery_unavailable') return { kind: 'delivery_unavailable' };
	if (status === 'delivery_failed') return { kind: 'delivery_failed' };
	if (!response.ok) return { kind: 'error' };

	switch (status) {
		case 'confirmation_sent':
			return {
				kind: 'sent',
				emailHint: emailHint(body.email_hint),
				retryAfter: seconds(body.retry_after)
			};
		case 'pending':
			return { kind: 'pending' };
		case 'failed':
			return { kind: 'failed' };
		case 'refunded':
			return { kind: 'refunded' };
	}
	return { kind: 'error' };
}

/**
 * Seconds to wait before asking about a pending payment again. The confirm
 * endpoint allows 10 requests a minute per visitor, and this schedule uses at
 * most four in the first minute (at 0, 10, 25 and 45 seconds), which leaves
 * room for the buyer's own button presses. Returns null once polling should
 * stop and hand over to a "Check again" button.
 */
export const CONFIRM_POLL_DELAYS = [10, 15, 20, 30, 45, 60, 60, 60] as const;

export function confirmPollDelay(attempt: number): number | null {
	return CONFIRM_POLL_DELAYS[attempt] ?? null;
}

/** The backend's resend cooldown, used when a resend doesn't report its own. */
export const ACCESS_EMAIL_RESEND_COOLDOWN = 120;

// ---------------------------------------------------------------------------
// Free-course enrolment (POST /api/courses/{slug}/enroll)
// ---------------------------------------------------------------------------

/**
 * What the enrolment form should do next. Only `redirect` moves the visitor
 * on, and only because the backend has already set the learner cookie. A
 * confirmation email grants nothing until its link is followed.
 */
export type EnrollmentResult =
	| { kind: 'redirect'; url: string }
	| { kind: 'confirmation_sent' }
	| { kind: 'error'; message: string };

/** The only redirect the enrolment endpoint is expected to return. */
export function isLearnRedirect(value: unknown, slug: string): value is string {
	return typeof value === 'string' && value === `/courses/${slug}/learn`;
}

function enrollmentErrorMessage(status: number, error: string, retryAfter: string | null): string {
	const e = error.toLowerCase();
	if (status === 429) {
		const minutes = Math.ceil(Number(retryAfter) / 60);
		return Number.isFinite(minutes) && minutes > 0
			? tr('errors.enrollment_rate_minutes', { count: minutes })
			: tr('errors.enrollment_rate_limited');
	}
	if (status === 503) return tr('errors.enrollment_email_unavailable');
	if (status === 502) return tr('errors.enrollment_email_failed');
	if (e.includes('email is required')) return tr('errors.email_required');
	if (e.includes('invalid email')) return tr('errors.email_invalid');
	if (e.includes('requires purchase')) return tr('errors.purchase_required');
	if (status === 404) return tr('errors.enrollment_closed');
	return tr('errors.enrollment_failed');
}

export async function startEnrollment(
	slug: string,
	email: string,
	name: string,
	fetchImpl: typeof fetch = fetch
): Promise<EnrollmentResult> {
	let response: Response;
	try {
		response = await fetchImpl(`/api/courses/${encodeURIComponent(slug)}/enroll`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'same-origin',
			body: JSON.stringify({ email: email.trim(), name: name.trim() })
		});
	} catch {
		return { kind: 'error', message: tr('errors.network') };
	}

	let body: Raw = {};
	try {
		body = (await response.json()) as Raw;
	} catch {
		// Fall through with an empty body.
	}

	if (!response.ok) {
		return {
			kind: 'error',
			message: enrollmentErrorMessage(
				response.status,
				String(body.error ?? ''),
				response.headers.get('Retry-After')
			)
		};
	}

	const status = String(body.status ?? '');
	if (
		(status === 'enrolled' || status === 'already_enrolled') &&
		isLearnRedirect(body.redirect_url, slug)
	) {
		return { kind: 'redirect', url: body.redirect_url };
	}
	// The backend answers the same way whether or not an email went out, so
	// the form can't be used to find out who is enrolled.
	if (
		response.status === 202 &&
		(status === 'verification_sent' || status === 'confirmation_required')
	) {
		return { kind: 'confirmation_sent' };
	}
	return { kind: 'error', message: tr('errors.enrollment_unexpected') };
}

// ---------------------------------------------------------------------------
// Quizzes (GET /api/courses/{slug}/lessons/{id}/quiz, POST .../quiz/submit)
// ---------------------------------------------------------------------------

export interface QuizQuestion {
	id: string;
	text: string;
	type: 'multiple_choice' | 'true_false';
	points: number;
	/** Option labels only. The learner endpoint never sends the answer key. */
	options: string[];
}

export interface LearnerQuiz {
	id: string;
	title: string;
	passing_score: number;
	max_attempts: number;
	attempts_used: number;
	previous_attempt: { score: number; passed: boolean } | null;
	questions: QuizQuestion[];
}

export interface QuizResult {
	score: number;
	passed: boolean;
	earned_points: number;
	total_points: number;
	results: Array<{ question_id: string; correct: boolean; explanation: string }>;
}

/**
 * Keeps only what a learner needs to answer. Anything else in an option,
 * such as a stray `is_correct`, is dropped rather than rendered.
 */
export function toLearnerQuiz(raw: Raw): LearnerQuiz {
	const previous =
		raw.previous_attempt && typeof raw.previous_attempt === 'object'
			? (raw.previous_attempt as Raw)
			: null;
	return {
		id: String(raw.id ?? ''),
		title: String(raw.title ?? ''),
		passing_score: Number(raw.passing_score) || 70,
		max_attempts: Math.max(0, Number(raw.max_attempts) || 0),
		attempts_used: Math.max(0, Number(raw.attempts_used) || 0),
		previous_attempt: previous
			? { score: Number(previous.score) || 0, passed: previous.passed === true }
			: null,
		questions: asArray(raw.questions)
			.map((q) => ({
				id: String(q.id ?? ''),
				text: String(q.question_text ?? ''),
				type:
					q.question_type === 'true_false' ? ('true_false' as const) : ('multiple_choice' as const),
				points: Number(q.points) || 10,
				options: asArray(q.options).map((o) => String(o.text ?? ''))
			}))
			.filter((q) => q.id && q.options.length > 0)
	};
}

/** Attempts remaining, or null when the quiz has no limit. */
export function attemptsLeft(quiz: LearnerQuiz): number | null {
	return quiz.max_attempts > 0 ? Math.max(0, quiz.max_attempts - quiz.attempts_used) : null;
}

/**
 * One answer per question, in question order. The backend rejects repeated
 * question IDs and scores unanswered questions as wrong.
 */
export function buildQuizSubmission(
	questions: QuizQuestion[],
	selected: Record<string, number | undefined>
) {
	return {
		answers: questions
			.filter((q) => Number.isInteger(selected[q.id]))
			.map((q) => ({ question_id: q.id, selected_answer: selected[q.id] as number }))
	};
}

export function toQuizResult(raw: Raw): QuizResult {
	return {
		score: Number(raw.score) || 0,
		passed: raw.passed === true,
		earned_points: Number(raw.earned_points) || 0,
		total_points: Number(raw.total_points) || 0,
		results: asArray(raw.results).map((r) => ({
			question_id: String(r.question_id ?? ''),
			correct: r.correct === true,
			explanation: typeof r.explanation === 'string' ? r.explanation : ''
		}))
	};
}

export type QuizErrorKind = 'access' | 'missing' | 'attempts' | 'other';

export function quizError(status: number, error: string): { kind: QuizErrorKind; message: string } {
	const e = error.toLowerCase();
	if (status === 401) return { kind: 'access', message: tr('errors.quiz_device') };
	if (status === 403 && e.includes('maximum attempts'))
		return { kind: 'attempts', message: tr('errors.quiz_attempts') };
	if (status === 403) return { kind: 'access', message: tr('errors.quiz_access') };
	if (status === 404) return { kind: 'missing', message: tr('errors.quiz_missing') };
	if (status === 429) return { kind: 'other', message: tr('errors.quiz_rate_limited') };
	if (e.includes('only be answered once'))
		return { kind: 'other', message: tr('errors.quiz_answered_once') };
	if (e.includes('no answers')) return { kind: 'other', message: tr('errors.quiz_no_answers') };
	return { kind: 'other', message: tr('errors.quiz_failed') };
}

// ---------------------------------------------------------------------------
// Certificates (GET /api/courses/{slug}/certificate, /api/certificates/verify/{id})
// ---------------------------------------------------------------------------

export interface Certificate {
	certificate_id: string;
	course_title: string;
	instructor_name: string;
	student_name: string;
	issued_at: string;
}

/** Keeps what is printed on the certificate and drops buyer_email. */
export function toCertificate(raw: Raw): Certificate | null {
	const id = typeof raw.certificate_id === 'string' ? raw.certificate_id : '';
	if (!/^[A-Za-z0-9_-]{1,128}$/.test(id)) return null;
	return {
		certificate_id: id,
		course_title: String(raw.course_title ?? ''),
		instructor_name: String(raw.instructor_name ?? ''),
		student_name: String(raw.student_name ?? ''),
		issued_at: typeof raw.issued_at === 'string' ? raw.issued_at : ''
	};
}

export function certificatePath(id: string): string {
	return `/certificates/${encodeURIComponent(id)}`;
}

export interface LearnerProgress {
	completedIds: string[];
	percent: number;
	/** Set by the server when the last lesson is done. */
	completed: boolean;
}

/** GET /api/courses/{slug}/progress, or null when it can't be read. */
export async function loadLearnerProgress(
	fetchImpl: typeof fetch,
	pbUrl: string,
	slug: string,
	headers: Record<string, string>
): Promise<LearnerProgress | null> {
	try {
		const res = await fetchImpl(`${pbUrl}/api/courses/${encodeURIComponent(slug)}/progress`, {
			headers
		});
		if (!res.ok) return null;
		const body = (await res.json()) as Raw;
		const course =
			body.course_progress && typeof body.course_progress === 'object'
				? (body.course_progress as Raw)
				: {};
		return {
			completedIds: asArray(body.lesson_progress)
				.filter((p) => p.completed === true)
				.map((p) => String(p.lesson_id ?? '')),
			percent: Number(course.progress_percent) || 0,
			completed: typeof course.completed_at === 'string' && course.completed_at !== ''
		};
	} catch (err) {
		console.error('Failed to load course progress:', err);
		return null;
	}
}

/**
 * The learner's own certificate as the server issued it. Issuing runs in
 * the background after the last lesson, so `missing` straight after
 * completion usually means "not yet".
 */
export type CertificateState =
	| { status: 'issued'; certificate: Certificate }
	| { status: 'missing' }
	| { status: 'unavailable' };

export async function loadLearnerCertificate(
	fetchImpl: typeof fetch,
	pbUrl: string,
	slug: string,
	headers: Record<string, string>
): Promise<CertificateState> {
	try {
		const res = await fetchImpl(`${pbUrl}/api/courses/${encodeURIComponent(slug)}/certificate`, {
			headers
		});
		if (res.status === 404) return { status: 'missing' };
		if (!res.ok) return { status: 'unavailable' };
		const certificate = toCertificate(await res.json());
		return certificate ? { status: 'issued', certificate } : { status: 'unavailable' };
	} catch (err) {
		console.error('Failed to load course certificate:', err);
		return { status: 'unavailable' };
	}
}

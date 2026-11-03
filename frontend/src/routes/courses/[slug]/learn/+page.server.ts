/**
 * Lesson player: /courses/[slug]/learn?lesson=<id>
 *
 * Sends the body of the selected lesson only, and only when the backend
 * included it for this visitor. Without a readable lesson the visitor goes
 * back to the course page, where the outline explains what is locked.
 */

import type { PageServerLoad } from './$types';
import { error, redirect } from '@sveltejs/kit';
import { internalApiHeaders } from '$lib/server/public-api';
import {
	findLessonContent,
	flattenLessons,
	isReadable,
	loadLearnerCertificate,
	loadLearnerProgress,
	PURCHASE_COOKIE,
	toCourseSummary
} from '$lib/commerce';

export const load: PageServerLoad = async ({ params, fetch, request, cookies, locals, url }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	const purchaseCookie = cookies.get(PURCHASE_COOKIE);
	const headers = {
		...internalApiHeaders(request),
		...(pbAuthToken ? { Authorization: `Bearer ${pbAuthToken}` } : {})
	};
	const coursePath = `/courses/${encodeURIComponent(params.slug)}`;

	let response: Response;
	try {
		response = await fetch(`${pbUrl}/api/courses/${encodeURIComponent(params.slug)}`, { headers });
	} catch (err) {
		console.error('Failed to load course:', err);
		throw error(503, 'Courses are unavailable right now');
	}
	if (response.status === 404) throw error(404, 'Not Found');
	if (!response.ok) throw error(502, 'Courses are unavailable right now');

	const raw = await response.json();
	const course = toCourseSummary(raw);
	const lessons = flattenLessons(course);
	const requested = url.searchParams.get('lesson');
	const lesson = requested
		? lessons.find((l) => l.id === requested)
		: lessons.find((l) => isReadable(l.state));

	if (!lesson || !isReadable(lesson.state)) throw redirect(303, coursePath);
	const content = findLessonContent(raw, lesson.id);
	if (!content) throw redirect(303, coursePath);

	// Progress needs the purchase or enrolment cookie; without it there is
	// nothing to fetch and the page simply hides progress controls.
	const progress =
		course.enable_progress && purchaseCookie
			? await loadLearnerProgress(fetch, pbUrl, params.slug, headers)
			: null;

	// Only ask for the certificate once the server says the course is done.
	const certificate =
		course.enable_certificates && progress?.completed
			? await loadLearnerCertificate(fetch, pbUrl, params.slug, headers)
			: null;

	return {
		course,
		lesson,
		content,
		completedIds: progress?.completedIds ?? [],
		progressAvailable: progress !== null,
		courseCompleted: progress?.completed ?? false,
		certificate
	};
};

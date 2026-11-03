/**
 * Course detail route: /courses/[slug]
 *
 * Shows the course outline and how to get access. Lesson bodies are never
 * sent to this page, even when the visitor can read them; they load on the
 * learn page one lesson at a time.
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';
import { internalApiHeaders } from '$lib/server/public-api';
import {
	loadLearnerCertificate,
	loadLearnerProgress,
	PURCHASE_COOKIE,
	toCourseSummary,
	type CertificateState,
	type LearnerProgress
} from '$lib/commerce';

export const load: PageServerLoad = async ({ params, fetch, request, cookies, locals }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const pbAuthToken = locals.pb?.authStore?.isValid ? locals.pb.authStore.token : null;
	const purchaseCookie = cookies.get(PURCHASE_COOKIE);
	const headers = {
		...internalApiHeaders(request),
		...(pbAuthToken ? { Authorization: `Bearer ${pbAuthToken}` } : {})
	};

	let response: Response;
	try {
		response = await fetch(`${pbUrl}/api/courses/${encodeURIComponent(params.slug)}`, { headers });
	} catch (err) {
		console.error('Failed to load course:', err);
		throw error(503, 'Courses are unavailable right now');
	}

	// The route is not registered when the courses feature is off, which
	// also comes back as a 404.
	if (response.status === 404) throw error(404, 'Not Found');
	if (!response.ok) throw error(502, 'Courses are unavailable right now');

	const course = toCourseSummary(await response.json());

	// Progress and certificates are keyed to the learner cookie. Without one
	// there is nothing to ask for.
	let progress: LearnerProgress | null = null;
	let certificate: CertificateState | null = null;
	if (purchaseCookie && course.has_access) {
		[progress, certificate] = await Promise.all([
			course.enable_progress ? loadLearnerProgress(fetch, pbUrl, params.slug, headers) : null,
			course.enable_certificates ? loadLearnerCertificate(fetch, pbUrl, params.slug, headers) : null
		]);
	}

	return { course, progress, certificate };
};

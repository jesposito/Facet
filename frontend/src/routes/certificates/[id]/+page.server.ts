/**
 * Certificate check: /certificates/[id]
 *
 * The public page behind a certificate link. It shows only what the backend
 * verify endpoint returns, which is what is printed on the certificate, so
 * learners can share it and anyone can confirm it was issued here.
 */

import type { PageServerLoad } from './$types';
import { error } from '@sveltejs/kit';
import { toCertificate } from '$lib/commerce';

export const load: PageServerLoad = async ({ params, fetch, setHeaders }) => {
	if (!/^[A-Za-z0-9_-]{1,128}$/.test(params.id)) throw error(404, 'Certificate not found');

	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	let response: Response;
	try {
		response = await fetch(`${pbUrl}/api/certificates/verify/${encodeURIComponent(params.id)}`, {
			headers: { 'X-Internal': 'true' }
		});
	} catch (err) {
		console.error('Failed to verify certificate:', err);
		throw error(503, 'Certificates are unavailable right now');
	}

	if (response.status === 404) throw error(404, 'Certificate not found');
	if (!response.ok) throw error(502, 'Certificates are unavailable right now');

	const certificate = toCertificate(await response.json());
	if (!certificate) throw error(502, 'Certificates are unavailable right now');

	// Certificates link to a person by name, so keep them out of shared caches.
	setHeaders({ 'cache-control': 'private, no-store' });
	return { certificate };
};

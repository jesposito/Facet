import type { RequestHandler } from './$types';
import { buildWebManifest, type ManifestProfile, type ManifestSiteSettings } from '$lib/web-manifest';

export const GET: RequestHandler = async ({ fetch }) => {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	let profile: ManifestProfile | null = null;
	let settings: ManifestSiteSettings | null = null;

	try {
		const [homepageRes, settingsRes] = await Promise.all([
			fetch(`${pbUrl}/api/homepage`, { headers: { 'X-Internal': 'true' } }),
			fetch(`${pbUrl}/api/site-settings`, { headers: { 'X-Internal': 'true' } })
		]);

		if (homepageRes.ok) {
			const homepage = await homepageRes.json();
			profile = homepage?.profile ?? null;
		}
		if (settingsRes.ok) {
			settings = await settingsRes.json();
		}
	} catch {
		// A manifest should still be valid while PocketBase is starting.
	}

	return new Response(JSON.stringify(buildWebManifest(profile, settings)), {
		headers: {
			'Content-Type': 'application/manifest+json',
			'Cache-Control': 'public, max-age=3600'
		}
	});
};

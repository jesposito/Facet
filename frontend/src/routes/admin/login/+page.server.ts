import PocketBase from 'pocketbase';
import type { PageServerLoad } from './$types';

// Server-side prefetch of enabled auth providers to avoid client-side
// discovery failures (e.g., mixed content, host mismatches).
export const load: PageServerLoad = async () => {
	const pbUrl =
		process.env.POCKETBASE_URL || process.env.VITE_POCKETBASE_URL || 'http://localhost:8090';

	try {
		const pb = new PocketBase(pbUrl);
		const methods = (await pb.collection('users').listAuthMethods()) as any;

		const oauthProviders =
			methods?.oauth2?.providers?.map((p: { name?: string }) => p.name).filter(Boolean) ??
			methods?.authProviders?.map((p: { name?: string }) => p.name).filter(Boolean) ??
			[];

		const passwordAuthEnabled = methods?.password?.enabled ?? true;

		return {
			oauthProviders,
			passwordAuthEnabled,
			initialAuthLoaded: true
		};
	} catch (err) {
		console.error('[LOGIN] Failed to prefetch auth methods', err);

		return {
			oauthProviders: [],
			passwordAuthEnabled: true,
			initialAuthLoaded: false
		};
	}
};

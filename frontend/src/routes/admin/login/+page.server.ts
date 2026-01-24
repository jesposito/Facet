import PocketBase from 'pocketbase';
import type { PageServerLoad } from './$types';
import type { AuthMethodsList, AuthProviderInfo } from 'pocketbase';

// Extended type to handle both old PocketBase API (oauth2.providers) and new (authProviders)
interface ExtendedAuthMethods extends Partial<AuthMethodsList> {
	oauth2?: { providers?: AuthProviderInfo[] };
	password?: { enabled?: boolean };
}

// Server-side prefetch of enabled auth providers to avoid client-side
// discovery failures (e.g., mixed content, host mismatches).
export const load: PageServerLoad = async () => {
	const pbUrl =
		process.env.POCKETBASE_URL || process.env.VITE_POCKETBASE_URL || 'http://localhost:8090';

	try {
		const pb = new PocketBase(pbUrl);
		const methods = (await pb.collection('users').listAuthMethods()) as ExtendedAuthMethods;

		const oauthProviders =
			methods?.oauth2?.providers?.map((p) => p.name).filter(Boolean) ??
			methods?.authProviders?.map((p) => p.name).filter(Boolean) ??
			[];

		// Handle both old (password.enabled) and new (usernamePassword/emailPassword) APIs
		const passwordAuthEnabled =
			methods?.password?.enabled ?? methods?.usernamePassword ?? methods?.emailPassword ?? true;

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

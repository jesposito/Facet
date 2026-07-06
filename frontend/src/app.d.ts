// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces

import type PocketBase from 'pocketbase';

declare global {
	const __APP_VERSION__: string;

	namespace App {
		interface SiteSettings {
			faviconUrl: string | null;
			customCSS: string | null;
			defaultLocale: string | null;
			showAvatar: boolean;
			defaultThemeMode: string;
			design: 'classic' | 'soft-premium';
		}

		// interface Error {}
		interface Locals {
			/** PocketBase instance with auth loaded from cookie (if present) */
			pb: PocketBase;
			/** Request-scoped site settings loaded once in hooks.server.ts */
			siteSettings?: SiteSettings;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};

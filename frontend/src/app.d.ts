// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces

import type PocketBase from 'pocketbase';

declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			/** PocketBase instance with auth loaded from cookie (if present) */
			pb: PocketBase;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};

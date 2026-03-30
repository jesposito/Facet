import { writable } from 'svelte/store';

export interface SiteNavItem {
	viewId: string;
	slug: string;
	label: string;
	name: string;
}

export interface SiteNavState {
	enabled: boolean;
	items: SiteNavItem[];
	loaded: boolean;
	mobileMenuOpen: boolean;
}

const initialState: SiteNavState = {
	enabled: false,
	items: [],
	loaded: false,
	mobileMenuOpen: false
};

function createSiteNavStore() {
	const { subscribe, update } = writable<SiteNavState>(initialState);

	return {
		subscribe,
		/** Initialize from SSR layout data. Preserves mobileMenuOpen across re-inits. */
		initFromSSR(data: { enabled: boolean; items: SiteNavItem[] } | undefined) {
			if (!data) return;
			update((state) => ({
				enabled: data.enabled,
				items: data.items || [],
				loaded: true,
				mobileMenuOpen: state.mobileMenuOpen
			}));
		},
		toggleMobileMenu() {
			update((state) => ({ ...state, mobileMenuOpen: !state.mobileMenuOpen }));
		},
		closeMobileMenu() {
			update((state) => ({ ...state, mobileMenuOpen: false }));
		}
	};
}

export const siteNavStore = createSiteNavStore();

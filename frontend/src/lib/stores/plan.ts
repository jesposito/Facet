import { writable, derived, get, type Readable } from 'svelte/store';
import { fetchWithTimeout } from '$lib/utils';

export interface PlanFeatures {
	basic_analytics: boolean;
	analytics: boolean;
	badge_forced: boolean;
	courses: boolean;
	api: boolean;
	custom_domain: boolean;
	newsletter: boolean;
	discussions: boolean;
	pricing: boolean;
}

export interface PlanConfig {
	managed: boolean;
	plan: string;
	features: PlanFeatures;
	storage_limit_mb?: number;
	custom_domain?: string;
	demo_mode?: boolean;
}

const defaultConfig: PlanConfig = {
	managed: false,
	plan: 'self-hosted',
	features: {
		basic_analytics: true,
		analytics: true,
		badge_forced: false,
		// Courses default OFF self-hosted: public /courses/[slug] route is not implemented
		// in this repo (cloud only). Backend authority via /api/plan can flip this on
		// when FACET_FEATURE_COURSES=true is set on the server.
		courses: false,
		api: true,
		custom_domain: true,
		newsletter: true,
		discussions: true,
		pricing: true
	}
};

export const planConfig = writable<PlanConfig>(defaultConfig);
export const planInitialized = writable<boolean>(false);

export const isManaged = derived(planConfig, ($config) => $config.managed);
export const currentPlan = derived(planConfig, ($config) => $config.plan);
export const planFeatures = derived(planConfig, ($config) => $config.features);
export const brandName = derived(planConfig, ($config) => ($config.managed ? 'Facet Cloud' : 'Facet'));
export const isDemoMode = derived(planConfig, ($config) => $config.demo_mode === true);

// Features that are not fully implemented in the self-hosted build and must
// honor the explicit flag value even when not in managed mode. Adding a feature
// here means "self-hosted does not unconditionally get this feature".
const selfHostedOptInFeatures: ReadonlySet<keyof PlanFeatures> = new Set(['courses']);

// Non-reactive version for use in script blocks
export function checkFeature(feature: keyof PlanFeatures): boolean {
	const config = get(planConfig);
	if (selfHostedOptInFeatures.has(feature)) return config.features[feature] === true;
	if (!config.managed) return true;
	return config.features[feature] ?? false;
}

// Reactive version for use in templates
export function hasFeature(feature: keyof PlanFeatures): Readable<boolean> {
	return derived(planConfig, ($config) => {
		if (selfHostedOptInFeatures.has(feature)) return $config.features[feature] === true;
		if (!$config.managed) return true;
		return $config.features[feature] ?? false;
	});
}

export async function initPlan() {
	try {
		const response = await fetchWithTimeout('/api/plan');
		if (response.ok) {
			const data: PlanConfig = await response.json();
			planConfig.set(data);
		}
	} catch {
		// On error, keep defaults (self-hosted with all features)
	}
	planInitialized.set(true);
}

/**
 * Initialize plan config from SSR data (if available)
 * This eliminates FOUC by setting the correct plan state before first render
 */
export function initPlanFromSSR(data: PlanConfig | null) {
	if (data) {
		planConfig.set(data);
		planInitialized.set(true);
	}
}

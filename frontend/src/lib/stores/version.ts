/**
 * Shared version checking store
 * Used by AdminSidebar and About page
 */
import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

const CACHE_KEY = 'facet_version_check';
const CACHE_DURATION = 4 * 60 * 60 * 1000; // 4 hours

// Get app version from global
const appVersion = typeof __APP_VERSION__ !== 'undefined' ? __APP_VERSION__ : 'dev';

// Stores
export const latestVersion = writable<string>('');
export const newVersionAvailable = writable<boolean>(false);
export const isCheckingVersion = writable<boolean>(false);

// Compare semver versions: returns 1 if a > b, -1 if a < b, 0 if equal
function compareVersions(a: string, b: string): number {
	const partsA = a.split('.').map(Number);
	const partsB = b.split('.').map(Number);

	for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
		const numA = partsA[i] || 0;
		const numB = partsB[i] || 0;
		if (numA > numB) return 1;
		if (numA < numB) return -1;
	}
	return 0;
}

// Clean version string for comparison
function cleanVersion(version: string): string {
	return version.replace(/^v/, '').split('-')[0];
}

// Calculate if update is available
function calculateHasUpdate(latest: string, current: string): boolean {
	const latestClean = cleanVersion(latest);
	const currentClean = cleanVersion(current);
	return Boolean(latestClean && currentClean !== latestClean && compareVersions(latestClean, currentClean) > 0);
}

/**
 * Check for new version (uses cache unless forced)
 * @param force - If true, bypasses cache and fetches fresh data
 */
export async function checkForNewVersion(force = false): Promise<void> {
	if (!browser) return;

	// Prevent concurrent checks
	if (get(isCheckingVersion)) return;
	isCheckingVersion.set(true);

	try {
		// Check cache first (unless forcing)
		if (!force) {
			const cached = localStorage.getItem(CACHE_KEY);
			if (cached) {
				const { timestamp, latest } = JSON.parse(cached);
				if (Date.now() - timestamp < CACHE_DURATION) {
					// Recalculate hasUpdate against current version (may have changed since cache)
					const hasUpdate = calculateHasUpdate(latest, appVersion);
					latestVersion.set(latest);
					newVersionAvailable.set(hasUpdate);
					isCheckingVersion.set(false);
					return;
				}
			}
		}

		// Fetch latest release from GitHub
		const response = await fetch('https://api.github.com/repos/jesposito/Facet/releases/latest', {
			headers: { Accept: 'application/vnd.github.v3+json' }
		});

		if (!response.ok) {
			isCheckingVersion.set(false);
			return;
		}

		const data = await response.json();
		const latest = data.tag_name || '';

		const hasUpdate = calculateHasUpdate(latest, appVersion);

		// Cache result
		localStorage.setItem(
			CACHE_KEY,
			JSON.stringify({
				timestamp: Date.now(),
				latest
			})
		);

		latestVersion.set(latest);
		newVersionAvailable.set(hasUpdate);
	} catch {
		// Silently fail - version check is non-critical
	} finally {
		isCheckingVersion.set(false);
	}
}

/**
 * Clear the version cache and reset state
 */
export function clearVersionCache(): void {
	if (!browser) return;
	localStorage.removeItem(CACHE_KEY);
	latestVersion.set('');
	newVersionAvailable.set(false);
}

/**
 * Force a fresh version check (clears cache first)
 */
export async function forceVersionCheck(): Promise<void> {
	clearVersionCache();
	await checkForNewVersion(true);
}

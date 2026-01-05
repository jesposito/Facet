import { writable, get } from 'svelte/store';
import { pb } from '$lib/pocketbase';

// Store for demo mode state
export const demoMode = writable(false);

// Initialize demo mode state with timeout protection
export async function initDemoMode() {
	try {
		// Add 5-second timeout to prevent hanging on slow/failed requests
		const controller = new AbortController();
		const timeoutId = setTimeout(() => controller.abort(), 5000);

		const response = await fetch('/api/demo/status', {
			headers: { Authorization: pb.authStore.token },
			signal: controller.signal
		});

		clearTimeout(timeoutId);

		if (response.ok) {
			const data = await response.json();
			const newDemoMode = data.demo_mode || false;
			demoMode.set(newDemoMode);
		} else {
			// Non-ok response (401, 403, 500, etc) - default to false
			demoMode.set(false);
		}
	} catch (err) {
		// Network error, timeout, or abort - default to false and continue
		console.error('[DEMO] Failed to check demo status:', err);
		demoMode.set(false);
	}
}

// Get collection name based on demo mode
export function getCollectionName(baseName: string): string {
	const currentDemoMode = get(demoMode);
	if (currentDemoMode) {
		const demoName = 'demo_' + baseName;
		return demoName;
	}
	return baseName;
}

// Wrapper for pb.collection() that routes to demo collections when demo mode is ON
export function collection(name: string) {
	const collectionName = getCollectionName(name);
	return pb.collection(collectionName);
}

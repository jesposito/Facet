import { writable, get } from 'svelte/store';
import { pb } from '$lib/pocketbase';

// Store for demo mode state
export const demoMode = writable(false);

// Initialize demo mode state
export async function initDemoMode() {
	console.log('[DEMO] initDemoMode() called');
	try {
		const response = await fetch('/api/demo/status', {
			headers: { Authorization: pb.authStore.token }
		});
		console.log('[DEMO] /api/demo/status response:', response.status, response.ok);
		if (response.ok) {
			const data = await response.json();
			console.log('[DEMO] Status response data:', data);
			const newDemoMode = data.demo_mode || false;
			console.log('[DEMO] Setting demo mode to:', newDemoMode);
			demoMode.set(newDemoMode);
			console.log('[DEMO] Demo mode store updated to:', newDemoMode);
		}
	} catch (err) {
		console.error('[DEMO] Failed to check demo status:', err);
	}
}

// Get collection name based on demo mode
export function getCollectionName(baseName: string): string {
	const currentDemoMode = get(demoMode);
	console.log(`[DEMO] getCollectionName("${baseName}") - demo mode is ${currentDemoMode}`);
	if (currentDemoMode) {
		const demoName = 'demo_' + baseName;
		console.log(`[DEMO] Routing to demo collection: ${demoName}`);
		return demoName;
	}
	console.log(`[DEMO] Routing to normal collection: ${baseName}`);
	return baseName;
}

// Wrapper for pb.collection() that routes to demo collections when demo mode is ON
export function collection(name: string) {
	const collectionName = getCollectionName(name);
	console.log(`[DEMO] collection("${name}") => pb.collection("${collectionName}")`);
	return pb.collection(collectionName);
}

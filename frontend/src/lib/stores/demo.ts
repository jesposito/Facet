import { writable, get } from 'svelte/store';
import { pb } from '$lib/pocketbase';

// Store for demo mode state
export const demoMode = writable(false);

// Initialize demo mode state
export async function initDemoMode() {
	try {
		const response = await fetch('/api/demo/status', {
			headers: { Authorization: pb.authStore.token }
		});
		if (response.ok) {
			const data = await response.json();
			const newDemoMode = data.demo_mode || false;
			demoMode.set(newDemoMode);
		}
	} catch (err) {
		console.error('[DEMO] Failed to check demo status:', err);
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

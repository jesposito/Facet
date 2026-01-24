import { writable, get } from 'svelte/store';
import { pb } from '$lib/pocketbase';
import { fetchWithTimeout } from '$lib/utils';

export const demoMode = writable(false);

export async function initDemoMode() {
	try {
		const response = await fetchWithTimeout('/api/demo/status', {
			headers: { Authorization: pb.authStore.token }
		});

		if (response.ok) {
			const data = await response.json();
			demoMode.set(data.demo_mode || false);
		} else {
			demoMode.set(false);
		}
	} catch {
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

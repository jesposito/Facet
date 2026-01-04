import { pb } from '$lib/pocketbase';

// Check if demo mode is currently enabled
export async function isDemoMode(): Promise<boolean> {
	try {
		const response = await fetch('/api/demo/status', {
			headers: {
				Authorization: pb.authStore.token
			}
		});
		if (response.ok) {
			const data = await response.json();
			return data.demo_mode || false;
		}
	} catch (err) {
		console.error('Failed to check demo mode:', err);
	}
	return false;
}

// Get the correct collection name based on demo mode
export async function getCollectionName(baseName: string): Promise<string> {
	const demoMode = await isDemoMode();
	if (demoMode) {
		return 'demo_' + baseName;
	}
	return baseName;
}

// Wrapper for pb.collection() that automatically routes to demo collections when demo mode is ON
export async function collection(name: string) {
	const collectionName = await getCollectionName(name);
	return pb.collection(collectionName);
}

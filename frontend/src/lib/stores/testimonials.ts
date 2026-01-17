import { writable } from 'svelte/store';

export const testimonialsStore = writable({
	pendingCount: 0
});

let refreshTimeout: ReturnType<typeof setTimeout> | null = null;

export async function refreshTestimonialsPendingCount() {
	try {
		const pb = await import('$lib/pocketbase').then(m => m.pb);
		if (!pb.authStore.isValid) {
			testimonialsStore.update(store => ({ ...store, pendingCount: 0 }));
			return;
		}

		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
			Authorization: `Bearer ${pb.authStore.token}`
		};

		const response = await fetch('/api/testimonials/pending-count', { headers });
		if (response.ok) {
			const data = await response.json();
			testimonialsStore.update(store => ({ ...store, pendingCount: data.count || 0 }));
		} else {
			testimonialsStore.update(store => ({ ...store, pendingCount: 0 }));
		}
	} catch {
		testimonialsStore.update(store => ({ ...store, pendingCount: 0 }));
	}
}

export function scheduleTestimonialsRefresh(delayMs: number = 500) {
	if (refreshTimeout) {
		clearTimeout(refreshTimeout);
	}
	refreshTimeout = setTimeout(() => {
		refreshTimeout = null;
		refreshTestimonialsPendingCount();
	}, delayMs);
}
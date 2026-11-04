import { beforeNavigate } from '$app/navigation';

// Call during component init. A tab close goes through SvelteKit's own beforeunload handler,
// where cancel() shows the browser's one prompt, so this adds no listener of its own.
export function guardUnsaved(isDirty: () => boolean, message: () => string) {
	beforeNavigate((navigation) => {
		if ((navigation.shallow && navigation.type === 'goto') || !isDirty()) return;
		if (navigation.type === 'leave' || !window.confirm(message())) navigation.cancel();
	});
}

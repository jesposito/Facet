/**
 * Svelte action for IntersectionObserver
 * Used to detect when elements enter/leave the viewport
 */
export type IntersectionParams = IntersectionObserverInit & {
	onChange: (entry: IntersectionObserverEntry) => void;
};

export function intersection(node: Element, params: IntersectionParams) {
	let observer: IntersectionObserver | null = null;

	function setup(p: IntersectionParams) {
		observer?.disconnect();
		observer = new IntersectionObserver((entries) => {
			if (entries[0]) p.onChange(entries[0]);
		}, p);
		observer.observe(node);
	}

	setup(params);

	return {
		update: setup,
		destroy() {
			observer?.disconnect();
		}
	};
}

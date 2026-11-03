import { expect, test } from '@playwright/test';
import { injectGA, removeGA, sanitizeGAId, shouldLoadGA } from '../src/lib/analytics/ga';

// Just enough of a document for the loader.
type FakeEl = {
	id: string;
	src?: string;
	async?: boolean;
	textContent?: string;
	remove: () => void;
};
function fakeDocument() {
	const head: FakeEl[] = [];
	const win: Record<string, unknown> = {};
	const doc = {
		defaultView: win,
		head: {
			appendChild(el: FakeEl) {
				el.remove = () => head.splice(head.indexOf(el), 1);
				head.push(el);
			}
		},
		createElement: () => ({ id: '' }) as FakeEl,
		getElementById: (id: string) => head.find((el) => el.id === id) ?? null
	};
	return { doc: doc as unknown as Document, head, win };
}

test.describe('Google Analytics loader', () => {
	test('the Analytics switch wins over a saved ID', () => {
		expect(shouldLoadGA({ analytics: false }, 'G-ABC123')).toBe(false);
		expect(shouldLoadGA({ analytics: true }, 'G-ABC123')).toBe(true);
		expect(shouldLoadGA({ analytics: true }, '')).toBe(false);
		expect(shouldLoadGA({ analytics: true }, null)).toBe(false);
		expect(shouldLoadGA({ analytics: true }, '"><script>'), 'sanitized to "script"').toBe(true);
	});

	test('IDs are reduced to safe characters', () => {
		expect(sanitizeGAId(" G-ABC'); alert(1);// ")).toBe('G-ABCalert1');
	});

	test('the snippet is added once and removed when switched off', () => {
		const { doc, head, win } = fakeDocument();
		injectGA(doc, 'G-ABC123');
		injectGA(doc, 'G-ABC123');
		expect(head, 'loader and inline config, once').toHaveLength(2);
		expect(head[0].src).toBe('https://www.googletagmanager.com/gtag/js?id=G-ABC123');

		removeGA(doc, 'G-ABC123');
		expect(head).toHaveLength(0);
		expect(win['ga-disable-G-ABC123'], 'opt-out flag set for an already loaded gtag').toBe(true);
	});

	test('nothing is added for an empty ID', () => {
		const { doc, head } = fakeDocument();
		injectGA(doc, '  ');
		expect(head).toHaveLength(0);
	});
});

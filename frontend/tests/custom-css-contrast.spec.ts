import { expect, test } from '@playwright/test';
import { analyzeCustomCSSContrast } from '../src/lib/a11y/custom-css-contrast';

test.describe('custom CSS contrast warning', () => {
	test('warns for obvious low contrast text and background pairs', () => {
		const result = analyzeCustomCSSContrast('body { color: #f5f5f5; background: #fff; }');

		expect(result.shouldWarn).toBe(true);
		expect(result.lowContrastPairs).toBe(1);
	});

	test('does not warn for high contrast text and background pairs', () => {
		const result = analyzeCustomCSSContrast('main { color: #111827; background-color: white; }');

		expect(result.shouldWarn).toBe(false);
		expect(result.lowContrastPairs).toBe(0);
	});

	test('warns when primary palette variables are overridden', () => {
		const result = analyzeCustomCSSContrast(':root { --color-primary-600: #ffffcc; }');

		expect(result.shouldWarn).toBe(true);
		expect(result.primaryColorOverrides).toBe(true);
	});

	test('parses rgb colors', () => {
		const result = analyzeCustomCSSContrast('.hero { color: rgb(230, 230, 230); background-color: rgb(255, 255, 255); }');

		expect(result.shouldWarn).toBe(true);
		expect(result.lowContrastPairs).toBe(1);
	});
});

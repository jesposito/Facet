import { test, expect, type Page } from '@playwright/test';

// Editorial utilities, always on under Soft Premium: .font-accent uses Newsreader
// italic and .text-gradient clips a gradient to the text.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';

// Inject a probe element and read computed values for the editorial utilities.
const probe = (page: Page) =>
	page.evaluate(() => {
		const wrap = document.createElement('div');
		wrap.innerHTML =
			'<span class="font-accent">x</span><span class="text-gradient">y</span><hr class="divider-editorial"/>';
		document.body.appendChild(wrap);
		const accent = getComputedStyle(wrap.querySelector('.font-accent')!);
		const gradient = getComputedStyle(wrap.querySelector('.text-gradient')!);
		const r = {
			font: accent.fontFamily.toLowerCase(),
			style: accent.fontStyle,
			// text-gradient clips a gradient to the text → transparent fill + a bg image.
			gradientFill: gradient.webkitTextFillColor || gradient.color,
			gradientImg: gradient.backgroundImage
		};
		wrap.remove();
		return r;
	});

test.describe('Soft Premium editorial utilities (always on)', () => {
	test('font-accent is Newsreader italic; gradient clips', async ({ page }) => {
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const r = await probe(page);
		expect(r.font).toContain('newsreader');
		expect(r.style).toBe('italic');
		expect(r.gradientImg).toContain('gradient');
	});
});

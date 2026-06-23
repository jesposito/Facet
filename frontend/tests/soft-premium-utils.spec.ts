import { test, expect, type Page } from '@playwright/test';

// B5: editorial utilities. .font-accent uses Newsreader under soft-premium and
// falls back to the heading font in classic; .grain only paints under soft-premium.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const LOGIN_EMAIL = process.env.ADMIN_EMAIL ?? 'admin@example.com';
const LOGIN_PASSWORD = process.env.ADMIN_PASSWORD ?? 'changeme123';

async function login(page: Page) {
	await page.goto(`${BASE_URL}/admin/login`);
	await page.waitForSelector('input[type="email"]', { timeout: 10_000 });
	await page.fill('input[type="email"]', LOGIN_EMAIL);
	await page.fill('input[type="password"]', LOGIN_PASSWORD);
	await page.click('button[type="submit"]');
	await page.waitForURL((url) => /\/admin(\/|$)/.test(url.pathname) && !url.pathname.includes('/login'), {
		timeout: 15_000
	});
}
async function setDesign(page: Page, label: 'Classic' | 'Soft Premium') {
	await page.goto(`${BASE_URL}/admin/settings/site`);
	await page.waitForSelector('[aria-labelledby="design-style-heading"]', { timeout: 10_000 });
	const radio = page.locator('[role="radio"]', { hasText: label }).first();
	if ((await radio.getAttribute('aria-checked')) === 'false') {
		await radio.click();
		await expect(radio).toHaveAttribute('aria-checked', 'true');
	}
}
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

test.describe('Soft Premium editorial utilities', () => {
	test.beforeEach(async ({ page }) => login(page));
	test.afterEach(async ({ page }) => setDesign(page, 'Classic'));

	test('classic: font-accent falls back to heading serif (no Newsreader)', async ({ page }) => {
		await setDesign(page, 'Classic');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const r = await probe(page);
		expect(r.font).not.toContain('newsreader');
		expect(r.font).toContain('lora'); // classic heading fallback
		// text-gradient is mode-independent (uses the accent ramp).
		expect(r.gradientImg).toContain('gradient');
	});

	test('soft-premium: font-accent is Newsreader italic; gradient clips', async ({ page }) => {
		await setDesign(page, 'Soft Premium');
		await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
		const r = await probe(page);
		expect(r.font).toContain('newsreader');
		expect(r.style).toBe('italic');
		expect(r.gradientImg).toContain('gradient');
	});
});

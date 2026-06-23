import { test, expect } from '@playwright/test';

// Verifies the universal AAA accent clamp. The clamp lives in
// generatePaletteFromHex (src/lib/colors.ts), which both the SSR accent emitter
// and the client accent applier use for custom-hex accents. We exercise the real
// bundled module through the Vite dev server so this tests the exact code the
// product ships — no mocks.
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';
const COLORS_MOD = '/src/lib/colors.ts';

function whiteContrast(hex: string) {
	const h = hex.trim().replace('#', '');
	const [r, g, b] = [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
	const f = (c: number) => {
		const cc = c / 255;
		return cc <= 0.03928 ? cc / 12.92 : Math.pow((cc + 0.055) / 1.055, 2.4);
	};
	const lum = 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
	return { ratio: 1.05 / (lum + 0.05), r, g, b };
}

test('generatePaletteFromHex clamps white-on-600 to AAA (7:1), preserving hue', async ({ page }) => {
	await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
	// Import the real source module via Vite (dev server transforms TS).
	const palettes = await page.evaluate(async (modUrl) => {
		const mod = (await import(/* @vite-ignore */ modUrl)) as {
			generatePaletteFromHex: (hex: string) => Record<string, string>;
		};
		const hexes = [
			'#ffe600', // pale yellow — worst case for white text
			'#22d3ee', // cyan
			'#a3e635', // lime
			'#f472b6', // pink
			'#0ea5e9', // sky (default-ish)
			'#7c3aed' // violet (already dark enough)
		];
		return hexes.map((h) => ({ h, scale: mod.generatePaletteFromHex(h) }));
	}, COLORS_MOD);

	for (const { h, scale } of palettes) {
		const six = scale['600'];
		expect(six, `${h} produced 600=${six}`).toMatch(/^#[0-9a-fA-F]{6}$/);
		const { ratio, r, g, b } = whiteContrast(six);
		// AAA (epsilon for 8-bit rounding).
		expect(ratio, `white-on-600 for ${h} -> ${six} = ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(6.9);
		// Hue preserved (mixing toward black keeps channel spread; not desaturated to gray).
		expect(Math.max(r, g, b) - Math.min(r, g, b), `${h} -> ${six} kept hue`).toBeGreaterThan(6);
	}
});

test('clamp only deepens — a hue already dark enough is left at its nominal 600', async ({ page }) => {
	await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
	const { darkSix, paleSix } = await page.evaluate(async (modUrl) => {
		const mod = (await import(/* @vite-ignore */ modUrl)) as {
			generatePaletteFromHex: (hex: string) => Record<string, string>;
		};
		return {
			// #1e3a8a (deep blue) already clears 7:1 at the nominal 0.15 darkening.
			darkSix: mod.generatePaletteFromHex('#1e3a8a')['600'],
			paleSix: mod.generatePaletteFromHex('#ffe600')['600']
		};
	}, COLORS_MOD);
	// Deep blue: nominal 0.15 mix toward black of #1e3a8a = #1a3175.
	expect(darkSix.toLowerCase()).toBe('#1a3175');
	// Pale yellow had to be deepened well past the nominal mix.
	expect(whiteContrast(paleSix).ratio).toBeGreaterThanOrEqual(6.9);
});

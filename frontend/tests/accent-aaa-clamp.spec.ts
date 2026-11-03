import { test, expect } from '@playwright/test';
import { generatePaletteFromHex } from '../src/lib/colors';

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

test('generatePaletteFromHex clamps white-on-600 to AAA (7:1), preserving hue', () => {
	const hexes = ['#ffe600', '#22d3ee', '#a3e635', '#f472b6', '#0ea5e9', '#7c3aed'];
	const palettes = hexes.map((h) => ({ h, scale: generatePaletteFromHex(h) }));

	for (const { h, scale } of palettes) {
		const six = scale['600'];
		expect(six, `${h} produced 600=${six}`).toMatch(/^#[0-9a-fA-F]{6}$/);
		const { ratio, r, g, b } = whiteContrast(six);
		expect(ratio, `white-on-600 for ${h} -> ${six} = ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(
			6.9
		);
		expect(Math.max(r, g, b) - Math.min(r, g, b), `${h} -> ${six} kept hue`).toBeGreaterThan(6);
	}
});

function relLuminance(hex: string) {
	const h = hex.trim().replace('#', '');
	const [r, g, b] = [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
	const f = (c: number) => {
		const cc = c / 255;
		return cc <= 0.03928 ? cc / 12.92 : Math.pow((cc + 0.055) / 1.055, 2.4);
	};
	return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}

test('palette ramp is monotonic from 600 to 950 even for pale accents whose 600 clamps deep', () => {
	const hexes = ['#ffe066', '#22d3ee', '#ffe600', '#a3e635', '#c2410c', '#1e3a8a'];
	const palettes = hexes.map((h) => ({ h, scale: generatePaletteFromHex(h) }));

	for (const { h, scale } of palettes) {
		const steps = ['500', '600', '700', '800', '900', '950'] as const;
		const lums = steps.map((s) => relLuminance(scale[s]));
		for (let i = 1; i < steps.length; i++) {
			expect(
				lums[i],
				`${h}: ${steps[i]}=${scale[steps[i]]} (L=${lums[i].toFixed(4)}) must be darker than ${steps[i - 1]}=${scale[steps[i - 1]]} (L=${lums[i - 1].toFixed(4)})`
			).toBeLessThan(lums[i - 1] + 1e-9);
		}
	}
});

test('clamp only deepens — a hue already dark enough is left at its nominal 600', () => {
	const darkSix = generatePaletteFromHex('#1e3a8a')['600'];
	const paleSix = generatePaletteFromHex('#ffe600')['600'];
	expect(darkSix.toLowerCase()).toBe('#1a3175');
	expect(whiteContrast(paleSix).ratio).toBeGreaterThanOrEqual(6.9);
});

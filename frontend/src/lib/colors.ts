/**
 * Accent Color Palette Constants
 *
 * Phase 6.5: Curated palette approach for accent color customization.
 * All colors are pre-tested for WCAG contrast compliance.
 * Uses Tailwind CSS color scale values (50-950) for each accent color.
 */

export type AccentColor = 'sky' | 'indigo' | 'emerald' | 'rose' | 'amber' | 'slate';

/**
 * Admin Tag Color Palette
 * Curated colors for admin-only tags to visually distinguish content items.
 * These are lighter/more subtle than accent colors for badge use.
 */
export type TagColor = 'blue' | 'green' | 'yellow' | 'red' | 'purple' | 'pink' | 'orange' | 'teal' | 'gray';

export interface TagColorInfo {
	name: TagColor;
	label: string;
	bg: string;       // Background class (light mode)
	bgDark: string;   // Background class (dark mode)
	text: string;     // Text class (light mode)
	textDark: string; // Text class (dark mode)
	border: string;   // Border class (light mode)
	borderDark: string; // Border class (dark mode)
}

export const TAG_COLORS: Record<TagColor, TagColorInfo> = {
	blue: {
		name: 'blue',
		label: 'Blue',
		bg: 'bg-blue-100',
		bgDark: 'dark:bg-blue-900/30',
		text: 'text-blue-700',
		textDark: 'dark:text-blue-300',
		border: 'border-blue-200',
		borderDark: 'dark:border-blue-800'
	},
	green: {
		name: 'green',
		label: 'Green',
		bg: 'bg-green-100',
		bgDark: 'dark:bg-green-900/30',
		text: 'text-green-700',
		textDark: 'dark:text-green-300',
		border: 'border-green-200',
		borderDark: 'dark:border-green-800'
	},
	yellow: {
		name: 'yellow',
		label: 'Yellow',
		bg: 'bg-yellow-100',
		bgDark: 'dark:bg-yellow-900/30',
		text: 'text-yellow-700',
		textDark: 'dark:text-yellow-300',
		border: 'border-yellow-200',
		borderDark: 'dark:border-yellow-800'
	},
	red: {
		name: 'red',
		label: 'Red',
		bg: 'bg-red-100',
		bgDark: 'dark:bg-red-900/30',
		text: 'text-red-700',
		textDark: 'dark:text-red-300',
		border: 'border-red-200',
		borderDark: 'dark:border-red-800'
	},
	purple: {
		name: 'purple',
		label: 'Purple',
		bg: 'bg-purple-100',
		bgDark: 'dark:bg-purple-900/30',
		text: 'text-purple-700',
		textDark: 'dark:text-purple-300',
		border: 'border-purple-200',
		borderDark: 'dark:border-purple-800'
	},
	pink: {
		name: 'pink',
		label: 'Pink',
		bg: 'bg-pink-100',
		bgDark: 'dark:bg-pink-900/30',
		text: 'text-pink-700',
		textDark: 'dark:text-pink-300',
		border: 'border-pink-200',
		borderDark: 'dark:border-pink-800'
	},
	orange: {
		name: 'orange',
		label: 'Orange',
		bg: 'bg-orange-100',
		bgDark: 'dark:bg-orange-900/30',
		text: 'text-orange-700',
		textDark: 'dark:text-orange-300',
		border: 'border-orange-200',
		borderDark: 'dark:border-orange-800'
	},
	teal: {
		name: 'teal',
		label: 'Teal',
		bg: 'bg-teal-100',
		bgDark: 'dark:bg-teal-900/30',
		text: 'text-teal-700',
		textDark: 'dark:text-teal-300',
		border: 'border-teal-200',
		borderDark: 'dark:border-teal-800'
	},
	gray: {
		name: 'gray',
		label: 'Gray',
		bg: 'bg-gray-100',
		bgDark: 'dark:bg-gray-700',
		text: 'text-gray-700',
		textDark: 'dark:text-gray-300',
		border: 'border-gray-200',
		borderDark: 'dark:border-gray-600'
	}
};

export const TAG_COLOR_LIST: TagColor[] = ['blue', 'green', 'yellow', 'red', 'purple', 'pink', 'orange', 'teal', 'gray'];

export const DEFAULT_TAG_COLOR: TagColor = 'blue';

export function getTagColor(name?: string): TagColorInfo {
	if (name && name in TAG_COLORS) {
		return TAG_COLORS[name as TagColor];
	}
	return TAG_COLORS[DEFAULT_TAG_COLOR];
}

export function getTagColorClasses(color?: string): string {
	const info = getTagColor(color);
	return `${info.bg} ${info.bgDark} ${info.text} ${info.textDark} ${info.border} ${info.borderDark}`;
}

export interface ColorScale {
	50: string;
	100: string;
	200: string;
	300: string;
	400: string;
	500: string;
	600: string;
	700: string;
	800: string;
	900: string;
	950: string;
}

export interface AccentColorInfo {
	name: AccentColor;
	label: string;
	description: string;
	scale: ColorScale;
}

/**
 * Curated accent color palette.
 * Each color has been selected for:
 * - Professional appearance across industries
 * - WCAG AA contrast compliance
 * - Distinct visual identity
 */
export const ACCENT_COLORS: Record<AccentColor, AccentColorInfo> = {
	sky: {
		name: 'sky',
		label: 'Sky',
		description: 'Tech, software, professional',
		scale: {
			50: '#f0f9ff',
			100: '#e0f2fe',
			200: '#bae6fd',
			300: '#7dd3fc',
			400: '#38bdf8',
			500: '#0ea5e9',
			600: '#0284c7',
			700: '#0369a1',
			800: '#075985',
			900: '#0c4a6e',
			950: '#082f49'
		}
	},
	indigo: {
		name: 'indigo',
		label: 'Indigo',
		description: 'Creative, design, consulting',
		scale: {
			50: '#eef2ff',
			100: '#e0e7ff',
			200: '#c7d2fe',
			300: '#a5b4fc',
			400: '#818cf8',
			500: '#6366f1',
			600: '#4f46e5',
			700: '#4338ca',
			800: '#3730a3',
			900: '#312e81',
			950: '#1e1b4b'
		}
	},
	emerald: {
		name: 'emerald',
		label: 'Emerald',
		description: 'Finance, sustainability, health',
		scale: {
			50: '#ecfdf5',
			100: '#d1fae5',
			200: '#a7f3d0',
			300: '#6ee7b7',
			400: '#34d399',
			500: '#10b981',
			600: '#059669',
			700: '#047857',
			800: '#065f46',
			900: '#064e3b',
			950: '#022c22'
		}
	},
	rose: {
		name: 'rose',
		label: 'Rose',
		description: 'Marketing, creative, personal branding',
		scale: {
			50: '#fff1f2',
			100: '#ffe4e6',
			200: '#fecdd3',
			300: '#fda4af',
			400: '#fb7185',
			500: '#f43f5e',
			600: '#e11d48',
			700: '#be123c',
			800: '#9f1239',
			900: '#881337',
			950: '#4c0519'
		}
	},
	amber: {
		name: 'amber',
		label: 'Amber',
		description: 'Education, construction, energy',
		scale: {
			50: '#fffbeb',
			100: '#fef3c7',
			200: '#fde68a',
			300: '#fcd34d',
			400: '#fbbf24',
			500: '#f59e0b',
			600: '#d97706',
			700: '#b45309',
			800: '#92400e',
			900: '#78350f',
			950: '#451a03'
		}
	},
	slate: {
		name: 'slate',
		label: 'Slate',
		description: 'Minimal, monochrome, conservative',
		scale: {
			50: '#f8fafc',
			100: '#f1f5f9',
			200: '#e2e8f0',
			300: '#cbd5e1',
			400: '#94a3b8',
			500: '#64748b',
			600: '#475569',
			700: '#334155',
			800: '#1e293b',
			900: '#0f172a',
			950: '#020617'
		}
	}
};

/**
 * Default accent color
 */
export const DEFAULT_ACCENT_COLOR: AccentColor = 'sky';

/**
 * Default accent for the Soft Premium design when the operator hasn't chosen
 * one — a warm terracotta that harmonizes with the stone surfaces. Operators
 * who set an accent keep it; classic is unaffected.
 */
export const SOFT_PREMIUM_DEFAULT_ACCENT = '#c2410c';

/**
 * List of available accent colors for UI iteration
 */
export const ACCENT_COLOR_LIST: AccentColor[] = ['sky', 'indigo', 'emerald', 'rose', 'amber', 'slate'];

/**
 * Get accent color info by name
 */
export function getAccentColor(name?: string): AccentColorInfo {
	if (name && name in ACCENT_COLORS) {
		return ACCENT_COLORS[name as AccentColor];
	}
	return ACCENT_COLORS[DEFAULT_ACCENT_COLOR];
}

/**
 * Generate CSS custom properties for a given accent color
 * These can be injected into the :root to override the default primary color
 */
export function generateAccentCssVariables(color: AccentColor): string {
	const info = ACCENT_COLORS[color];
	if (!info) return '';

	return `
		--color-primary-50: ${info.scale[50]};
		--color-primary-100: ${info.scale[100]};
		--color-primary-200: ${info.scale[200]};
		--color-primary-300: ${info.scale[300]};
		--color-primary-400: ${info.scale[400]};
		--color-primary-500: ${info.scale[500]};
		--color-primary-600: ${info.scale[600]};
		--color-primary-700: ${info.scale[700]};
		--color-primary-800: ${info.scale[800]};
		--color-primary-900: ${info.scale[900]};
		--color-primary-950: ${info.scale[950]};
		--color-primary-50-rgb: ${hexToRgbChannels(info.scale[50])};
		--color-primary-100-rgb: ${hexToRgbChannels(info.scale[100])};
		--color-primary-200-rgb: ${hexToRgbChannels(info.scale[200])};
		--color-primary-300-rgb: ${hexToRgbChannels(info.scale[300])};
		--color-primary-400-rgb: ${hexToRgbChannels(info.scale[400])};
		--color-primary-500-rgb: ${hexToRgbChannels(info.scale[500])};
		--color-primary-600-rgb: ${hexToRgbChannels(info.scale[600])};
		--color-primary-700-rgb: ${hexToRgbChannels(info.scale[700])};
		--color-primary-800-rgb: ${hexToRgbChannels(info.scale[800])};
		--color-primary-900-rgb: ${hexToRgbChannels(info.scale[900])};
		--color-primary-950-rgb: ${hexToRgbChannels(info.scale[950])};
	`.trim();
}

/**
 * Convert "#rrggbb" → "R G B" channel string (space-separated decimal RGB).
 * Required for CSS variables consumed by Tailwind opacity modifiers, since
 * `rgb(var(--x) / <alpha>)` only resolves when the var holds raw channels.
 */
export function hexToRgbChannels(hex: string): string {
	if (!isValidHexColor(hex)) return '0 0 0';
	const r = parseInt(hex.slice(1, 3), 16);
	const g = parseInt(hex.slice(3, 5), 16);
	const b = parseInt(hex.slice(5, 7), 16);
	return `${r} ${g} ${b}`;
}

/**
 * Set both `--color-primary-N` (hex, for direct CSS consumers) and the
 * companion `--color-primary-N-rgb` (channels, for Tailwind opacity utilities)
 * on document.documentElement. Caller must run in a browser context.
 */
export function applyPaletteToRoot(scale: ColorScale): void {
	const root = document.documentElement;
	for (const step of [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950] as const) {
		const hex = scale[step];
		root.style.setProperty(`--color-primary-${step}`, hex);
		root.style.setProperty(`--color-primary-${step}-rgb`, hexToRgbChannels(hex));
	}
}

/**
 * Build the contents of a `:root { ... }` block setting both hex and RGB
 * channel forms for every palette step. Used by callers that inject a
 * `<style>` element instead of writing individual properties.
 */
export function buildPaletteRootBlock(scale: ColorScale): string {
	const lines: string[] = [];
	for (const step of [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950] as const) {
		lines.push(`  --color-primary-${step}: ${scale[step]};`);
	}
	for (const step of [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950] as const) {
		lines.push(`  --color-primary-${step}-rgb: ${hexToRgbChannels(scale[step])};`);
	}
	return `:root {\n${lines.join('\n')}\n}`;
}

/**
 * Check if a string is a valid accent color
 */
export function isValidAccentColor(value: unknown): value is AccentColor {
	return typeof value === 'string' && value in ACCENT_COLORS;
}

/**
 * Validate a hex color string (#RRGGBB format)
 */
export function isValidHexColor(value: string): boolean {
	return /^#[0-9a-fA-F]{6}$/.test(value);
}

/**
 * Relative luminance of a hex color per WCAG 2.x.
 * Used to decide whether dark or light text is more readable on top.
 * Returns 0 (black) to 1 (white).
 */
export function hexLuminance(hex: string): number {
	if (!isValidHexColor(hex)) return 0;
	const r = parseInt(hex.slice(1, 3), 16) / 255;
	const g = parseInt(hex.slice(3, 5), 16) / 255;
	const b = parseInt(hex.slice(5, 7), 16) / 255;
	const [rs, gs, bs] = [r, g, b].map((c) =>
		c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
	);
	return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

/**
 * Relative luminance from 0-255 channels (WCAG 2.x), without a hex round-trip.
 */
function channelLuminance(r: number, g: number, b: number): number {
	const [rs, gs, bs] = [r, g, b].map((c) => {
		const cc = c / 255;
		return cc <= 0.03928 ? cc / 12.92 : Math.pow((cc + 0.055) / 1.055, 2.4);
	});
	return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

/** Contrast ratio of white (#fff) text on a given color luminance. */
function whiteOnColorContrast(luminance: number): number {
	return 1.05 / (luminance + 0.05);
}

/**
 * AAA accent contrast target for the 600 shade. The 600 shade carries white
 * text on primary buttons/links/CTAs (`.btn-primary`, accent links), so white
 * on it must clear 7:1. Pale accents (yellow/cyan) otherwise fail badly.
 */
export const ACCENT_MIN_CONTRAST = 7;

/**
 * WCAG luminance crossover (0.179) — backgrounds above this need dark text;
 * backgrounds at or below need light text. Derived from the 4.5:1 contrast
 * inflection between white (L=1.0) and black (L=0.0) text.
 */
export const DARK_TEXT_THRESHOLD = 0.179;

/**
 * Generate a full color scale from a single hex color.
 * Produces light-to-dark shades by mixing with white (for lighter steps)
 * and black (for darker steps), preserving the hue of the input color.
 * The input color is used as the 500 step.
 */
export function generatePaletteFromHex(hex: string): ColorScale {
	if (!isValidHexColor(hex)) {
		return ACCENT_COLORS[DEFAULT_ACCENT_COLOR].scale;
	}
	const r = parseInt(hex.slice(1, 3), 16);
	const g = parseInt(hex.slice(3, 5), 16);
	const b = parseInt(hex.slice(5, 7), 16);

	function mix(base: number, target: number, amount: number): number {
		return Math.round(base + (target - base) * amount);
	}

	function toHex(red: number, green: number, blue: number): string {
		return '#' + [red, green, blue].map(c => c.toString(16).padStart(2, '0')).join('');
	}

	// AAA clamp for the 600 shade: darken (mix toward black, which preserves hue
	// because all channels scale toward 0 equally) until white text on it clears
	// 7:1. Starts at the nominal 0.15 darkening and only ever goes darker, so the
	// accent hue is preserved and the change is strictly more accessible. Black
	// always passes, so the search terminates.
	function clamp600(): string {
		const baseAmount = 0.15;
		const lum = (amount: number) =>
			channelLuminance(mix(r, 0, amount), mix(g, 0, amount), mix(b, 0, amount));
		if (whiteOnColorContrast(lum(baseAmount)) >= ACCENT_MIN_CONTRAST) {
			return toHex(mix(r, 0, baseAmount), mix(g, 0, baseAmount), mix(b, 0, baseAmount));
		}
		let lo = baseAmount;
		let hi = 1;
		for (let i = 0; i < 24; i++) {
			const m = (lo + hi) / 2;
			if (whiteOnColorContrast(lum(m)) >= ACCENT_MIN_CONTRAST) hi = m;
			else lo = m;
		}
		return toHex(mix(r, 0, hi), mix(g, 0, hi), mix(b, 0, hi));
	}

	return {
		50:  toHex(mix(r, 255, 0.95), mix(g, 255, 0.95), mix(b, 255, 0.95)),
		100: toHex(mix(r, 255, 0.88), mix(g, 255, 0.88), mix(b, 255, 0.88)),
		200: toHex(mix(r, 255, 0.73), mix(g, 255, 0.73), mix(b, 255, 0.73)),
		300: toHex(mix(r, 255, 0.55), mix(g, 255, 0.55), mix(b, 255, 0.55)),
		400: toHex(mix(r, 255, 0.30), mix(g, 255, 0.30), mix(b, 255, 0.30)),
		500: hex,
		600: clamp600(),
		700: toHex(mix(r, 0, 0.30), mix(g, 0, 0.30), mix(b, 0, 0.30)),
		800: toHex(mix(r, 0, 0.45), mix(g, 0, 0.45), mix(b, 0, 0.45)),
		900: toHex(mix(r, 0, 0.60), mix(g, 0, 0.60), mix(b, 0, 0.60)),
		950: toHex(mix(r, 0, 0.80), mix(g, 0, 0.80), mix(b, 0, 0.80)),
	};
}

/**
 * Generate CSS custom properties for accent color (works server-side, no DOM needed).
 * Returns a CSS string that can be injected into <style> tags for SSR.
 */
export function generateAccentCSSVars(colorName?: string | null, customHex?: string | null): string {
	let scale: ColorScale | null = null;

	if (customHex) {
		scale = generatePaletteFromHex(customHex);
	} else if (colorName && colorName in ACCENT_COLORS) {
		scale = ACCENT_COLORS[colorName as AccentColor].scale;
	}

	if (!scale) return '';

	return `:root {
		--color-primary-50: ${scale[50]};
		--color-primary-100: ${scale[100]};
		--color-primary-200: ${scale[200]};
		--color-primary-300: ${scale[300]};
		--color-primary-400: ${scale[400]};
		--color-primary-500: ${scale[500]};
		--color-primary-600: ${scale[600]};
		--color-primary-700: ${scale[700]};
		--color-primary-800: ${scale[800]};
		--color-primary-900: ${scale[900]};
		--color-primary-950: ${scale[950]};
		--color-primary-50-rgb: ${hexToRgbChannels(scale[50])};
		--color-primary-100-rgb: ${hexToRgbChannels(scale[100])};
		--color-primary-200-rgb: ${hexToRgbChannels(scale[200])};
		--color-primary-300-rgb: ${hexToRgbChannels(scale[300])};
		--color-primary-400-rgb: ${hexToRgbChannels(scale[400])};
		--color-primary-500-rgb: ${hexToRgbChannels(scale[500])};
		--color-primary-600-rgb: ${hexToRgbChannels(scale[600])};
		--color-primary-700-rgb: ${hexToRgbChannels(scale[700])};
		--color-primary-800-rgb: ${hexToRgbChannels(scale[800])};
		--color-primary-900-rgb: ${hexToRgbChannels(scale[900])};
		--color-primary-950-rgb: ${hexToRgbChannels(scale[950])};
	}`;
}

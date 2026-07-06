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
	// always passes, so the search terminates. Returns the resolved mix-toward-black
	// AMOUNT (not the hex) so the darker shades can be floored above it (see below).
	function clamp600Amount(): number {
		const baseAmount = 0.15;
		const lum = (amount: number) =>
			channelLuminance(mix(r, 0, amount), mix(g, 0, amount), mix(b, 0, amount));
		if (whiteOnColorContrast(lum(baseAmount)) >= ACCENT_MIN_CONTRAST) {
			return baseAmount;
		}
		let lo = baseAmount;
		let hi = 1;
		for (let i = 0; i < 24; i++) {
			const m = (lo + hi) / 2;
			if (whiteOnColorContrast(lum(m)) >= ACCENT_MIN_CONTRAST) hi = m;
			else lo = m;
		}
		return hi;
	}

	// For a PALE custom accent (e.g. #ffe066), the AAA clamp above can drive 600's
	// darkening amount well past the nominal 0.30 used for 700. Left unguarded, that
	// makes 600 DARKER than 700, a non-monotonic (inverted) ramp. Floor each darker
	// shade strictly above the clamped 600 amount so the scale stays monotonic from
	// 600 through 950 regardless of how deep the clamp had to go. The nominal mixes
	// (0.30/0.45/0.60/0.80) are preserved for normal accents where 600's amount sits
	// at or below 0.30.
	const a600 = clamp600Amount();
	const a700 = Math.max(0.30, a600 + 0.07);
	const a800 = Math.max(0.45, a700 + 0.10);
	const a900 = Math.max(0.60, a800 + 0.12);
	const a950 = Math.max(0.80, a900 + 0.10);
	const mixBlack = (amount: number) => toHex(mix(r, 0, amount), mix(g, 0, amount), mix(b, 0, amount));

	return {
		50:  toHex(mix(r, 255, 0.95), mix(g, 255, 0.95), mix(b, 255, 0.95)),
		100: toHex(mix(r, 255, 0.88), mix(g, 255, 0.88), mix(b, 255, 0.88)),
		200: toHex(mix(r, 255, 0.73), mix(g, 255, 0.73), mix(b, 255, 0.73)),
		300: toHex(mix(r, 255, 0.55), mix(g, 255, 0.55), mix(b, 255, 0.55)),
		400: toHex(mix(r, 255, 0.30), mix(g, 255, 0.30), mix(b, 255, 0.30)),
		500: hex,
		600: mixBlack(a600),
		700: mixBlack(a700),
		800: mixBlack(a800),
		900: mixBlack(a900),
		950: mixBlack(a950),
	};
}

/**
 * AAA text-ink targets and reference surfaces.
 *
 * Body/heading text on the public profile can land on any of several surfaces in
 * each mode. We clamp the operator's hue against the WORST-CASE surface — the one
 * that yields the LOWEST contrast for that ink — so clearing 7:1 there guarantees
 * AAA on every other surface text can land on, by construction.
 *
 * The worst case differs by mode because of which side of the surface the ink is:
 *
 *  - Light mode (ink is DARKER than the surface): contrast = (L_surf+0.05)/(L_ink+0.05),
 *    which INCREASES with surface luminance. So the DARKEST light surface is the
 *    worst case. Text lands on `--surface` (#ffffff), but also directly on the warm
 *    `--bg` (#fbf8f4) and soft-premium `--bg` (#fef9f6), which are DARKER than white
 *    and therefore LOWER-contrast for dark ink. The darkest of these is #fbf8f4
 *    (L≈0.9418), so we clamp against it; clearing 7:1 on #fbf8f4 also clears the
 *    lighter #fef9f6 and #ffffff automatically.
 *  - Dark mode (ink is LIGHTER than the surface): contrast = (L_ink+0.05)/(L_surf+0.05),
 *    which DECREASES with surface luminance. So the LIGHTEST dark surface is the
 *    worst case. Text lands on Soft Premium `--surface` (#221e1a, L≈0.0134) and
 *    `--bg` (#1a1613, L≈0.0084), but Classic cards use Tailwind gray-800
 *    (#1f2937, L≈0.0210), which is lighter still. Clearing 7:1 on #1f2937 clears
 *    it on the darker Soft Premium surfaces automatically.
 *
 * NOTE: if a future design token introduces a light-mode text surface DARKER than
 * #fbf8f4, or a dark-mode surface LIGHTER than #1f2937, these constants must be
 * updated — the guarantee is only as strong as "clamp against the worst-case
 * surface." See tests/text-color-aaa.spec.ts for the regression proof.
 */
export const TEXT_INK_MIN_CONTRAST = 7;
// Worst-case (lowest-contrast) surface per mode — see block comment above.
// Light: darkest surface tinted text lands on (warm --bg), NOT white.
const TEXT_INK_LIGHT_SURFACE = '#fbf8f4';
// Dark: lightest dark surface tinted text lands on (classic gray-800 card).
const TEXT_INK_DARK_SURFACE = '#1f2937';

/** Contrast ratio between two relative luminances (WCAG 2.x), order-independent. */
function contrastRatio(l1: number, l2: number): number {
	const hi = Math.max(l1, l2);
	const lo = Math.min(l1, l2);
	return (hi + 0.05) / (lo + 0.05);
}

/**
 * Derive a per-mode, hue-preserved, AAA-clamped text ink from an operator's hue.
 *
 * Returns `{ light, dark }` hex strings, both guaranteed to clear 7:1 against the
 * WORST-CASE (lowest-contrast) surface text sits on in their respective modes (the
 * warm `--bg` #fbf8f4 in light mode, the classic dark card #1f2937 in dark mode).
 *
 * Mechanism (mirrors `clamp600`'s binary search): we only ever move lightness in
 * the AAA-safe direction while keeping hue:
 *  - Light ink: mix the hue toward BLACK (all channels scale toward 0 equally →
 *    hue preserved) until contrast vs the warm --bg ≥ 7:1.
 *  - Dark ink: mix the hue toward WHITE (all channels scale toward 255 equally →
 *    hue preserved) until contrast vs the lightest dark surface ≥ 7:1.
 *
 * Black (light) and white (dark) always satisfy 7:1, so each search terminates.
 * Invalid input falls back to a neutral readable ink so the default never breaks.
 */
export function deriveTextInk(hex: string): { light: string; dark: string } {
	if (!isValidHexColor(hex)) {
		// Neutral, AAA-safe defaults (near-black on light, near-white on dark).
		return { light: '#1a1613', dark: '#f4eee4' };
	}
	const r = parseInt(hex.slice(1, 3), 16);
	const g = parseInt(hex.slice(3, 5), 16);
	const b = parseInt(hex.slice(5, 7), 16);

	const mix = (base: number, target: number, amount: number): number =>
		Math.round(base + (target - base) * amount);
	const toHex = (red: number, green: number, blue: number): string =>
		'#' + [red, green, blue].map((c) => c.toString(16).padStart(2, '0')).join('');

	const surfaceLum = (surface: string): number => {
		const sr = parseInt(surface.slice(1, 3), 16);
		const sg = parseInt(surface.slice(3, 5), 16);
		const sb = parseInt(surface.slice(5, 7), 16);
		return channelLuminance(sr, sg, sb);
	};

	// Binary-search the smallest mix `amount` (toward `target`) whose resulting ink
	// clears 7:1 against `surface`. amount=0 is the raw hue; amount=1 is the target
	// (black or white), which always passes — so the bracket [0,1] always contains
	// a solution and the search converges.
	const clamp = (target: number, surface: string): string => {
		const surfL = surfaceLum(surface);
		const inkLum = (amount: number): number =>
			channelLuminance(mix(r, target, amount), mix(g, target, amount), mix(b, target, amount));
		// Already AAA at the raw hue? Honor it exactly.
		if (contrastRatio(inkLum(0), surfL) >= TEXT_INK_MIN_CONTRAST) {
			return toHex(r, g, b);
		}
		let lo = 0;
		let hi = 1;
		for (let i = 0; i < 24; i++) {
			const mAmt = (lo + hi) / 2;
			if (contrastRatio(inkLum(mAmt), surfL) >= TEXT_INK_MIN_CONTRAST) hi = mAmt;
			else lo = mAmt;
		}
		return toHex(mix(r, target, hi), mix(g, target, hi), mix(b, target, hi));
	};

	return {
		light: clamp(0, TEXT_INK_LIGHT_SURFACE), // darken toward black
		dark: clamp(255, TEXT_INK_DARK_SURFACE) // lighten toward white
	};
}

/**
 * Build the SSR `<style>`/inline CSS-var declarations for the operator's text
 * color. Returns `--text-ink` (light value) and `--text-ink-dark` (dark value)
 * as a `:root { ... }` block, or '' when no color is set (so the default render
 * is byte-identical to today — nothing is injected, the scoped rule no-ops).
 */
export function generateTextInkCSSVars(textColor?: string | null): string {
	if (!textColor || !isValidHexColor(textColor)) return '';
	const { light, dark } = deriveTextInk(textColor);
	return `:root {
		--text-ink: ${light};
		--text-ink-dark: ${dark};
	}`;
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

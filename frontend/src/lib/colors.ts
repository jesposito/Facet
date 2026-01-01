/**
 * Accent Color Palette Constants
 *
 * Phase 6.5: Curated palette approach for accent color customization.
 * All colors are pre-tested for WCAG contrast compliance.
 * Uses Tailwind CSS color scale values (50-950) for each accent color.
 */

export type AccentColor = 'sky' | 'indigo' | 'emerald' | 'rose' | 'amber' | 'slate';

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
	`.trim();
}

/**
 * Check if a string is a valid accent color
 */
export function isValidAccentColor(value: unknown): value is AccentColor {
	return typeof value === 'string' && value in ACCENT_COLORS;
}

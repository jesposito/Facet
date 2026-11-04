/**
 * Font Pack Constants
 *
 * Curated font pairings for typography customization.
 * Each pack includes heading, body, and code fonts with
 * locally bundled faces for every pack.
 */

export type FontPack =
	| 'editorial'
	| 'modern'
	| 'classic'
	| 'technical'
	| 'editorial-bold'
	| 'humanist'
	| 'geometric'
	| 'soft-premium';

export interface FontPackInfo {
	name: FontPack;
	label: string;
	description: string;
	heading: string;
	headingFallback: string;
	accent?: string;
	accentFallback?: string;
	body: string;
	bodyFallback: string;
	code: string;
	codeFallback: string;
}

export const FONT_PACKS: Record<FontPack, FontPackInfo> = {
	editorial: {
		name: 'editorial',
		label: 'Clean & Contemporary',
		description: 'Sophisticated editorial feel with serif headings',
		heading: 'Lora',
		headingFallback: "Georgia, 'Times New Roman', serif",
		body: 'Plus Jakarta Sans',
		bodyFallback: 'Inter, system-ui, sans-serif',
		code: 'JetBrains Mono',
		codeFallback: 'monospace'
	},
	modern: {
		name: 'modern',
		label: 'Modern & Minimal',
		description: 'Clean, tech-forward with a single sans-serif',
		heading: 'Inter',
		headingFallback: 'system-ui, sans-serif',
		body: 'Inter',
		bodyFallback: 'system-ui, sans-serif',
		code: 'JetBrains Mono',
		codeFallback: 'monospace'
	},
	classic: {
		name: 'classic',
		label: 'Classic & Refined',
		description: 'Traditional, academic, literary',
		heading: 'Playfair Display',
		headingFallback: "Georgia, 'Times New Roman', serif",
		body: 'Source Sans 3',
		bodyFallback: 'Inter, system-ui, sans-serif',
		code: 'Source Code Pro',
		codeFallback: 'monospace'
	},
	technical: {
		name: 'technical',
		label: 'Precise & Functional',
		description: 'Engineering, technical documentation',
		heading: 'IBM Plex Sans',
		headingFallback: 'system-ui, sans-serif',
		body: 'IBM Plex Sans',
		bodyFallback: 'system-ui, sans-serif',
		code: 'IBM Plex Mono',
		codeFallback: 'monospace'
	},
	'editorial-bold': {
		name: 'editorial-bold',
		label: 'Bold & Expressive',
		description: 'Energetic, startup feel',
		heading: 'Sora',
		headingFallback: 'system-ui, sans-serif',
		body: 'Nunito Sans',
		bodyFallback: 'Inter, system-ui, sans-serif',
		code: 'Fira Code',
		codeFallback: 'monospace'
	},
	humanist: {
		name: 'humanist',
		label: 'Warm & Approachable',
		description: 'Friendly, readable, personal',
		heading: 'Merriweather',
		headingFallback: "Georgia, 'Times New Roman', serif",
		body: 'Lato',
		bodyFallback: 'Inter, system-ui, sans-serif',
		code: 'Source Code Pro',
		codeFallback: 'monospace'
	},
	geometric: {
		name: 'geometric',
		label: 'Structured & Confident',
		description: 'Modern, geometric, bold',
		heading: 'Space Grotesk',
		headingFallback: 'system-ui, sans-serif',
		body: 'DM Sans',
		bodyFallback: 'system-ui, sans-serif',
		code: 'DM Mono',
		codeFallback: 'monospace'
	},
	'soft-premium': {
		name: 'soft-premium',
		label: 'Soft Premium',
		description: 'Warm geometric sans with a serif accent — the redesign pairing',
		heading: 'Hanken Grotesk',
		headingFallback: "'Plus Jakarta Sans', system-ui, sans-serif",
		accent: 'Newsreader',
		accentFallback: "Georgia, 'Times New Roman', serif",
		body: 'Hanken Grotesk',
		bodyFallback: "'Plus Jakarta Sans', system-ui, sans-serif",
		code: 'JetBrains Mono',
		codeFallback: 'monospace'
	}
};

// Soft Premium's default pairing. Classic keeps the static app.css typography
// until the operator explicitly chooses a pack.
export const DEFAULT_FONT_PACK: FontPack = 'soft-premium';

export const FONT_PACK_LIST: FontPack[] = [
	'editorial',
	'modern',
	'classic',
	'technical',
	'editorial-bold',
	'humanist',
	'geometric',
	'soft-premium'
];

export function getFontPack(name?: string | null): FontPackInfo {
	if (name && name in FONT_PACKS) {
		return FONT_PACKS[name as FontPack];
	}
	return FONT_PACKS[DEFAULT_FONT_PACK];
}

export function isValidFontPack(value: unknown): value is FontPack {
	return typeof value === 'string' && value in FONT_PACKS;
}

/**
 * Generate CSS custom properties for font families.
 * Returns a :root block with --font-heading, --font-body, --font-code.
 */
export function generateFontCSSVars(packName?: string | null): string {
	const pack = getFontPack(packName);
	const accent = pack.accent || pack.heading;
	const accentFallback = pack.accentFallback || pack.headingFallback;

	return `:root {
    --font-heading: '${pack.heading}', ${pack.headingFallback};
    --font-accent: '${accent}', ${accentFallback};
    --font-body: '${pack.body}', ${pack.bodyFallback};
    --font-code: '${pack.code}', ${pack.codeFallback};
  }`;
}

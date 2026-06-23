/**
 * Font Pack Constants
 *
 * Curated font pairings for typography customization.
 * Each pack includes heading, body, and code fonts with
 * pre-built Google Fonts URLs for performance.
 */

export type FontPack = 'editorial' | 'modern' | 'classic' | 'technical' | 'editorial-bold' | 'humanist' | 'geometric' | 'soft-premium';

export interface FontPackInfo {
  name: FontPack;
  label: string;
  description: string;
  heading: string;
  headingFallback: string;
  body: string;
  bodyFallback: string;
  code: string;
  codeFallback: string;
  googleFontsUrl: string;
}

export const FONT_PACKS: Record<FontPack, FontPackInfo> = {
  editorial: {
    name: 'editorial',
    label: 'Clean & Contemporary',
    description: 'Sophisticated editorial feel with serif headings',
    heading: 'Lora',
    headingFallback: "Georgia, 'Times New Roman', serif",
    body: 'Plus Jakarta Sans',
    bodyFallback: "Inter, system-ui, sans-serif",
    code: 'JetBrains Mono',
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Lora:ital,wght@0,400;0,500;0,600;0,700;1,400;1,500;1,600;1,700&family=Plus+Jakarta+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap'
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
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap'
  },
  classic: {
    name: 'classic',
    label: 'Classic & Refined',
    description: 'Traditional, academic, literary',
    heading: 'Playfair Display',
    headingFallback: "Georgia, 'Times New Roman', serif",
    body: 'Source Sans 3',
    bodyFallback: "Inter, system-ui, sans-serif",
    code: 'Source Code Pro',
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Playfair+Display:wght@400;500;600;700&family=Source+Sans+3:wght@400;500;600;700&family=Source+Code+Pro:wght@400;500&display=swap'
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
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap'
  },
  'editorial-bold': {
    name: 'editorial-bold',
    label: 'Bold & Expressive',
    description: 'Energetic, startup feel',
    heading: 'Sora',
    headingFallback: 'system-ui, sans-serif',
    body: 'Nunito Sans',
    bodyFallback: "Inter, system-ui, sans-serif",
    code: 'Fira Code',
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Sora:wght@400;500;600;700&family=Nunito+Sans:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap'
  },
  humanist: {
    name: 'humanist',
    label: 'Warm & Approachable',
    description: 'Friendly, readable, personal',
    heading: 'Merriweather',
    headingFallback: "Georgia, 'Times New Roman', serif",
    body: 'Lato',
    bodyFallback: "Inter, system-ui, sans-serif",
    code: 'Source Code Pro',
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Merriweather:wght@400;700&family=Lato:wght@400;700&family=Source+Code+Pro:wght@400;500&display=swap'
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
    codeFallback: 'monospace',
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;600;700&family=DM+Sans:wght@400;500;600;700&family=DM+Mono:wght@400;500&display=swap'
  },
  'soft-premium': {
    name: 'soft-premium',
    label: 'Soft Premium',
    description: 'Warm geometric sans with a serif accent — the redesign pairing',
    heading: 'Hanken Grotesk',
    headingFallback: "'Plus Jakarta Sans', system-ui, sans-serif",
    body: 'Hanken Grotesk',
    bodyFallback: "'Plus Jakarta Sans', system-ui, sans-serif",
    code: 'JetBrains Mono',
    codeFallback: 'monospace',
    // Includes Newsreader for the --font-accent editorial flourishes.
    googleFontsUrl: 'https://fonts.googleapis.com/css2?family=Hanken+Grotesk:wght@400;500;600;700&family=Newsreader:ital,wght@0,400;0,500;1,400;1,500&family=JetBrains+Mono:wght@400;500&display=swap'
  }
};

export const DEFAULT_FONT_PACK: FontPack = 'editorial';

export const FONT_PACK_LIST: FontPack[] = ['editorial', 'modern', 'classic', 'technical', 'editorial-bold', 'humanist', 'geometric', 'soft-premium'];

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

  return `:root {
    --font-heading: '${pack.heading}', ${pack.headingFallback};
    --font-body: '${pack.body}', ${pack.bodyFallback};
    --font-code: '${pack.code}', ${pack.codeFallback};
  }`;
}

/**
 * Get the Google Fonts URL for a given font pack.
 */
export function getGoogleFontsUrl(packName?: string | null): string {
  return getFontPack(packName).googleFontsUrl;
}

type RGB = { r: number; g: number; b: number };

export type CustomCSSContrastAnalysis = {
	shouldWarn: boolean;
	lowContrastPairs: number;
	primaryColorOverrides: boolean;
};

const MIN_TEXT_CONTRAST = 4.5;
const PRIMARY_COLOR_VAR_RE = /--color-primary-(?:50|100|200|300|400|500|600|700|800|900|950)(?:-rgb)?\s*:/i;
const NAMED_COLORS: Record<string, RGB> = {
	black: { r: 0, g: 0, b: 0 },
	blue: { r: 0, g: 0, b: 255 },
	gray: { r: 128, g: 128, b: 128 },
	green: { r: 0, g: 128, b: 0 },
	grey: { r: 128, g: 128, b: 128 },
	orange: { r: 255, g: 165, b: 0 },
	purple: { r: 128, g: 0, b: 128 },
	red: { r: 255, g: 0, b: 0 },
	white: { r: 255, g: 255, b: 255 },
	yellow: { r: 255, g: 255, b: 0 }
};

export function analyzeCustomCSSContrast(css: string): CustomCSSContrastAnalysis {
	const cleaned = css.replace(/\/\*[\s\S]*?\*\//g, '');
	const primaryColorOverrides = PRIMARY_COLOR_VAR_RE.test(cleaned);
	let lowContrastPairs = 0;

	for (const block of cleaned.matchAll(/\{([^{}]*)\}/g)) {
		const declarations = parseDeclarations(block[1]);
		const color = declarations.get('color');
		const background = declarations.get('background-color') ?? declarations.get('background');
		if (!color || !background) continue;

		const foregroundRgb = parseCSSColor(color);
		const backgroundRgb = parseCSSColor(background);
		if (!foregroundRgb || !backgroundRgb) continue;

		if (contrastRatio(foregroundRgb, backgroundRgb) < MIN_TEXT_CONTRAST) {
			lowContrastPairs += 1;
		}
	}

	return {
		shouldWarn: primaryColorOverrides || lowContrastPairs > 0,
		lowContrastPairs,
		primaryColorOverrides
	};
}

function parseDeclarations(block: string): Map<string, string> {
	const declarations = new Map<string, string>();
	for (const declaration of block.split(';')) {
		const separatorIndex = declaration.indexOf(':');
		if (separatorIndex === -1) continue;
		const property = declaration.slice(0, separatorIndex).trim().toLowerCase();
		const value = declaration.slice(separatorIndex + 1).trim();
		if (property && value) {
			declarations.set(property, value);
		}
	}
	return declarations;
}

function parseCSSColor(value: string): RGB | null {
	const lower = value.toLowerCase();
	if (/\btransparent\b/.test(lower)) return null;

	const hexMatch = lower.match(/#([0-9a-f]{3,8})\b/);
	if (hexMatch) return parseHexColor(hexMatch[1]);

	const rgbMatch = lower.match(/rgba?\(\s*([0-9.]+%?)\s*[, ]\s*([0-9.]+%?)\s*[, ]\s*([0-9.]+%?)/);
	if (rgbMatch) {
		return {
			r: parseRGBChannel(rgbMatch[1]),
			g: parseRGBChannel(rgbMatch[2]),
			b: parseRGBChannel(rgbMatch[3])
		};
	}

	for (const [name, rgb] of Object.entries(NAMED_COLORS)) {
		if (new RegExp(`\\b${name}\\b`).test(lower)) {
			return rgb;
		}
	}

	return null;
}

function parseHexColor(hex: string): RGB | null {
	if (hex.length === 3 || hex.length === 4) {
		return {
			r: parseInt(hex[0] + hex[0], 16),
			g: parseInt(hex[1] + hex[1], 16),
			b: parseInt(hex[2] + hex[2], 16)
		};
	}
	if (hex.length === 6 || hex.length === 8) {
		return {
			r: parseInt(hex.slice(0, 2), 16),
			g: parseInt(hex.slice(2, 4), 16),
			b: parseInt(hex.slice(4, 6), 16)
		};
	}
	return null;
}

function parseRGBChannel(channel: string): number {
	if (channel.endsWith('%')) {
		return clampByte((parseFloat(channel) / 100) * 255);
	}
	return clampByte(parseFloat(channel));
}

function clampByte(value: number): number {
	if (!Number.isFinite(value)) return 0;
	return Math.max(0, Math.min(255, Math.round(value)));
}

function relativeLuminance({ r, g, b }: RGB): number {
	const channel = (value: number) => {
		const normalized = value / 255;
		return normalized <= 0.03928
			? normalized / 12.92
			: Math.pow((normalized + 0.055) / 1.055, 2.4);
	};
	return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

function contrastRatio(a: RGB, b: RGB): number {
	const l1 = relativeLuminance(a);
	const l2 = relativeLuminance(b);
	const hi = Math.max(l1, l2);
	const lo = Math.min(l1, l2);
	return (hi + 0.05) / (lo + 0.05);
}

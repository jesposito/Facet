import { ACCENT_COLORS, DEFAULT_ACCENT_COLOR, SOFT_PREMIUM_DEFAULT_ACCENT, type AccentColor } from '$lib/colors';

export type ManifestProfile = {
	name?: unknown;
	avatar_url?: unknown;
	custom_hex_color?: unknown;
	accent_color?: unknown;
	hero_bg_color?: unknown;
};

export type ManifestSiteSettings = {
	favicon?: unknown;
	design?: unknown;
};

type ManifestIcon = {
	src: string;
	sizes: string;
	type?: string;
	purpose?: string;
};

export type FacetWebManifest = {
	name: string;
	short_name: string;
	start_url: string;
	scope: string;
	display: 'standalone';
	background_color: string;
	theme_color: string;
	icons: ManifestIcon[];
};

export function buildWebManifest(profile: ManifestProfile | null, settings: ManifestSiteSettings | null): FacetWebManifest {
	const name = stringOrEmpty(profile?.name) || 'Facet';
	const icons = uniqueIcons([
		iconFromUrl(stringOrEmpty(profile?.avatar_url), 'any'),
		iconFromUrl(stringOrEmpty(settings?.favicon), 'any'),
		{ src: '/icon.png', sizes: '498x512', type: 'image/png', purpose: 'any' },
		{ src: '/favicon.png', sizes: '32x32', type: 'image/png', purpose: 'any' }
	]);

	return {
		name,
		short_name: name.length > 12 ? name.slice(0, 12) : name,
		start_url: '/',
		scope: '/',
		display: 'standalone',
		background_color: isHexColor(profile?.hero_bg_color) ? String(profile?.hero_bg_color) : '#ffffff',
		theme_color: resolveThemeColor(profile, settings),
		icons
	};
}

function resolveThemeColor(profile: ManifestProfile | null, settings: ManifestSiteSettings | null): string {
	if (isHexColor(profile?.custom_hex_color)) {
		return String(profile?.custom_hex_color);
	}

	const accent = stringOrEmpty(profile?.accent_color);
	if (accent && accent in ACCENT_COLORS) {
		return ACCENT_COLORS[accent as AccentColor].scale[600];
	}

	if (settings?.design === 'soft-premium') {
		return SOFT_PREMIUM_DEFAULT_ACCENT;
	}

	return ACCENT_COLORS[DEFAULT_ACCENT_COLOR].scale[600];
}

function iconFromUrl(src: string, sizes: string): ManifestIcon | null {
	if (!src) return null;
	return { src, sizes, purpose: 'any' };
}

function uniqueIcons(icons: Array<ManifestIcon | null>): ManifestIcon[] {
	const seen = new Set<string>();
	const result: ManifestIcon[] = [];
	for (const icon of icons) {
		if (!icon || seen.has(icon.src)) continue;
		seen.add(icon.src);
		result.push(icon);
	}
	return result;
}

function stringOrEmpty(value: unknown): string {
	return typeof value === 'string' ? value.trim() : '';
}

function isHexColor(value: unknown): value is string {
	return typeof value === 'string' && /^#[0-9a-fA-F]{3,8}$/.test(value);
}

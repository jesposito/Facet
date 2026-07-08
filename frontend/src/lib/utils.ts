import { marked } from 'marked';
import DOMPurify from 'isomorphic-dompurify';
import { get } from 'svelte/store';
import { locale } from 'svelte-i18n';

type EmbedMatch = {
	provider: string;
	url: string;
};

// Configure DOMPurify with iframe domain whitelist for security.
// URL-parsing-based check (not substring) so that hostile URLs like
// `https://www.youtube.com.attacker.tld/embed/...` cannot satisfy the allowlist
// by happening to start with a trusted prefix.
DOMPurify.addHook('uponSanitizeElement', (node, data) => {
	// Duck-type the Element check: `node instanceof Element` throws
	// `ReferenceError: Element is not defined` during SvelteKit SSR because
	// the jsdom-backed `Element` global isn't always materialized in the
	// hook's evaluation context. Checking for the methods we need is
	// equivalent for our purposes and works in both Node and the browser.
	const el = node as unknown as { getAttribute?: (n: string) => string | null; parentNode?: { removeChild: (n: unknown) => void } } | null;
	if (data.tagName === 'iframe' && el && typeof el.getAttribute === 'function') {
		const src = el.getAttribute('src') || '';

		// Allowlist of trusted embed origins + required path prefix per provider.
		const allowedOrigins: Array<{ hostname: string; pathPrefix: string }> = [
			{ hostname: 'www.youtube.com', pathPrefix: '/embed/' },
			{ hostname: 'www.youtube-nocookie.com', pathPrefix: '/embed/' },
			{ hostname: 'player.vimeo.com', pathPrefix: '/video/' },
			{ hostname: 'www.loom.com', pathPrefix: '/embed/' },
			{ hostname: 'w.soundcloud.com', pathPrefix: '/player/' },
			{ hostname: 'open.spotify.com', pathPrefix: '/embed/' },
			{ hostname: 'codepen.io', pathPrefix: '/embed/' },
			// Figma's embed endpoint is `/embed?embed_host=...` (no trailing slash).
			{ hostname: 'www.figma.com', pathPrefix: '/embed' },
			{ hostname: 'calendly.com', pathPrefix: '/' },
			{ hostname: 'cal.com', pathPrefix: '/' }
		];

		let isAllowed = false;
		try {
			const url = new URL(src);
			if (url.protocol === 'https:') {
				isAllowed = allowedOrigins.some(
					({ hostname, pathPrefix }) =>
						url.hostname === hostname && url.pathname.startsWith(pathPrefix)
				);
			}
		} catch {
			// Invalid URL → not allowed.
		}

		if (!isAllowed) {
			el.parentNode?.removeChild(node);
			console.warn('[Security] Blocked iframe with untrusted source:', src);
		}
	}
});

// WCAG 1.4.2 (Audio Control): autoplay must not start automatically. Strip the
// `autoplay` token from any iframe `allow=` value that survives sanitization.
// Done as a separate hook because DOMPurify cannot filter individual tokens
// inside an attribute value list.
DOMPurify.addHook('uponSanitizeAttribute', (_node, data) => {
	if (data.attrName === 'allow' && typeof data.attrValue === 'string') {
		data.attrValue = data.attrValue
			.split(';')
			.map((t) => t.trim())
			.filter((t) => t && !/^autoplay\b/i.test(t))
			.join('; ');
	}
});

const RICH_ADD_TAGS = [
	'iframe',
	'video',
	'audio',
	'figure',
	'figcaption',
	'picture',
	'source'
] as const;

const RICH_ADD_ATTR = [
	'allowfullscreen',
	'loading',
	'controls',
	'target',
	'rel',
	'allow',
	'title',
	'name',
	'poster',
	'sandbox'
] as const;

export function sanitizeRichHtml(html: string): string {
	if (!html) return '';
	return DOMPurify.sanitize(html, {
		ADD_TAGS: [...RICH_ADD_TAGS],
		ADD_ATTR: [...RICH_ADD_ATTR],
		ALLOW_DATA_ATTR: true,
		ALLOW_ARIA_ATTR: true
	});
}

function currentLocale(): string {
	return get(locale) || 'en';
}

export function formatDate(dateString: string | undefined, options?: Intl.DateTimeFormatOptions): string {
	if (!dateString) return '';
	if (/^\d{4}$/.test(dateString)) return dateString;
	const formatOptions = options || { month: 'short', year: 'numeric' };
	if (/^\d{4}-\d{2}$/.test(dateString)) {
		const [year, month] = dateString.split('-');
		const date = new Date(parseInt(year), parseInt(month) - 1);
		return date.toLocaleDateString(currentLocale(), formatOptions);
	}
	const date = new Date(dateString);
	return date.toLocaleDateString(currentLocale(), formatOptions);
}

export function formatDateRange(startDate?: string, endDate?: string, presentText: string = 'Present'): string {
	const start = formatDate(startDate);
	const end = endDate ? formatDate(endDate) : presentText;
	return `${start} - ${end}`;
}

// Extract date value for text input (supports flexible formats: YYYY, YYYY-MM, YYYY-MM-DD)
export function toDateInputValue(dateString: string | undefined | null): string {
	if (!dateString) return '';
	// If it's already a short format (year or year-month), return as-is
	if (/^\d{4}(-\d{2})?$/.test(dateString)) return dateString;
	// For ISO dates, extract just the date part (YYYY-MM-DD)
	if (dateString.includes('T') || dateString.includes(' ')) {
		return dateString.slice(0, 10);
	}
	return dateString;
}

// Markdown parsing with XSS protection
// Converts markdown to HTML, applies shortcode embeds, and sanitizes output
export function parseMarkdown(content: string): string {
	if (!content) return '';

	// Step 1: Apply shortcode transformations ({{youtube:...}}, etc.)
	const withEmbeds = applyShortcodes(content);

	// Step 2: Convert markdown to HTML
	const html = marked.parse(withEmbeds, { async: false }) as string;

	return sanitizeRichHtml(html);
}

// Media shortcodes -> embed HTML
// Usage: {{youtube:https://www.youtube.com/watch?v=...}}
// Supported providers: youtube, vimeo, loom, soundcloud, spotify, codepen,
// figma, image, video, pdf, immich, embed, calendly, calcom, googlecal, booking.
function applyShortcodes(content: string): string {
	return content.replace(/\{\{\s*([a-zA-Z0-9_]+)\s*:\s*([^}]+?)\s*\}\}/g, (_, rawProvider, rawUrl) => {
		const match: EmbedMatch = {
			provider: (rawProvider || '').toLowerCase().trim(),
			url: (rawUrl || '').trim()
		};
		return buildEmbed(match) ?? _;
	});
}

function buildEmbed(match: EmbedMatch): string | null {
	const url = sanitizeUrl(match.url);
	if (!url) return null;

	switch (match.provider) {
		case 'youtube': {
			const id = extractYouTubeId(url);
			if (!id) return null;
			return `<div class="embed video"><iframe src="https://www.youtube.com/embed/${id}" title="YouTube video" allowfullscreen loading="lazy"></iframe></div>`;
		}
		case 'vimeo': {
			const id = url.split('/').pop();
			if (!id) return null;
			return `<div class="embed video"><iframe src="https://player.vimeo.com/video/${id}" title="Vimeo video" allowfullscreen loading="lazy"></iframe></div>`;
		}
		case 'loom': {
			const id = url.split('/').pop();
			if (!id) return null;
			return `<div class="embed video"><iframe src="https://www.loom.com/embed/${id}" title="Loom video" allowfullscreen loading="lazy"></iframe></div>`;
		}
		case 'soundcloud':
			return `<div class="embed audio"><iframe scrolling="no" frameborder="no" allow="autoplay" src="https://w.soundcloud.com/player/?url=${encodeURIComponent(
				url
			)}"></iframe></div>`;
		case 'spotify':
			return `<div class="embed audio"><iframe src="${url.replace(
				'https://open.spotify.com/',
				'https://open.spotify.com/embed/'
			)}" allow="encrypted-media"></iframe></div>`;
		case 'codepen':
			return `<div class="embed code"><iframe src="${url.replace(
				'/pen/',
				'/embed/'
			)}?default-tab=result" title="CodePen" loading="lazy" allowfullscreen></iframe></div>`;
		case 'figma':
			return `<div class="embed design"><iframe src="https://www.figma.com/embed?embed_host=share&url=${encodeURIComponent(
				url
			)}" allowfullscreen></iframe></div>`;
		case 'immich':
		case 'image':
			return `<figure class="embed image"><img src="${url}" alt=""></figure>`;
		case 'video':
			return `<div class="embed video"><video src="${url}" controls></video></div>`;
		case 'pdf':
			return `<div class="embed document"><iframe src="${url}" title="PDF document" loading="lazy"></iframe></div>`;
		case 'calendly': {
			const inlineUrl = toCalendlyEmbedUrl(url);
			if (!inlineUrl) return null;
			return `<section class="embed booking calendly" aria-label="Book a meeting via Calendly"><iframe src="${inlineUrl}" title="Calendly booking calendar" loading="lazy" sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"></iframe></section>`;
		}
		case 'cal':
		case 'calcom': {
			const embedUrl = toCalcomEmbedUrl(url);
			if (!embedUrl) return null;
			return `<section class="embed booking calcom" aria-label="Book a meeting via Cal.com"><iframe src="${embedUrl}" title="Cal.com booking calendar" loading="lazy" sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"></iframe></section>`;
		}
		case 'googlecal':
		case 'googlecalendar':
		case 'booking':
			return `<div class="embed booking link"><a href="${url}" target="_blank" rel="noopener noreferrer" class="btn btn-primary">Book a time <span class="sr-only">(opens in new tab)</span></a></div>`;
		case 'embed':
		default:
			return `<div class="embed link"><a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a></div>`;
	}
}

function toCalendlyEmbedUrl(raw: string): string | null {
	try {
		const u = new URL(raw);
		if (u.protocol !== 'https:' || u.hostname !== 'calendly.com') return null;
		if (!u.searchParams.has('embed_type')) {
			u.searchParams.set('embed_type', 'Inline');
		}
		return u.toString();
	} catch {
		return null;
	}
}

function toCalcomEmbedUrl(raw: string): string | null {
	try {
		const u = new URL(raw);
		if (u.protocol !== 'https:' || u.hostname !== 'cal.com') return null;
		if (!u.pathname.endsWith('/embed')) {
			u.pathname = u.pathname.replace(/\/$/, '') + '/embed';
		}
		return u.toString();
	} catch {
		return null;
	}
}

function extractYouTubeId(url: string): string | null {
	try {
		const u = new URL(url);
		if (u.hostname.includes('youtu.be')) {
			return u.pathname.replace('/', '');
		}
		if (u.searchParams.get('v')) return u.searchParams.get('v');
		if (u.pathname.startsWith('/embed/')) return u.pathname.replace('/embed/', '');
		return null;
	} catch {
		return null;
	}
}

function sanitizeUrl(url: string): string | null {
	try {
		const u = new URL(url.trim());
		if (!u.protocol.startsWith('http')) return null;
		return u.toString();
	} catch {
		return null;
	}
}

// Skill grouping
export function groupSkillsByCategory(skills: Array<{ category?: string; name: string }>): Record<string, string[]> {
	const grouped: Record<string, string[]> = {};
	for (const skill of skills) {
		const category = skill.category || 'Other';
		if (!grouped[category]) {
			grouped[category] = [];
		}
		grouped[category].push(skill.name);
	}
	return grouped;
}

// URL helpers
export function isValidUrl(url: string): boolean {
	try {
		new URL(url);
		return true;
	} catch {
		return false;
	}
}

/**
 * Normalizes a user-provided external URL by:
 * 1. Returning null for empty/invalid values
 * 2. Prepending https:// if no protocol is present
 * 3. Returning the URL as-is if already valid
 *
 * Use this for user-provided URLs like award links, project links, etc.
 * to prevent broken relative links (e.g., "github.com/user" -> "/current-page/github.com/user")
 */
export function normalizeExternalUrl(url: string | undefined | null): string | null {
	if (!url || typeof url !== 'string') return null;

	const trimmed = url.trim();
	if (!trimmed) return null;

	// Already has a valid protocol
	if (/^(https?|mailto|tel|ftp):\/?\/?/i.test(trimmed)) {
		try {
			// Validate it's actually parseable
			new URL(trimmed);
			return trimmed;
		} catch {
			return null;
		}
	}

	// Looks like a domain (contains a dot and no spaces)
	if (trimmed.includes('.') && !trimmed.includes(' ')) {
		const withProtocol = `https://${trimmed}`;
		try {
			new URL(withProtocol);
			return withProtocol;
		} catch {
			return null;
		}
	}

	// Doesn't look like a URL (e.g., plain username, random text)
	return null;
}

/**
 * Supported external media providers. The list mirrors the iframe allowlist
 * enforced by the markdown sanitizer above (see `allowedOrigins`) so the
 * media library can only accept links that are renderable in posts/projects.
 */
export type ExternalMediaProvider =
	| 'youtube'
	| 'vimeo'
	| 'spotify'
	| 'loom'
	| 'soundcloud'
	| 'codepen'
	| 'figma';

type ExternalProviderSpec = {
	provider: ExternalMediaProvider;
	hostnames: string[]; // matched as exact or subdomain
	label: string;
};

const EXTERNAL_MEDIA_PROVIDERS: ExternalProviderSpec[] = [
	{ provider: 'youtube', hostnames: ['youtube.com', 'youtu.be', 'youtube-nocookie.com'], label: 'YouTube' },
	{ provider: 'vimeo', hostnames: ['vimeo.com', 'player.vimeo.com'], label: 'Vimeo' },
	{ provider: 'spotify', hostnames: ['spotify.com', 'open.spotify.com'], label: 'Spotify' },
	{ provider: 'loom', hostnames: ['loom.com'], label: 'Loom' },
	{ provider: 'soundcloud', hostnames: ['soundcloud.com', 'w.soundcloud.com'], label: 'SoundCloud' },
	{ provider: 'codepen', hostnames: ['codepen.io'], label: 'CodePen' },
	{ provider: 'figma', hostnames: ['figma.com'], label: 'Figma' }
];

/**
 * Detect the embed provider for a given URL by matching its hostname against
 * the same allowlist used by the markdown sanitizer iframe policy.
 *
 * Returns `null` when the URL is invalid, non-https, or not in the allowlist.
 */
export function detectExternalMediaProvider(raw: string): ExternalMediaProvider | null {
	const trimmed = (raw || '').trim();
	if (!trimmed) return null;
	let parsed: URL;
	try {
		parsed = new URL(trimmed);
	} catch {
		return null;
	}
	if (parsed.protocol !== 'https:') return null;
	const host = parsed.hostname.toLowerCase();
	for (const spec of EXTERNAL_MEDIA_PROVIDERS) {
		if (spec.hostnames.some((h) => host === h || host.endsWith(`.${h}`))) {
			return spec.provider;
		}
	}
	return null;
}

/**
 * Human-readable label for a provider (e.g., "youtube" -> "YouTube").
 */
export function getExternalMediaProviderLabel(provider: ExternalMediaProvider | null | undefined): string {
	if (!provider) return '';
	const spec = EXTERNAL_MEDIA_PROVIDERS.find((p) => p.provider === provider);
	return spec?.label ?? provider;
}

/**
 * Extracts a YouTube video ID from any common YouTube URL shape (watch, embed,
 * youtu.be, /shorts/). Returns null if no ID can be derived.
 */
export function extractYouTubeVideoId(raw: string): string | null {
	try {
		const u = new URL(raw);
		if (u.hostname.endsWith('youtu.be')) {
			const id = u.pathname.replace(/^\//, '').split('/')[0];
			return id || null;
		}
		const v = u.searchParams.get('v');
		if (v) return v;
		if (u.pathname.startsWith('/embed/')) return u.pathname.replace('/embed/', '').split('/')[0] || null;
		if (u.pathname.startsWith('/shorts/')) return u.pathname.replace('/shorts/', '').split('/')[0] || null;
		return null;
	} catch {
		return null;
	}
}

/**
 * Predictable thumbnail URL for supported providers. Today only YouTube has
 * predictable thumbnail URLs without an extra oEmbed call — others (Vimeo,
 * Spotify, Loom, etc.) return null and the UI falls back to a provider icon.
 */
export function getExternalMediaThumbnail(url: string): string | null {
	const provider = detectExternalMediaProvider(url);
	if (!provider) return null;
	if (provider === 'youtube') {
		const id = extractYouTubeVideoId(url);
		if (!id) return null;
		return `https://i.ytimg.com/vi/${id}/hqdefault.jpg`;
	}
	return null;
}

/**
 * Contact types that cannot be linked (copy-only)
 * These platforms don't support direct profile links from usernames
 */
export const NON_LINKABLE_CONTACT_TYPES = ['discord', 'slack'] as const;

/**
 * Checks if a contact type should be rendered as a link or copy-only
 */
export function isLinkableContactType(type: string): boolean {
	return !NON_LINKABLE_CONTACT_TYPES.includes(type.toLowerCase() as typeof NON_LINKABLE_CONTACT_TYPES[number]);
}

/**
 * Checks if a URL string has a hostname that matches or is a subdomain of the expected domain.
 * This prevents attacks like "evil.github.com.attacker.com" passing a check for "github.com".
 *
 * @param value - The URL string to check (may or may not have protocol)
 * @param expectedDomain - The domain to check for (e.g., "github.com")
 * @returns true if the hostname matches or is a subdomain of the expected domain
 */
function isValidDomainUrl(value: string, expectedDomain: string): boolean {
	try {
		// Ensure value has a protocol for URL parsing
		const urlString = value.startsWith('http://') || value.startsWith('https://')
			? value
			: `https://${value}`;
		const url = new URL(urlString);
		const hostname = url.hostname.toLowerCase();
		const domain = expectedDomain.toLowerCase();

		// Exact match or subdomain (e.g., "www.github.com" matches "github.com")
		return hostname === domain || hostname.endsWith(`.${domain}`);
	} catch {
		return false;
	}
}

/**
 * Builds the appropriate href for a contact method based on its type.
 * Returns null for non-linkable types (Discord, Slack).
 *
 * @param type - Contact type (email, phone, github, discord, etc.)
 * @param value - The contact value (email address, phone number, URL, username)
 * @returns The href string or null if not linkable
 */
export function buildContactHref(type: string, value: string): string | null {
	if (!value) return null;

	const normalizedType = type.toLowerCase();

	// Non-linkable types - return null (should render as copy-only)
	if (!isLinkableContactType(normalizedType)) {
		return null;
	}

	switch (normalizedType) {
		case 'email':
			return `mailto:${value}`;

		case 'phone':
			return `tel:${value.replace(/\s/g, '')}`;

		case 'whatsapp':
			// wa.me links work better than tel: for WhatsApp
			const cleanNumber = value.replace(/\s/g, '').replace(/^\+/, '');
			return `https://wa.me/${cleanNumber}`;

		case 'telegram':
			// Support both @username and full URLs
			if (isValidDomainUrl(value, 't.me') || isValidDomainUrl(value, 'telegram.me')) {
				return normalizeExternalUrl(value);
			}
			const telegramUsername = value.replace(/^@/, '');
			return `https://t.me/${telegramUsername}`;

		case 'github':
			// Support both username and full URLs
			if (isValidDomainUrl(value, 'github.com')) {
				return normalizeExternalUrl(value);
			}
			return `https://github.com/${value.replace(/^@/, '')}`;

		case 'twitter':
			// Support both @username and full URLs
			if (isValidDomainUrl(value, 'twitter.com') || isValidDomainUrl(value, 'x.com')) {
				return normalizeExternalUrl(value);
			}
			return `https://twitter.com/${value.replace(/^@/, '')}`;

		case 'instagram':
			if (isValidDomainUrl(value, 'instagram.com')) {
				return normalizeExternalUrl(value);
			}
			return `https://instagram.com/${value.replace(/^@/, '')}`;

		case 'linkedin':
		case 'facebook':
		case 'website':
		default:
			// For these, expect users to provide full URLs
			return normalizeExternalUrl(value);
	}
}

export function getLinkIcon(type: string): string {
	const icons: Record<string, string> = {
		github: '🔗',
		website: '🌐',
		linkedin: '💼',
		twitter: '🐦',
		email: '📧',
		demo: '🎮',
		docs: '📚',
		npm: '📦'
	};
	return icons[type.toLowerCase()] || '🔗';
}

// Truncation
export function truncate(text: string, maxLength: number): string {
	if (!text || text.length <= maxLength) return text;
	return text.slice(0, maxLength).trim() + '...';
}

/** Check if a string looks like a filename (has a known file extension and no spaces) */
export function isFilename(title?: string): boolean {
	return !!title && /\.(png|jpe?g|gif|webp|avif|svg|mp4|mov|webm|mkv|avi|mp3|wav|ogg|pdf|doc|docx|xls|xlsx|ppt|pptx|zip|tar|gz|rar|7z)$/i.test(title) && !title.includes(' ');
}

// Class helper
export function cn(...classes: (string | undefined | false)[]): string {
	return classes.filter(Boolean).join(' ');
}

// Theme
export function getTheme(): 'light' | 'dark' {
	if (typeof window === 'undefined') return 'light';
	if (localStorage.getItem('theme') === 'dark') return 'dark';
	if (localStorage.getItem('theme') === 'light') return 'light';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function setTheme(theme: 'light' | 'dark'): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem('theme', theme);
	document.documentElement.classList.toggle('dark', theme === 'dark');
}

export function toggleTheme(): void {
	const current = getTheme();
	setTheme(current === 'dark' ? 'light' : 'dark');
}

const DEFAULT_FETCH_TIMEOUT_MS = 5000;

export async function fetchWithTimeout(
	url: string,
	options: RequestInit = {},
	timeoutMs: number = DEFAULT_FETCH_TIMEOUT_MS
): Promise<Response> {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

	try {
		const response = await fetch(url, {
			...options,
			signal: controller.signal
		});
		clearTimeout(timeoutId);
		return response;
	} catch (err) {
		clearTimeout(timeoutId);
		throw err;
	}
}

import { marked } from 'marked';
import DOMPurify from 'isomorphic-dompurify';

type EmbedMatch = {
	provider: string;
	url: string;
};

// Configure DOMPurify with iframe domain whitelist for security
// This hook validates iframe sources to prevent malicious embeds
DOMPurify.addHook('uponSanitizeElement', (node, data) => {
	if (data.tagName === 'iframe' && node instanceof Element) {
		const src = node.getAttribute('src') || '';

		// Whitelist of trusted embed domains matching our shortcode providers
		const allowedDomains = [
			'https://www.youtube.com/embed/',
			'https://www.youtube-nocookie.com/embed/',
			'https://player.vimeo.com/video/',
			'https://www.loom.com/embed/',
			'https://w.soundcloud.com/player/',
			'https://open.spotify.com/embed/',
			'https://codepen.io/embed/',
			'https://www.figma.com/embed/'
		];

		// Remove iframe if source doesn't match whitelisted domains
		const isAllowed = allowedDomains.some(domain => src.startsWith(domain));
		if (!isAllowed) {
			node.parentNode?.removeChild(node);
			console.warn('[Security] Blocked iframe with untrusted source:', src);
		}
	}
});

export function formatDate(dateString: string | undefined, options?: Intl.DateTimeFormatOptions): string {
	if (!dateString) return '';
	if (/^\d{4}$/.test(dateString)) return dateString;
	if (/^\d{4}-\d{2}$/.test(dateString)) {
		const [year, month] = dateString.split('-');
		const date = new Date(parseInt(year), parseInt(month) - 1);
		return date.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
	}
	const date = new Date(dateString);
	return date.toLocaleDateString('en-US', options || { month: 'short', year: 'numeric' });
}

export function formatDateRange(startDate?: string, endDate?: string): string {
	const start = formatDate(startDate);
	const end = endDate ? formatDate(endDate) : 'Present';
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

	// Step 3: Sanitize HTML to prevent XSS attacks
	// This protects against malicious content in markdown (scripts, dangerous attributes, etc.)
	return DOMPurify.sanitize(html, {
		// Extend default safe tags with media/embed elements
		ADD_TAGS: ['iframe', 'video', 'audio', 'figure', 'figcaption', 'picture', 'source'],

		// Add attributes needed for media embeds and accessibility
		ADD_ATTR: [
			'allowfullscreen', // YouTube/Vimeo fullscreen capability
			'frameborder',     // iframe styling (legacy but still used)
			'loading',         // lazy loading for performance
			'controls',        // video/audio playback controls
			'target',          // open links in new tab
			'rel',             // link security (noopener, noreferrer)
			'allow',           // iframe permissions (autoplay, encrypted-media)
			'scrolling'        // iframe scrolling behavior
		],

		// Keep data-* and aria-* attributes for accessibility and functionality
		ALLOW_DATA_ATTR: true,
		ALLOW_ARIA_ATTR: true
	});
}

// Media shortcodes -> embed HTML
// Usage: {{youtube:https://www.youtube.com/watch?v=...}}
// Supported providers: youtube, vimeo, loom, soundcloud, spotify, codepen, figma, image, video, pdf, immich, embed
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
		case 'embed':
		default:
			return `<div class="embed link"><a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a></div>`;
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
			if (value.includes('t.me/') || value.includes('telegram.me/')) {
				return normalizeExternalUrl(value);
			}
			const telegramUsername = value.replace(/^@/, '');
			return `https://t.me/${telegramUsername}`;

		case 'github':
			// Support both username and full URLs
			if (value.includes('github.com')) {
				return normalizeExternalUrl(value);
			}
			return `https://github.com/${value.replace(/^@/, '')}`;

		case 'twitter':
			// Support both @username and full URLs
			if (value.includes('twitter.com') || value.includes('x.com')) {
				return normalizeExternalUrl(value);
			}
			return `https://twitter.com/${value.replace(/^@/, '')}`;

		case 'instagram':
			if (value.includes('instagram.com')) {
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

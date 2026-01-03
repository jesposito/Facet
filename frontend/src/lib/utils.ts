import { marked } from 'marked';

type EmbedMatch = {
	provider: string;
	url: string;
};

// Date formatting
export function formatDate(dateString: string | undefined, options?: Intl.DateTimeFormatOptions): string {
	if (!dateString) return '';
	const date = new Date(dateString);
	return date.toLocaleDateString('en-US', options || { month: 'short', year: 'numeric' });
}

export function formatDateRange(startDate?: string, endDate?: string): string {
	const start = formatDate(startDate);
	const end = endDate ? formatDate(endDate) : 'Present';
	return `${start} - ${end}`;
}

// Markdown parsing
export function parseMarkdown(content: string): string {
	if (!content) return '';
	const withEmbeds = applyShortcodes(content);
	return marked.parse(withEmbeds, { async: false }) as string;
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

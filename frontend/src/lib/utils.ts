import { marked } from 'marked';

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
	return marked.parse(content, { async: false }) as string;
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

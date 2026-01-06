import PocketBase from 'pocketbase';
import { writable } from 'svelte/store';
import { browser } from '$app/environment';

function getBrowserPbUrl(): string {
	if (import.meta.env.VITE_POCKETBASE_URL) {
		return import.meta.env.VITE_POCKETBASE_URL as string;
	}
	const origin = new URL(window.location.origin);
	// If running frontend dev server on 5173, talk to backend on 8090
	if (origin.port === '5173') {
		origin.port = '8090';
		return origin.toString();
	}
	return origin.toString();
}

// Initialize PocketBase client
const runtimePbUrl = browser ? getBrowserPbUrl() : process.env.POCKETBASE_URL || 'http://localhost:8090';
export const pb = new PocketBase(runtimePbUrl);

// Expose for debugging (development only)
if (browser && import.meta.env.DEV) {
	(window as unknown as { pb: PocketBase }).pb = pb;
}

// Auth store (SDK 0.21.x uses 'model')
export const currentUser = writable(pb.authStore.model);

// Update store when auth changes
pb.authStore.onChange((token, model) => {
	currentUser.set(model);
});

// Types
export interface Profile {
	id: string;
	name: string;
	headline?: string;
	location?: string;
	summary?: string;
	hero_image?: string;
	avatar?: string;
	contact_email?: string;
	contact_links?: ContactLink[];
	visibility: 'public' | 'unlisted' | 'private';
	accent_color?: 'sky' | 'indigo' | 'emerald' | 'rose' | 'amber' | 'slate';
}

export interface ContactLink {
	type: string;
	url: string;
	label?: string;
}

export interface Experience {
	id: string;
	company: string;
	title: string;
	location?: string;
	start_date?: string;
	end_date?: string;
	description?: string;
	bullets?: string[];
	skills?: string[];
	media?: string[];
	visibility: 'public' | 'unlisted' | 'private' | 'password';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	sort_order: number;
}

export interface Project {
	id: string;
	title: string;
	slug?: string;
	summary?: string;
	description?: string;
	tech_stack?: string[];
	links?: ProjectLink[];
	media?: string[];
	media_refs?: string[];
	cover_image?: string;
	categories?: string[];
	visibility: 'public' | 'unlisted' | 'private' | 'password';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	is_featured: boolean;
	sort_order: number;
	source_id?: string;
	field_locks?: Record<string, boolean>;
}

export interface ProjectLink {
	type: string;
	url: string;
}

export interface Education {
	id: string;
	institution: string;
	degree?: string;
	field?: string;
	start_date?: string;
	end_date?: string;
	description?: string;
	visibility: 'public' | 'unlisted' | 'private';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	sort_order: number;
}

export interface Skill {
	id: string;
	name: string;
	category?: string;
	proficiency?: 'expert' | 'proficient' | 'familiar';
	visibility: 'public' | 'unlisted' | 'private';
	view_visibility?: Record<string, boolean>;
	sort_order: number;
}

export interface Post {
	id: string;
	title: string;
	slug?: string;
	excerpt?: string;
	content?: string;
	cover_image?: string;
	tags?: string[];
	media_refs?: string[];
	visibility: 'public' | 'unlisted' | 'private';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	published_at?: string;
}

export interface Talk {
	id: string;
	title: string;
	slug?: string;
	event?: string;
	event_url?: string;
	date?: string;
	location?: string;
	description?: string;
	slides_url?: string;
	video_url?: string;
	media_refs?: string[];
	visibility: 'public' | 'unlisted' | 'private';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	sort_order: number;
}

export interface Certification {
	id: string;
	name: string;
	issuer?: string;
	issue_date?: string;
	expiry_date?: string;
	credential_id?: string;
	credential_url?: string;
	visibility: 'public' | 'unlisted' | 'private';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	sort_order: number;
}

export interface Award {
	id: string;
	title: string;
	issuer?: string;
	awarded_at?: string;
	description?: string;
	url?: string;
	visibility: 'public' | 'unlisted' | 'private';
	view_visibility?: Record<string, boolean>;
	is_draft: boolean;
	sort_order: number;
}

export type ContactMethodType =
	| 'email'
	| 'phone'
	| 'linkedin'
	| 'github'
	| 'twitter'
	| 'facebook'
	| 'instagram'
	| 'website'
	| 'whatsapp'
	| 'telegram'
	| 'discord'
	| 'slack'
	| 'other';

export type ProtectionLevel = 'none' | 'obfuscation' | 'click_to_reveal' | 'captcha';

export interface ContactMethod {
	id: string;
	type: ContactMethodType;
	value: string;
	label?: string;
	icon?: string;
	protection_level: ProtectionLevel;
	view_visibility?: Record<string, boolean>;
	is_primary: boolean;
	sort_order: number;
}

export interface View {
	id: string;
	name: string;
	slug: string;
	description?: string;
	visibility: 'public' | 'unlisted' | 'private' | 'password';
	hero_headline?: string;
	hero_summary?: string;
	cta_text?: string;
	cta_url?: string;
	sections?: ViewSection[];
	is_active: boolean;
	is_default?: boolean;
	accent_color?: 'sky' | 'indigo' | 'emerald' | 'rose' | 'amber' | 'slate' | null;
}

export interface ItemConfig {
	order?: number;
	overrides?: Record<string, string | string[]>;
}

// Layout types for each section
export type SectionLayout = string;

// Width options for sections (Phase 6.3)
export type SectionWidth = 'full' | 'half' | 'third';

export interface ViewSection {
	section: string;
	enabled: boolean;
	items?: string[];
	order?: number;
	layout?: SectionLayout;
	width?: SectionWidth;
	itemConfig?: Record<string, ItemConfig>;
}

// Valid width options with labels
export const VALID_WIDTHS: { value: SectionWidth; label: string }[] = [
	{ value: 'full', label: 'Full Width' },
	{ value: 'half', label: 'Half Width' },
	{ value: 'third', label: 'Third Width' }
];

// Valid layouts per section type (curated presets)
export const VALID_LAYOUTS: Record<string, { layouts: string[]; default: string; labels: Record<string, string> }> = {
	experience: {
		layouts: ['default', 'timeline', 'compact'],
		default: 'default',
		labels: {
			default: 'Default',
			timeline: 'Timeline',
			compact: 'Compact'
		}
	},
	projects: {
		layouts: ['grid-3', 'grid-2', 'list', 'featured'],
		default: 'grid-3',
		labels: {
			'grid-3': '3-Column Grid',
			'grid-2': '2-Column Grid',
			list: 'List',
			featured: 'Featured + Grid'
		}
	},
	education: {
		layouts: ['default', 'timeline'],
		default: 'default',
		labels: {
			default: 'Default',
			timeline: 'Timeline'
		}
	},
	certifications: {
		layouts: ['grouped', 'grid', 'timeline'],
		default: 'grouped',
		labels: {
			grouped: 'Grouped by Issuer',
			grid: 'Grid',
			timeline: 'Timeline'
		}
	},
	skills: {
		layouts: ['grouped', 'cloud', 'bars', 'flat'],
		default: 'grouped',
		labels: {
			grouped: 'Grouped by Category',
			cloud: 'Tag Cloud',
			bars: 'Skill Bars',
			flat: 'Flat List'
		}
	},
	awards: {
		layouts: ['grouped', 'timeline', 'grid'],
		default: 'grouped',
		labels: {
			grouped: 'Grouped by Issuer',
			timeline: 'Timeline',
			grid: 'Grid'
		}
	},
	posts: {
		layouts: ['grid-3', 'grid-2', 'list', 'featured'],
		default: 'grid-3',
		labels: {
			'grid-3': '3-Column Grid',
			'grid-2': '2-Column Grid',
			list: 'List',
			featured: 'Featured + Grid'
		}
	},
	talks: {
		layouts: ['default', 'cards', 'list'],
		default: 'default',
		labels: {
			default: 'Default',
			cards: 'Cards',
			list: 'List'
		}
	},
	contacts: {
		layouts: ['vertical', 'horizontal', 'grid'],
		default: 'vertical',
		labels: {
			vertical: 'Vertical List',
			horizontal: 'Horizontal',
			grid: 'Grid'
		}
	}
};

// Helper to get section layout with fallback to default
export function getSectionLayout(section: string, layout?: string): string {
	const config = VALID_LAYOUTS[section];
	if (!config) return 'default';
	if (layout && config.layouts.includes(layout)) return layout;
	return config.default;
}

// Layout-to-width restrictions: which layouts only work at full width
// Layouts not listed here support all widths (full, half, third)
export const FULL_WIDTH_ONLY_LAYOUTS: Record<string, string[]> = {
	experience: ['timeline'],      // Timeline visual needs horizontal space
	education: ['timeline'],       // Timeline visual needs horizontal space
	certifications: ['timeline'],  // Timeline visual needs horizontal space
	projects: ['featured'],        // Featured item takes full width
	posts: ['featured'],           // Featured item takes full width
	skills: ['cloud', 'bars'],     // Cloud/bars need space to look good
	talks: []                      // All talks layouts work at any width
};

// Get valid widths for a given section and layout
export function getValidWidthsForLayout(
	sectionKey: string,
	layout: string
): { value: SectionWidth; label: string }[] {
	const fullOnlyLayouts = FULL_WIDTH_ONLY_LAYOUTS[sectionKey] || [];

	if (fullOnlyLayouts.includes(layout)) {
		// Only full width is valid for this layout
		return [{ value: 'full', label: 'Full Width' }];
	}

	// All widths are valid
	return VALID_WIDTHS;
}

// Check if a width is valid for a given section and layout
export function isWidthValidForLayout(
	sectionKey: string,
	layout: string,
	width: SectionWidth
): boolean {
	const fullOnlyLayouts = FULL_WIDTH_ONLY_LAYOUTS[sectionKey] || [];

	if (fullOnlyLayouts.includes(layout)) {
		return width === 'full';
	}

	return true;
}

// Define which fields can be overridden per collection
export const OVERRIDABLE_FIELDS: Record<string, string[]> = {
	experience: ['title', 'description', 'bullets'],
	projects: ['title', 'summary', 'description'],
	education: ['degree', 'field', 'description'],
	talks: ['title', 'description']
};

export interface AIProvider {
	id: string;
	name: string;
	type: 'openai' | 'anthropic' | 'ollama' | 'custom';
	base_url?: string;
	model?: string;
	is_default: boolean;
	is_active: boolean;
	test_status?: string;
	last_test?: string;
}

export interface Source {
	id: string;
	type: 'github';
	identifier: string;
	project_id?: string;
	last_sync?: string;
	sync_status?: 'pending' | 'success' | 'error';
	sync_log?: string;
}

export interface ImportProposal {
	id: string;
	source_id: string;
	project_id?: string;
	proposed_data: Record<string, unknown>;
	diff?: Record<string, { type: string; old?: unknown; new?: unknown }>;
	ai_enriched: boolean;
	status: 'pending' | 'applied' | 'rejected';
}

export interface ShareToken {
	id: string;
	view_id: string;
	token_hash: string;
	token_prefix?: string;
	name?: string;
	expires_at?: string;
	max_uses?: number;
	use_count: number;
	is_active: boolean;
	last_used_at?: string;
	created: string;
	updated: string;
	expand?: {
		view_id?: View;
	};
}

// API helpers
export async function fetchProfile(): Promise<Profile | null> {
	try {
		const records = await pb.collection('profile').getList(1, 1);
		return records.items[0] as unknown as Profile;
	} catch {
		return null;
	}
}

export async function fetchExperience(): Promise<Experience[]> {
	try {
		const records = await pb.collection('experience').getList(1, 100, {
			filter: "visibility != 'private' && is_draft = false",
			sort: '-sort_order,-start_date'
		});
		return records.items as unknown as Experience[];
	} catch {
		return [];
	}
}

export async function fetchProjects(): Promise<Project[]> {
	try {
		const records = await pb.collection('projects').getList(1, 100, {
			filter: "visibility != 'private' && is_draft = false",
			sort: '-is_featured,-sort_order'
		});
		return records.items as unknown as Project[];
	} catch {
		return [];
	}
}

export async function fetchEducation(): Promise<Education[]> {
	try {
		const records = await pb.collection('education').getList(1, 100, {
			filter: "visibility != 'private' && is_draft = false",
			sort: '-sort_order,-end_date'
		});
		return records.items as unknown as Education[];
	} catch {
		return [];
	}
}

export async function fetchSkills(): Promise<Skill[]> {
	try {
		const records = await pb.collection('skills').getList(1, 200, {
			filter: "visibility != 'private'",
			sort: 'category,sort_order'
		});
		return records.items as unknown as Skill[];
	} catch {
		return [];
	}
}

export async function fetchPosts(): Promise<Post[]> {
	try {
		const records = await pb.collection('posts').getList(1, 100, {
			filter: "visibility != 'private' && is_draft = false",
			sort: '-published_at'
		});
		return records.items as unknown as Post[];
	} catch {
		return [];
	}
}

export async function fetchTalks(): Promise<Talk[]> {
	try {
		const records = await pb.collection('talks').getList(1, 100, {
			filter: "visibility != 'private' && is_draft = false",
			sort: '-sort_order,-date'
		});
		return records.items as unknown as Talk[];
	} catch {
		return [];
	}
}

export async function fetchCertifications(): Promise<Certification[]> {
	try {
		const records = await pb.collection('certifications').getList(1, 100, {
			filter: "visibility != 'private' && is_draft = false",
			sort: 'issuer,sort_order,-issue_date'
		});
		return records.items as unknown as Certification[];
	} catch {
		return [];
	}
}

export async function fetchContactMethods(): Promise<ContactMethod[]> {
	try {
		const records = await pb.collection('contact_methods').getList(1, 100, {
			sort: '-is_primary,sort_order'
		});
		return records.items as unknown as ContactMethod[];
	} catch {
		return [];
	}
}

export function getFileUrl(record: { id: string; collectionId?: string; collectionName?: string }, filename: string): string {
	if (!filename) return '';
	const collectionId = record.collectionId || record.collectionName;
	// Use relative URL - works behind any reverse proxy
	return `/api/files/${collectionId}/${record.id}/${filename}`;
}

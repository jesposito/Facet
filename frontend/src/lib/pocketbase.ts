import PocketBase from 'pocketbase';
import { writable } from 'svelte/store';
import { browser } from '$app/environment';

// Initialize PocketBase client
const pbUrl = browser ? window.location.origin : (process.env.POCKETBASE_URL || 'http://localhost:8090');
export const pb = new PocketBase(pbUrl);

// Force Bearer prefix on auth header (required for PocketBase 0.23+)
pb.beforeSend = function (url, options) {
	const token = pb.authStore.token;
	if (token) {
		// Always set Authorization header with Bearer prefix
		options.headers = Object.assign({}, options.headers, {
			'Authorization': 'Bearer ' + token
		});
	}
	return { url, options };
};

// Auth store (SDK 0.22+ uses 'record' instead of 'model')
export const currentUser = writable(pb.authStore.record);

// Update store when auth changes
pb.authStore.onChange((token, record) => {
	currentUser.set(record);
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
	cover_image?: string;
	categories?: string[];
	visibility: 'public' | 'unlisted' | 'private' | 'password';
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
	is_draft: boolean;
	sort_order: number;
}

export interface Skill {
	id: string;
	name: string;
	category?: string;
	proficiency?: 'expert' | 'proficient' | 'familiar';
	visibility: 'public' | 'unlisted' | 'private';
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
	visibility: 'public' | 'unlisted' | 'private';
	is_draft: boolean;
	published_at?: string;
}

export interface Talk {
	id: string;
	title: string;
	event?: string;
	event_url?: string;
	date?: string;
	location?: string;
	description?: string;
	slides_url?: string;
	video_url?: string;
	visibility: 'public' | 'unlisted' | 'private';
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
	is_draft: boolean;
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
}

export interface ItemConfig {
	order?: number;
	overrides?: Record<string, string | string[]>;
}

export interface ViewSection {
	section: string;
	enabled: boolean;
	items?: string[];
	order?: number;
	itemConfig?: Record<string, ItemConfig>;
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

export function getFileUrl(record: { id: string; collectionId?: string; collectionName?: string }, filename: string): string {
	if (!filename) return '';
	const collectionId = record.collectionId || record.collectionName;
	// Use relative URL - works behind any reverse proxy
	return `/api/files/${collectionId}/${record.id}/${filename}`;
}

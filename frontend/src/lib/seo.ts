import type { RecordModel } from 'pocketbase';

// Type definitions for JSON-LD schemas
export interface JsonLdPerson {
	'@context': 'https://schema.org';
	'@type': 'Person';
	name?: string;
	jobTitle?: string;
	description?: string;
	url?: string;
	image?: string;
	sameAs?: string[];
}

export interface JsonLdArticle {
	'@context': 'https://schema.org';
	'@type': 'BlogPosting' | 'Article';
	headline: string;
	description?: string;
	image?: string;
	datePublished?: string;
	dateModified?: string;
	author?: JsonLdPerson;
	url?: string;
}

export interface JsonLdWebSite {
	'@context': 'https://schema.org';
	'@type': 'WebSite';
	name: string;
	description?: string;
	url: string;
}

/**
 * Generate JSON-LD for a person profile (homepage)
 */
export function generatePersonJsonLd(profile: RecordModel, baseUrl: string): JsonLdPerson {
	const socialLinks: string[] = [];

	// Extract social links
	if (profile.contact_links && Array.isArray(profile.contact_links)) {
		for (const link of profile.contact_links) {
			if (link.url) {
				socialLinks.push(link.url);
			}
		}
	}

	return {
		'@context': 'https://schema.org',
		'@type': 'Person',
		name: profile.name || undefined,
		jobTitle: profile.headline || undefined,
		description: profile.summary || undefined,
		url: baseUrl,
		image: profile.avatar_url || undefined,
		sameAs: socialLinks.length > 0 ? socialLinks : undefined
	};
}

/**
 * Generate JSON-LD for a blog post
 */
export function generateArticleJsonLd(
	post: RecordModel,
	baseUrl: string,
	author?: RecordModel
): JsonLdArticle {
	const authorData: JsonLdPerson | undefined = author
		? {
				'@context': 'https://schema.org',
				'@type': 'Person',
				name: author.name || undefined,
				url: baseUrl
		  }
		: undefined;

	return {
		'@context': 'https://schema.org',
		'@type': 'BlogPosting',
		headline: post.title,
		description: post.excerpt || undefined,
		image: post.cover_image_url || undefined,
		datePublished: post.published_at || post.created,
		dateModified: post.updated,
		author: authorData,
		url: `${baseUrl}/posts/${post.slug}`
	};
}

/**
 * Generate JSON-LD for the website itself
 */
export function generateWebSiteJsonLd(profile: RecordModel, baseUrl: string): JsonLdWebSite {
	return {
		'@context': 'https://schema.org',
		'@type': 'WebSite',
		name: profile.name ? `${profile.name}'s Profile` : 'Profile',
		description: profile.headline || profile.summary || undefined,
		url: baseUrl
	};
}

/**
 * Serialize JSON-LD to HTML script tag content
 */
export function serializeJsonLd(data: object): string {
	return JSON.stringify(data, null, 0); // Minified for production
}

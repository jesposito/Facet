export interface SitemapRecord {
	slug?: string;
	updated?: string;
	visibility?: string;
	is_draft?: boolean;
}

export interface SitemapUrl {
	loc: string;
	lastmod: string;
	priority: string;
	changefreq: string;
}

/** Only published, public records belong in a sitemap; private and unlisted ones stay out. */
export function publicRecords(records: SitemapRecord[]): SitemapRecord[] {
	return records.filter(
		(record) => record.slug && record.visibility === 'public' && !record.is_draft
	);
}

export function lastmod(value?: string): string {
	const date = value ? new Date(value) : new Date();
	return (Number.isNaN(date.getTime()) ? new Date() : date).toISOString().split('T')[0];
}

export function escapeXml(value: string): string {
	return value
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;')
		.replaceAll("'", '&apos;');
}

export function renderSitemap(urls: SitemapUrl[]): string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls
	.map(
		(url) => `  <url>
    <loc>${escapeXml(url.loc)}</loc>
    <lastmod>${url.lastmod}</lastmod>
    <changefreq>${url.changefreq}</changefreq>
    <priority>${url.priority}</priority>
  </url>`
	)
	.join('\n')}
</urlset>`;
}

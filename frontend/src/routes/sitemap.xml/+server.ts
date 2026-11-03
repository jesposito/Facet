import type { RequestHandler } from './$types';
import { publicOrigin } from '$lib/server/robots';
import {
	lastmod,
	publicRecords,
	renderSitemap,
	type SitemapRecord,
	type SitemapUrl
} from '$lib/server/sitemap';

async function getJson<T>(fetcher: typeof fetch, path: string): Promise<T> {
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	const response = await fetcher(`${pbUrl}${path}`, { headers: { 'X-Internal': 'true' } });
	if (!response.ok) throw new Error(`Sitemap data request failed: ${path} (${response.status})`);
	return (await response.json()) as T;
}

export const GET: RequestHandler = async ({ fetch, url }) => {
	// Same origin choice as robots.txt, so the two never disagree.
	const baseUrl = publicOrigin(process.env, url.origin);
	try {
		const [homepage, postsData, talksData, navData] = await Promise.all([
			getJson<{ projects?: SitemapRecord[] }>(fetch, '/api/homepage'),
			getJson<{ posts?: SitemapRecord[] }>(fetch, '/api/posts'),
			getJson<{ talks?: SitemapRecord[] }>(fetch, '/api/talks'),
			getJson<{ items?: Array<{ slug?: string; updated?: string; visibility?: string }> }>(
				fetch,
				'/api/site-nav'
			)
		]);
		const projects = publicRecords(homepage.projects || []);
		const posts = publicRecords(postsData.posts || []);
		const talks = publicRecords(talksData.talks || []);
		const urls: SitemapUrl[] = [
			{ loc: baseUrl, lastmod: lastmod(), priority: '1.0', changefreq: 'weekly' }
		];

		for (const view of navData.items || []) {
			// Unlisted and private views are reachable only by link or login.
			if (!view.slug || (view.visibility && view.visibility !== 'public')) continue;
			urls.push({
				loc: `${baseUrl}/${encodeURIComponent(view.slug)}`,
				lastmod: lastmod(view.updated),
				priority: '0.9',
				changefreq: 'weekly'
			});
		}
		for (const project of projects) {
			urls.push({
				loc: `${baseUrl}/projects/${encodeURIComponent(project.slug!)}`,
				lastmod: lastmod(project.updated),
				priority: '0.7',
				changefreq: 'monthly'
			});
		}
		if (posts.length > 0) {
			urls.push({
				loc: `${baseUrl}/posts`,
				lastmod: lastmod(posts[0].updated),
				priority: '0.8',
				changefreq: 'weekly'
			});
		}
		for (const post of posts) {
			urls.push({
				loc: `${baseUrl}/posts/${encodeURIComponent(post.slug!)}`,
				lastmod: lastmod(post.updated),
				priority: '0.7',
				changefreq: 'monthly'
			});
		}
		if (talks.length > 0) {
			urls.push({
				loc: `${baseUrl}/talks`,
				lastmod: lastmod(talks[0].updated),
				priority: '0.6',
				changefreq: 'monthly'
			});
		}
		for (const talk of talks) {
			urls.push({
				loc: `${baseUrl}/talks/${encodeURIComponent(talk.slug!)}`,
				lastmod: lastmod(talk.updated),
				priority: '0.5',
				changefreq: 'monthly'
			});
		}

		return new Response(renderSitemap(urls), {
			headers: { 'Content-Type': 'application/xml', 'Cache-Control': 'public, max-age=3600' }
		});
	} catch (error) {
		console.error('Sitemap generation error:', error);
		return new Response(
			renderSitemap([{ loc: baseUrl, lastmod: lastmod(), priority: '1.0', changefreq: 'weekly' }]),
			{ headers: { 'Content-Type': 'application/xml', 'Cache-Control': 'public, max-age=3600' } }
		);
	}
};

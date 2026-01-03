import { pb } from '$lib/pocketbase';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	try {
		// Fetch all public content
		const [projects, posts, talks, views] = await Promise.all([
			pb.collection('projects').getFullList({
				filter: 'visibility = "public" && is_draft = false',
				fields: 'slug,updated'
			}),
			pb.collection('posts').getFullList({
				filter: 'visibility = "public" && is_draft = false',
				fields: 'slug,updated'
			}),
			pb.collection('talks').getFullList({
				filter: 'visibility = "public"',
				fields: 'slug,updated'
			}),
			pb.collection('views').getFullList({
				filter: 'visibility = "public" && is_active = true',
				fields: 'slug,updated'
			})
		]);

		const baseUrl = process.env.PUBLIC_APP_URL || process.env.APP_URL || 'http://localhost:8080';

		const urls: Array<{ loc: string; lastmod: string; priority: string; changefreq: string }> = [];

		// Homepage - highest priority
		urls.push({
			loc: baseUrl,
			lastmod: new Date().toISOString().split('T')[0],
			priority: '1.0',
			changefreq: 'weekly'
		});

		// Public views
		for (const view of views) {
			urls.push({
				loc: `${baseUrl}/${view.slug}`,
				lastmod: new Date(view.updated).toISOString().split('T')[0],
				priority: '0.9',
				changefreq: 'weekly'
			});
		}

		// Projects index
		if (projects.length > 0) {
			urls.push({
				loc: `${baseUrl}/projects`,
				lastmod: new Date(projects[0].updated).toISOString().split('T')[0],
				priority: '0.8',
				changefreq: 'weekly'
			});
		}

		// Individual projects
		for (const project of projects) {
			urls.push({
				loc: `${baseUrl}/projects/${project.slug}`,
				lastmod: new Date(project.updated).toISOString().split('T')[0],
				priority: '0.7',
				changefreq: 'monthly'
			});
		}

		// Posts index
		if (posts.length > 0) {
			urls.push({
				loc: `${baseUrl}/posts`,
				lastmod: new Date(posts[0].updated).toISOString().split('T')[0],
				priority: '0.8',
				changefreq: 'weekly'
			});
		}

		// Individual posts
		for (const post of posts) {
			urls.push({
				loc: `${baseUrl}/posts/${post.slug}`,
				lastmod: new Date(post.updated).toISOString().split('T')[0],
				priority: '0.7',
				changefreq: 'monthly'
			});
		}

		// Talks index
		if (talks.length > 0) {
			urls.push({
				loc: `${baseUrl}/talks`,
				lastmod: new Date(talks[0].updated).toISOString().split('T')[0],
				priority: '0.6',
				changefreq: 'monthly'
			});
		}

		// Individual talks
		for (const talk of talks) {
			if (talk.slug) {
				urls.push({
					loc: `${baseUrl}/talks/${talk.slug}`,
					lastmod: new Date(talk.updated).toISOString().split('T')[0],
					priority: '0.5',
					changefreq: 'monthly'
				});
			}
		}

		// Generate XML
		const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls
	.map(
		(url) => `  <url>
    <loc>${url.loc}</loc>
    <lastmod>${url.lastmod}</lastmod>
    <changefreq>${url.changefreq}</changefreq>
    <priority>${url.priority}</priority>
  </url>`
	)
	.join('\n')}
</urlset>`;

		return new Response(sitemap, {
			headers: {
				'Content-Type': 'application/xml',
				'Cache-Control': 'public, max-age=3600' // Cache for 1 hour
			}
		});
	} catch (error) {
		console.error('Sitemap generation error:', error);
		// Return minimal sitemap on error
		const baseUrl = process.env.PUBLIC_APP_URL || process.env.APP_URL || 'http://localhost:8080';
		const fallbackSitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>${baseUrl}</loc>
    <lastmod>${new Date().toISOString().split('T')[0]}</lastmod>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
</urlset>`;

		return new Response(fallbackSitemap, {
			headers: {
				'Content-Type': 'application/xml'
			}
		});
	}
};

import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	const baseUrl = process.env.PUBLIC_APP_URL || process.env.APP_URL || 'http://localhost:8080';

	const robotsTxt = `# Facet - Personal Profile Platform
# Your profile, your data, your rules.

User-agent: *
Allow: /
Allow: /projects
Allow: /posts
Allow: /talks

# Disallow admin and private areas
Disallow: /admin
Disallow: /s/
Disallow: /v/

# Sitemap
Sitemap: ${baseUrl}/sitemap.xml
`;

	return new Response(robotsTxt, {
		headers: {
			'Content-Type': 'text/plain',
			'Cache-Control': 'public, max-age=86400' // Cache for 24 hours
		}
	});
};

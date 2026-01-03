import { test, expect } from '@playwright/test';
import { apiBaseURL } from './config';

test.describe('SEO features', () => {
	test('sitemap.xml is valid and contains URLs', async ({ request }) => {
		// Sitemap is served by SvelteKit frontend via baseURL (default: http://localhost:5173)
		const sitemapRes = await request.get('/sitemap.xml');
		expect(sitemapRes.status()).toBe(200);
		expect(sitemapRes.headers()['content-type']).toContain('xml');

		const sitemapText = await sitemapRes.text();
		expect(sitemapText).toContain('<?xml');
		expect(sitemapText).toContain('<urlset');
		expect(sitemapText).toContain('http://www.sitemaps.org/schemas/sitemap/0.9');
		// Should contain at least the homepage
		expect(sitemapText).toContain('<loc>');
	});

	test('robots.txt is accessible and valid', async ({ request }) => {
		// Robots.txt is served by SvelteKit frontend
		const robotsRes = await request.get('/robots.txt');
		expect(robotsRes.status()).toBe(200);
		expect(robotsRes.headers()['content-type']).toContain('text/plain');

		const robotsText = await robotsRes.text();
		expect(robotsText).toContain('User-agent:');
		expect(robotsText).toContain('Sitemap:');
		expect(robotsText).toMatch(/sitemap\.xml/i);
	});

	test('homepage has canonical URL and Open Graph tags', async ({ page }) => {
		await page.goto('/');

		// Check for canonical URL
		const canonical = page.locator('link[rel="canonical"]');
		await expect(canonical).toHaveCount(1);
		const canonicalHref = await canonical.getAttribute('href');
		expect(canonicalHref).toBeTruthy();

		// Check for Open Graph tags
		const ogTitle = page.locator('meta[property="og:title"]');
		await expect(ogTitle).toHaveCount(1);

		const ogUrl = page.locator('meta[property="og:url"]');
		await expect(ogUrl).toHaveCount(1);

		const ogType = page.locator('meta[property="og:type"]');
		await expect(ogType).toHaveCount(1);

		// Check for Twitter Card
		const twitterCard = page.locator('meta[property="twitter:card"]');
		await expect(twitterCard).toHaveCount(1);
	});

	test('project pages have proper SEO meta tags', async ({ request, page }) => {
		// First, fetch a public project
		const projectsRes = await request.get(`${apiBaseURL}/api/projects`);
		if (!projectsRes.ok()) {
			test.skip(true, 'No projects API available');
			return;
		}

		const projectsData = await projectsRes.json();
		const projects = projectsData?.projects || [];

		if (projects.length === 0) {
			test.skip(true, 'No projects available for testing');
			return;
		}

		const project = projects[0];
		if (!project.slug) {
			test.skip(true, 'Project has no slug');
			return;
		}

		// Visit the project page
		await page.goto(`/projects/${project.slug}`);

		// Check canonical URL
		const canonical = page.locator('link[rel="canonical"]');
		await expect(canonical).toHaveCount(1);
		const canonicalHref = await canonical.getAttribute('href');
		expect(canonicalHref).toContain(`projects/${project.slug}`);

		// Check Open Graph type is article
		const ogType = page.locator('meta[property="og:type"]');
		const typeContent = await ogType.getAttribute('content');
		expect(typeContent).toBe('article');

		// Check for basic OG tags
		await expect(page.locator('meta[property="og:title"]')).toHaveCount(1);
		await expect(page.locator('meta[property="og:url"]')).toHaveCount(1);
	});

	test('post pages have article metadata', async ({ request, page }) => {
		// First, fetch a public post
		const postsRes = await request.get(`${apiBaseURL}/api/posts`);
		if (!postsRes.ok()) {
			test.skip(true, 'No posts API available');
			return;
		}

		const postsData = await postsRes.json();
		const posts = postsData?.posts || [];

		if (posts.length === 0) {
			test.skip(true, 'No posts available for testing');
			return;
		}

		const post = posts[0];
		if (!post.slug) {
			test.skip(true, 'Post has no slug');
			return;
		}

		// Visit the post page
		await page.goto(`/posts/${post.slug}`);

		// Check canonical URL
		const canonical = page.locator('link[rel="canonical"]');
		await expect(canonical).toHaveCount(1);
		const canonicalHref = await canonical.getAttribute('href');
		expect(canonicalHref).toContain(`posts/${post.slug}`);

		// Check Open Graph type is article
		const ogType = page.locator('meta[property="og:type"]');
		const typeContent = await ogType.getAttribute('content');
		expect(typeContent).toBe('article');

		// Check for published time if post has it
		if (post.published_at) {
			const publishedTime = page.locator('meta[property="article:published_time"]');
			await expect(publishedTime).toHaveCount(1);
		}
	});

	test('JSON-LD structured data is present when profile exists', async ({ page }) => {
		await page.goto('/');

		// Check if page has profile data (not just "This profile is being set up")
		const bodyText = await page.locator('body').textContent();
		const hasProfile = bodyText && !bodyText.includes('This profile is being set up');

		if (hasProfile) {
			// Profile exists, should have JSON-LD
			const jsonLdScripts = page.locator('script[type="application/ld+json"]');
			const count = await jsonLdScripts.count();

			// Should have at least Person or WebSite schema
			expect(count).toBeGreaterThan(0);

			// Validate JSON-LD is parseable
			const firstScript = jsonLdScripts.first();
			const content = await firstScript.textContent();
			expect(content).toBeTruthy();

			// Should be valid JSON
			const jsonLd = JSON.parse(content!);
			expect(jsonLd['@context']).toBe('https://schema.org');
			expect(jsonLd['@type']).toBeTruthy();
		} else {
			// No profile configured yet - JSON-LD not expected
			// This is correct behavior, test passes
			console.log('[TEST] No profile data, JSON-LD correctly omitted');
		}
	});
});

test.describe('Error pages', () => {
	test('404 page shows for non-existent routes', async ({ page }) => {
		const response = await page.goto('/this-page-definitely-does-not-exist-12345');

		// Should return 404 status
		expect(response?.status()).toBe(404);

		// Should show custom error page content
		const content = await page.textContent('body');
		expect(content).toBeTruthy();

		// Should have the custom 404 message
		await expect(page.locator('h1')).toContainText('404');
	});

	test('500 error page is accessible via test route', async ({ page }) => {
		const response = await page.goto('/test-500');

		// Should return 500 status
		expect(response?.status()).toBe(500);

		// Should show custom error page content
		const content = await page.textContent('body');
		expect(content).toBeTruthy();

		// Should have the custom 500 message
		await expect(page.locator('h1')).toContainText('500');
	});

	test('error pages have SVG illustrations', async ({ page }) => {
		await page.goto('/this-does-not-exist');

		// Check for SVG element (our custom illustrations)
		const svg = page.locator('svg');
		const svgCount = await svg.count();

		// Should have at least one SVG (the illustration)
		expect(svgCount).toBeGreaterThan(0);
	});

	test('error pages have "Go Home" button', async ({ page }) => {
		await page.goto('/this-does-not-exist');

		// Should have a "Go Home" button
		const homeButton = page.getByTestId('go-home-button');
		await expect(homeButton).toBeVisible();

		// Button should navigate home when clicked
		await homeButton.click();
		await expect(page).toHaveURL('/');
	});
});

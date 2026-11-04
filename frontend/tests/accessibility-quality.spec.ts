import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

const WCAG_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];

const MODES = [
	{ name: 'desktop light', theme: 'light', viewport: { width: 1440, height: 900 } },
	{ name: 'mobile dark', theme: 'dark', viewport: { width: 390, height: 844 } }
] as const;

const ADMIN_ROUTES = [
	'/admin/audience',
	'/admin/content',
	'/admin/help',
	'/admin/homepage',
	'/admin/import',
	'/admin/media',
	'/admin/newsletter/compose',
	'/admin/pricing',
	'/admin/profile',
	'/admin/settings',
	'/admin/settings/about',
	'/admin/settings/audit',
	'/admin/settings/integrations',
	'/admin/settings/payments',
	'/admin/testimonials',
	'/admin/testimonials/requests',
	'/admin/views/new'
];

// Uses the real toggle, then waits for the theme's colour transitions to end so
// axe measures the settled colours and not a frame from the middle of a fade.
async function settleTheme(page: Page, theme: 'light' | 'dark') {
	const opposite = theme === 'dark' ? 'light' : 'dark';
	const switchToTheme = page.getByRole('button', { name: `Switch to ${theme} theme` });
	const switchBack = page.getByRole('button', { name: `Switch to ${opposite} theme` });

	await expect(switchToTheme.or(switchBack).first()).toBeVisible();
	await expect(async () => {
		if (await switchToTheme.isVisible()) await switchToTheme.click();
		await expect(switchBack).toBeVisible({ timeout: 1_000 });
	}).toPass();

	await page.evaluate(async () => {
		const nextFrame = () => new Promise((resolve) => requestAnimationFrame(resolve));
		await nextFrame();
		await nextFrame();
		await Promise.allSettled(
			document
				.getAnimations()
				.filter((animation) => animation instanceof CSSTransition)
				.map((animation) => animation.finished)
		);
	});
}

async function expectNoViolations(page: Page, theme: 'light' | 'dark') {
	await expect(page.getByRole('heading').first()).toBeVisible();
	await expect(page.locator('.animate-spin')).toHaveCount(0);
	await settleTheme(page, theme);

	const { violations } = await new AxeBuilder({ page }).withTags(WCAG_TAGS).analyze();
	expect(violations.map(({ id, nodes }) => ({ id, targets: nodes.map((n) => n.target) }))).toEqual(
		[]
	);
}

test.describe('settled-theme accessibility', () => {
	const suffix = randomUUID().slice(0, 8);
	const projectTitle = `Accessibility project ${suffix}`;
	const mediaTitle = `Accessibility image ${suffix}`;
	const viewSlug = `accessibility-quality-${suffix}`;
	const created: { collection: string; id: string }[] = [];
	let token: string;
	let viewId: string;

	test.beforeEach(async ({ page }) => {
		// Accessibility readiness does not depend on the external release lookup.
		await page.route('**/api/version-check', (route) => route.fulfill({ json: { latest: '' } }));
	});

	// A public Facet with a CTA and enough technologies to render the overflow
	// badge, so these checks do not depend on what the site happens to contain.
	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		const headers = { Authorization: token };

		const project = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
			headers,
			data: {
				title: projectTitle,
				slug: `accessibility-project-${suffix}`,
				summary: 'Accessibility fixture.',
				description: 'Accessibility fixture.',
				tech_stack: ['Svelte', 'TypeScript', 'Go', 'SQLite', 'Docker', 'Playwright', 'Axe'],
				visibility: 'public',
				is_draft: false
			}
		});
		expect(project.ok(), await project.text()).toBe(true);
		const projectId = (await project.json()).id as string;
		created.push({ collection: 'projects', id: projectId });

		const view = await request.post(`${apiBaseURL}/api/collections/views/records`, {
			headers,
			data: {
				name: `Accessibility Facet ${suffix}`,
				slug: viewSlug,
				visibility: 'public',
				is_active: true,
				cta_text: 'Accessibility fixture',
				cta_url: 'https://example.com',
				sections: [{ section: 'projects', enabled: true, layout: 'grid-2', items: [projectId] }]
			}
		});
		expect(view.ok(), await view.text()).toBe(true);
		viewId = (await view.json()).id as string;
		created.push({ collection: 'views', id: viewId });

		const document = await request.post(`${apiBaseURL}/api/collections/posts/records`, {
			headers,
			multipart: {
				title: `Accessibility document ${suffix}`,
				slug: `accessibility-document-${suffix}`,
				visibility: 'private',
				downloadable_files: {
					name: `accessibility-document-${suffix}.pdf`,
					mimeType: 'application/pdf',
					buffer: Buffer.from('%PDF-1.4\n% Accessibility fixture\n%%EOF')
				}
			}
		});
		expect(document.ok(), await document.text()).toBe(true);
		created.push({ collection: 'posts', id: (await document.json()).id });

		const upload = await request.post(`${apiBaseURL}/api/collections/uploads/records`, {
			headers,
			multipart: {
				title: mediaTitle,
				mime: 'image/png',
				file: {
					name: `accessibility-image-${suffix}.png`,
					mimeType: 'image/png',
					buffer: Buffer.from(
						'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
						'base64'
					)
				}
			}
		});
		expect(upload.ok(), await upload.text()).toBe(true);
		created.push({ collection: 'uploads', id: (await upload.json()).id });
	});

	test.afterAll(async ({ request }) => {
		for (const { collection, id } of [...created].reverse()) {
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
		}
	});

	for (const { name, theme, viewport } of MODES) {
		test.describe(name, () => {
			test.use({ viewport });

			test('public homepage', async ({ page }) => {
				await page.goto('/');
				await expectNoViolations(page, theme);
			});

			test('public Facet', async ({ page }) => {
				await page.goto(`/${viewSlug}`);
				await expect(page.getByText(projectTitle).first()).toBeVisible();
				await expectNoViolations(page, theme);
			});

			for (const route of ADMIN_ROUTES) {
				test(route, async ({ page, request }) => {
					await seedAdminSession(page, request);
					await page.goto(route);
					const main = page.getByRole('main');
					if (route === '/admin/settings/payments') {
						await expect(main.locator('fieldset')).toBeEnabled();
					}
					const titles: Record<string, string> = {
						'/admin/settings/integrations': 'Integrations',
						'/admin/settings/payments': 'Payment Settings',
						'/admin/testimonials/requests': 'Request Links'
					};
					if (titles[route]) {
						await expect(main.getByRole('heading', { level: 1 })).toHaveCount(1);
						await expect(
							main.getByRole('heading', { level: 1, name: titles[route], exact: true })
						).toBeVisible();
					}
					if (route === '/admin/media') {
						await expect(
							page.getByRole('link', { name: `Accessibility document ${suffix}`, exact: true })
						).toBeVisible();
						await expect(main.getByRole('link', { name: mediaTitle, exact: true })).toBeVisible();
						const search = main.getByRole('textbox', { name: 'Search', exact: true });
						await expect(search).toHaveAttribute('aria-label', 'Search');
						await search.fill(mediaTitle);
						await search.press('Enter');
						await expect(main.getByRole('link', { name: mediaTitle, exact: true })).toBeVisible();
						await expect(
							main.getByRole('link', { name: `Accessibility document ${suffix}`, exact: true })
						).toHaveCount(0);
					}
					await expectNoViolations(page, theme);
				});
			}

			test('/admin/views/:id', async ({ page, request }) => {
				await seedAdminSession(page, request);
				await page.goto(`/admin/views/${viewId}`);
				await expectNoViolations(page, theme);
			});
		});
	}
});

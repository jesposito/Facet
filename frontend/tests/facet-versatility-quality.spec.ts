import { randomUUID } from 'node:crypto';
import { test, expect, type APIRequestContext, type Page, type Route } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import {
	createPublicProjectAndPost,
	deletePublicProjectAndPost,
	loginAsAdmin,
	seedAdminSession
} from './helpers';

const VIEWPORTS = [
	{ width: 390, height: 844 },
	{ width: 768, height: 1024 },
	{ width: 1440, height: 900 }
];

const GUIDANCE_HEADING = 'One Facet, many uses';
const ERROR_TITLE = "Couldn't load all of your dashboard";
const WELCOME_TITLE = 'This is your space.';
const EMPTY_LIST = { page: 1, perPage: 100, totalItems: 0, totalPages: 0, items: [] };

type Fixture = { projectId: string; postId: string; viewId: string; name: string; slug: string };

function records(name: string) {
	return new RegExp(`/api/collections/(demo_)?${name}/records(\\?|$)`);
}

// In dev the SDK calls the backend on another port, so mocks need the CORS header.
function fulfillJson(route: Route, body: unknown, status = 200) {
	return route.fulfill({
		status,
		contentType: 'application/json',
		headers: { 'access-control-allow-origin': '*' },
		body: JSON.stringify(body)
	});
}

// The sidebar reads views too; only the dashboard asks for them by view count.
function isDashboardRead(route: Route, collection: string) {
	const request = route.request();
	if (request.method() !== 'GET') return false;
	if (collection !== 'views') return true;
	return new URL(request.url()).searchParams.get('sort') === '-view_count';
}

async function createFacet(request: APIRequestContext, token: string, name: string, slug: string) {
	const response = await request.post(`${apiBaseURL}/api/collections/views/records`, {
		headers: { Authorization: token },
		data: { name, slug, visibility: 'public', is_active: true }
	});
	expect(response.ok(), await response.text()).toBeTruthy();
	return (await response.json()).id as string;
}

async function deleteFacet(request: APIRequestContext, token: string, id: string) {
	await request.delete(`${apiBaseURL}/api/collections/views/records/${id}`, {
		headers: { Authorization: token }
	});
}

async function mockNoContent(page: Page) {
	for (const name of ['projects', 'experience', 'views']) {
		await page.route(records(name), (route) =>
			route.request().method() === 'GET' ? fulfillJson(route, EMPTY_LIST) : route.continue()
		);
	}
}

function guidance(page: Page) {
	return page.getByRole('region', { name: GUIDANCE_HEADING });
}

// Match the actual lazy TipTap core module by its package error marker, not a build hash.
async function holdRichTextImport(page: Page, fail: boolean) {
	let release!: () => void;
	const released = new Promise<void>((resolve) => (release = resolve));
	let held = false;
	let completed = false;
	let moduleURL = '';
	await page.route(/\.js(?:\?|$)/, async (route) => {
		const response = await route.fetch();
		const body = await response.text();
		if (!body.includes('[tiptap error]: Invalid JSON content')) return route.fulfill({ response });
		held = true;
		moduleURL = route.request().url();
		await released;
		if (fail) {
			await route.fulfill({ status: 503, contentType: 'application/javascript', body: '' });
		} else {
			await route.fulfill({ response });
		}
		completed = true;
	});
	return { release, held: () => held, completed: () => completed, moduleURL: () => moduleURL };
}

async function expectAnchorRevealed(page: Page, id: string, title: RegExp) {
	const section = page.locator(`#${id}`);
	const trigger = section.getByRole('button', { name: title });
	await expect(trigger).toHaveAttribute('aria-expanded', 'true');
	await expect(trigger).toBeFocused();
	await expect(trigger).toBeInViewport({ ratio: 1 });
	await expect(section).toBeInViewport();
	const [target, controls] = await Promise.all([
		trigger.boundingBox(),
		page.locator('main .sticky').first().boundingBox()
	]);
	expect(target).not.toBeNull();
	expect(controls).not.toBeNull();
	expect(target!.y).toBeGreaterThanOrEqual(controls!.y + controls!.height);
	return trigger;
}

function metric(page: Page, label: string) {
	return page.locator('dl.stat-band > div').filter({ has: page.getByText(label, { exact: true }) });
}

// Scoped to main: the sidebar lists recent Facets too, and comes first in the DOM.
function facetLink(page: Page, name: string) {
	return page.getByRole('main').getByRole('link', { name }).first();
}

function createAction(page: Page) {
	return page.getByRole('main').getByRole('link', { name: 'Create a Facet' }).first();
}

async function expectGuidanceFits(page: Page, width: number) {
	const region = guidance(page);
	await expect(region).toBeVisible();
	await expect(region).toContainText(/Any Facet can be shared on its own|Share a Facet on its own/);
	const examples = region.locator('summary');
	if (await examples.count()) {
		expect((await region.boundingBox())!.height).toBeLessThan(240);
		await examples.click();
	}
	await expect(region).toContainText('Digital resume for a recruiter');
	await expect(region).toContainText('Client proposal');
	await expect(region).toContainText('Event or resource page');
	await expect(region).toContainText('Project page');
	await expect(region).toContainText('About page');
	await expect(region).toContainText('Writing page');

	// The admin shell clips overflow, so the page width alone would hide a bad layout.
	const box = await region.boundingBox();
	expect(box).not.toBeNull();
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(width);
	const overflowing = await region.evaluate(
		(element) =>
			[element, ...element.querySelectorAll('*')].filter(
				(node) => node.clientWidth > 0 && node.scrollWidth > node.clientWidth + 1
			).length
	);
	expect(overflowing).toBe(0);
	const pageWidth = await page.evaluate(() => document.documentElement.scrollWidth);
	expect(pageWidth).toBeLessThanOrEqual(width);
}

test.describe('Facet versatility guidance and dashboard resilience', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	let token: string;
	let fixture: Fixture;

	test.beforeEach(async ({ page, request }) => {
		({ token } = await loginAsAdmin(request));
		const suffix = randomUUID().slice(0, 8);
		const name = `Versatility ${suffix}`;
		const slug = `versatility-${suffix}`;
		const content = await createPublicProjectAndPost(request, token);
		const viewId = await createFacet(request, token, name, slug);
		fixture = { ...content, viewId, name, slug };
		await seedAdminSession(page, request);
	});

	test.afterEach(async ({ request }) => {
		await deleteFacet(request, token, fixture.viewId);
		await deletePublicProjectAndPost(request, token, fixture);
	});

	for (const viewport of VIEWPORTS) {
		test(`guidance fits with no Facets at ${viewport.width}px`, async ({ page }) => {
			await page.setViewportSize(viewport);
			await mockNoContent(page);

			await page.goto('/admin/views');
			await expect(
				page.getByRole('heading', { name: "You haven't created any facets yet." })
			).toBeVisible();
			await expectGuidanceFits(page, viewport.width);
			await expect(guidance(page)).toContainText('The experience and skills that fit one role.');

			await page.goto('/admin');
			await expect(page.getByRole('heading', { level: 1, name: WELCOME_TITLE })).toBeVisible();
			await expectGuidanceFits(page, viewport.width);
			await expect(guidance(page).getByRole('link', { name: 'Create a Facet' })).toBeVisible();
		});

		test(`guidance fits alongside Facets at ${viewport.width}px`, async ({ page }) => {
			await page.setViewportSize(viewport);

			await page.goto('/admin/views');
			await expect(facetLink(page, fixture.name)).toBeAttached();
			await expectGuidanceFits(page, viewport.width);

			await page.goto('/admin');
			await expect(facetLink(page, fixture.name)).toBeVisible();
			await expectGuidanceFits(page, viewport.width);
		});
	}

	test('profile loading preserves Page Help focus after the actual profile read', async ({
		page
	}) => {
		let release!: () => void;
		const released = new Promise<void>((resolve) => (release = resolve));
		let held = false;
		await page.route(records('profile'), async (route) => {
			if (
				route.request().method() !== 'GET' ||
				new URL(route.request().url()).searchParams.get('perPage') !== '100'
			)
				return route.continue();
			const response = await route.fetch();
			expect(response.ok()).toBe(true);
			held = true;
			await released;
			await route.fulfill({ response });
		});
		try {
			await page.goto('/admin/homepage#nav-section');
			await expect.poll(() => held).toBe(true);
			await expect(page.getByText('Loading profile...', { exact: true })).toBeVisible();
			const help = page.getByRole('button', { name: 'Page Help', exact: true });
			await help.focus();
			release();
			await expect(
				page.getByRole('textbox', { name: 'Rich text editor', exact: true })
			).toBeVisible();
			await expect(page.getByRole('button', { name: 'Save Homepage', exact: true })).toBeEnabled();
			await expect(help).toBeFocused();
			await expect(
				page.locator('#nav-section').getByRole('button', { name: /^Site Navigation\b/ })
			).not.toBeFocused();
		} finally {
			release();
		}
	});

	for (const entry of ['fresh hash', 'guide click', 'profile retry'] as const) {
		test(`${entry} reveals and focuses navigation after the actual profile read`, async ({
			page
		}) => {
			await page.setViewportSize(
				entry === 'fresh hash' ? { width: 390, height: 844 } : { width: 1440, height: 900 }
			);
			await page.emulateMedia({
				reducedMotion: entry === 'fresh hash' ? 'reduce' : 'no-preference'
			});
			if (entry === 'guide click') {
				await page.goto('/admin/views');
				await guidance(page).locator('summary').click();
			}

			let releaseSupporting!: () => void;
			const supportingReleased = new Promise<void>((resolve) => (releaseSupporting = resolve));
			let supportingHeld = false;
			let supportingCompleted = false;
			await page.route(records('skills'), async (route) => {
				if (
					route.request().method() !== 'GET' ||
					new URL(route.request().url()).searchParams.get('perPage') !== '100'
				)
					return route.continue();
				const response = await route.fetch();
				expect(response.ok()).toBe(true);
				supportingHeld = true;
				await supportingReleased;
				await route.fulfill({ response });
				supportingCompleted = true;
			});

			let releaseProfile!: () => void;
			const released = new Promise<void>((resolve) => (releaseProfile = resolve));
			let profileHeld = false;
			let profileAttempts = 0;
			await page.route(records('profile'), async (route) => {
				if (
					route.request().method() !== 'GET' ||
					new URL(route.request().url()).searchParams.get('perPage') !== '100'
				)
					return route.continue();
				const response = await route.fetch();
				expect(response.ok()).toBe(true);
				profileHeld = true;
				const fail = entry === 'profile retry' && ++profileAttempts === 1;
				await released;
				if (fail) return fulfillJson(route, { error: 'Profile temporarily unavailable' }, 503);
				await route.fulfill({ response });
			});

			try {
				if (entry === 'guide click') {
					await guidance(page).locator('a[href="/admin/homepage#nav-section"]').click();
				} else {
					await page.goto(
						entry === 'profile retry'
							? '/admin/homepage#brand-section'
							: '/admin/homepage#nav-section'
					);
				}
				await expect.poll(() => profileHeld).toBe(true);
				await expect(page.getByText('Loading profile...', { exact: true })).toBeVisible();
				await expect(page.locator('#nav-section')).toHaveCount(0);
				releaseProfile();

				const section = page.locator(entry === 'profile retry' ? '#brand-section' : '#nav-section');
				const trigger = section.getByRole('button', {
					includeHidden: true,
					name: entry === 'profile retry' ? /^Brand\b/ : /^Site Navigation\b/
				});
				if (entry === 'profile retry') {
					const alert = page.getByRole('alert').filter({ hasText: "Couldn't load your profile" });
					await expect(alert).toBeVisible();
					await expect(section).not.toBeVisible();
					await expect(trigger).not.toBeFocused();
					await alert.getByRole('button', { name: 'Retry', exact: true }).click();
				}

				// These assertions precede any test action that could focus or scroll the target.
				await expect(page).toHaveURL(
					entry === 'profile retry'
						? /\/admin\/homepage#brand-section$/
						: /\/admin\/homepage#nav-section$/
				);
				await expect(trigger).toHaveAttribute('aria-expanded', 'true');
				await expect(trigger).toBeFocused();
				await expect(trigger).toBeInViewport({ ratio: 1 });
				await expect(section).toBeInViewport();
				const [target, controls] = await Promise.all([
					trigger.boundingBox(),
					page.locator('main .sticky').first().boundingBox()
				]);
				expect(target).not.toBeNull();
				expect(controls).not.toBeNull();
				expect(target!.y).toBeGreaterThanOrEqual(controls!.y + controls!.height);

				// Settling other reads must not repeat the deep-link focus or overwrite a draft.
				const name = page.locator('#name');
				await name.fill(`Unsaved navigation draft ${fixture.name}`);
				await expect.poll(() => supportingHeld).toBe(true);
				await expect(
					page.getByRole('button', { name: 'Save Homepage', exact: true })
				).toBeDisabled();
				releaseSupporting();
				await expect.poll(() => supportingCompleted).toBe(true);
				await expect(
					page.getByRole('button', { name: 'Save Homepage', exact: true })
				).toBeEnabled();
				await expect(name).toHaveValue(`Unsaved navigation draft ${fixture.name}`);
				await expect(name).toBeFocused();
			} finally {
				releaseProfile();
				releaseSupporting();
			}
		});
	}

	for (const outcome of [
		'initialized',
		'failed import and hints',
		'hint timeout',
		'import timeout',
		'editing',
		'header focus',
		'closed profile'
	] as const) {
		test(`late rich-text ${outcome} respects the requested anchor and user focus`, async ({
			page
		}) => {
			await page.setViewportSize({ width: 1440, height: 900 });
			if (outcome === 'hint timeout' || outcome === 'import timeout') await page.clock.install();
			await page.goto('/admin/views');
			await guidance(page).locator('summary').click();
			const editor = await holdRichTextImport(
				page,
				outcome === 'failed import and hints' || outcome === 'closed profile'
			);
			let releaseLayout!: () => void;
			const layoutReleased = new Promise<void>((resolve) => (releaseLayout = resolve));
			let layoutHeld = false;
			if (outcome === 'initialized') {
				await page.route('**/api/site-settings', async (route) => {
					if (route.request().method() !== 'GET') return route.continue();
					const response = await route.fetch();
					expect(response.ok()).toBe(true);
					layoutHeld = true;
					await layoutReleased;
					await route.fulfill({ response });
				});
			} else if (outcome === 'failed import and hints' || outcome === 'hint timeout') {
				await page.route(records('views'), async (route) => {
					const url = new URL(route.request().url());
					if (
						route.request().method() !== 'GET' ||
						url.searchParams.get('perPage') !== '100' ||
						url.searchParams.has('filter') ||
						url.searchParams.has('sort')
					)
						return route.continue();
					layoutHeld = true;
					await layoutReleased;
					if (outcome === 'hint timeout') {
						await route.continue().catch(() => undefined);
						return;
					}
					await fulfillJson(route, { error: 'Optional hints unavailable' }, 503);
				});
			}
			try {
				await guidance(page).locator('a[href="/admin/homepage#nav-section"]').click();
				await expect.poll(editor.held).toBe(true);
				const nav = page
					.locator('#nav-section')
					.getByRole('button', { name: /^Site Navigation\b/ });
				await expect(nav).toHaveAttribute('aria-expanded', 'true');
				await expect(nav).not.toBeFocused();
				const name = page.locator('#name');
				const draft = `Late editor draft ${fixture.name}`;
				if (outcome === 'import timeout') {
					await page.clock.fastForward(15_001);
					const markdown = page.getByRole('button', { name: /Switch to Markdown/ });
					await expect(markdown).toBeVisible();
					await expectAnchorRevealed(page, 'nav-section', /^Site Navigation\b/);
					editor.release();
					await expect.poll(editor.completed).toBe(true);
					await page.evaluate(async (url) => {
						await import(url);
					}, editor.moduleURL());
					await expect(
						page.getByRole('textbox', { name: 'Rich text editor', exact: true })
					).toHaveCount(0);
					await expectAnchorRevealed(page, 'nav-section', /^Site Navigation\b/);
					await markdown.click();
					const input = page.getByRole('textbox', { name: 'Rich text editor', exact: true });
					await input.fill(draft);
					await expect(input).toHaveValue(draft);
					await expect(input).toBeFocused();
					return;
				}
				if (outcome === 'editing' || outcome === 'closed profile') await name.fill(draft);
				const save = page.getByRole('button', { name: 'Save Homepage', exact: true });
				if (outcome === 'header focus') {
					await expect(save).toBeEnabled();
					await save.focus();
				}
				if (outcome === 'closed profile') {
					await page
						.locator('#profile-section')
						.getByRole('button', { name: /^Profile & Contact\b/ })
						.click();
					await page.evaluate(() => {
						const a = document.createElement('a');
						a.href = '#brand-section';
						a.textContent = 'Jump to Brand while editor is closed';
						a.style.cssText = 'position:fixed;bottom:0;right:0;z-index:1000;background:white';
						document.body.append(a);
					});
					await page
						.getByRole('link', { name: 'Jump to Brand while editor is closed', exact: true })
						.click();
					await expectAnchorRevealed(page, 'brand-section', /^Brand\b/);
				}
				editor.release();
				await expect.poll(editor.completed).toBe(true);
				if (outcome !== 'closed profile') {
					if (outcome === 'failed import and hints') {
						await expect(page.getByRole('button', { name: /Switch to Markdown/ })).toBeVisible();
					} else {
						await expect(
							page.getByRole('textbox', { name: 'Rich text editor', exact: true })
						).toBeVisible();
					}
				}
				if (outcome === 'hint timeout') {
					await expect.poll(() => layoutHeld).toBe(true);
					await expect(nav).not.toBeFocused();
					const aborted = page.waitForEvent('requestfailed', (request) => {
						const url = new URL(request.url());
						return (
							url.pathname.endsWith('/collections/views/records') &&
							url.searchParams.get('perPage') === '100' &&
							!url.searchParams.has('filter') &&
							!url.searchParams.has('sort') &&
							request.failure()?.errorText === 'net::ERR_ABORTED'
						);
					});
					await page.clock.fastForward(15_001);
					await aborted;
					await expectAnchorRevealed(page, 'nav-section', /^Site Navigation\b/);
				} else if (outcome === 'initialized' || outcome === 'failed import and hints') {
					await expect.poll(() => layoutHeld).toBe(true);
					await expect(nav).not.toBeFocused();
					releaseLayout();
					await page.evaluate(() => document.fonts.ready);
					await expectAnchorRevealed(page, 'nav-section', /^Site Navigation\b/);
				} else if (outcome === 'header focus') {
					await expect(save).toBeFocused();
					await expect(nav).not.toBeFocused();
				} else if (outcome === 'editing') {
					await expect(
						page.getByRole('button', { name: 'Save Homepage', exact: true })
					).toBeEnabled();
					await expect(name).toHaveValue(draft);
					await expect(name).toBeFocused();
					await expect(nav).not.toBeFocused();
				} else {
					await expectAnchorRevealed(page, 'brand-section', /^Brand\b/);
					const profile = page
						.locator('#profile-section')
						.getByRole('button', { name: /^Profile & Contact\b/ });
					await profile.click();
					await expect(page.getByRole('button', { name: /Switch to Markdown/ })).toBeVisible();
					await expect(name).toHaveValue(draft);
					await expect(profile).toBeFocused();
				}
			} finally {
				editor.release();
				releaseLayout();
			}
		});
	}

	test('retained Homepage anchors preserve an unsaved draft across all six sections', async ({
		page
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.emulateMedia({ reducedMotion: 'reduce' });
		let profileReads = 0;
		await page.route(records('profile'), (route) => {
			if (
				route.request().method() === 'GET' &&
				new URL(route.request().url()).searchParams.get('perPage') === '100'
			)
				profileReads++;
			return route.continue();
		});
		await page.goto('/admin/homepage');
		await expect(page.getByRole('button', { name: 'Save Homepage', exact: true })).toBeEnabled();
		const name = page.locator('#name');
		const draft = `Retained navigation draft ${fixture.name}`;
		await name.fill(draft);
		const originalReads = profileReads;
		expect(originalReads).toBeGreaterThan(0);
		const anchors = [
			{ id: 'profile-section', title: 'Profile & Contact' },
			{ id: 'brand-section', title: 'Brand' },
			{ id: 'hero-section', title: 'Hero Section' },
			{ id: 'nav-section', title: 'Site Navigation' },
			{ id: 'section-order', title: 'Section Order' },
			{ id: 'visibility-section', title: 'Visibility' }
		];
		await page.evaluate((anchors) => {
			const nav = document.createElement('nav');
			nav.setAttribute('aria-label', 'Deep-link test links');
			nav.style.cssText =
				'position:fixed;bottom:0;left:0;z-index:1000;background:white;display:flex;gap:8px';
			for (const { id, title } of anchors) {
				const link = document.createElement('a');
				link.href = `#${id}`;
				link.textContent = `Jump to ${title}`;
				nav.append(link);
			}
			document.body.append(nav);
		}, anchors);
		for (const { id, title } of anchors) {
			await page.getByRole('link', { name: `Jump to ${title}`, exact: true }).click();
			const section = page.locator(`#${id}`);
			const trigger = section.getByRole('button', { name: new RegExp(`^${title}\\b`) });
			await expect(page).toHaveURL(new RegExp(`#${id}$`));
			await expect(trigger).toHaveAttribute('aria-expanded', 'true');
			await expect(trigger).toBeFocused();
			await expect(trigger).toBeInViewport({ ratio: 1 });
			await expect(section).toBeInViewport();
			const [target, controls] = await Promise.all([
				trigger.boundingBox(),
				page.locator('main .sticky').first().boundingBox()
			]);
			expect(target).not.toBeNull();
			expect(controls).not.toBeNull();
			expect(target!.y).toBeGreaterThanOrEqual(controls!.y + controls!.height);
			await expect(name).toHaveValue(draft);
			expect(profileReads).toBe(originalReads);
		}
	});

	test('mobile owner work sits above the fold, below the backup warning', async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.route('**/api/system/encryption-status', (route) =>
			fulfillJson(route, { source: 'auto' })
		);

		await page.goto('/admin');

		await expect(page.getByRole('heading', { name: /Backup Reminder/ })).toBeVisible();
		const firstFacet = page
			.getByRole('region', { name: 'Facets', exact: true })
			.getByRole('listitem')
			.first();
		await expect(createAction(page)).toBeInViewport({ ratio: 1 });
		await expect(firstFacet).toBeInViewport({ ratio: 1 });
	});

	test('metrics are a valid two-column description list on mobile', async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto('/admin');

		const groups = page.locator('dl.stat-band > div');
		await expect(groups).toHaveCount(4);
		const shapes = await groups.evaluateAll((elements) =>
			elements.map((element) => [...element.children].map((child) => child.tagName))
		);
		expect(shapes).toEqual(Array(4).fill(['DT', 'DD']));

		const [first, second, third] = await Promise.all(
			[0, 1, 2].map((index) => groups.nth(index).boundingBox())
		);
		expect(second!.y).toBe(first!.y);
		expect(second!.x).toBeGreaterThan(first!.x);
		expect(third!.y).toBeGreaterThan(first!.y);
	});

	test('recent activity follows update times, not record IDs', async ({ page }) => {
		const recent = (id: string, title: string, updated: string) => ({
			id,
			title,
			company: 'Fixture Co',
			updated
		});
		let sort: string | null = null;
		await page.route(records('projects'), (route) => {
			if (!isDashboardRead(route, 'projects')) return route.continue();
			sort = new URL(route.request().url()).searchParams.get('sort');
			return fulfillJson(route, {
				...EMPTY_LIST,
				totalItems: 2,
				items: [
					recent('zzzzzzzzzzzzzzz', 'Oldest project', '2024-01-01 09:00:00.000Z'),
					recent('aaaaaaaaaaaaaaa', 'Newest project', '2026-06-01 09:00:00.000Z')
				]
			});
		});
		await page.route(records('experience'), (route) =>
			isDashboardRead(route, 'experience')
				? fulfillJson(route, {
						...EMPTY_LIST,
						totalItems: 1,
						items: [recent('mmmmmmmmmmmmmmm', 'Middle role', '2025-01-01 09:00:00.000Z')]
					})
				: route.continue()
		);

		await page.goto('/admin');

		const activity = page
			.locator('div.card')
			.filter({ has: page.getByRole('heading', { name: 'Recent Activity' }) })
			.getByRole('listitem');
		await expect(activity).toHaveCount(3);
		await expect(activity.nth(0)).toContainText('Newest project');
		await expect(activity.nth(1)).toContainText('Middle role');
		await expect(activity.nth(2)).toContainText('Oldest project');
		expect(sort).toBe('-updated');
	});

	test('create, edit, search and delete stay available with no category gate', async ({
		page,
		request
	}) => {
		const disposableName = `Disposable ${randomUUID().slice(0, 8)}`;
		const disposableId = await createFacet(
			request,
			token,
			disposableName,
			disposableName.toLowerCase().replace(' ', '-')
		);

		try {
			await page.setViewportSize({ width: 1440, height: 900 });

			await page.goto('/admin');
			await createAction(page).click();
			await expect(page).toHaveURL(/\/admin\/views\/new$/);
			await expect(page.getByRole('heading', { name: 'Create Facet' })).toBeVisible();
			await expect(page.locator('#name')).toBeEditable();
			await expect(page.locator('#slug')).toBeEditable();

			await page.goto('/admin/views');
			await expect(
				page.getByRole('main').getByRole('link', { name: '+ New Facet' })
			).toHaveAttribute('href', '/admin/views/new');

			const search = page.locator('#views-search');
			await search.fill(fixture.slug);
			await expect(page.getByRole('status').filter({ hasText: /^1 of/ })).toBeVisible();
			const row = page.locator('tr', { hasText: fixture.name });
			await expect(row).toBeVisible();

			await search.fill(disposableName);
			const disposableRow = page.locator('tr', { hasText: disposableName });
			await disposableRow.getByRole('button', { name: 'Delete' }).click();
			await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();
			await expect(disposableRow).toHaveCount(0);

			await search.fill(fixture.slug);
			await row.getByRole('link', { name: 'Edit' }).click();
			await expect(page).toHaveURL(new RegExp(`/admin/views/${fixture.viewId}$`));
		} finally {
			await deleteFacet(request, token, disposableId);
		}
	});

	for (const failing of ['projects', 'views']) {
		test(`a ${failing} 503 shows Retry, keeps known data and recovers`, async ({ page }) => {
			let unavailable = true;
			let failedRequests = 0;
			await page.route(records(failing), (route) => {
				if (!unavailable || !isDashboardRead(route, failing)) return route.continue();
				failedRequests += 1;
				return fulfillJson(route, { status: 503, message: 'Unavailable' }, 503);
			});

			await page.goto('/admin');

			const alert = page.getByRole('alert').filter({ hasText: ERROR_TITLE });
			await expect(alert).toBeVisible();
			await expect(page.getByRole('heading', { name: WELCOME_TITLE })).toHaveCount(0);

			const unknownMetric = metric(page, failing === 'projects' ? 'Projects' : 'Total Visitors');
			await expect(unknownMetric.getByRole('definition')).toContainText('Not available');
			await expect(unknownMetric.getByRole('definition')).not.toContainText('0');

			// Exact, so this only matches once the button has left "Retrying...".
			const retry = alert.getByRole('button', { name: 'Retry', exact: true });

			// Still failing: what loaded the first time has to survive this retry.
			const failedAttempts = failedRequests;
			await retry.click();
			await expect.poll(() => failedRequests).toBeGreaterThan(failedAttempts);
			await expect(retry).toBeVisible();
			if (failing === 'projects') {
				await expect(facetLink(page, fixture.name)).toBeVisible();
			} else {
				await expect(metric(page, 'Projects').getByRole('definition')).not.toContainText(
					'Not available'
				);
			}

			unavailable = false;
			await retry.click();

			await expect(alert).toHaveCount(0);
			await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeFocused();
			await expect(facetLink(page, fixture.name)).toBeVisible();
			await expect(metric(page, 'Projects').getByRole('definition')).not.toContainText(
				'Not available'
			);
			await expect(metric(page, 'Total Visitors').getByRole('definition')).not.toContainText(
				'Not available'
			);
		});
	}

	test('a failed site settings read is an error, not zero visitors', async ({ page }) => {
		await page.route('**/api/site-settings', (route) =>
			route.request().method() === 'GET'
				? route.fulfill({ status: 503, body: 'Unavailable' })
				: route.continue()
		);

		await page.goto('/admin');

		await expect(page.getByRole('alert').filter({ hasText: ERROR_TITLE })).toBeVisible();
		await expect(metric(page, 'Total Visitors').getByRole('definition')).toContainText(
			'Not available'
		);
	});

	test('layout and dashboard reads do not cancel each other', async ({ page }) => {
		const cancelled: string[] = [];
		page.on('requestfailed', (request) => {
			const url = request.url();
			// perPage=4 is the sidebar, which debounces and cancels its own reloads.
			if (records('(projects|experience|views)').test(url) && !url.includes('perPage=4')) {
				cancelled.push(url);
			}
		});
		// Runs after the session seed, so the layout's wizard check reads views as well.
		await page.addInitScript(() => localStorage.removeItem('facet_setup_wizard'));

		await page.goto('/admin');

		await expect(facetLink(page, fixture.name)).toBeVisible();
		await expect(page.getByRole('alert').filter({ hasText: ERROR_TITLE })).toHaveCount(0);
		expect(cancelled).toEqual([]);
	});

	test('the Facet list shows a recoverable failure instead of an empty collection', async ({
		page
	}) => {
		let unavailable = true;
		await page.route(records('views'), (route) =>
			unavailable && route.request().method() === 'GET'
				? fulfillJson(route, { message: 'Unavailable' }, 503)
				: route.continue()
		);
		await page.goto('/admin/views');
		const alert = page.getByRole('alert').filter({ hasText: 'Failed to load facets' });
		await expect(alert).toBeVisible();
		await expect(
			page.getByRole('heading', { name: "You haven't created any facets yet." })
		).toHaveCount(0);
		await expect(page.getByRole('heading', { name: 'No matching facets' })).toHaveCount(0);
		unavailable = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(facetLink(page, fixture.name)).toBeVisible();
	});

	test('a stalled dashboard request times out into a retryable error', async ({ page }) => {
		await page.clock.install();
		let stalled = true;
		const held: Route[] = [];
		await page.route(records('projects'), (route) => {
			if (!stalled || !isDashboardRead(route, 'projects')) return route.continue();
			held.push(route);
		});
		const started = page.waitForRequest(records('projects'));
		await page.goto('/admin');
		await started;
		await page.clock.fastForward(21_000);
		const alert = page.getByRole('alert').filter({ hasText: ERROR_TITLE });
		await expect(alert).toBeVisible();
		stalled = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(facetLink(page, fixture.name)).toBeVisible();
		await Promise.all(held.map((route) => route.continue().catch(() => undefined)));
	});

	test('leaving during a load cancels it without an error', async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		const dashboardErrors: string[] = [];
		page.on('console', (message) => {
			if (message.type() === 'error' && message.text().includes('dashboard')) {
				dashboardErrors.push(message.text());
			}
		});
		const held: Route[] = [];
		await page.route(records('projects'), (route) => {
			if (!isDashboardRead(route, 'projects')) return route.continue();
			held.push(route);
		});

		const projectsRequested = page.waitForRequest(records('projects'));
		await page.goto('/admin');
		await projectsRequested;

		await page.locator('a[href="/admin/views"]').first().click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		await expect(page.getByRole('heading', { level: 1, name: 'Facets' })).toBeVisible();

		await Promise.all(held.map((route) => route.continue().catch(() => undefined)));
		await expect(page.getByRole('alert').filter({ hasText: ERROR_TITLE })).toHaveCount(0);
		expect(dashboardErrors).toEqual([]);
	});
});

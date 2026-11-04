import AxeBuilder from '@axe-core/playwright';
import { expect, test, type APIRequestContext, type Page, type Route } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

type NavItem = { viewId: string; enabled: boolean; label: string };
type Facet = { id: string; name: string; slug: string };

const WCAG_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];
const MODES = [
	{ name: 'desktop light', theme: 'light', viewport: { width: 1440, height: 900 } },
	{ name: 'mobile dark', theme: 'dark', viewport: { width: 390, height: 844 } }
] as const;
const CATALOG_FILTER = 'is_active = true && visibility = "public"';
const READ_FAULTS = [
	{
		name: 'HTTP 503',
		respond: (route: Route) => json(route, { error: 'temporarily unavailable' }, 503)
	},
	{
		name: 'malformed JSON',
		respond: (route: Route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: '{not-json' })
	}
];

function json(route: Route, body: unknown, status = 200) {
	return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

const isSettings = (url: URL) => url.pathname === '/api/site-settings';
const isCatalog = (url: URL) =>
	url.pathname === '/api/collections/views/records' &&
	url.searchParams.get('filter') === CATALOG_FILTER;
const showInNav = (page: Page, name: string) =>
	page.getByRole('checkbox', { name: `Show ${name} in navigation`, exact: true });
const labelInput = (page: Page, name: string) =>
	page.getByRole('textbox', { name: `Button label for ${name}`, exact: true });
const saveHomepage = (page: Page) =>
	page.getByRole('button', { name: 'Save Homepage', exact: true });

function settingsPut(page: Page) {
	return page.waitForResponse(
		(response) => isSettings(new URL(response.url())) && response.request().method() === 'PUT'
	);
}

async function openNav(page: Page) {
	await page.goto('/admin/homepage#nav-section');
	await expect(
		page.locator('#nav-section').getByRole('button', { name: /^Site Navigation\b/ })
	).toHaveAttribute('aria-expanded', 'true');
}

// Uses the real toggle, then waits for colour transitions so axe sees settled colours.
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

test.describe('site navigation', () => {
	test.describe.configure({ mode: 'serial' });

	const suffix = randomUUID().slice(0, 8);
	const created: string[] = [];
	const createdLinks: string[] = [];
	let token: string;
	let original: Record<string, unknown>;
	let sequence = 0;

	async function readSettings(request: APIRequestContext) {
		const response = await request.get(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token }
		});
		expect(response.ok(), await response.text()).toBe(true);
		return (await response.json()) as Record<string, any>;
	}

	async function writeSettings(request: APIRequestContext, data: Record<string, unknown>) {
		const response = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data
		});
		expect(response.ok(), await response.text()).toBe(true);
	}

	async function savedItems(request: APIRequestContext, facets: Facet[]) {
		const ids = new Set(facets.map((facet) => facet.id));
		const items = ((await readSettings(request)).site_nav_items ?? []) as NavItem[];
		return items.filter((item) => ids.has(item.viewId));
	}

	async function createFacet(
		request: APIRequestContext,
		visibility: 'public' | 'unlisted' = 'public'
	): Promise<Facet> {
		const number = String(++sequence).padStart(3, '0');
		const name = `zzzz Site nav ${suffix} ${number}`;
		const slug = `site-nav-${suffix}-${number}`;
		const response = await request.post(`${apiBaseURL}/api/collections/views/records`, {
			headers: { Authorization: token },
			data: { name, slug, visibility, is_active: true, sections: [] }
		});
		expect(response.ok(), await response.text()).toBe(true);
		const id = (await response.json()).id as string;
		created.push(id);
		return { id, name, slug };
	}

	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		const settings = await readSettings(request);
		original = {
			site_nav_enabled: settings.site_nav_enabled === true,
			site_nav_mode: settings.site_nav_mode || 'bar',
			site_nav_position: settings.site_nav_position || 'below',
			site_cta_enabled: settings.site_cta_enabled !== false,
			footer_cta_enabled: settings.footer_cta_enabled !== false,
			site_nav_items: settings.site_nav_items ?? [],
			homepage_section_order: settings.homepage_section_order ?? [],
			homepage_sections: settings.homepage_sections ?? {},
			homepage_custom_content: settings.homepage_custom_content ?? []
		};
	});

	test.beforeEach(async ({ request }) => {
		await writeSettings(request, { site_nav_enabled: true, site_nav_items: [] });
	});

	test.afterEach(async ({ request }) => {
		await writeSettings(request, original);
		for (const id of createdLinks.splice(0).reverse()) {
			const response = await request.delete(
				`${apiBaseURL}/api/collections/share_tokens/records/${id}`,
				{
					headers: { Authorization: token }
				}
			);
			expect(response.ok(), await response.text()).toBe(true);
		}
		for (const id of created.splice(0).reverse()) {
			const response = await request.delete(`${apiBaseURL}/api/collections/views/records/${id}`, {
				headers: { Authorization: token }
			});
			expect(response.ok(), await response.text()).toBe(true);
		}
	});

	for (const { name, theme, viewport } of MODES) {
		test.describe(name, () => {
			test.use({ viewport });

			test('enabled navigation controls pass axe with names from the Facet', async ({
				page,
				request
			}) => {
				await seedAdminSession(page, request);
				const facet = await createFacet(request);
				await writeSettings(request, {
					site_nav_items: [{ viewId: facet.id, enabled: true, label: '' }]
				});
				await openNav(page);
				await expect(showInNav(page, facet.name)).toBeChecked();
				await expect(labelInput(page, facet.name)).toBeVisible();
				await expect(page.locator('.animate-spin')).toHaveCount(0);
				await settleTheme(page, theme);

				const { violations } = await new AxeBuilder({ page }).withTags(WCAG_TAGS).analyze();
				expect(
					violations.map(({ id, nodes }) => ({ id, targets: nodes.map((n) => n.target) }))
				).toEqual([]);
			});
		});
	}

	test('a Facet switch works by keyboard and pointer', async ({ page, request }) => {
		await seedAdminSession(page, request);
		const facet = await createFacet(request);
		await openNav(page);
		const toggle = showInNav(page, facet.name);
		await expect(toggle).not.toBeChecked();

		await toggle.focus();
		const keyboardSave = settingsPut(page);
		await page.keyboard.press('Space');
		expect((await keyboardSave).ok()).toBe(true);
		await expect(toggle).toBeChecked();

		const pointerSave = settingsPut(page);
		await toggle.click();
		expect((await pointerSave).ok()).toBe(true);
		await expect(toggle).not.toBeChecked();
	});

	test('a shared Facet joins navigation only by choice and keeps its link after leaving', async ({
		page,
		request,
		browser
	}) => {
		await seedAdminSession(page, request);
		const facet = await createFacet(request, 'unlisted');
		const generated = await request.post(`${apiBaseURL}/api/share/generate`, {
			headers: { Authorization: token },
			data: { view_id: facet.id, name: 'site-navigation-quality', max_uses: 0 }
		});
		expect(generated.ok(), await generated.text()).toBe(true);
		const generatedLink = await generated.json();
		createdLinks.push(generatedLink.id);
		const shareToken = generatedLink.token as string;

		const visitorContext = await browser.newContext();
		try {
			const visitor = await visitorContext.newPage();
			expect((await visitor.goto(`/s/${shareToken}`))?.ok()).toBe(true);
			await expect(visitor).toHaveURL(new RegExp(`/${facet.slug}$`));

			await page.goto(`/admin/views/${facet.id}`);
			await page.locator('#visibility').selectOption('public');
			await page.getByRole('button', { name: 'Save', exact: true }).click();
			await expect
				.poll(async () => {
					const response = await request.get(
						`${apiBaseURL}/api/collections/views/records/${facet.id}`,
						{ headers: { Authorization: token } }
					);
					return (await response.json()).visibility;
				})
				.toBe('public');

			const siteNav = visitor.getByRole('navigation', { name: 'Site navigation', exact: true });
			await visitor.goto('/');
			await expect(visitor.getByRole('link', { name: facet.name, exact: true })).toHaveCount(0);

			await openNav(page);
			const toggle = showInNav(page, facet.name);
			await expect(toggle).not.toBeChecked();
			const joined = settingsPut(page);
			await toggle.click();
			expect((await joined).ok()).toBe(true);

			await visitor.goto('/');
			await siteNav.getByRole('link', { name: facet.name, exact: true }).click();
			await expect(visitor).toHaveURL(new RegExp(`/${facet.slug}$`));

			const left = settingsPut(page);
			await toggle.click();
			expect((await left).ok()).toBe(true);

			await visitor.goto('/');
			await expect(visitor.getByRole('link', { name: facet.name, exact: true })).toHaveCount(0);
			expect((await visitor.goto(`/${facet.slug}`))?.ok()).toBe(true);
		} finally {
			await visitorContext.close();
		}
	});

	for (const fault of READ_FAULTS) {
		test(`catalog ${fault.name} is not an empty list and Retry keeps drafts`, async ({
			page,
			request
		}) => {
			await seedAdminSession(page, request);
			const facet = await createFacet(request);
			const saved = [{ viewId: facet.id, enabled: true, label: 'Saved label' }];
			await writeSettings(request, { site_nav_items: saved });
			let broken = true;
			await page.route(isCatalog, (route) => (broken ? fault.respond(route) : route.fallback()));

			await openNav(page);
			const alert = page.getByRole('alert').filter({ hasText: "Couldn't load your public Facets" });
			await expect(alert).toBeVisible();
			await expect(page.getByText('No public facets available', { exact: false })).toHaveCount(0);
			const name = page.locator('#name');
			await expect(name).toBeEditable();
			await name.fill(`${await name.inputValue()} draft`);
			const draft = await name.inputValue();

			broken = false;
			await alert.getByRole('button', { name: 'Retry', exact: true }).click();
			await expect(alert).toHaveCount(0);
			await expect(labelInput(page, facet.name)).toHaveValue('Saved label');
			await expect(showInNav(page, facet.name)).toBeChecked();
			await expect(name).toHaveValue(draft);
			expect(await savedItems(request, [facet])).toEqual(saved);
		});

		test(`settings ${fault.name} pauses every write and Retry keeps drafts`, async ({
			page,
			request
		}) => {
			await seedAdminSession(page, request);
			const writes: string[] = [];
			let broken = true;
			await page.route(isSettings, (route) => {
				if (route.request().method() !== 'GET') {
					writes.push(route.request().method());
					return route.fallback();
				}
				return broken ? fault.respond(route) : route.fallback();
			});

			await openNav(page);
			const alert = page
				.getByRole('alert')
				.filter({ hasText: "Couldn't load your homepage settings" });
			await expect(alert).toBeVisible();
			await expect(saveHomepage(page)).toBeDisabled();
			const toggle = page.getByRole('checkbox', { name: 'Enable site navigation' });
			await expect(toggle).toBeDisabled();
			await toggle.dispatchEvent('change');
			const name = page.locator('#name');
			await expect(name).toBeEditable();
			await name.fill(`${await name.inputValue()} draft`);
			const draft = await name.inputValue();
			expect(writes).toEqual([]);

			broken = false;
			await alert.getByRole('button', { name: 'Retry', exact: true }).click();
			await expect(alert).toHaveCount(0);
			await expect(toggle).toBeEnabled();
			await expect(toggle).toBeChecked();
			await expect(saveHomepage(page)).toBeEnabled();
			await expect(name).toHaveValue(draft);
			expect(writes).toEqual([]);
		});
	}

	test('the 101st public Facet can be chosen', async ({ page, request }) => {
		test.setTimeout(180_000);
		await seedAdminSession(page, request);
		let last = await createFacet(request);
		for (let count = 1; count < 101; count++) last = await createFacet(request);

		await openNav(page);
		const toggle = showInNav(page, last.name);
		await expect(toggle).toBeVisible();
		const saved = settingsPut(page);
		await toggle.click();
		expect((await saved).ok()).toBe(true);
		expect(await savedItems(request, [last])).toEqual([
			{ viewId: last.id, enabled: true, label: '' }
		]);
	});

	test('a failed held write cannot undo a newer toggle or label', async ({ page, request }) => {
		await seedAdminSession(page, request);
		const first = await createFacet(request);
		const second = await createFacet(request);
		let puts = 0;
		let release!: () => void;
		const held = new Promise<void>((resolve) => {
			release = resolve;
		});
		await page.route(isSettings, async (route) => {
			if (route.request().method() !== 'PUT') return route.fallback();
			if (++puts > 1) return route.fallback();
			await held;
			return json(route, { error: 'Settings unavailable' }, 500);
		});

		await openNav(page);
		await showInNav(page, first.name).click();
		await expect.poll(() => puts).toBe(1);
		await labelInput(page, second.name).fill('Newer label');
		await showInNav(page, second.name).click();
		expect(puts).toBe(1);

		const failed = settingsPut(page);
		release();
		expect((await failed).status()).toBe(500);
		await expect.poll(() => puts).toBe(2);
		await expect
			.poll(() => savedItems(request, [first, second]))
			.toEqual([
				{ viewId: first.id, enabled: true, label: '' },
				{ viewId: second.id, enabled: true, label: 'Newer label' }
			]);
		await expect(showInNav(page, first.name)).toBeChecked();
		await expect(showInNav(page, second.name)).toBeChecked();
		await expect(labelInput(page, second.name)).toHaveValue('Newer label');
		await expect(page.getByRole('alert').filter({ hasText: "Couldn't save" })).toHaveCount(0);
	});

	test('a failed toggle keeps the typed label and Retry saves both', async ({ page, request }) => {
		await seedAdminSession(page, request);
		const facet = await createFacet(request);
		let failing = true;
		await page.route(isSettings, (route) =>
			route.request().method() === 'PUT' && failing
				? json(route, { error: 'Settings unavailable' }, 500)
				: route.fallback()
		);

		await openNav(page);
		const toggle = showInNav(page, facet.name);
		await labelInput(page, facet.name).fill('Kept label');
		await toggle.click();
		const alert = page.getByRole('alert').filter({ hasText: "Couldn't save" });
		await expect(alert).toBeVisible();
		await expect(toggle).not.toBeChecked();
		await expect(labelInput(page, facet.name)).toHaveValue('Kept label');

		failing = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(alert).toHaveCount(0);
		await expect(toggle).toBeChecked();
		await expect
			.poll(() => savedItems(request, [facet]))
			.toEqual([{ viewId: facet.id, enabled: true, label: 'Kept label' }]);
	});

	test('Save Homepage waits for a held nav write and sends the newest nav state', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const facet = await createFacet(request);
		const headers = { Authorization: token };
		const profiles = await request.get(`${apiBaseURL}/api/collections/profile/records?perPage=1`, {
			headers
		});
		const profile = (await profiles.json()).items[0] as Record<string, unknown>;
		const bodies: Array<Record<string, any>> = [];
		let release!: () => void;
		const held = new Promise<void>((resolve) => {
			release = resolve;
		});
		await page.route(isSettings, async (route) => {
			if (route.request().method() !== 'PUT') return route.fallback();
			bodies.push(route.request().postDataJSON());
			if (bodies.length === 1) await held;
			return route.fallback();
		});

		try {
			await openNav(page);
			await showInNav(page, facet.name).click();
			await expect.poll(() => bodies.length).toBe(1);
			await labelInput(page, facet.name).fill('Saved by button');
			const profileSaved = page.waitForResponse(
				(response) =>
					response.url().includes('/api/collections/profile/records/') &&
					response.request().method() === 'PATCH'
			);
			await saveHomepage(page).click();
			expect((await profileSaved).ok()).toBe(true);
			expect(bodies).toHaveLength(1);

			release();
			await expect(page.getByText('Homepage saved successfully', { exact: true })).toBeVisible();
			expect(bodies).toHaveLength(2);
			expect(bodies[1].site_nav_items).toContainEqual({
				viewId: facet.id,
				enabled: true,
				label: 'Saved by button'
			});
		} finally {
			release();
			const restored = await request.patch(
				`${apiBaseURL}/api/collections/profile/records/${profile.id}`,
				{
					headers,
					data: {
						hero_layout: profile.hero_layout,
						hero_spacing: profile.hero_spacing,
						hero_bg_color: profile.hero_bg_color
					}
				}
			);
			expect(restored.ok(), await restored.text()).toBe(true);
		}
	});

	test('keyboard reorder persists from the buttons and the grabbed row', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const facets = [
			await createFacet(request),
			await createFacet(request),
			await createFacet(request)
		];
		const [a, b, c] = facets;
		await writeSettings(request, {
			site_nav_items: facets.map((facet) => ({ viewId: facet.id, enabled: true, label: '' }))
		});
		const savedOrder = async () => (await savedItems(request, facets)).map((item) => item.viewId);

		await openNav(page);
		const down = page.getByRole('button', { name: `Move ${a.name} down`, exact: true });
		await down.focus();
		const moved = settingsPut(page);
		await page.keyboard.press('Enter');
		expect((await moved).ok()).toBe(true);
		await expect(down).toBeFocused();
		expect(await savedOrder()).toEqual([b.id, a.id, c.id]);

		await page.getByRole('listitem', { name: c.name, exact: true }).focus();
		await page.keyboard.press('Space');
		const grabbed = settingsPut(page);
		await page.keyboard.press('ArrowUp');
		expect((await grabbed).ok()).toBe(true);
		await page.keyboard.press('Space');
		await expect.poll(savedOrder).toEqual([b.id, c.id, a.id]);

		await page.reload();
		await expect(
			page.getByRole('button', { name: `Move ${b.name} up`, exact: true })
		).toBeDisabled();
	});
});

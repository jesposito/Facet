import { expect, test, type APIRequestContext, type Page, type Route } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

type Fixture = { id: string };

const READ_FAULTS = [
	{
		name: 'HTTP 503',
		respond: (route: Route) => json(route, { error: 'temporarily unavailable' }, 503)
	},
	{ name: 'a 200 with no list', respond: (route: Route) => json(route, {}) }
];
const CATALOG_TARGETS = ['custom_content', 'experience'];

function json(route: Route, body: unknown, status = 200) {
	return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

const isSettings = (url: URL) => url.pathname === '/api/site-settings';
const isProfile = (url: URL) => url.pathname.startsWith('/api/collections/profile/records');
const isList = (collection: string) => (url: URL) =>
	url.pathname === `/api/collections/${collection}/records`;
const isGet = (route: Route) => route.request().method() === 'GET';
const saveHomepage = (page: Page) =>
	page.getByRole('button', { name: 'Save Homepage', exact: true });
const alertWith = (page: Page, text: string) => page.getByRole('alert').filter({ hasText: text });
const projectsToggle = (page: Page) =>
	page.getByRole('button', { name: 'Toggle Projects section', exact: true });

// A disabled button drops real clicks, so this reaches the handler behind it.
function attemptSave(page: Page) {
	return saveHomepage(page).evaluate((button: HTMLButtonElement) => {
		button.disabled = false;
		button.click();
		button.disabled = true;
	});
}

// While on, this records and refuses writes, so a missing guard can't touch real data.
async function guardWrites(page: Page) {
	const guard = { writes: [] as string[], on: true };
	const refuse = (route: Route) => {
		if (isGet(route) || !guard.on) return route.fallback();
		guard.writes.push(`${route.request().method()} ${new URL(route.request().url()).pathname}`);
		return json(route, { error: 'Unexpected write' }, 500);
	};
	await page.route(isProfile, refuse);
	await page.route(isSettings, refuse);
	return guard;
}

// A valid raw list, so reaching a page past the old caps needs no real records.
function pagedList(
	route: Route,
	total: number,
	item: (position: number) => Record<string, unknown>
) {
	const url = new URL(route.request().url());
	const page = Number(url.searchParams.get('page'));
	const perPage = Number(url.searchParams.get('perPage'));
	const first = (page - 1) * perPage + 1;
	const last = Math.min(first + perPage - 1, total);
	const items = [];
	for (let position = first; position <= last; position++) items.push(item(position));
	const totalPages = Math.ceil(total / perPage);
	return json(route, { page, perPage, totalItems: total, totalPages, items });
}

test.describe('homepage required reads', () => {
	test.describe.configure({ mode: 'serial' });

	const suffix = randomUUID().slice(0, 8);
	const created: { collection: string; id: string }[] = [];
	let token: string;
	let profile: Record<string, any>;
	let original: Record<string, unknown>;

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

	async function createRecord(
		request: APIRequestContext,
		collection: string,
		data: Record<string, unknown>
	): Promise<Fixture> {
		const response = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
			headers: { Authorization: token },
			data
		});
		expect(response.ok(), await response.text()).toBe(true);
		const { id } = (await response.json()) as Fixture;
		created.push({ collection, id });
		return { id };
	}

	async function readProfiles(request: APIRequestContext) {
		const response = await request.get(`${apiBaseURL}/api/collections/profile/records?perPage=1`, {
			headers: { Authorization: token }
		});
		expect(response.ok(), await response.text()).toBe(true);
		return (await response.json()) as { totalItems: number; items: Record<string, any>[] };
	}

	// A public block, a private block and a private project pick, all saved on the homepage.
	async function seedSavedChoices(request: APIRequestContext) {
		const custom = { content: 'Homepage recovery fixture', is_draft: false };
		const shown = await createRecord(request, 'custom_content', {
			...custom,
			title: `Recovery shown ${suffix}`,
			visibility: 'public'
		});
		const hidden = await createRecord(request, 'custom_content', {
			...custom,
			title: `Recovery hidden ${suffix}`,
			visibility: 'private'
		});
		const pick = await createRecord(request, 'projects', {
			title: `Recovery pick ${suffix}`,
			slug: `homepage-recovery-${randomUUID()}`,
			summary: 'Homepage recovery fixture.',
			description: 'Homepage recovery fixture.',
			visibility: 'private',
			is_draft: false
		});
		await writeSettings(request, {
			homepage_section_order: ['projects', `custom:${shown.id}`, `custom:${hidden.id}`],
			homepage_sections: {
				projects: { enabled: true, items: [pick.id], layout: 'default', width: 'full' },
				[`custom:${hidden.id}`]: { enabled: true, items: [], layout: 'default', width: 'half' }
			},
			homepage_custom_content: [
				{ id: shown.id, enabled: true },
				{ id: hidden.id, enabled: true }
			]
		});
		return { shown, hidden, pick };
	}

	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		[profile] = (await readProfiles(request)).items;
		expect(profile?.name, 'these tests need a seeded profile with a name').toBeTruthy();
		const settings = await readSettings(request);
		original = {
			homepage_section_order: settings.homepage_section_order ?? [],
			homepage_sections: settings.homepage_sections ?? {},
			homepage_custom_content: settings.homepage_custom_content ?? []
		};
	});

	test.afterEach(async ({ request }) => {
		await writeSettings(request, original);
		const restored = await request.patch(
			`${apiBaseURL}/api/collections/profile/records/${profile.id}`,
			{
				headers: { Authorization: token },
				data: {
					name: profile.name,
					hero_layout: profile.hero_layout,
					hero_spacing: profile.hero_spacing,
					hero_bg_color: profile.hero_bg_color
				}
			}
		);
		expect(restored.ok(), await restored.text()).toBe(true);
		for (const { collection, id } of created.splice(0).reverse()) {
			const response = await request.delete(
				`${apiBaseURL}/api/collections/${collection}/records/${id}`,
				{ headers: { Authorization: token } }
			);
			expect(response.ok(), await response.text()).toBe(true);
		}
	});

	for (const fault of READ_FAULTS) {
		test(`profile ${fault.name} pauses the full save and Retry loads the real profile`, async ({
			page,
			request
		}) => {
			await seedAdminSession(page, request);
			const before = (await readProfiles(request)).totalItems;
			const guard = await guardWrites(page);
			let broken = true;
			let catalogReads = 0;
			await page.route(isList('custom_content'), (route) => {
				catalogReads++;
				return route.fallback();
			});
			await page.route(isProfile, (route) =>
				isGet(route) && broken ? fault.respond(route) : route.fallback()
			);

			await page.goto('/admin/homepage#nav-section');
			const alert = alertWith(page, "Couldn't load your profile");
			await expect(alert).toBeVisible();
			await expect(saveHomepage(page)).toBeDisabled();
			await expect(page.locator('#name')).toBeHidden();
			await expect(page.getByRole('checkbox', { name: 'Enable site navigation' })).toBeEnabled();
			await page.getByRole('button', { name: /^Section Order\b/ }).click();
			await expect(projectsToggle(page)).toBeVisible();
			const reads = catalogReads;

			// Only the save guard stands between this hidden name and a second profile.
			await page.locator('#name').evaluate((input: HTMLInputElement) => {
				input.value = 'Second profile';
				input.dispatchEvent(new Event('input', { bubbles: true }));
			});
			await attemptSave(page);
			await page
				.locator('form:has(#name)')
				.evaluate((form: HTMLFormElement) => form.requestSubmit());

			broken = false;
			await alert.getByRole('button', { name: 'Retry', exact: true }).click();
			await expect(alert).toHaveCount(0);
			await expect(page.locator('#name')).toHaveValue(profile.name);
			await expect(page.locator('#name')).toBeEditable();
			await expect(saveHomepage(page)).toBeEnabled();
			expect(guard.writes).toEqual([]);
			expect(catalogReads).toBe(reads);
			expect((await readProfiles(request)).totalItems).toBe(before);
		});

		for (const collection of CATALOG_TARGETS) {
			test(`${collection} ${fault.name} hides sections and Retry keeps typed and saved choices`, async ({
				page,
				request
			}) => {
				await seedAdminSession(page, request);
				const { shown, hidden, pick } = await seedSavedChoices(request);
				const guard = await guardWrites(page);
				let broken = true;
				let profileReads = 0;
				await page.route(isProfile, (route) => {
					if (isGet(route)) profileReads++;
					return route.fallback();
				});
				await page.route(isList(collection), (route) =>
					isGet(route) && broken ? fault.respond(route) : route.fallback()
				);

				await page.goto('/admin/homepage#section-order');
				const alert = alertWith(page, "Couldn't load your homepage sections");
				await expect(alert).toBeVisible();
				await expect(saveHomepage(page)).toBeDisabled();
				await expect(projectsToggle(page)).toHaveCount(0);
				await expect(page.getByText(/all items \(\d+\)|\d+ available/)).toHaveCount(0);

				const name = page.locator('#name');
				const draft = `${profile.name} draft`;
				await name.fill(draft);
				await attemptSave(page);
				const reads = profileReads;

				broken = false;
				await alert.getByRole('button', { name: 'Retry', exact: true }).click();
				await expect(alert).toHaveCount(0);
				await expect(projectsToggle(page)).toBeVisible();
				await expect(name).toHaveValue(draft);
				await expect(saveHomepage(page)).toBeEnabled();
				expect(guard.writes).toEqual([]);
				expect(profileReads).toBe(reads);

				guard.on = false;
				const put = page.waitForRequest(
					(sent) => isSettings(new URL(sent.url())) && sent.method() === 'PUT'
				);
				await saveHomepage(page).click();
				const body = (await put).postDataJSON();
				expect(body.homepage_custom_content).toEqual(
					expect.arrayContaining([
						{ id: shown.id, enabled: true },
						{ id: hidden.id, enabled: true }
					])
				);
				expect(body.homepage_sections.projects.items).toEqual([pick.id]);
				expect(body.homepage_sections[`custom:${hidden.id}`].width).toBe('half');
				expect(body.homepage_section_order).toContain(`custom:${hidden.id}`);
				await expect(page.getByText('Homepage saved successfully', { exact: true })).toBeVisible();
			});
		}
	}

	async function saveAndRead(page: Page, request: APIRequestContext) {
		const response = page.waitForResponse(
			(sent) => isSettings(new URL(sent.url())) && sent.request().method() === 'PUT'
		);
		await expect(saveHomepage(page)).toBeEnabled();
		await saveHomepage(page).click();
		expect((await response).ok()).toBe(true);
		await expect(saveHomepage(page)).toBeEnabled();
		return readSettings(request);
	}

	test('Select All and visible reorder keep hidden picks until Clear is chosen', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const { pick } = await seedSavedChoices(request);
		const firstTitle = `Recovery first ${suffix}`;
		const secondTitle = `Recovery second ${suffix}`;
		const first = await createRecord(request, 'projects', {
			title: firstTitle,
			slug: `homepage-first-${randomUUID()}`,
			visibility: 'public',
			is_draft: false
		});
		const second = await createRecord(request, 'projects', {
			title: secondTitle,
			slug: `homepage-second-${randomUUID()}`,
			visibility: 'public',
			is_draft: false
		});
		await page.route(isList('projects'), (route) =>
			json(route, {
				page: 1,
				perPage: 100,
				totalItems: 3,
				totalPages: 1,
				items: [
					{ id: first.id, title: firstTitle, visibility: 'public' },
					{ id: second.id, title: secondTitle, visibility: 'public' },
					{ id: pick.id, title: 'Hidden pick', visibility: 'private' }
				]
			})
		);
		const settings = await readSettings(request);
		await writeSettings(request, {
			homepage_sections: {
				...settings.homepage_sections,
				projects: { enabled: true, items: [pick.id, first.id], layout: 'default', width: 'full' }
			}
		});
		await page.goto('/admin/homepage#section-order');
		await page.getByRole('button', { name: 'Expand Projects section', exact: true }).click();
		const card = page
			.locator('#section-order .border.rounded-lg')
			.filter({ has: projectsToggle(page) });
		await expect(card.getByRole('checkbox', { name: firstTitle, exact: true })).toBeChecked();
		await card.getByRole('button', { name: 'Select All', exact: true }).click();
		let saved = await saveAndRead(page, request);
		expect(saved.homepage_sections.projects.items).toEqual([pick.id, first.id, second.id]);
		await card.getByRole('button', { name: `Move ${secondTitle} up`, exact: true }).click();
		saved = await saveAndRead(page, request);
		expect(saved.homepage_sections.projects.items).toEqual([second.id, first.id, pick.id]);
		await card.getByRole('checkbox', { name: firstTitle, exact: true }).uncheck();
		saved = await saveAndRead(page, request);
		expect(saved.homepage_sections.projects.items).toEqual([second.id, pick.id]);
		await card.getByRole('button', { name: 'Clear', exact: true }).click();
		saved = await saveAndRead(page, request);
		expect(saved.homepage_sections.projects.items ?? []).toEqual([]);
	});

	test('testimonial choice and skills settings survive Save and stay scoped publicly', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const first = await createRecord(request, 'testimonials', {
			author_name: 'Recovery author A',
			content: `Homepage first ${suffix}`,
			status: 'approved',
			visibility: 'public',
			featured: true
		});
		const chosen = await createRecord(request, 'testimonials', {
			author_name: 'Recovery author B',
			content: `Homepage chosen ${suffix}`,
			status: 'approved',
			visibility: 'public'
		});
		const outside = await createRecord(request, 'testimonials', {
			author_name: 'Recovery outside author',
			content: `Homepage outside ${suffix}`,
			status: 'approved',
			visibility: 'public'
		});
		const skills = {
			enabled: true,
			items: [],
			layout: 'grouped',
			width: 'full',
			categoryOrder: ['Languages', 'Tools'],
			disabledCategories: ['Tools'],
			categoryDisplayModes: { Languages: 'tags' }
		};
		await writeSettings(request, {
			homepage_section_order: ['testimonials', 'skills'],
			homepage_sections: {
				testimonials: {
					enabled: true,
					items: [first.id, chosen.id],
					layout: 'featured',
					width: 'full',
					featuredId: chosen.id
				},
				skills
			}
		});
		await page.goto('/admin/homepage#section-order');
		await page.getByRole('button', { name: 'Expand Testimonials section', exact: true }).click();
		await expect(page.locator('#homepage-featured-testimonial')).toHaveValue(chosen.id);
		const saved = await saveAndRead(page, request);
		expect(saved.homepage_sections.testimonials.featuredId).toBe(chosen.id);
		expect({
			...saved.homepage_sections.skills,
			items: saved.homepage_sections.skills.items ?? []
		}).toMatchObject(skills);
		await page.reload();
		await page.getByRole('button', { name: 'Expand Testimonials section', exact: true }).click();
		await expect(page.locator('#homepage-featured-testimonial')).toHaveValue(chosen.id);
		await page.goto('/');
		await expect(page.locator('#testimonials blockquote')).toContainText(
			`Homepage chosen ${suffix}`
		);
		await expect(page.locator('#testimonials')).not.toContainText(`Homepage outside ${suffix}`);
		await writeSettings(request, {
			homepage_sections: {
				...saved.homepage_sections,
				testimonials: { ...saved.homepage_sections.testimonials, featuredId: outside.id }
			}
		});
		await page.reload();
		await expect(page.locator('#testimonials blockquote')).toContainText(
			`Homepage first ${suffix}`
		);
		await expect(page.locator('#testimonials')).not.toContainText(`Homepage outside ${suffix}`);
	});

	test('the 501st and 1001st items and the 101st custom block can be chosen', async ({
		page,
		request
	}) => {
		test.setTimeout(120_000);
		await seedAdminSession(page, request);
		await writeSettings(request, {
			homepage_section_order: [],
			homepage_sections: {},
			homepage_custom_content: []
		});
		const pages = new Set<string | null>();
		await page.route(isList('projects'), (route) => {
			if (!isGet(route)) return route.fallback();
			pages.add(new URL(route.request().url()).searchParams.get('page'));
			return pagedList(route, 1001, (position) => ({
				id: `paged${position}`,
				title: `Paged project ${position}`,
				visibility: 'public',
				is_draft: false
			}));
		});
		await page.route(isList('custom_content'), (route) =>
			pagedList(route, 101, (position) => ({
				id: `pagedcustom${position}`,
				title: `Paged custom ${position}`,
				visibility: 'public',
				is_draft: false
			}))
		);

		await page.goto('/admin/homepage#section-order');
		await page.getByRole('button', { name: 'Expand Projects section', exact: true }).click();
		for (const position of [501, 1001]) {
			await page.getByRole('checkbox', { name: `Paged project ${position}`, exact: true }).check();
		}
		await expect(page.getByText('Custom: Paged custom 101', { exact: true })).toBeVisible();
		expect(pages.size).toBe(11);

		const put = page.waitForRequest(
			(sent) => isSettings(new URL(sent.url())) && sent.method() === 'PUT'
		);
		await saveHomepage(page).click();
		const body = (await put).postDataJSON();
		expect(body.homepage_sections.projects.items).toEqual(['paged501', 'paged1001']);
		expect(body.homepage_custom_content).toContainEqual({ id: 'pagedcustom101', enabled: true });
		await expect(page.getByText('Homepage saved successfully', { exact: true })).toBeVisible();
	});

	test('an empty profile list is a real first run and a failed save cannot create twice', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const writes: string[] = [];
		let failing = true;
		await page.route(isSettings, (route) => {
			if (route.request().method() !== 'PUT') return route.fallback();
			if (failing) return json(route, { error: 'Settings unavailable' }, 500);
			return json(route, { site_nav_enabled: false, site_nav_items: [] });
		});
		await page.route(isProfile, (route) => {
			if (isGet(route)) {
				return json(route, { page: 1, perPage: 100, totalItems: 0, totalPages: 0, items: [] });
			}
			writes.push(route.request().method());
			return json(route, { id: 'firstprofile', collectionId: 'profile', name: 'First profile' });
		});

		await page.goto('/admin/homepage');
		const name = page.locator('#name');
		await expect(name).toHaveValue('');
		await name.fill('First profile');
		await saveHomepage(page).click();
		await expect(page.getByText('Your profile changes were saved', { exact: false })).toBeVisible();

		failing = false;
		await saveHomepage(page).click();
		await expect(page.getByText('Homepage saved successfully', { exact: true })).toBeVisible();
		expect(writes).toEqual(['POST', 'PATCH']);
	});

	test('leaving mid-read aborts it and coming back reads again', async ({ page, request }) => {
		await seedAdminSession(page, request);
		const errors: string[] = [];
		page.on('console', (message) => {
			if (message.type() === 'error') errors.push(message.text());
		});
		let reads = 0;
		let release!: () => void;
		const held = new Promise<void>((resolve) => {
			release = resolve;
		});
		await page.route(isProfile, async (route) => {
			const pageRead = isGet(route) && route.request().url().includes('perPage=100');
			if (!pageRead || ++reads > 1) return route.fallback();
			await held;
			return json(route, {}).catch(() => undefined);
		});

		await page.goto('/admin/homepage');
		await expect.poll(() => reads).toBe(1);
		const aborted = page.waitForEvent('requestfailed', (sent) => isProfile(new URL(sent.url())));
		await page.locator('a[href="/admin/views"]').first().click();
		await expect(page).toHaveURL(/\/admin\/views$/);
		await aborted;
		release();

		await page.goBack();
		await expect(page.locator('#name')).toHaveValue(profile.name);
		expect(errors.filter((text) => text.includes('Failed to load profile'))).toEqual([]);
	});
});

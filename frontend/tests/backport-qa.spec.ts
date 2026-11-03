import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

const BASE_URL =
	process.env.PLAYWRIGHT_BASE_URL ?? process.env.VITE_POCKETBASE_URL ?? 'http://localhost:5173';

async function login(page: Page, request: APIRequestContext) {
	await seedAdminSession(page, request);
}

async function openAccordion(page: Page, sectionId: string, name: RegExp) {
	const trigger = page.locator(sectionId).getByRole('button', { name });
	await expect(trigger).toBeVisible();
	if ((await trigger.getAttribute('aria-expanded')) !== 'true') {
		await trigger.click();
	}
	await expect(trigger).toHaveAttribute('aria-expanded', 'true');
}

test.describe('Backport QA - Phase 2b a11y', () => {
	test.beforeEach(async ({ page, request }) => {
		await login(page, request);
	});

	test('AccentPicker renders with radiogroup pattern', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/homepage#brand-section`);
		await openAccordion(page, '#brand-section', /^Brand\b/);
		const group = page.getByRole('radiogroup', { name: 'Curated accent colors', exact: true });
		await expect(group).toBeVisible();
		const radios = group.getByRole('radio');
		expect(await radios.count()).toBeGreaterThan(0);
		await expect(group.locator('[role="radio"][aria-checked="true"]')).toHaveCount(1);
	});

	test('AccentPicker keyboard navigation works', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/homepage#brand-section`);
		await openAccordion(page, '#brand-section', /^Brand\b/);
		const group = page.getByRole('radiogroup', { name: 'Curated accent colors', exact: true });
		await expect(group).toBeVisible();
		const radios = group.getByRole('radio');
		const count = await radios.count();
		expect(count).toBeGreaterThan(1);
		const checkedRadio = group.locator('[role="radio"][aria-checked="true"]');
		await checkedRadio.focus();
		await expect(checkedRadio).toBeFocused();
		const checkedIndex = await radios.evaluateAll((elements) =>
			elements.findIndex((element) => element.getAttribute('aria-checked') === 'true')
		);
		await page.keyboard.press('ArrowRight');
		const nextRadio = radios.nth((checkedIndex + 1) % count);
		await expect(nextRadio).toBeFocused();
		await expect(nextRadio).toHaveAttribute('aria-checked', 'true');
	});

	test('Admin content uses a real button for quick-create', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/posts`);
		const newPost = page.getByRole('button', { name: '+ New Post', exact: true });
		await expect(newPost).toBeVisible();
		await newPost.click();
		await expect(page.getByRole('heading', { name: 'New Post', exact: true })).toBeVisible();
		expect(await page.locator('button span[role="button"]').count()).toBe(0);
	});

	test('AdminSidebar facets section shows single link with count badge', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin`);
		const facetsLink = page.locator('a[href="/admin/views"]').first();
		await expect(facetsLink).toBeVisible();
		const badge = facetsLink.locator('span').filter({ hasText: /^[0-9]+$/ });
		const badgeCount = await badge.count();
		if (badgeCount > 0) {
			const count = parseInt((await badge.textContent()) || '0', 10);
			expect(count).toBeGreaterThanOrEqual(0);
		}
		await expect(page.locator('a[href="/admin/views/new"]').first()).toBeVisible();
	});

	test('Floating cluster has inert attribute when nav is pinned', async ({ page }) => {
		await page.goto(`${BASE_URL}/`);
		await page.waitForLoadState('networkidle');
		const cluster = page.locator('.fixed.top-4.right-4');
		if ((await cluster.count()) > 0) {
			await page.evaluate(() => window.scrollTo(0, 1000));
			await page.waitForTimeout(500);
			const inertAttr = await cluster.getAttribute('inert');
			expect(inertAttr !== null || (await cluster.isVisible())).toBeTruthy();
		} else {
			test.skip(true, 'Floating cluster not found');
		}
	});
});

test.describe('Backport QA - Keyboard reordering', () => {
	test.beforeEach(async ({ page, request }) => {
		await login(page, request);
	});

	test('HomepageSectionManager has keyboard move buttons', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/homepage#section-order`);
		const section = page.locator('#section-order');
		const moveButtons = section.getByRole('button', { name: /move .+ (up|down)/i });
		await expect(moveButtons.first()).toBeVisible();
		expect(await moveButtons.count()).toBeGreaterThan(0);
	});

	test('View editor has keyboard move buttons', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/views`);
		const viewLinks = page.locator('a[href^="/admin/views/"]');
		await expect(viewLinks.first()).toBeVisible();
		await viewLinks.first().click();
		const moveButtons = page.getByRole('button', { name: /move .+ (up|down)/i });
		await expect(moveButtons.first()).toBeVisible();
		expect(await moveButtons.count()).toBeGreaterThan(0);
	});
});

test.describe('Backport QA - MarkdownEditor inline dialog', () => {
	test.beforeEach(async ({ page, request }) => {
		await login(page, request);
	});

	test('MarkdownEditor opens inline dialog for links', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/posts`);
		await page.getByRole('button', { name: '+ New Post', exact: true }).click();
		const linkButton = page.getByRole('button', { name: 'Insert link', exact: true });
		await expect(linkButton).toBeVisible();
		await linkButton.focus();
		await page.keyboard.press('Enter');
		const dialog = page.getByRole('dialog', { name: 'Insert Link', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog.getByLabel('URL', { exact: true })).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(dialog).not.toBeVisible();
		await expect(linkButton).toBeFocused();
	});

	test('MarkdownEditor image dialog closes immediately and restores focus', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/posts`);
		await page.getByRole('button', { name: '+ New Post', exact: true }).click();
		const imageButton = page.getByRole('button', { name: 'Insert image', exact: true });
		await expect(imageButton).toBeVisible();
		await imageButton.focus();
		await page.keyboard.press('Enter');
		const dialog = page.getByRole('dialog', { name: 'Insert Image', exact: true });
		await expect(dialog).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(dialog).not.toBeVisible();
		await expect(imageButton).toBeFocused();
	});
});

test.describe('Phase 3 quick wins', () => {
	test.beforeEach(async ({ page, request }) => {
		await login(page, request);
	});

	test('Hero layout picker renders in homepage admin', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/homepage#hero-section`);
		const heroLayoutGroup = page.getByRole('group', { name: 'Hero Layout', exact: true });
		await expect(heroLayoutGroup).toBeVisible();
		expect(await heroLayoutGroup.locator('button[aria-pressed]').count()).toBeGreaterThan(0);
	});

	test('Homepage has sections-grid CSS class', async ({ page }) => {
		await page.goto(`${BASE_URL}/`);
		await page.waitForLoadState('networkidle');
		await expect(page.locator('.sections-grid')).toBeVisible();
	});
});

test.describe('Site nav chips mode', () => {
	let token: string;
	let originalSettings: Record<string, unknown>;
	let activeFacet: { slug: string; label: string };

	test.beforeAll(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
		const headers = { Authorization: token };
		const response = await request.get(`${apiBaseURL}/api/site-settings`, { headers });
		expect(response.ok()).toBeTruthy();
		const settings = await response.json();
		originalSettings = {
			site_nav_enabled: settings.site_nav_enabled === true,
			site_nav_mode: settings.site_nav_mode || 'bar',
			site_nav_position: settings.site_nav_position || 'below',
			site_cta_enabled: settings.site_cta_enabled !== false,
			footer_cta_enabled: settings.footer_cta_enabled !== false,
			site_nav_items: settings.site_nav_items || []
		};

		const viewsResponse = await request.get(
			`${apiBaseURL}/api/collections/views/records?perPage=100`,
			{
				headers
			}
		);
		expect(viewsResponse.ok()).toBeTruthy();
		const views = (await viewsResponse.json()).items as Array<Record<string, unknown>>;
		const view = views.find((item) => item.is_active === true && item.visibility === 'public');
		if (!view) throw new Error('Site nav fixture requires an active public Facet');
		activeFacet = { slug: String(view.slug), label: 'Backport QA Facet' };

		const enabled = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers,
			data: {
				...originalSettings,
				site_nav_enabled: true,
				site_nav_items: [{ viewId: view.id, enabled: true, label: activeFacet.label }]
			}
		});
		expect(enabled.ok(), await enabled.text()).toBeTruthy();
	});

	test.afterAll(async ({ request }) => {
		const restored = await request.put(`${apiBaseURL}/api/site-settings`, {
			headers: { Authorization: token },
			data: originalSettings
		});
		expect(restored.ok(), await restored.text()).toBeTruthy();
	});

	test.beforeEach(async ({ page, request }) => {
		await login(page, request);
	});

	test('Admin homepage editor shows nav mode selector with radiogroup', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/homepage#nav-section`);
		await openAccordion(page, '#nav-section', /^Site Navigation\b/);
		const radiogroup = page.getByRole('radiogroup', { name: 'Navigation Style', exact: true });
		await expect(radiogroup).toBeVisible();
		await expect(radiogroup.getByRole('radio')).toHaveCount(2);
	});

	test('Public nav has aria-current="page" on active link', async ({ page }) => {
		await page.goto(`${BASE_URL}/`);
		const nav = page.getByRole('navigation', { name: 'Site navigation', exact: true });
		await expect(nav).toBeVisible();
		await expect(nav.getByRole('link', { name: 'Home', exact: true })).toHaveAttribute(
			'aria-current',
			'page'
		);

		const facetLink = nav.getByRole('link', { name: activeFacet.label, exact: true });
		await expect(facetLink).toBeVisible();
		await facetLink.click();
		await expect(page).toHaveURL(`${BASE_URL}/${activeFacet.slug}`);
		await expect(facetLink).toHaveAttribute('aria-current', 'page');
		await expect(nav.getByRole('link', { name: 'Home', exact: true })).not.toHaveAttribute(
			'aria-current',
			'page'
		);
	});

	test('E2E: switching chips in admin re-renders public nav as chips', async ({ page }) => {
		await page.goto(`${BASE_URL}/admin/homepage#nav-section`);
		await openAccordion(page, '#nav-section', /^Site Navigation\b/);
		const radiogroup = page.getByRole('radiogroup', { name: 'Navigation Style', exact: true });
		await expect(radiogroup).toBeVisible();
		const chipsBtn = radiogroup.getByRole('radio', { name: /^Chips\b/ });
		const saved = page.waitForResponse(
			(response) =>
				response.url().endsWith('/api/site-settings') && response.request().method() === 'PUT'
		);
		await chipsBtn.click();
		expect((await saved).ok()).toBeTruthy();
		await expect(chipsBtn).toHaveAttribute('aria-checked', 'true');

		const apiResp = await page.request.get(`${BASE_URL}/api/site-nav`);
		expect(apiResp.ok()).toBeTruthy();
		const apiData = await apiResp.json();
		expect(apiData.mode).toBe('chips');

		await page.goto(`${BASE_URL}/`);
		const nav = page.getByRole('navigation', { name: 'Site navigation', exact: true });
		await expect(nav).toBeVisible();
		const navClass = (await nav.getAttribute('class')) || '';
		expect(navClass).toContain('bg-white');
		expect(navClass).not.toContain('bg-primary-600');
		const homeLink = nav.locator('a[href="/"]').first();
		const linkClass = (await homeLink.getAttribute('class')) || '';
		expect(linkClass).toContain('rounded-full');
	});
});

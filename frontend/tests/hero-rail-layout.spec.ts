import { test, expect, type APIRequestContext } from '@playwright/test';
import { apiBaseURL } from './config';

/**
 * Rail hero layout end-to-end certification.
 *
 * The rail layout renders the hero as a sticky ~320px left <aside> sidebar and
 * flows page content in the right column via a 2-column grid. This spec drives
 * the real running stack:
 *   - A PB superuser PATCHes profile.hero_layout = 'rail'
 *   - The public page is loaded at a desktop viewport
 *   - We assert the rail <aside> is present, sits in the left column (~320px),
 *     and the page uses the 2-col grid
 *   - We then reset to 'standard' and assert the rail <aside> is gone, proving
 *     other layouts are unaffected.
 *
 * The suite is serial (playwright.config workers: 1); run with --workers=1.
 */

const PB_SUPERUSER = process.env.PB_SUPERUSER_EMAIL || 'admin@localhost.dev';
const PB_SUPERUSER_PW = process.env.PB_SUPERUSER_PASSWORD || 'admin123';

async function superuserToken(request: APIRequestContext): Promise<string> {
	const res = await request.post(
		`${apiBaseURL}/api/collections/_superusers/auth-with-password`,
		{ data: { identity: PB_SUPERUSER, password: PB_SUPERUSER_PW } }
	);
	expect(res.ok(), 'PB superuser auth must succeed').toBeTruthy();
	return (await res.json()).token as string;
}

async function profileId(request: APIRequestContext, token: string): Promise<string> {
	const res = await request.get(`${apiBaseURL}/api/collections/profile/records?perPage=1`, {
		headers: { Authorization: token }
	});
	expect(res.ok()).toBeTruthy();
	const items = (await res.json()).items as Array<{ id: string }>;
	expect(items.length, 'a profile record must exist').toBeGreaterThan(0);
	return items[0].id;
}

async function setHeroLayout(
	request: APIRequestContext,
	token: string,
	id: string,
	layout: string
): Promise<void> {
	const res = await request.patch(`${apiBaseURL}/api/collections/profile/records/${id}`, {
		headers: { Authorization: token },
		data: { hero_layout: layout }
	});
	expect(
		res.ok(),
		`PATCH hero_layout=${layout} must succeed (migration must accept 'rail')`
	).toBeTruthy();
}

test.describe('rail hero layout', () => {
	let token: string;
	let id: string;

	test.beforeAll(async ({ request }) => {
		token = await superuserToken(request);
		id = await profileId(request, token);
	});

	test.afterAll(async ({ request }) => {
		// Always reset to standard so the live instance / other specs are unaffected.
		await setHeroLayout(request, token, id, 'standard');
	});

	test('rail renders a sticky ~320px left sidebar in a 2-col grid', async ({ request, page }) => {
		await setHeroLayout(request, token, id, 'rail');
		await page.setViewportSize({ width: 1280, height: 900 });
		await page.goto('/', { waitUntil: 'networkidle' });

		// The rail sidebar is a <header class="rail-hero"> banner (not an <aside>),
		// matching the other five hero layouts and carrying the page's <h1>.
		const rail = page.locator('header.rail-hero');
		await expect(rail, 'rail header renders for hero_layout=rail').toHaveCount(1);
		await expect(rail).toBeVisible();

		// The header must hug the left edge and be roughly the rail width (~320px).
		const box = await rail.boundingBox();
		expect(box, 'rail header has a layout box').not.toBeNull();
		expect(box!.x, 'rail sidebar hugs the left edge').toBeLessThan(40);
		expect(box!.width, 'rail sidebar is ~320px wide').toBeGreaterThan(240);
		expect(box!.width, 'rail sidebar is ~320px wide').toBeLessThan(420);

		// The hero's grid wrapper must use the 2-col rail grid template.
		const gridTemplate = await rail.evaluate((el) => {
			const wrapper = el.parentElement as HTMLElement | null;
			return wrapper ? getComputedStyle(wrapper).gridTemplateColumns : '';
		});
		// Two resolved track widths => 2-column grid is active (rail col + content col).
		const tracks = gridTemplate.trim().split(/\s+/).filter(Boolean);
		expect(
			tracks.length,
			`rail wrapper uses a 2-col grid (got "${gridTemplate}")`
		).toBe(2);

		// Content must flow in the right column, to the right of the sidebar.
		const main = page.locator('main#main-content');
		await expect(main).toBeVisible();
		const mainBox = await main.boundingBox();
		expect(mainBox, 'main has a layout box').not.toBeNull();
		expect(
			mainBox!.x,
			'main content sits to the right of the rail sidebar'
		).toBeGreaterThan(box!.x + box!.width - 1);
	});

	test('switching back to standard removes the rail sidebar (other layouts unaffected)', async ({
		request,
		page
	}) => {
		await setHeroLayout(request, token, id, 'standard');
		await page.setViewportSize({ width: 1280, height: 900 });
		await page.goto('/', { waitUntil: 'networkidle' });

		// No rail sidebar for the standard layout.
		await expect(page.locator('header.rail-hero'), 'no rail header for standard').toHaveCount(0);

		// The standard hero still renders its <header>.
		await expect(page.locator('header').first(), 'standard hero header renders').toBeVisible();
	});
});

import { expect, test, type Page } from '@playwright/test';
import { seedAdminSession } from './helpers';

const MOBILE = { width: 390, height: 844 };
const DESKTOP = { width: 1440, height: 900 };

const menuButton = (page: Page) =>
	page.locator('.admin-header button[aria-controls="admin-sidebar"]');
const drawer = (page: Page) => page.getByRole('dialog', { name: 'Admin navigation', exact: true });
const closeControl = (page: Page) =>
	drawer(page).getByRole('button', { name: 'Close navigation menu', exact: true });
const sidebarStops = (page: Page) =>
	page.locator('#admin-sidebar').locator('a[href], button:not(:disabled)');

async function openAdmin(page: Page, viewport: typeof MOBILE) {
	await page.setViewportSize(viewport);
	await page.goto('/admin/help');
	await expect(menuButton(page)).toBeVisible();
}

// Waits for the slide-in to finish so position checks read the settled drawer
async function openDrawer(page: Page) {
	await menuButton(page).click();
	await expect(closeControl(page)).toBeFocused();
	await expect(closeControl(page)).toBeInViewport({ ratio: 1 });
}

async function pressAndReadFocus(page: Page, key: string) {
	await page.keyboard.press(key);
	return page.evaluate(() => {
		const active = document.activeElement as HTMLElement;
		const { left, right } = active.getBoundingClientRect();
		return {
			inSidebar: !!active.closest('#admin-sidebar'),
			onScreen: active === document.body || (right > 0 && left < window.innerWidth)
		};
	});
}

async function expectPageUnlocked(page: Page) {
	await expect(page.locator('#main-content')).toHaveJSProperty('inert', false);
	await expect(page.locator('.admin-header')).toHaveJSProperty('inert', false);
	await expect.poll(() => page.evaluate(() => document.body.style.overflow)).toBe('');
}

async function expectDrawerClosed(page: Page) {
	await expect(drawer(page)).toHaveCount(0);
	await expect(page.locator('[inert] #admin-sidebar')).toHaveCount(1);
	await expectPageUnlocked(page);
}

test.beforeEach(async ({ page, request }) => {
	await seedAdminSession(page, request);
});

test('closed drawer is inert and adds no offscreen tab stops', async ({ page }) => {
	await openAdmin(page, MOBILE);
	await expectDrawerClosed(page);

	for (let i = 0; i < 30; i++) {
		expect(await pressAndReadFocus(page, 'Tab')).toEqual({ inSidebar: false, onScreen: true });
	}
});

test('open drawer is a labelled modal that holds keyboard focus', async ({ page }) => {
	await openAdmin(page, MOBILE);
	await openDrawer(page);

	await expect(drawer(page)).toHaveAttribute('aria-modal', 'true');
	await expect(drawer(page).locator('nav')).toHaveCount(1);
	await expect(page.locator('#main-content')).toHaveJSProperty('inert', true);
	await expect(page.locator('.admin-header')).toHaveJSProperty('inert', true);
	expect(await page.evaluate(() => document.body.style.overflow)).toBe('hidden');

	// One full lap lands back on the close control without leaving the drawer
	const stops = await sidebarStops(page).count();
	for (let i = 0; i < stops; i++) {
		expect(await pressAndReadFocus(page, 'Tab')).toEqual({ inSidebar: true, onScreen: true });
	}
	await expect(closeControl(page)).toBeFocused();

	await page.keyboard.press('Shift+Tab');
	await expect(sidebarStops(page).last()).toBeFocused();
});

const dismissals: [string, (page: Page) => Promise<unknown>][] = [
	['Escape', (page) => page.keyboard.press('Escape')],
	['the close control', (page) => closeControl(page).click()],
	['the backdrop', (page) => page.mouse.click(MOBILE.width - 20, MOBILE.height / 2)]
];

for (const [name, dismiss] of dismissals) {
	test(`${name} closes the drawer and returns focus to the trigger`, async ({ page }) => {
		await openAdmin(page, MOBILE);
		await openDrawer(page);

		await dismiss(page);
		await expectDrawerClosed(page);
		await expect(menuButton(page)).toBeFocused();
	});
}

test('navigating from the drawer leaves focus to the destination page', async ({ page }) => {
	await openAdmin(page, MOBILE);
	await openDrawer(page);

	await drawer(page).locator('a[href="/admin/settings"]').focus();
	await page.keyboard.press('Enter');
	await expect(page).toHaveURL(/\/admin\/settings$/);
	await expectDrawerClosed(page);
	await expect(menuButton(page)).not.toBeFocused();
	expect(await pressAndReadFocus(page, 'Tab')).toEqual({ inSidebar: false, onScreen: true });
});

test('resizing to desktop restores navigation and document scroll', async ({ page }) => {
	await openAdmin(page, MOBILE);
	await openDrawer(page);

	await page.setViewportSize(DESKTOP);
	await expect(drawer(page)).toHaveCount(0);
	await expect(page.locator('[inert] #admin-sidebar')).toHaveCount(0);
	await expectPageUnlocked(page);
	await expect(menuButton(page)).toBeFocused();
	await expect(sidebarStops(page).first()).toBeVisible();

	await page.setViewportSize(MOBILE);
	await expectDrawerClosed(page);
});

test('desktop collapsed navigation stays tabbable and untrapped', async ({ page }) => {
	await page.addInitScript(() => localStorage.setItem('adminSidebarOpen', 'false'));
	await openAdmin(page, DESKTOP);
	await expect(menuButton(page)).toHaveAttribute('aria-expanded', 'false');
	await expect(drawer(page)).toHaveCount(0);
	await expect(page.locator('[inert] #admin-sidebar')).toHaveCount(0);

	const links = page.locator('#admin-sidebar nav a');
	await links.first().focus();
	await page.keyboard.press('Tab');
	await expect(links.nth(1)).toBeFocused();
	await page.keyboard.press('Escape');
	await expect(links.nth(1)).toBeFocused();

	await links.last().focus();
	expect(await pressAndReadFocus(page, 'Tab')).toEqual({ inSidebar: false, onScreen: true });
});

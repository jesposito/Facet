import {
	expect,
	test,
	type APIRequestContext,
	type Locator,
	type Page,
	type Request,
	type Route
} from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { seedAdminSession } from './helpers';

// Every dialog opens through the real UI. Only AI status, exports and the submits held
// open for busy checks are intercepted, so no AI, mail or payment service is called.

const DESKTOP = { width: 1280, height: 800 };
const MOBILE = { width: 390, height: 640 };
const WRITE_CALL =
	/\/api\/(share(?:\/|$)|view(?:\/|$)|collections\/[^/]+\/records(?:\/|\?|$)|site-settings(?:\?|$))/;

const shell = (page: Page) => page.locator('.admin-shell').first();
const tokenOpener = (page: Page) =>
	page.getByRole('button', { name: 'Generate Token', exact: true }).first();
const resumeOpener = (page: Page) => page.getByTitle('Generate AI-powered resume');
const modal = (page: Page, name: string) => page.getByRole('dialog', { name, exact: true });

// The view select renders once views load, and it is the expected first field.
async function openTokens(page: Page) {
	const viewsLoaded = page.waitForResponse((res) => res.url().includes('/collections/views/'));
	await page.goto('/admin/tokens');
	await viewsLoaded;
}

async function mockResumeUI(page: Page, exports: Record<string, unknown>[] = []) {
	await page.route('**/api/ai-print/status', (route) =>
		route.fulfill({ json: { available: true, pandoc_installed: true, ai_configured: true } })
	);
	await page.route('**/api/view/*/exports', (route) => route.fulfill({ json: { exports } }));
}

// Parks the first matching request until the test fulfills it.
async function holdRequest(page: Page, url: string) {
	let release!: (route: Route) => void;
	const held = new Promise<Route>((resolve) => (release = resolve));
	await page.route(url, (route) => release(route));
	return { held };
}

async function expectPageLocked(page: Page) {
	await expect(shell(page)).toHaveJSProperty('inert', true);
	const state = await page.evaluate(() => ({
		overflow: getComputedStyle(document.documentElement).overflowY,
		sideways: document.documentElement.scrollWidth > document.documentElement.clientWidth
	}));
	expect(state).toEqual({ overflow: 'hidden', sideways: false });
}

async function expectPageRestored(page: Page) {
	await expect(shell(page)).toHaveJSProperty('inert', false);
	const locks = await page.evaluate(() => [
		document.documentElement.style.overflow,
		document.body.style.overflow
	]);
	expect(locks).toEqual(['', '']);
}

async function expectTabStaysInside(page: Page, dialog: Locator) {
	for (const key of ['Tab', 'Shift+Tab']) {
		for (let press = 0; press < 12; press++) {
			await page.keyboard.press(key);
			const focus = await dialog.evaluate((el) => {
				const active = document.activeElement as HTMLElement;
				const box = active.getBoundingClientRect();
				return {
					inside: el.contains(active),
					onScreen:
						box.top >= 0 && box.left >= 0 && box.bottom <= innerHeight && box.right <= innerWidth
				};
			});
			expect(focus).toEqual({ inside: true, onScreen: true });
		}
	}
}

async function expectModalContract(
	page: Page,
	opener: Locator,
	name: string,
	initialFocus: (dialog: Locator) => Locator
) {
	const writes: string[] = [];
	const recordWrite = (request: Request) => {
		if (request.method() !== 'GET' && WRITE_CALL.test(request.url())) writes.push(request.url());
	};
	page.on('request', recordWrite);

	await opener.click();
	const dialog = modal(page, name);
	await expect(dialog).toHaveAttribute('aria-modal', 'true');
	await expect(initialFocus(dialog)).toBeFocused();
	await expectPageLocked(page);
	await expectTabStaysInside(page, dialog);

	await page.keyboard.press('Escape');
	await expect(dialog).toHaveCount(0);
	await expect(opener).toBeFocused();
	await expectPageRestored(page);

	page.off('request', recordWrite);
	expect(writes).toEqual([]);
}

test.describe('admin dialogs', () => {
	let token: string;
	let projectId: string;
	let projectTitle: string;
	let viewId: string;

	async function createRecord(
		request: APIRequestContext,
		collection: string,
		data: Record<string, unknown>
	) {
		const res = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
			headers: { Authorization: token },
			data
		});
		expect(res.ok(), await res.text()).toBe(true);
		return (await res.json()).id as string;
	}

	test.beforeEach(async ({ page, request }) => {
		({ token } = await seedAdminSession(page, request));
		const suffix = randomUUID().slice(0, 8);
		projectTitle = `dialog project ${suffix}`;
		projectId = await createRecord(request, 'projects', {
			title: projectTitle,
			slug: `dialog-project-${suffix}`,
			summary: 'Dialog fixture.',
			description: 'Dialog fixture.',
			visibility: 'public',
			is_draft: false
		});
		viewId = await createRecord(request, 'views', {
			name: `Dialog view ${suffix}`,
			slug: `dialog-view-${suffix}`,
			visibility: 'unlisted',
			is_active: true,
			sections: [
				{ section: 'projects', enabled: true, layout: 'grid-2', itemSelection: 'all_public' }
			]
		});
	});

	test.afterEach(async ({ request }) => {
		const headers = { Authorization: token };
		await request.delete(`${apiBaseURL}/api/collections/views/records/${viewId}`, { headers });
		await request.delete(`${apiBaseURL}/api/collections/projects/records/${projectId}`, {
			headers
		});
	});

	for (const [label, viewport] of [
		['desktop', DESKTOP],
		['mobile', MOBILE]
	] as const) {
		test.describe(label, () => {
			test.use({ viewport });

			test('share token dialog', async ({ page }) => {
				await openTokens(page);
				await expectModalContract(page, tokenOpener(page), 'Generate Share Token', (dialog) =>
					dialog.locator('#view_id')
				);
			});

			test('resume dialog', async ({ page }) => {
				await mockResumeUI(page);
				await page.goto(`/admin/views/${viewId}`);
				await expectModalContract(page, resumeOpener(page), 'Generate Resume', (dialog) =>
					dialog.locator('#format')
				);
			});

			test('override dialog', async ({ page }) => {
				await page.goto(`/admin/views/${viewId}`);
				await page.getByRole('button', { name: 'Expand Projects section' }).click();
				const opener = page
					.getByRole('list', { name: 'Content for Projects' })
					.getByRole('listitem')
					.filter({ hasText: projectTitle })
					.getByRole('button', { name: 'Customize' });
				await expectModalContract(page, opener, 'Customize for this View', (dialog) =>
					dialog.locator('#override_title')
				);
			});

			test('custom CSS help dialog', async ({ page }) => {
				await page.goto('/admin/settings/site');
				const opener = page.getByRole('button', { name: 'Selectors', exact: true });
				await expectModalContract(page, opener, 'Public CSS hooks', (dialog) =>
					dialog.getByRole('heading', { name: 'Public CSS hooks' })
				);
			});
		});
	}

	test('a pending share token survives Escape; Enter still submits', async ({ page }) => {
		const { held } = await holdRequest(page, '**/api/share/generate');
		await openTokens(page);
		await tokenOpener(page).click();
		const dialog = modal(page, 'Generate Share Token');
		await dialog.locator('#view_id').selectOption(viewId);
		await dialog.locator('#name').fill('dialog quality');
		await dialog.locator('#name').press('Enter');
		const route = await held;

		await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeDisabled();
		await page.keyboard.press('Escape');
		await page.keyboard.press('Escape');
		await expect(dialog.locator('#name')).toHaveValue('dialog quality');

		await route.fulfill({ json: { token: 'dialog-quality-token' } });
		await expect(dialog).toHaveCount(0);
		await expect(page.locator('code', { hasText: '/s/dialog-quality-token' })).toBeVisible();
		await expect(tokenOpener(page)).toBeFocused();
		await expectPageRestored(page);
	});

	test('a pending resume survives Escape and backdrop clicks', async ({ page }) => {
		await mockResumeUI(page);
		const { held } = await holdRequest(page, '**/api/view/*/generate');
		await page.goto(`/admin/views/${viewId}`);
		await resumeOpener(page).click();
		const dialog = modal(page, 'Generate Resume');
		await dialog.getByRole('button', { name: 'Generate', exact: true }).click();
		const route = await held;

		await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeDisabled();
		await page.keyboard.press('Escape');
		await page.mouse.click(5, 5);
		await expect(dialog).toBeVisible();
		await page.keyboard.press('Tab');
		await expect(dialog).toBeFocused();

		await route.fulfill({
			json: {
				export_id: 'dialog-export',
				format: 'pdf',
				status: 'completed',
				generated_at: new Date().toISOString(),
				download_url: ''
			}
		});
		await expect(dialog).toHaveCount(0);
		await expect(page.getByText('Resume generated successfully!')).toBeVisible();
		await expect(resumeOpener(page)).toBeFocused();
		await expectPageRestored(page);
	});

	test('a failed generation explains the error and keeps the entered role', async ({ page }) => {
		await mockResumeUI(page);
		await page.route('**/api/view/*/generate', (route) =>
			route.fulfill({ status: 503, json: { error: 'Temporary generation failure' } })
		);
		await page.goto('/admin/views/' + viewId);
		await resumeOpener(page).click();
		const dialog = modal(page, 'Generate Resume');
		await dialog.locator('#target_role').fill('Learning designer');
		await dialog.getByRole('button', { name: 'Generate', exact: true }).click();
		await expect(dialog.getByRole('alert')).toHaveText('Temporary generation failure');
		await expect(dialog.locator('#target_role')).toHaveValue('Learning designer');
		await expect(dialog.getByRole('button', { name: 'Generate', exact: true })).toBeEnabled();
		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
		await expectPageRestored(page);
	});

	test('the export delete confirm works over the resume dialog', async ({ page }) => {
		await mockResumeUI(page, [
			{
				id: 'dialog-export',
				format: 'pdf',
				status: 'completed',
				generated_at: new Date().toISOString(),
				download_url: ''
			}
		]);
		await page.route('**/api/view/*/exports/*', (route) => route.fulfill({ json: {} }));
		await page.goto(`/admin/views/${viewId}`);
		await resumeOpener(page).click();
		const dialog = modal(page, 'Generate Resume');
		const remove = dialog.getByRole('button', { name: 'Delete', exact: true });
		const confirmation = page.getByRole('alertdialog', { name: 'Delete Export' });

		await remove.click();
		await expect(confirmation.getByRole('button', { name: 'Cancel' })).toBeFocused();
		await page.keyboard.press('Escape');
		await expect(confirmation).toHaveCount(0);
		await expect(remove).toBeFocused();
		await expectPageLocked(page);

		await remove.click();
		await confirmation.getByRole('button', { name: 'Delete', exact: true }).click();
		await expect(remove).toHaveCount(0);
		await expect(page.getByText('Export deleted')).toBeVisible();
		await page.keyboard.press('Tab');
		await expect(dialog).toBeFocused();
		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
		await expect(resumeOpener(page)).toBeFocused();
		await expectPageRestored(page);
	});

	test('browser back releases the page', async ({ page }) => {
		await page.goto('/admin/views');
		await page.locator('a[href="/admin/tokens"]').filter({ visible: true }).first().click();
		await expect(page).toHaveURL(/\/admin\/tokens$/);
		await tokenOpener(page).click();
		await expect(modal(page, 'Generate Share Token')).toBeVisible();

		await page.goBack();
		await expect(page).toHaveURL(/\/admin\/views$/);
		await expect(modal(page, 'Generate Share Token')).toHaveCount(0);
		await expect(page.locator('.admin-shell')).toHaveCount(1);
		await expectPageRestored(page);
	});

	test('the mobile drawer still works after a dialog closes', async ({ page }) => {
		await page.setViewportSize(MOBILE);
		await page.goto('/admin/tokens');
		await tokenOpener(page).click();
		await expect(modal(page, 'Generate Share Token')).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(tokenOpener(page)).toBeFocused();

		const menu = page.locator('.admin-header button[aria-controls="admin-sidebar"]');
		await menu.click();
		await expect(modal(page, 'Admin navigation')).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(menu).toBeFocused();
		await expectPageRestored(page);
	});
});

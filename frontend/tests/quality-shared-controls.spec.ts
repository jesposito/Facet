import AxeBuilder from '@axe-core/playwright';
import { randomUUID } from 'node:crypto';
import { expect, test } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession, setSiteDesign } from './helpers';

const createdAwardIds: string[] = [];
let token: string;

test.beforeEach(async ({ page, request }) => {
	({ token } = await loginAsAdmin(request));
	await seedAdminSession(page, request);
});

test.afterEach(async ({ request }) => {
	for (const id of createdAwardIds.splice(0)) {
		await request.delete(`${apiBaseURL}/api/collections/awards/records/${id}`, {
			headers: { Authorization: token }
		});
	}
});

test('named award search filters and clears results', async ({ page, request }) => {
	const suffix = randomUUID();
	const matchingTitle = `Shared controls match ${suffix}`;
	const otherTitle = `Shared controls other ${suffix}`;

	for (const title of [matchingTitle, otherTitle]) {
		const response = await request.post(`${apiBaseURL}/api/collections/awards/records`, {
			headers: { Authorization: token },
			data: {
				title,
				issuer: 'Playwright',
				visibility: 'public',
				is_draft: false,
				sort_order: 0
			}
		});
		expect(response.ok(), await response.text()).toBeTruthy();
		createdAwardIds.push((await response.json()).id);
	}

	await page.goto('/admin/awards');
	const search = page.getByRole('textbox', { name: 'Search awards...', exact: true });
	await expect(search).toBeVisible();

	await search.fill(matchingTitle);
	await expect(page.getByRole('heading', { name: matchingTitle, exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: otherTitle, exact: true })).toHaveCount(0);

	await page.getByRole('button', { name: 'Clear search', exact: true }).click();
	await expect(search).toHaveValue('');
	await expect(page.getByRole('heading', { name: matchingTitle, exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: otherTitle, exact: true })).toBeVisible();
});

test('logout accessible name contains its visible label', async ({ page }) => {
	await page.goto('/admin');
	await expect(page.getByRole('button', { name: 'Logout', exact: true })).toBeVisible();

	const results = await new AxeBuilder({ page })
		.include('.admin-header .btn-ghost')
		.withRules(['label-content-name-mismatch'])
		.analyze();
	expect(results.violations).toEqual([]);
});

test('sidebar version metadata meets contrast in light and dark themes', async ({ page }) => {
	await page.goto('/admin');

	for (const theme of ['light', 'dark']) {
		const dark = theme === 'dark';
		if ((await page.locator('html').evaluate((el) => el.classList.contains('dark'))) !== dark) {
			await page.getByRole('button', { name: `Switch to ${theme} theme`, exact: true }).click();
		}
		await page.evaluate(async () => {
			await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
			await Promise.all(
				document
					.getAnimations()
					.filter((a) => a instanceof CSSTransition)
					.map((a) => a.finished.catch(() => {}))
			);
		});
		const results = await new AxeBuilder({ page })
			.include('#admin-sidebar > div')
			.withRules(['color-contrast'])
			.analyze();
		expect(results.violations, `${theme} sidebar footer contrast`).toEqual([]);
	}
});

test('rich and markdown editables share a name and heading controls include visible text', async ({
	page
}) => {
	await page.addInitScript(() => localStorage.setItem('editor-preferred-mode', 'wysiwyg'));
	await page.goto('/admin/custom');
	await page.getByRole('button', { name: /New Custom Content/ }).click();

	const richEditor = page.getByRole('textbox', { name: 'Rich text editor', exact: true });
	await expect(richEditor).toBeVisible();
	await expect(richEditor).toHaveAttribute('contenteditable', 'true');
	await expect(richEditor).toHaveAttribute('aria-multiline', 'true');
	await expect(page.locator('.markdown-editor [role="textbox"]')).toHaveCount(1);

	for (const level of [1, 2, 3]) {
		await expect(
			page.getByRole('button', { name: `H${level} Heading ${level}`, exact: true })
		).toBeVisible();
	}

	await page.getByRole('button', { name: 'Insert link', exact: true }).click();
	await expect(page.getByRole('dialog', { name: 'Insert Link', exact: true })).toBeVisible();
	await page
		.getByRole('dialog', { name: 'Insert Link', exact: true })
		.getByRole('button', { name: 'Cancel', exact: true })
		.click();

	await richEditor.fill('Shared editor content');
	await page
		.getByRole('button', { name: 'Switch to Markdown (Ctrl+Shift+M)', exact: true })
		.click();
	const markdownEditor = page.getByRole('textbox', { name: 'Rich text editor', exact: true });
	await expect(markdownEditor).toBeVisible();
	await expect(markdownEditor).toHaveValue('Shared editor content');

	const results = await new AxeBuilder({ page })
		.include('.markdown-editor')
		.withRules(['label', 'label-content-name-mismatch'])
		.analyze();
	expect(results.violations).toEqual([]);

	await markdownEditor.fill('# Updated markdown');
	await page
		.getByRole('button', { name: 'Switch to rich text (Ctrl+Shift+M)', exact: true })
		.click();
	await expect(
		page.getByRole('textbox', { name: 'Rich text editor', exact: true })
	).toHaveAttribute('contenteditable', 'true');
	await expect(page.getByRole('heading', { name: 'Updated markdown', exact: true })).toBeVisible();
});

test('default primary buttons meet text contrast', async ({ page, request }) => {
	await setSiteDesign(request, token, 'classic');
	await page.goto('/admin/awards');

	const results = await new AxeBuilder({ page })
		.include('.btn-primary')
		.withRules(['color-contrast'])
		.analyze();
	expect(results.violations).toEqual([]);
});

import { randomUUID } from 'node:crypto';
import { expect, test, type APIRequestContext, type Locator, type Page } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

type CreatedRecord = { id: string };
type Fixture = { collection: string; id: string };
type Screen = {
	route: string;
	collection: string;
	data: (label: string, run: string) => Record<string, unknown>;
};

const screens: Screen[] = [
	{
		route: 'awards',
		collection: 'awards',
		data: (label) => ({ title: label, issuer: 'Bulk selection fixture' })
	},
	{
		route: 'certifications',
		collection: 'certifications',
		data: (label) => ({ name: label, issuer: 'Bulk selection fixture' })
	},
	{
		route: 'projects',
		collection: 'projects',
		data: (label, run) => ({
			title: label,
			slug: `bulk-${run}-${label.startsWith('Shown') ? 'shown' : 'hidden'}`,
			summary: 'Bulk selection fixture'
		})
	},
	{
		route: 'posts',
		collection: 'posts',
		data: (label, run) => ({
			title: label,
			slug: `bulk-${run}-${label.startsWith('Shown') ? 'shown' : 'hidden'}`,
			content: 'Bulk selection fixture'
		})
	},
	{
		route: 'courses',
		collection: 'courses',
		data: (label, run) => ({
			title: label,
			slug: `bulk-${run}-${label.startsWith('Shown') ? 'shown' : 'hidden'}`,
			access_tier: 'free',
			price: 0
		})
	},
	{
		route: 'education',
		collection: 'education',
		data: (label) => ({ degree: label, institution: 'Bulk selection fixture' })
	},
	{
		route: 'skills',
		collection: 'skills',
		data: (label) => ({ name: label, category: 'Bulk selection fixture' })
	},
	{
		route: 'experience',
		collection: 'experience',
		data: (label) => ({ title: label, company: 'Bulk selection fixture' })
	},
	{
		route: 'talks',
		collection: 'talks',
		data: (label) => ({ title: label, event: 'Bulk selection fixture' })
	},
	{
		route: 'custom',
		collection: 'custom_content',
		data: (label) => ({ title: label, content: 'Bulk selection fixture' })
	}
];

function rowCheckbox(page: Page, label: string) {
	return page.getByRole('checkbox', { name: `Select ${label}`, exact: true });
}

function bulkBar(page: Page) {
	return page.getByRole('region', { name: 'Bulk selection actions', exact: true });
}

async function expectCount(bar: Locator, selected: number, total: number) {
	await expect(bar.getByRole('status')).toHaveText(`${selected} of ${total} selected`);
}

async function createRecord(
	request: APIRequestContext,
	token: string,
	screen: Screen,
	label: string,
	run: string
) {
	const response = await request.post(
		`${apiBaseURL}/api/collections/${screen.collection}/records`,
		{
			headers: { Authorization: token },
			data: {
				visibility: 'private',
				...(screen.collection === 'skills' ? {} : { is_draft: false }),
				...screen.data(label, run)
			}
		}
	);
	expect(response.ok(), await response.text()).toBe(true);
	return (await response.json()) as CreatedRecord;
}

test.describe('Reliable admin bulk selection', () => {
	test.skip(
		(!adminEmail || !adminPassword) && !process.env.FACET_OWNER_AUTH_FILE,
		'Admin credentials required'
	);

	let token: string;
	let created: Fixture[] = [];

	test.beforeEach(async ({ page, request }) => {
		({ token } = await loginAsAdmin(request));
		await seedAdminSession(page, request);
	});

	test.afterEach(async ({ request }) => {
		for (const fixture of [...created].reverse()) {
			await request.delete(
				`${apiBaseURL}/api/collections/${fixture.collection}/records/${fixture.id}`,
				{ headers: { Authorization: token } }
			);
		}
		created = [];
	});

	for (const screen of screens) {
		test(`${screen.route} keeps selection reliable and visible-only`, async ({ page, request }) => {
			const run = randomUUID();
			const shownLabel = `Shown ${run}`;
			const hiddenLabel = `Hidden ${run}`;
			for (const label of [shownLabel, hiddenLabel]) {
				const record = await createRecord(request, token, screen, label, run);
				created.push({ collection: screen.collection, id: record.id });
			}

			if (screen.route === 'awards') await page.setViewportSize({ width: 390, height: 500 });
			await page.goto(`/admin/${screen.route}`);
			const main = page.getByRole('main');
			const search = main.getByRole('textbox', { name: /^Search / });
			await search.fill(run);
			await main.getByRole('button', { name: 'Select', exact: true }).click();

			const bar = bulkBar(page);
			const shown = rowCheckbox(page, shownLabel);
			const hidden = rowCheckbox(page, hiddenLabel);
			await expect(bar).toBeVisible();
			await expectCount(bar, 0, 2);
			await expect(bar.getByRole('button', { name: 'Change Visibility' })).toBeDisabled();
			await expect(bar.getByRole('button', { name: 'Delete', exact: true })).toBeDisabled();
			await expect(shown).toBeVisible();
			await expect(hidden).toBeVisible();

			await shown.check();
			await expectCount(bar, 1, 2);
			await hidden.focus();
			await hidden.press('Space');
			await expectCount(bar, 2, 2);
			await expect(shown).toBeChecked();
			await expect(hidden).toBeChecked();

			await bar.getByRole('button', { name: 'Clear', exact: true }).click();
			await expectCount(bar, 0, 2);
			await expect(shown).not.toBeChecked();
			await expect(hidden).not.toBeChecked();

			await search.fill(shownLabel);
			await expect(hidden).toHaveCount(0);
			await bar.getByRole('button', { name: 'Select all', exact: true }).click();
			await expectCount(bar, 1, 1);
			await expect(shown).toBeChecked();

			await search.fill(run);
			await expect(hidden).toBeVisible();
			await expectCount(bar, 1, 2);
			await expect(hidden).not.toBeChecked();
			await bar.getByRole('button', { name: 'Select all', exact: true }).click();
			await expectCount(bar, 2, 2);

			await search.fill(hiddenLabel);
			await expect(shown).toHaveCount(0);
			await expectCount(bar, 1, 1);
			await expect(hidden).toBeChecked();
			await search.fill(shownLabel);
			await expect(hidden).toHaveCount(0);
			await expectCount(bar, 0, 1);
			await expect(shown).not.toBeChecked();

			await search.fill(run);
			await expect(hidden).toBeVisible();
			await expectCount(bar, 0, 2);
			await bar.getByRole('button', { name: 'Select all', exact: true }).click();
			await bar.getByRole('button', { name: 'Clear', exact: true }).click();
			await expectCount(bar, 0, 2);

			if (screen.route === 'awards') {
				await shown.check();
				const visibility = bar.getByRole('button', { name: 'Change Visibility' });
				await visibility.click();
				await expect(bar.getByRole('button', { name: 'Public', exact: true })).toBeVisible();
				await bar.getByRole('button', { name: 'Public', exact: true }).press('Escape');
				await expect(bar.getByRole('button', { name: 'Public', exact: true })).toHaveCount(0);
				await expect(visibility).toBeFocused();
				const clear = bar.getByRole('button', { name: 'Clear', exact: true });
				await clear.focus();
				await clear.press('Escape');
				await expect(clear).toBeFocused();
			}

			await bar.getByRole('button', { name: 'Cancel', exact: true }).click();
			await expect(bar).toHaveCount(0);
			await expect(shown).toHaveCount(0);
			await main.getByRole('button', { name: 'Select', exact: true }).click();
			await expectCount(bar, 0, 2);
			await expect(shown).not.toBeChecked();
			await expect(hidden).not.toBeChecked();

			if (screen.route === 'awards') {
				await bar.locator('button').evaluateAll((buttons) => {
					for (const button of buttons) {
						button.textContent =
							'SichtbarkeitsänderungsoptionenFürAlleAusgewähltenEinträge gleichzeitig ändern';
					}
				});
				// Every action must stay inside the bar after its hover transition,
				// including disabled actions. Pointer position must not hide overflow.
				for (const action of await bar.getByRole('button').all()) {
					await action.hover();
					await action.evaluate(async (element) => {
						await Promise.all(
							element.getAnimations().map((animation) => animation.finished.catch(() => undefined))
						);
					});
					const overflowing = await bar.evaluate((element) =>
						[element, ...element.querySelectorAll('*')]
							.filter((node) => node.clientWidth > 0 && node.scrollWidth > node.clientWidth + 1)
							.map((node) => ({
								tag: node.tagName,
								classes: node.getAttribute('class'),
								width: node.clientWidth,
								scroll: node.scrollWidth
							}))
					);
					expect(overflowing).toEqual([]);
				}
				expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
					390
				);
				await expect(bar).toHaveCSS('position', 'sticky');
				await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
				const barBox = await bar.boundingBox();
				const headerBox = await page.locator('.admin-header').boundingBox();
				expect(barBox).not.toBeNull();
				expect(headerBox).not.toBeNull();
				expect(barBox!.y).toBeGreaterThanOrEqual(headerBox!.y + headerBox!.height);
				expect(barBox!.y).toBeLessThan(headerBox!.y + headerBox!.height + 20);
			}
		});
	}
});

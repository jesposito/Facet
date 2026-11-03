import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { adminEmail, adminPassword } from './config';
import { loginAsAdmin } from './helpers';

const shouldRunAdmin = Boolean(adminEmail && adminPassword);

async function openAdminSession(page: Page, request: APIRequestContext) {
	const { token, record } = await loginAsAdmin(request);
	await page.context().addCookies([
		{
			name: 'pb_auth',
			value: JSON.stringify({ token, record }),
			url: process.env.PLAYWRIGHT_BASE_URL!
		}
	]);
	await page.addInitScript(
		({ token, record }) => {
			localStorage.setItem('pocketbase_auth', JSON.stringify({ token, record }));
			localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
		},
		{ token, record }
	);
}

test.describe('audited admin workflows', () => {
	test.skip(!shouldRunAdmin, 'ADMIN_EMAIL and ADMIN_PASSWORD are required for admin tests');

	test('resume import reviews selections and puts returned IDs into the new Facet', async ({
		page,
		request
	}) => {
		await openAdminSession(page, request);

		await page.route('**/api/collections/ai_providers/records?*', async (route) => {
			await route.fulfill({
				json: {
					page: 1,
					perPage: 1,
					totalItems: 1,
					totalPages: 1,
					items: [{ id: 'provider-1', name: 'Fixture provider', type: 'stub', is_active: true }]
				}
			});
		});
		await page.route('**/api/resume/upload', async (route) => {
			await route.fulfill({
				json: {
					status: 'success',
					filename: 'resume.pdf',
					parsed: {
						projects: [
							{ title: 'Keep project', summary: 'Selected for import' },
							{ title: 'Skip project', summary: 'Excluded during review' }
						]
					}
				}
			});
		});

		let importPayload: {
			visibility?: string;
			parsed?: { projects?: Array<{ title?: string }> };
		} | null = null;
		await page.route('**/api/resume/import', async (route) => {
			importPayload = route.request().postDataJSON();
			await route.fulfill({
				json: {
					status: 'success',
					imported: { projects: ['imported-project-id'] },
					deduplicated: 0,
					counts: { projects: 1 }
				}
			});
		});

		let createdView: {
			sections?: Array<{ section: string; enabled: boolean; items: string[] }>;
		} | null = null;
		await page.route('**/api/collections/views/records', async (route) => {
			if (route.request().method() !== 'POST') return route.continue();
			createdView = route.request().postDataJSON();
			await route.fulfill({
				json: { id: 'created-view-id', name: 'Resume review Facet', slug: 'resume-review-facet' }
			});
		});

		await page.goto('/admin/views/new');
		await page.getByLabel('Name *', { exact: true }).fill('Resume review Facet');
		await page.getByLabel('URL Slug *', { exact: true }).fill('resume-review-facet');
		await page.getByRole('button', { name: 'Import Resume', exact: true }).click();
		await page.locator('#resumeFileModal').setInputFiles({
			name: 'resume.pdf',
			mimeType: 'application/pdf',
			buffer: Buffer.from('disposable resume fixture')
		});
		await page.getByRole('button', { name: 'Upload & parse', exact: true }).click();

		await expect(page.getByRole('heading', { name: 'Review parsed items' })).toBeVisible();
		await page.getByRole('checkbox', { name: 'Select item Skip project' }).uncheck();
		await page.getByRole('radio', { name: /Public/ }).check();
		await page.getByRole('button', { name: 'Import 1 selected', exact: true }).click();

		await expect(page.getByRole('dialog')).toHaveCount(0);
		expect(importPayload).toEqual({
			filename: 'resume.pdf',
			visibility: 'public',
			parsed: expect.objectContaining({
				projects: [{ title: 'Keep project', summary: 'Selected for import' }]
			})
		});

		await page.getByRole('button', { name: 'Create', exact: true }).click();
		await expect.poll(() => createdView).not.toBeNull();
		const projectSection = createdView!.sections!.find((section) => section.section === 'projects');
		expect(projectSection).toMatchObject({ enabled: true, items: ['imported-project-id'] });
	});

	test('proposal JSON editing preserves object arrays and rejects malformed JSON', async ({
		page,
		request
	}) => {
		await openAdminSession(page, request);
		await page.route(
			'**/api/collections/import_proposals/records/proposal-links*',
			async (route) => {
				await route.fulfill({
					json: {
						id: 'proposal-links',
						status: 'pending',
						proposed_data: {
							links: [{ type: 'website', url: 'https://old.example.test' }]
						},
						diff: {}
					}
				});
			}
		);

		let applyCount = 0;
		let appliedEdits: { links?: unknown } | null = null;
		await page.route('**/api/proposals/proposal-links/apply', async (route) => {
			applyCount += 1;
			const body = route.request().postDataJSON() as { edits: { links?: unknown } };
			appliedEdits = body.edits;
			await route.fulfill({ json: { status: 'success' } });
		});

		await page.goto('/admin/review/proposal-links');
		const editor = page.getByLabel('Edit links as JSON', { exact: true });
		await expect(editor).toBeVisible();
		await editor.fill('[{"type":');
		await page.getByRole('button', { name: 'Apply Selected Changes', exact: true }).click();
		await expect(editor).toHaveAttribute('aria-invalid', 'true');
		await expect(page.getByText('Links must be a valid JSON array of objects.')).toBeVisible();
		expect(applyCount).toBe(0);

		const links = [
			{ type: 'website', url: 'https://new.example.test' },
			{ type: 'repository', url: 'https://github.com/example/project' }
		];
		await editor.fill(JSON.stringify(links, null, 2));
		await expect(editor).not.toHaveAttribute('aria-invalid', 'true');
		await page.getByRole('button', { name: 'Apply Selected Changes', exact: true }).click();
		await expect.poll(() => applyCount).toBe(1);
		expect(appliedEdits).toEqual({ links });
	});

	test('homepage settings failure does not claim a complete save', async ({ page, request }) => {
		await openAdminSession(page, request);
		await page.route('**/api/site-settings', async (route) => {
			if (route.request().method() === 'PUT') {
				await route.fulfill({ status: 500, json: { error: 'Settings unavailable' } });
			} else {
				await route.continue();
			}
		});

		await page.goto('/admin/homepage');
		await page.getByRole('button', { name: 'Save Homepage', exact: true }).click();
		await expect(page.getByText(/Failed to save homepage settings/)).toBeVisible();
		await expect(page.getByText('Homepage saved successfully', { exact: true })).toHaveCount(0);
	});

	test('optimistic navigation toggles roll back when settings cannot be saved', async ({
		page,
		request
	}) => {
		await openAdminSession(page, request);
		await page.route('**/api/site-settings', async (route) => {
			if (route.request().method() === 'PUT') {
				await route.fulfill({ status: 500, json: { error: 'Settings unavailable' } });
			} else {
				await route.continue();
			}
		});

		await page.goto('/admin/homepage#nav-section');
		const toggle = page.getByRole('checkbox', { name: 'Enable site navigation' });
		await expect(toggle).toBeEnabled();
		const original = await toggle.isChecked();
		const failedSave = page.waitForResponse(
			(response) =>
				response.url().endsWith('/api/site-settings') && response.request().method() === 'PUT'
		);
		await toggle.click();
		expect((await failedSave).status()).toBe(500);
		await expect(page.getByText('Failed to save site nav settings', { exact: true })).toBeVisible();
		await expect(toggle).toBeChecked({ checked: original });
	});

	test('an uncertain auth refresh never renders protected admin content', async ({
		page,
		request
	}) => {
		await openAdminSession(page, request);
		await page.route('**/api/collections/users/auth-refresh', async (route) => {
			await route.fulfill({ status: 503, json: { error: 'Temporarily unavailable' } });
		});

		await page.goto('/admin/homepage');
		await expect(page.getByText(/Couldn't verify your admin session/)).toBeVisible();
		await expect(page.getByRole('button', { name: 'Save Homepage', exact: true })).toHaveCount(0);
	});
});

import { expect, test, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

const contactsApi = /\/api\/collections\/contact_methods\/records(?:\/([^/?]+))?(?:\?|$)/;
const viewsApi = /\/api\/collections\/views\/records(?:\?|$)/;
const retiredFacet = 'retiredfacet0000';

async function installDraft(page: Page, id: string) {
	await page.addInitScript((editingId) => {
		localStorage.setItem(
			'autosave-admin-contacts',
			JSON.stringify({
				data: {
					type: 'email',
					value: 'draft@example.test',
					label: 'Recovered contact',
					protectionLevel: 'none',
					viewVisibility: { retiredfacet0000: true },
					isPrimary: false,
					sortOrder: 0
				},
				savedAt: Date.now(),
				isEditing: true,
				editingId
			})
		);
	}, id);
}

test.describe('Contact method recovery', () => {
	let token: string;
	let created: string[];
	test.beforeEach(async ({ page, request }) => {
		({ token } = await loginAsAdmin(request));
		created = [];
		await seedAdminSession(page, request);
	});
	test.afterEach(async ({ request }) => {
		for (const id of created.reverse()) {
			const response = await request.delete(
				apiBaseURL + '/api/collections/contact_methods/records/' + id,
				{ headers: { Authorization: token } }
			);
			expect(response.ok(), await response.text()).toBe(true);
		}
	});

	for (const failedRead of ['contacts', 'views'] as const) {
		test(`${failedRead} read failure pauses draft restore and Retry updates the original contact`, async ({
			page,
			request
		}) => {
			const contact = await request.post(apiBaseURL + '/api/collections/contact_methods/records', {
				headers: { Authorization: token },
				data: {
					type: 'email',
					value: 'old@example.test',
					label: 'Contact ' + randomUUID(),
					protection_level: 'none',
					view_visibility: { [retiredFacet]: true }
				}
			});
			expect(contact.ok(), await contact.text()).toBe(true);
			const { id } = await contact.json();
			created.push(id);
			await installDraft(page, id);
			let broken = true;
			await page.route(failedRead === 'contacts' ? contactsApi : viewsApi, (route) =>
				broken && route.request().method() === 'GET'
					? route.fulfill({ status: 503, json: { message: 'Injected read failure' } })
					: route.continue()
			);
			const writes: { method: string; path: string }[] = [];
			page.on('request', (sent) => {
				if (contactsApi.test(sent.url()) && sent.method() !== 'GET')
					writes.push({ method: sent.method(), path: new URL(sent.url()).pathname });
			});
			await page.goto('/admin/contacts');
			const alert = page.getByRole('alert').filter({ hasText: "Couldn't load" });
			await expect(alert).toBeVisible();
			await expect(page.getByRole('button', { name: 'Restore Draft', exact: true })).toHaveCount(0);
			await expect(page.getByRole('button', { name: '+ New Contact', exact: true })).toBeDisabled();
			expect(writes).toEqual([]);
			broken = false;
			await alert.getByRole('button', { name: 'Retry', exact: true }).click();
			await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
			await expect(
				page.getByRole('heading', { name: 'Edit Contact Method', exact: true })
			).toBeVisible();
			await expect(page.locator('#contact-value')).toHaveValue('draft@example.test');
			await page.getByRole('button', { name: 'Deselect All', exact: true }).click();
			await page.getByRole('button', { name: 'Update Contact', exact: true }).click();
			await expect(
				page.getByText('Contact method updated successfully', { exact: true })
			).toBeVisible();
			expect(writes).toEqual([
				{ method: 'PATCH', path: '/api/collections/contact_methods/records/' + id }
			]);
			const stored = await request.get(
				apiBaseURL + '/api/collections/contact_methods/records/' + id,
				{ headers: { Authorization: token } }
			);
			expect(stored.ok(), await stored.text()).toBe(true);
			expect(await stored.json()).toMatchObject({
				value: 'draft@example.test',
				view_visibility: { [retiredFacet]: true }
			});
		});
	}

	test('a missing edited contact keeps its draft instead of becoming a new contact', async ({
		page
	}) => {
		await installDraft(page, 'missingcontact0');
		let creates = 0;
		page.on('request', (sent) => {
			if (contactsApi.test(sent.url()) && sent.method() === 'POST') creates++;
		});
		await page.goto('/admin/contacts');
		await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
		await expect(
			page.getByText('This draft belongs to a contact that is no longer available.', {
				exact: false
			})
		).toBeVisible();
		await expect(page.locator('#contact-value')).toHaveCount(0);
		expect(await page.evaluate(() => localStorage.getItem('autosave-admin-contacts'))).toBeTruthy();
		expect(creates).toBe(0);
	});

	test('an earlier create cannot close or clear the next contact draft', async ({ page }) => {
		let release!: () => void;
		const held = new Promise<void>((resolve) => {
			release = resolve;
		});
		let posted = false;
		await page.route(contactsApi, async (route) => {
			if (route.request().method() !== 'POST') return route.continue();
			created.push(route.request().postDataJSON().id);
			posted = true;
			await held;
			return route.continue();
		});
		await page.goto('/admin/contacts');
		await page.getByRole('button', { name: '+ New Contact', exact: true }).click();
		await page.locator('#contact-value').fill('first@example.test');
		await page.getByRole('button', { name: 'Add Contact', exact: true }).click();
		await expect.poll(() => posted).toBe(true);
		await expect(page.locator('#contact-value')).toBeDisabled();
		await page.getByRole('button', { name: 'Cancel', exact: true }).first().click();
		await page.getByRole('button', { name: '+ New Contact', exact: true }).click();
		await page.locator('#contact-value').fill('second@example.test');
		await page.locator('#contact-label').fill('Next contact draft');
		release();
		await expect(
			page.getByText('Contact method added successfully', { exact: true })
		).toBeVisible();
		await expect(page.locator('#contact-value')).toHaveValue('second@example.test');
		await expect(page.locator('#contact-label')).toHaveValue('Next contact draft');
		await expect
			.poll(async () =>
				page.evaluate(
					() => JSON.parse(localStorage.getItem('autosave-admin-contacts') || '{}').data?.value
				)
			)
			.toBe('second@example.test');
	});

	test('a lost create response is read back by its stable ID without another POST', async ({
		page,
		request
	}) => {
		let posts = 0;
		await page.route(contactsApi, async (route) => {
			if (route.request().method() !== 'POST') return route.continue();
			posts++;
			const data = route.request().postDataJSON();
			const response = await request.post(apiBaseURL + '/api/collections/contact_methods/records', {
				headers: { Authorization: token },
				data
			});
			expect(response.ok(), await response.text()).toBe(true);
			created.push(data.id);
			return route.abort('connectionfailed');
		});
		await page.goto('/admin/contacts');
		await page.getByRole('button', { name: '+ New Contact', exact: true }).click();
		await page.locator('#contact-value').fill('lost-response@example.test');
		await page.getByRole('button', { name: 'Add Contact', exact: true }).click();
		await expect(
			page.getByText('Contact method added successfully', { exact: true })
		).toBeVisible();
		await expect(page.locator('#contact-value')).toHaveCount(0);
		expect(posts).toBe(1);
		expect(created[0]).toMatch(/^[a-z0-9]{15}$/);
	});
});

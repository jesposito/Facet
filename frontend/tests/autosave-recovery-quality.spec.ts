import { expect, test } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { loginAsAdmin, seedAdminSession } from './helpers';

test('a new experience draft is offered on direct load and again after reload', async ({
	page,
	request
}) => {
	await seedAdminSession(page, request);
	await page.addInitScript(() => {
		localStorage.setItem(
			'autosave-admin-experience',
			JSON.stringify({
				data: {
					company: 'Draft company',
					title: 'Draft role',
					visibility: 'private',
					isDraft: true
				},
				savedAt: Date.now(),
				isEditing: false
			})
		);
	});
	const writes: string[] = [];
	page.on('request', (sent) => {
		if (
			new URL(sent.url()).pathname.startsWith('/api/collections/experience/records') &&
			sent.method() !== 'GET'
		)
			writes.push(sent.method());
	});
	await page.goto('/admin/experience');
	for (let attempt = 0; attempt < 2; attempt++) {
		await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
		await expect(page.locator('#company')).toHaveValue('Draft company');
		await expect(page.locator('#title')).toHaveValue('Draft role');
		await page.getByRole('button', { name: 'Settings', exact: true }).click();
		await expect(page.locator('#is_draft')).toBeChecked();
		if (attempt === 0) await page.reload();
	}
	expect(writes).toEqual([]);
});

test('an existing skill draft waits for a successful list read and updates its original record', async ({
	page,
	request
}) => {
	const { token } = await loginAsAdmin(request);
	const headers = { Authorization: token };
	const beforeName = `Recovery skill ${randomUUID()}`;
	const created = await request.post(apiBaseURL + '/api/collections/skills/records', {
		headers,
		data: { name: beforeName, category: 'Tools', proficiency: 'proficient', visibility: 'private' }
	});
	expect(created.ok(), await created.text()).toBe(true);
	const { id } = await created.json();
	try {
		await seedAdminSession(page, request);
		await page.addInitScript((editingId) => {
			localStorage.setItem(
				'autosave-admin-skills',
				JSON.stringify({
					data: {
						name: 'Recovered skill draft',
						category: 'Languages',
						proficiency: 'expert',
						visibility: 'private'
					},
					savedAt: Date.now(),
					isEditing: true,
					editingId
				})
			);
		}, id);
		let failed = true;
		await page.route('**/api/collections/skills/records?*', (route) =>
			failed && route.request().method() === 'GET'
				? route.fulfill({ status: 503, json: { message: 'Injected temporary list failure' } })
				: route.continue()
		);
		const writes: { method: string; path: string }[] = [];
		page.on('request', (sent) => {
			const path = new URL(sent.url()).pathname;
			if (path.startsWith('/api/collections/skills/records') && sent.method() !== 'GET')
				writes.push({ method: sent.method(), path });
		});
		await page.goto('/admin/skills');
		const alert = page.getByRole('alert').filter({ hasText: 'Changes are paused' });
		await expect(alert).toBeVisible();
		await expect(page.getByRole('button', { name: 'Restore Draft', exact: true })).toHaveCount(0);
		expect(
			await page.evaluate(
				() => JSON.parse(localStorage.getItem('autosave-admin-skills')!).editingId
			)
		).toBe(id);
		failed = false;
		await alert.getByRole('button', { name: 'Retry', exact: true }).click();
		await page.getByRole('button', { name: 'Restore Draft', exact: true }).click();
		await expect(page.locator('#name')).toHaveValue('Recovered skill draft');
		await expect(page.locator('#category')).toHaveValue('Languages');
		await page.getByRole('button', { name: 'Update Skill', exact: true }).click();
		await expect(page.getByText('Skill updated successfully', { exact: true })).toBeVisible();
		expect(writes).toEqual([{ method: 'PATCH', path: '/api/collections/skills/records/' + id }]);
		const stored = await request.get(apiBaseURL + '/api/collections/skills/records/' + id, {
			headers
		});
		expect(stored.ok(), await stored.text()).toBe(true);
		expect(await stored.json()).toMatchObject({
			name: 'Recovered skill draft',
			category: 'Languages'
		});
	} finally {
		const deleted = await request.delete(apiBaseURL + '/api/collections/skills/records/' + id, {
			headers
		});
		expect(deleted.ok(), await deleted.text()).toBe(true);
	}
});

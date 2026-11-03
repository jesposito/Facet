import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { apiBaseURL } from './config';
import { getWithRateLimitRetry, loginAsAdmin, seedAdminSession } from './helpers';

type CreatedRecord = { id: string };
type CreatedView = CreatedRecord & { slug: string };
type SavedRecord = {
	name?: string;
	visibility?: string;
	sections?: { section: string; items?: string[] }[];
	view_visibility?: Record<string, boolean>;
};

// Saving a Facet changes that Facet. It must not rescope contacts, opt private
// or draft items in through a bulk action, or leave a password view unusable.
test.describe('Facet editor privacy', () => {
	let created: { collection: string; id: string }[] = [];
	let token: string;

	test.beforeEach(async ({ request }) => {
		({ token } = await loginAsAdmin(request));
	});

	test.afterEach(async ({ request }) => {
		for (const { collection, id } of [...created].reverse()) {
			await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
				headers: { Authorization: token }
			});
		}
		created = [];
	});

	async function createRecord<T extends CreatedRecord = CreatedRecord>(
		request: APIRequestContext,
		collection: string,
		data: Record<string, unknown>
	) {
		const response = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
			headers: { Authorization: token },
			data
		});
		expect(response.ok(), await response.text()).toBe(true);
		const record = (await response.json()) as T;
		created.push({ collection, id: record.id });
		return record;
	}

	async function readRecord(request: APIRequestContext, collection: string, id: string) {
		const response = await request.get(
			`${apiBaseURL}/api/collections/${collection}/records/${id}`,
			{ headers: { Authorization: token } }
		);
		expect(response.ok(), await response.text()).toBe(true);
		return (await response.json()) as SavedRecord;
	}

	async function createProject(
		request: APIRequestContext,
		title: string,
		fields: { visibility: string; is_draft?: boolean }
	) {
		return createRecord(request, 'projects', {
			title,
			slug: `facet-privacy-${randomUUID()}`,
			summary: 'Facet privacy fixture.',
			description: 'Facet privacy fixture.',
			is_draft: false,
			...fields
		});
	}

	async function publicSectionIds(request: APIRequestContext, slug: string, section: string) {
		const response = await getWithRateLimitRetry(request, `${apiBaseURL}/api/view/${slug}/data`);
		expect(response.ok(), await response.text()).toBe(true);
		const body = await response.json();
		return ((body.sections?.[section] ?? []) as { id: string }[]).map((item) => item.id);
	}

	async function savedItems(request: APIRequestContext, viewId: string, section: string) {
		const view = await readRecord(request, 'views', viewId);
		return view.sections?.find((s) => s.section === section)?.items ?? [];
	}

	// Waiting for an item row means the editor has the data a save acts on.
	async function openSection(page: Page, viewId: string, sectionLabel: string, itemLabel: string) {
		await page.goto(`/admin/views/${viewId}`);
		await page.getByRole('button', { name: `Expand ${sectionLabel} section` }).click();
		await expect(page.getByRole('checkbox', { name: itemLabel })).toBeVisible();
	}

	async function saveAndFinish(page: Page) {
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Facet updated successfully')).toBeVisible();
	}

	async function saveRequest(page: Page, viewId: string) {
		const saved = page.waitForResponse(
			(r) =>
				r.request().method() === 'PATCH' &&
				new URL(r.url()).pathname === `/api/collections/views/records/${viewId}`
		);
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		return saved;
	}

	test('saving one Facet leaves contact scope alone', async ({ page, request }) => {
		const suffix = randomUUID().slice(0, 8);
		const contactSection = (items: string[]) => [
			{ section: 'contacts', enabled: true, layout: 'vertical', width: 'full', items }
		];
		const viewA = await createRecord<CreatedView>(request, 'views', {
			name: `Contact Facet A ${suffix}`,
			slug: `contact-a-${suffix}`,
			visibility: 'public',
			is_active: true
		});
		const viewB = await createRecord<CreatedView>(request, 'views', {
			name: `Contact Facet B ${suffix}`,
			slug: `contact-b-${suffix}`,
			visibility: 'public',
			is_active: true
		});
		const contact = (name: string, scope: Record<string, boolean>) =>
			createRecord(request, 'contact_methods', {
				type: 'email',
				value: `${name}-${suffix}@example.test`,
				label: `${name} ${suffix}`,
				protection_level: 'none',
				view_visibility: scope
			});
		const everywhere = await contact('everywhere', {});
		const onlyB = await contact('only-b', { [viewB.id]: true });
		const items = [everywhere.id, onlyB.id];
		for (const view of [viewA, viewB]) {
			const response = await request.patch(
				`${apiBaseURL}/api/collections/views/records/${view.id}`,
				{ headers: { Authorization: token }, data: { sections: contactSection(items) } }
			);
			expect(response.ok(), await response.text()).toBe(true);
		}
		const before = {
			everywhere: (await readRecord(request, 'contact_methods', everywhere.id)).view_visibility,
			onlyB: (await readRecord(request, 'contact_methods', onlyB.id)).view_visibility
		};
		expect(await publicSectionIds(request, viewB.slug, 'contacts')).toEqual(items);

		await seedAdminSession(page, request);
		await openSection(page, viewA.id, 'Contact Methods', `everywhere ${suffix}`);
		await saveAndFinish(page);

		const after = {
			everywhere: (await readRecord(request, 'contact_methods', everywhere.id)).view_visibility,
			onlyB: (await readRecord(request, 'contact_methods', onlyB.id)).view_visibility
		};
		expect(after).toEqual(before);
		expect(await publicSectionIds(request, viewB.slug, 'contacts')).toEqual(items);
		expect(await publicSectionIds(request, viewA.slug, 'contacts')).toEqual([everywhere.id]);
	});

	test('bulk selection adds only public published items', async ({ page, request }) => {
		const suffix = randomUUID().slice(0, 8);
		const chosen = await createProject(request, `Chosen private ${suffix}`, {
			visibility: 'private'
		});
		const first = await createProject(request, `Ordered public ${suffix}`, {
			visibility: 'public'
		});
		const added = await createProject(request, `Added public ${suffix}`, { visibility: 'public' });
		const hidden = await createProject(request, `Other private ${suffix}`, {
			visibility: 'private'
		});
		const draft = await createProject(request, `Draft ${suffix}`, {
			visibility: 'public',
			is_draft: true
		});
		const view = await createRecord<CreatedView>(request, 'views', {
			name: `Bulk Facet ${suffix}`,
			slug: `bulk-${suffix}`,
			visibility: 'public',
			is_active: true,
			sections: [
				{
					section: 'projects',
					enabled: true,
					layout: 'grid-3',
					width: 'full',
					items: [chosen.id, first.id]
				}
			]
		});

		await seedAdminSession(page, request);
		await openSection(page, view.id, 'Projects', `Added public ${suffix}`);
		await page.getByRole('button', { name: 'Select All', exact: true }).click();
		await expect(page.getByRole('checkbox', { name: `Added public ${suffix}` })).toBeChecked();
		await expect(page.getByRole('checkbox', { name: `Other private ${suffix}` })).not.toBeChecked();
		await expect(page.getByRole('checkbox', { name: `Draft ${suffix}` })).not.toBeChecked();
		await saveAndFinish(page);

		const items = await savedItems(request, view.id, 'projects');
		expect(items.slice(0, 2)).toEqual([chosen.id, first.id]);
		expect(items).toContain(added.id);
		expect(items).not.toContain(hidden.id);
		expect(items).not.toContain(draft.id);

		expect((await readRecord(request, 'projects', hidden.id)).view_visibility?.[view.id]).not.toBe(
			true
		);
		expect((await readRecord(request, 'projects', draft.id)).view_visibility?.[view.id]).not.toBe(
			true
		);

		const shown = await publicSectionIds(request, view.slug, 'projects');
		expect(shown.slice(0, 2)).toEqual([chosen.id, first.id]);
		expect(shown).toContain(added.id);
		expect(shown).not.toContain(hidden.id);
		expect(shown).not.toContain(draft.id);
	});

	test('a private item chosen by hand is opted in; a draft is not', async ({ page, request }) => {
		const suffix = randomUUID().slice(0, 8);
		const chosen = await createProject(request, `Manual private ${suffix}`, {
			visibility: 'private'
		});
		const draft = await createProject(request, `Manual draft ${suffix}`, {
			visibility: 'private',
			is_draft: true
		});
		const view = await createRecord<CreatedView>(request, 'views', {
			name: `Manual Facet ${suffix}`,
			slug: `manual-${suffix}`,
			visibility: 'public',
			is_active: true,
			sections: [{ section: 'projects', enabled: true, layout: 'grid-3', width: 'full', items: [] }]
		});

		await seedAdminSession(page, request);
		await openSection(page, view.id, 'Projects', `Manual private ${suffix}`);
		await page.getByRole('checkbox', { name: `Manual private ${suffix}` }).check();
		await page.getByRole('checkbox', { name: `Manual draft ${suffix}` }).check();
		await saveAndFinish(page);

		expect((await readRecord(request, 'projects', chosen.id)).view_visibility?.[view.id]).toBe(
			true
		);
		expect((await readRecord(request, 'projects', draft.id)).view_visibility?.[view.id]).not.toBe(
			true
		);
		expect(await publicSectionIds(request, view.slug, 'projects')).toEqual([chosen.id]);
	});

	test('the editor asks for a password only when one is not configured', async ({
		page,
		request
	}) => {
		const suffix = randomUUID().slice(0, 8);
		const view = await createRecord<CreatedView>(request, 'views', {
			name: `Password Facet ${suffix}`,
			slug: `password-${suffix}`,
			visibility: 'public',
			is_active: true
		});

		await seedAdminSession(page, request);
		await page.goto(`/admin/views/${view.id}`);
		await page.locator('#visibility').selectOption('password');
		await page.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(page.getByText('Password is required for password-protected views')).toBeVisible();
		expect((await readRecord(request, 'views', view.id)).visibility).toBe('public');

		await page.locator('#password').fill('a facet password');
		expect((await saveRequest(page, view.id)).ok()).toBe(true);
		expect((await readRecord(request, 'views', view.id)).visibility).toBe('password');

		// The configured password stays through an edit that doesn't retype it.
		await page.locator('#name').fill(`Renamed Facet ${suffix}`);
		expect((await saveRequest(page, view.id)).ok()).toBe(true);
		expect((await readRecord(request, 'views', view.id)).name).toBe(`Renamed Facet ${suffix}`);

		const access = await getWithRateLimitRetry(
			request,
			`${apiBaseURL}/api/view/${view.slug}/access`
		);
		expect(access.ok(), await access.text()).toBe(true);
		expect((await access.json()).requires_password).toBe(true);
	});
});

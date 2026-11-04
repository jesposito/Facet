import { randomUUID } from 'node:crypto';
import { expect, test, type APIRequestContext, type Page, type Route } from '@playwright/test';
import { apiBaseURL } from './config';
import { loginAsAdmin } from './helpers';

const uiBaseURL =
	process.env.PLAYWRIGHT_BASE_URL || process.env.VITE_POCKETBASE_URL || 'http://localhost:5173';
const storageKey = 'facet_setup_wizard';
const profileRecords = /\/api\/collections\/profile\/records(\/|\?|$)/;
const viewsRecords = /\/api\/collections\/views\/records(\/|\?|$)/;

const savedProfile = {
	id: 'mockprofile0001',
	name: 'Saved Name',
	headline: 'Saved headline',
	summary: 'Saved summary'
};
const savedView = {
	id: 'mockview0000001',
	name: 'Saved Facet',
	slug: 'saved-facet',
	visibility: 'public',
	is_active: true,
	view_count: 0
};

function listBody(items: object[]) {
	return { page: 1, perPage: 100, totalItems: items.length, totalPages: 1, items };
}

function isSetupCheck(route: Route) {
	return new URL(route.request().url()).searchParams.get('perPage') === '100';
}

function viewLookupId(route: Route) {
	return /\/views\/records\/([^/?]+)/.exec(route.request().url())?.[1];
}

function wizard(page: Page) {
	return page.getByRole('dialog', { name: 'Quick Setup' });
}

function resumeButton(page: Page) {
	return page.getByRole('button', { name: 'Resume setup', exact: true });
}

// seedAdminSession rewrites the wizard flag on every page load, which would wipe
// the draft these tests reload to find. This keeps its auth fixture and writes the
// wizard storage once, before the first load.
async function seedOwnerSession(page: Page, request: APIRequestContext, storedWizard?: object) {
	const { token, record: model } = await loginAsAdmin(request);
	await page.context().addCookies([
		{
			name: 'pb_auth',
			value: JSON.stringify({ token, model }),
			url: `${new URL(uiBaseURL).origin}/`
		}
	]);
	await page.addInitScript(
		({ token, model, storageKey, wizardJSON }) => {
			localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
			if (wizardJSON && !sessionStorage.getItem('first-run-seeded')) {
				localStorage.setItem(storageKey, wizardJSON);
				sessionStorage.setItem('first-run-seeded', 'true');
			}
		},
		{
			token,
			model,
			storageKey,
			wizardJSON: storedWizard ? JSON.stringify(storedWizard) : ''
		}
	);
}

async function mockList(page: Page, pattern: RegExp, items: object[]) {
	await page.route(pattern, (route) =>
		route.request().method() === 'GET' ? route.fulfill({ json: listBody(items) }) : route.fallback()
	);
}

async function readStoredWizard(page: Page) {
	const stored = await page.evaluate((key) => localStorage.getItem(key), storageKey);
	return stored ? (JSON.parse(stored) as Record<string, unknown>) : null;
}

function launchButton(page: Page) {
	return page.getByRole('button', { name: 'Launch Profile', exact: true });
}

function launchDraft(slug: string) {
	return {
		dismissed: false,
		draft: {
			currentStep: 3,
			profile: { name: 'Ada Example', headline: 'Teacher' },
			selectedTemplate: 'consulting',
			viewName: '履歴書',
			viewSlug: slug
		}
	};
}

async function mockSavedProfile(page: Page) {
	await page.route(profileRecords, (route) =>
		route.request().method() === 'PATCH'
			? route.fulfill({ json: savedProfile })
			: route.fulfill({ json: listBody([savedProfile]) })
	);
}

// Sends the Facet create to the real server, then drops the reply. Lists stay empty so the
// wizard keeps its place, and lookups by ID reach the server unless a test makes them fail.
async function loseViewResponse(page: Page) {
	const lost = { posts: [] as { id: string }[], patches: 0, failLookup: false };
	await page.route(viewsRecords, async (route) => {
		const method = route.request().method();
		if (method === 'POST') {
			lost.posts.push(route.request().postDataJSON());
			await route.fetch();
			return route.abort('connectionfailed');
		}
		if (method === 'PATCH') lost.patches += 1;
		if (!viewLookupId(route)) return route.fulfill({ json: listBody([]) });
		return lost.failLookup && lost.posts.length > 0
			? route.fulfill({ status: 500, json: { status: 500, message: 'Mock failure', data: {} } })
			: route.continue();
	});
	return lost;
}

async function storedFacets(request: APIRequestContext, token: string, slug: string) {
	const response = await request.get(`${apiBaseURL}/api/collections/views/records`, {
		headers: { Authorization: token },
		params: { filter: `slug="${slug}"` }
	});
	expect(response.ok(), await response.text()).toBe(true);
	return (await response.json()).items as Record<string, any>[];
}

async function removeFacets(request: APIRequestContext, token: string, slug: string) {
	for (const facet of await storedFacets(request, token, slug)) {
		await request.delete(`${apiBaseURL}/api/collections/views/records/${facet.id}`, {
			headers: { Authorization: token }
		});
	}
}

test.describe('first-run setup', () => {
	test('setup check survives other requests for the same collections', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request);
		await mockList(page, profileRecords, []);

		const setupErrors: string[] = [];
		page.on('console', (message) => {
			if (message.text().includes('Failed to check setup wizard')) setupErrors.push(message.text());
		});

		// Hold the setup request until another views request starts, which is the
		// moment the SDK used to cancel it
		let releaseSetupCheck = () => {};
		const otherViewsRequest = new Promise<void>((resolve) => {
			releaseSetupCheck = resolve;
		});
		await page.route(viewsRecords, async (route) => {
			if (route.request().method() !== 'GET') return route.fallback();
			if (isSetupCheck(route)) await otherViewsRequest;
			else releaseSetupCheck();
			await route.fulfill({ json: listBody([]) }).catch(() => undefined);
		});

		await page.goto('/admin');
		await expect(wizard(page)).toBeVisible();
		expect(setupErrors).toEqual([]);
	});

	test('a failed setup check keeps the wizard closed and offers a retry', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request);
		await mockList(page, profileRecords, []);

		let failing = true;
		await page.route(viewsRecords, (route) => {
			if (route.request().method() !== 'GET') return route.fallback();
			if (failing && isSetupCheck(route)) {
				return route.fulfill({
					status: 500,
					json: { status: 500, message: 'Mock failure', data: {} }
				});
			}
			return route.fulfill({ json: listBody([]) });
		});

		await page.goto('/admin');
		const notice = page.getByRole('alert').filter({ hasText: 'whether setup is finished' });
		await expect(notice).toBeVisible();
		await expect(wizard(page)).toBeHidden();

		failing = false;
		await notice.getByRole('button', { name: 'Try again', exact: true }).click();
		await expect(wizard(page)).toBeVisible();
	});

	test('Escape changes nothing while the wizard is closed', async ({ page, request }) => {
		await seedOwnerSession(page, request);
		await mockList(page, profileRecords, [savedProfile]);
		await mockList(page, viewsRecords, [savedView]);

		await page.goto('/admin');
		await expect(page.locator('#main-content')).toBeVisible();
		await page.keyboard.press('Escape');

		await expect(wizard(page)).toBeHidden();
		await expect(resumeButton(page)).toBeHidden();
		expect(await readStoredWizard(page)).toMatchObject({ dismissed: false });
	});

	test('Escape keeps the draft, and Resume brings it back after a reload', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request);
		await mockList(page, profileRecords, []);
		await mockList(page, viewsRecords, []);

		await page.goto('/admin');
		const name = page.locator('#wizard-name');
		await expect(name).toBeFocused();
		await name.fill('Ada Example');
		await page.locator('#wizard-headline').fill('Teacher');
		await page.keyboard.press('Escape');

		await expect(wizard(page)).toBeHidden();
		await expect(resumeButton(page)).toBeFocused();

		await page.reload();
		await expect(resumeButton(page)).toBeVisible();
		await expect(wizard(page)).toBeHidden();

		await resumeButton(page).click();
		await expect(name).toHaveValue('Ada Example');
		await page.getByRole('button', { name: 'Continue', exact: true }).click();
		await expect(page.getByRole('heading', { name: 'Create your first Facet' })).toBeVisible();

		// Resuming clears the dismissal, so a reload lands back on the same step
		await page.reload();
		await expect(page.getByRole('heading', { name: 'Create your first Facet' })).toBeVisible();
		expect(await readStoredWizard(page)).not.toHaveProperty('isOpen');
	});

	test('a non-Latin Facet name can use a separate address and resume it', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request, {
			dismissed: false,
			draft: {
				currentStep: 2,
				profile: { name: 'Ada', headline: 'Teacher' },
				selectedTemplate: 'recruiter',
				viewName: 'Recruiter'
			}
		});
		await mockList(page, profileRecords, []);
		await mockList(page, viewsRecords, []);
		await page.goto('/admin');
		await page.locator('#view-name').fill('履歴書');
		const next = wizard(page).getByRole('button', { name: 'Continue', exact: true });
		await expect(next).toBeDisabled();
		await page.getByLabel('URL Slug', { exact: true }).fill('my-resume');
		await expect(next).toBeEnabled();
		await page.keyboard.press('Escape');
		await page.reload();
		await resumeButton(page).click();
		await expect(page.locator('#view-name')).toHaveValue('履歴書');
		await expect(page.getByLabel('URL Slug', { exact: true })).toHaveValue('my-resume');
		await next.click();
		await expect(page.getByRole('button', { name: 'Launch Profile', exact: true })).toBeVisible();
	});

	test('keyboard focus stays in the wizard and returns to Resume', async ({ page, request }) => {
		// The flag older builds wrote: no auto-open, Resume available
		await seedOwnerSession(page, request, { dismissedPermanently: true });
		await mockList(page, profileRecords, []);
		await mockList(page, viewsRecords, []);

		await page.goto('/admin');
		await expect(resumeButton(page)).toBeVisible();
		await expect(wizard(page)).toBeHidden();

		await resumeButton(page).click();
		await expect(page.locator('#wizard-name')).toBeFocused();
		await expect(page.locator('body')).toHaveCSS('overflow', 'hidden');

		const focusInside = wizard(page).locator(':focus');
		for (const key of ['Tab', 'Tab', 'Tab', 'Tab', 'Tab', 'Shift+Tab', 'Shift+Tab', 'Shift+Tab']) {
			await page.keyboard.press(key);
			await expect(focusInside).toHaveCount(1);
		}

		await page.keyboard.press('Escape');
		await expect(wizard(page)).toBeHidden();
		await expect(resumeButton(page)).toBeFocused();
		await expect(page.locator('body')).not.toHaveCSS('overflow', 'hidden');
	});

	test('a failed Facet save is reported and finishing clears the draft', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request, {
			dismissed: false,
			isOpen: true,
			token: 'must-not-survive',
			draft: {
				currentStep: 3,
				profile: { name: 'Ada Example', headline: 'Teacher' },
				selectedTemplate: 'recruiter',
				viewName: 'Recruiter',
				viewId: 'Not A Record ID'
			}
		});
		await mockList(page, viewsRecords, []);

		const profileUpdates: unknown[] = [];
		await page.route(profileRecords, (route) => {
			if (route.request().method() === 'PATCH') {
				profileUpdates.push(route.request().postDataJSON());
				return route.fulfill({ json: savedProfile });
			}
			return route.fulfill({ json: listBody([savedProfile]) });
		});

		const createIds: string[] = [];
		await page.route(viewsRecords, (route) => {
			if (route.request().method() !== 'POST') {
				return viewLookupId(route)
					? route.fulfill({ status: 404, json: { status: 404, message: 'Missing', data: {} } })
					: route.fallback();
			}
			createIds.push(route.request().postDataJSON().id);
			return createIds.length === 1
				? route.fulfill({
						status: 500,
						json: { status: 500, message: 'Mock failure', data: {} }
					})
				: route.fulfill({ json: savedView });
		});

		await page.goto('/admin');
		const launch = page.getByRole('button', { name: 'Launch Profile', exact: true });
		await expect(launch).toBeVisible();
		const seeded = await readStoredWizard(page);
		expect(seeded).not.toHaveProperty('token');
		expect(seeded).not.toHaveProperty('isOpen');

		await launch.click();
		await expect(wizard(page).getByRole('alert')).toContainText('Your profile is saved');
		await expect(wizard(page)).toBeVisible();
		expect(await readStoredWizard(page)).toMatchObject({
			draft: { profile: { name: 'Ada Example' }, viewId: createIds[0] }
		});

		await launch.click();
		await expect(wizard(page)).toBeHidden();
		expect(await readStoredWizard(page)).toEqual({ dismissed: false, completedWizard: true });
		expect(createIds[0]).toMatch(/^[a-z0-9]{15}$/);
		expect(createIds[1]).toBe(createIds[0]);
		expect(profileUpdates[0]).toEqual({
			name: 'Ada Example',
			headline: 'Teacher',
			summary: 'Saved summary'
		});
	});

	test('a Facet that landed with a lost reply is confirmed, not created twice', async ({
		page,
		request
	}) => {
		const { token } = await loginAsAdmin(request);
		const slug = `lost-reply-${randomUUID()}`;
		await seedOwnerSession(page, request, launchDraft(slug));
		await mockSavedProfile(page);
		const lost = await loseViewResponse(page);

		try {
			await page.goto('/admin');
			await launchButton(page).click();
			await expect(wizard(page)).toBeHidden();

			expect(lost.posts).toHaveLength(1);
			expect(lost.posts[0].id).toMatch(/^[a-z0-9]{15}$/);
			const facets = await storedFacets(request, token, slug);
			expect(facets).toHaveLength(1);
			expect(facets[0]).toMatchObject({
				id: lost.posts[0].id,
				name: '履歴書',
				visibility: 'unlisted',
				is_active: true
			});
			expect(facets[0].sections[0].itemSelection).toBe('all_public');
			expect(await readStoredWizard(page)).toEqual({ dismissed: false, completedWizard: true });
		} finally {
			await removeFacets(request, token, slug);
		}
	});

	test('a failed check keeps the draft, and Retry finds the Facet', async ({ page, request }) => {
		const { token } = await loginAsAdmin(request);
		const slug = `lost-check-${randomUUID()}`;
		await seedOwnerSession(page, request, launchDraft(slug));
		await mockSavedProfile(page);
		const lost = await loseViewResponse(page);
		lost.failLookup = true;

		try {
			await page.goto('/admin');
			await launchButton(page).click();
			await expect(wizard(page).getByRole('alert')).toContainText("couldn't check");
			expect(await readStoredWizard(page)).toMatchObject({
				draft: { viewName: '履歴書', viewSlug: slug, viewId: lost.posts[0].id }
			});

			lost.failLookup = false;
			await wizard(page).getByRole('button', { name: 'Retry', exact: true }).click();
			await expect(wizard(page)).toBeHidden();
			expect(lost.posts).toHaveLength(1);
			expect(await storedFacets(request, token, slug)).toHaveLength(1);
		} finally {
			await removeFacets(request, token, slug);
		}
	});

	test('a reload finds the landed Facet through the saved draft ID', async ({ page, request }) => {
		const { token } = await loginAsAdmin(request);
		const slug = `lost-reload-${randomUUID()}`;
		await seedOwnerSession(page, request, launchDraft(slug));
		await mockSavedProfile(page);
		const lost = await loseViewResponse(page);
		lost.failLookup = true;

		try {
			await page.goto('/admin');
			await launchButton(page).click();
			await expect(wizard(page).getByRole('alert')).toContainText("couldn't check");

			lost.failLookup = false;
			await page.reload();
			await launchButton(page).click();
			await expect(wizard(page)).toBeHidden();
			expect(lost.posts).toHaveLength(1);
			expect(await storedFacets(request, token, slug)).toHaveLength(1);
		} finally {
			await removeFacets(request, token, slug);
		}
	});

	test('a launch that never landed resends the same ID after a reload', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request, launchDraft('never-landed'));
		await mockSavedProfile(page);
		const createIds: string[] = [];
		await page.route(viewsRecords, (route) => {
			if (route.request().method() === 'POST') {
				createIds.push(route.request().postDataJSON().id);
				return createIds.length === 1
					? route.abort('connectionfailed')
					: route.fulfill({ json: savedView });
			}
			return viewLookupId(route)
				? route.fulfill({ status: 404, json: { status: 404, message: 'Missing', data: {} } })
				: route.fulfill({ json: listBody([]) });
		});

		await page.goto('/admin');
		await launchButton(page).click();
		await expect(wizard(page).getByRole('alert')).toContainText("couldn't reach the server");

		await page.reload();
		await launchButton(page).click();
		await expect(wizard(page)).toBeHidden();
		expect(createIds).toHaveLength(2);
		expect(createIds[1]).toBe(createIds[0]);
	});

	test('edits after a lost launch wait for the owner to replace the Facet', async ({
		page,
		request
	}) => {
		const { token } = await loginAsAdmin(request);
		const slug = `lost-edit-${randomUUID()}`;
		await seedOwnerSession(page, request, launchDraft(slug));
		await mockSavedProfile(page);
		const lost = await loseViewResponse(page);
		lost.failLookup = true;

		try {
			await page.goto('/admin');
			const alert = wizard(page).getByRole('alert');
			await launchButton(page).click();
			await expect(alert).toContainText("couldn't check");

			lost.failLookup = false;
			await wizard(page).getByRole('button', { name: 'Back', exact: true }).click();
			await page.locator('#view-name').fill('職務経歴書');
			await wizard(page).getByRole('button', { name: 'Continue', exact: true }).click();
			await launchButton(page).click();
			await expect(alert).toContainText('different details');
			expect(lost.patches).toBe(0);
			expect((await storedFacets(request, token, slug))[0].name).toBe('履歴書');

			await launchButton(page).click();
			await expect(wizard(page)).toBeHidden();
			expect(lost.patches).toBe(1);
			expect(lost.posts).toHaveLength(1);
			const facets = await storedFacets(request, token, slug);
			expect(facets).toHaveLength(1);
			expect(facets[0]).toMatchObject({ name: '職務経歴書', visibility: 'unlisted' });
		} finally {
			await removeFacets(request, token, slug);
		}
	});

	test('the legacy password gate starts blank on the current password', async ({
		page,
		request
	}) => {
		await seedOwnerSession(page, request);
		await page.route('**/api/auth/check-default-password', (route) =>
			route.fulfill({ json: { has_default_password: true } })
		);

		await page.goto('/admin');
		const current = page.locator('#current-password');
		await expect(current).toBeFocused();
		await expect(current).toHaveValue('');
		await expect(current).toHaveAttribute('autocomplete', 'current-password');
		await expect(page.locator('#main-content')).toHaveCount(0);
	});
});

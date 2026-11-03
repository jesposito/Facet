import { expect, type APIRequestContext, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { setTimeout as delay } from 'node:timers/promises';
import { adminEmail, adminPassword, apiBaseURL } from './config';

const uiBaseURL =
	process.env.PLAYWRIGHT_BASE_URL || process.env.VITE_POCKETBASE_URL || 'http://localhost:5173';
const expectedOrigin = new URL(uiBaseURL).origin;
const expectedAPIOrigin = new URL(apiBaseURL).origin;

let cachedAdminAuth:
	{ context: string; promise: Promise<{ token: string; record: any }> } | undefined;

async function authenticateAdmin(request: APIRequestContext) {
	const fixturePath = process.env.FACET_OWNER_AUTH_FILE;
	if (fixturePath) {
		const fixture = JSON.parse(await readFile(fixturePath, 'utf8'));
		if (fixture.origin !== expectedOrigin || fixture.apiOrigin !== expectedAPIOrigin) {
			throw new Error(
				`Owner auth fixture context mismatch: expected UI ${expectedOrigin} and API ${expectedAPIOrigin}`
			);
		}
		if (typeof fixture.token !== 'string' || !fixture.token || !fixture.record) {
			throw new Error('Owner auth fixture must contain a token and record');
		}
		return { token: fixture.token as string, record: fixture.record };
	}

	if (!adminEmail || !adminPassword) {
		throw new Error('ADMIN_EMAIL and ADMIN_PASSWORD are required for admin tests');
	}

	const res = await request.post(`${apiBaseURL}/api/collections/users/auth-with-password`, {
		data: {
			identity: adminEmail,
			password: adminPassword
		}
	});

	expect(res.ok()).toBeTruthy();
	const body = await res.json();
	return { token: body?.token as string, record: body?.record };
}

export async function loginAsAdmin(request: APIRequestContext) {
	const context = `${expectedOrigin}\n${expectedAPIOrigin}`;
	if (!cachedAdminAuth || cachedAdminAuth.context !== context) {
		cachedAdminAuth = { context, promise: authenticateAdmin(request) };
	}
	try {
		return await cachedAdminAuth.promise;
	} catch (error) {
		cachedAdminAuth = undefined;
		throw error;
	}
}

export async function seedAdminSession(page: Page, request: APIRequestContext) {
	const { token, record: model } = await loginAsAdmin(request);
	await page
		.context()
		.addCookies([
			{ name: 'pb_auth', value: JSON.stringify({ token, model }), url: `${expectedOrigin}/` }
		]);
	await page.addInitScript(
		({ token, model }) => {
			localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
			localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
		},
		{ token, model }
	);
	return { token, record: model };
}

export async function fetchFirstView(request: APIRequestContext, token: string) {
	const res = await request.get(`${apiBaseURL}/api/collections/views/records?page=1&perPage=5`, {
		headers: { Authorization: token }
	});

	if (!res.ok()) {
		return null;
	}

	const body = await res.json();
	const items = (body?.items ?? []) as Array<Record<string, any>>;
	return items.length ? items[0] : null;
}

export async function setSiteDesign(
	request: APIRequestContext,
	token: string,
	design: 'classic' | 'soft-premium'
) {
	const res = await request.put(`${apiBaseURL}/api/site-settings`, {
		headers: { Authorization: token },
		data: { design }
	});

	expect(res.ok(), `PUT site design=${design} must succeed`).toBeTruthy();
}

export async function patchFirstProfile(
	request: APIRequestContext,
	token: string,
	data: Record<string, unknown>
) {
	const list = await request
		.get(`${apiBaseURL}/api/collections/profile/records?perPage=1`, {
			headers: { Authorization: token }
		})
		.then((r) => r.json());
	const id = list.items[0].id as string;
	const res = await request.patch(`${apiBaseURL}/api/collections/profile/records/${id}`, {
		headers: { Authorization: token },
		data
	});

	expect(res.ok(), 'PATCH first profile must succeed').toBeTruthy();
}

export async function getWithRateLimitRetry(
	request: APIRequestContext,
	url: string,
	options?: Parameters<APIRequestContext['get']>[1]
) {
	for (let attempt = 0; attempt < 3; attempt++) {
		const response = await request.get(url, options);
		if (response.status() !== 429 || attempt === 2) return response;

		const retryAfter = Number(response.headers()['retry-after']);
		const waitMs =
			Number.isFinite(retryAfter) && retryAfter > 0
				? Math.min(Math.ceil(retryAfter * 1000), 5_000)
				: 1_000;
		await response.dispose();
		await delay(waitMs);
	}
	throw new Error('Rate-limit retry loop ended unexpectedly');
}

export async function createPublicProjectAndPost(request: APIRequestContext, token: string) {
	const suffix = randomUUID();
	const projectSlug = `playwright-project-${suffix}`;
	const postSlug = `playwright-post-${suffix}`;
	const projectRes = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
		headers: { Authorization: token },
		data: {
			title: 'Playwright public project',
			slug: projectSlug,
			summary: 'Disposable public rendering fixture',
			description: 'Disposable public rendering fixture',
			visibility: 'public',
			is_draft: false
		}
	});
	expect(projectRes.ok(), await projectRes.text()).toBeTruthy();
	const project = await projectRes.json();

	try {
		const postRes = await request.post(`${apiBaseURL}/api/collections/posts/records`, {
			headers: { Authorization: token },
			data: {
				title: 'Playwright public post',
				slug: postSlug,
				excerpt: 'Disposable public rendering fixture',
				content: 'Disposable public rendering fixture',
				published_at: '2026-01-02 03:04:05.000Z',
				visibility: 'public',
				is_draft: false
			}
		});
		expect(postRes.ok(), await postRes.text()).toBeTruthy();
		const post = await postRes.json();
		return {
			projectId: project.id as string,
			postId: post.id as string,
			projectSlug,
			postSlug
		};
	} catch (error) {
		await request.delete(`${apiBaseURL}/api/collections/projects/records/${project.id}`, {
			headers: { Authorization: token }
		});
		throw error;
	}
}

export async function deletePublicProjectAndPost(
	request: APIRequestContext,
	token: string,
	fixture: { projectId: string; postId: string }
) {
	for (const [collection, id] of [
		['posts', fixture.postId],
		['projects', fixture.projectId]
	]) {
		await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, {
			headers: { Authorization: token }
		});
	}
}

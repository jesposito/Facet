import { expect, type APIRequestContext } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';

export async function loginAsAdmin(request: APIRequestContext) {
	if (!adminEmail || !adminPassword) {
		throw new Error('ADMIN_EMAIL and ADMIN_PASSWORD are required for admin tests');
	}

	const res = await request.post(`${apiBaseURL}/api/collections/users/auth-with-password`, {
		data: {
			identity: adminEmail,
			password: adminPassword,
		},
	});

	expect(res.ok()).toBeTruthy();
	const body = await res.json();

	return {
		token: body?.token as string,
		record: body?.record,
	};
}

export async function fetchFirstView(request: APIRequestContext, token: string) {
	const res = await request.get(`${apiBaseURL}/api/collections/views/records?page=1&perPage=5`, {
		headers: { Authorization: token },
	});

	if (!res.ok()) {
		return null;
	}

	const body = await res.json();
	const items = (body?.items ?? []) as Array<Record<string, any>>;
	return items.length ? items[0] : null;
}

export async function setSiteDesign(request: APIRequestContext, token: string, design: 'classic' | 'soft-premium') {
	const res = await request.put(`${apiBaseURL}/api/site-settings`, {
		headers: { Authorization: token },
		data: { design },
	});

	expect(res.ok(), `PUT site design=${design} must succeed`).toBeTruthy();
}

export async function patchFirstProfile(request: APIRequestContext, token: string, data: Record<string, unknown>) {
	const list = await request
		.get(`${apiBaseURL}/api/collections/profile/records?perPage=1`, {
			headers: { Authorization: token },
		})
		.then((r) => r.json());
	const id = list.items[0].id as string;
	const res = await request.patch(`${apiBaseURL}/api/collections/profile/records/${id}`, {
		headers: { Authorization: token },
		data,
	});

	expect(res.ok(), 'PATCH first profile must succeed').toBeTruthy();
}

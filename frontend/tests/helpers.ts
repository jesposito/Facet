import { expect, APIRequestContext } from '@playwright/test';
import { adminEmail, adminPassword, apiBaseURL } from './config';

export async function loginAsAdmin(request: APIRequestContext) {
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

	return {
		token: body?.token as string,
		record: body?.record
	};
}

export async function fetchFirstView(request: APIRequestContext, token: string) {
	const res = await request.get(
		`${apiBaseURL}/api/collections/views/records?page=1&perPage=5`,
		{
			headers: { Authorization: token }
		}
	);

	if (!res.ok()) {
		return null;
	}

	const body = await res.json();
	const items = (body?.items ?? []) as Array<Record<string, any>>;
	return items.length ? items[0] : null;
}

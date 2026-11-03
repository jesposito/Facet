import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { publicCacheHeaders } from '$lib/server/public-api';
import {
	checkUnsubscribeToken,
	confirmUnsubscribe,
	isValidUnsubscribeToken,
	unsubscribeHeaders
} from '$lib/server/unsubscribe';

function layoutSetCacheControl(request: Request, locals: App.Locals): boolean {
	const signedIn = Boolean(locals.pb?.authStore?.isValid);
	return 'Cache-Control' in publicCacheHeaders(request, signedIn);
}

export const load: PageServerLoad = async ({ params, fetch, request, locals, setHeaders }) => {
	setHeaders(unsubscribeHeaders(layoutSetCacheControl(request, locals)));

	// /unsubscribe with no token explains that the signed link from the email
	// is required, e.g. for someone whose link was cut off by their mail client.
	if (params.token === undefined) {
		return { check: { state: 'invalid' as const }, hasToken: false };
	}
	return { check: await checkUnsubscribeToken(fetch, params.token), hasToken: true };
};

export const actions: Actions = {
	confirm: async ({ params, fetch }) => {
		if (!isValidUnsubscribeToken(params.token)) {
			return fail(400, { action: 'confirm' as const, result: 'invalid' as const });
		}
		const result = await confirmUnsubscribe(fetch, params.token);
		if (result === 'invalid') return fail(404, { action: 'confirm' as const, result });
		if (result === 'unavailable') return fail(503, { action: 'confirm' as const, result });
		return { action: 'confirm' as const, result };
	}
};

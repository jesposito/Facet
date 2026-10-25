import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	PUBLIC_TURNSTILE_SITE_KEY: { public: true, schema: (input) => input ?? '' }
});

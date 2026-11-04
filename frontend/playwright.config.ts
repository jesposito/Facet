import { defineConfig, devices } from '@playwright/test';

const uiBaseURL =
	process.env.PLAYWRIGHT_BASE_URL || process.env.VITE_POCKETBASE_URL || 'http://localhost:5173';

const sourceUnitTests = [
	'accent-aaa-clamp.spec.ts',
	'access-context.spec.ts',
	'api-key-scopes.spec.ts',
	'calendar-dates.spec.ts',
	'custom-css-contrast.spec.ts',
	'google-analytics.spec.ts',
	'health-route.spec.ts',
	'hooks-server-retry.spec.ts',
	'media-upload-guidance.spec.ts',
	'newsletter-helpers.spec.ts',
	'smoke-proxy.spec.ts',
	'smoke-script.spec.ts',
	'web-manifest.spec.ts'
].map((file) => `**/${file}`);

export default defineConfig({
	testDir: './tests',
	timeout: 30_000,
	expect: {
		timeout: 5_000
	},
	// The soft-premium specs mutate shared single-tenant state (site settings,
	// profile font_pack). Run serially with a single worker so they don't race.
	fullyParallel: false,
	workers: 1,
	retries: process.env.CI ? 2 : 0,
	use: {
		baseURL: uiBaseURL,
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure'
	},
	projects: [
		{
			name: 'source-unit',
			testMatch: sourceUnitTests
		},
		{
			name: 'public-api',
			testMatch: '**/public-api.spec.ts'
		},
		{
			name: 'chromium',
			testIgnore: [...sourceUnitTests, '**/public-api.spec.ts'],
			use: { ...devices['Desktop Chrome'] }
		}
	]
});

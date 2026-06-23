import { defineConfig, devices } from '@playwright/test';

const uiBaseURL =
	process.env.PLAYWRIGHT_BASE_URL ||
	process.env.VITE_POCKETBASE_URL ||
	'http://localhost:5173';

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
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});

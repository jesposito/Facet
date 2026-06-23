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
	// These E2E specs drive ONE single-tenant instance with shared global state
	// (the site_settings.design mode, nav mode, etc.). Parallel workers mutating
	// that single record interleave and contaminate each other — e.g. one spec
	// sets soft-premium while another asserts classic. Run serially so each test
	// owns the instance state for its duration.
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

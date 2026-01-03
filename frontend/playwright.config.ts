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

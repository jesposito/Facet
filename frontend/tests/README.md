# Browser tests

Facet's end-to-end tests use [Playwright](https://playwright.dev/) with Chromium. They exercise public pages, the admin, and the backend API together, so they need a running Facet. They're separate from `make test`, which covers script regressions, Go tests and type checks ([development guide](../../docs/DEV.md#tests-and-checks)).

## Before you run them

1. **Install the browser** once:
   ```bash
   cd frontend
   npx playwright install chromium
   ```
2. **Start Facet** in another terminal with `make dev`. By default the tests expect the frontend at `http://localhost:5173` and the backend at `http://localhost:8090`.
3. **Use a database you don't mind changing.** Many tests create, edit and delete content, and some change site-wide design settings. Don't point them at an instance with real content.
4. **Give the tests an owner account** for anything behind sign-in:
   ```bash
   export ADMIN_EMAIL=admin@example.com
   export ADMIN_PASSWORD='your-changed-password'
   ```
   Sign in once in the browser first and change the default password. The admin dashboard won't do anything else until you have, and some tests need that done. Tests that need an account are skipped when these variables are missing.

Some tests need data that only the development profile provides. If a test reports that no views, projects or posts exist, load it with `make seed-dev`. That command **deletes `pb_data/` and rewrites `.env`** ([details](../../docs/DEV.md#make-targets)).

### Superuser tests

`hero-rail-layout.spec.ts` and `text-color-aaa.spec.ts` change fields through a PocketBase superuser. They default to `admin@localhost.dev` / `admin123`. Either create that superuser in your development database ([how](../../docs/DEV.md#what-make-dev-does)) or point the tests at yours with `PB_SUPERUSER_EMAIL` and `PB_SUPERUSER_PASSWORD`.

## Running

From `frontend/`:

| Command                                      | Runs                                                 |
| -------------------------------------------- | ---------------------------------------------------- |
| `npm test` (or `npm run test:e2e`)           | Every spec                                           |
| `npm run test:public`                        | `public-api.spec.ts` and `seo-and-errors.spec.ts`    |
| `npm run test:admin`                         | `admin-flows.spec.ts` and `media-management.spec.ts` |
| `npx playwright test tests/security.spec.ts` | One file                                             |
| `npx playwright test -g "share token"`       | Tests whose names match                              |
| `npm run test:ui`                            | Playwright's interactive runner                      |
| `npm run test:headed`                        | With a visible browser                               |
| `npm run test:debug`                         | Step through with the inspector                      |

Tests run one at a time (`workers: 1`) because several change shared site settings ([config](../playwright.config.ts)). On CI, failed tests are retried twice. Traces, screenshots and videos are kept for failures under `test-results/`. Open a trace with `npx playwright show-trace <path-to-trace.zip>`.

## Configuration

| Variable                                      | Default                                                         | Used for                    |
| --------------------------------------------- | --------------------------------------------------------------- | --------------------------- |
| `PLAYWRIGHT_BASE_URL`                         | `VITE_POCKETBASE_URL` if set, otherwise `http://localhost:5173` | Where pages are loaded from |
| `API_BASE_URL`                                | `POCKETBASE_URL` if set, otherwise `http://localhost:8090`      | Direct backend calls        |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD`               | (none)                                                          | Owner sign-in               |
| `PB_SUPERUSER_EMAIL`, `PB_SUPERUSER_PASSWORD` | `admin@localhost.dev`, `admin123`                               | Superuser-only tests        |
| `CI`                                          | (unset)                                                         | Enables retries             |

`PLAYWRIGHT_BASE_URL` falls back to `VITE_POCKETBASE_URL`, which is a backend address in normal use. If you've set `VITE_POCKETBASE_URL`, set `PLAYWRIGHT_BASE_URL` explicitly too.

## What CI runs

The [Security and tests workflow](../../.github/workflows/security-tests.yml) builds the frontend, then [`scripts/ci/run-browser-smoke.sh`](../../scripts/ci/run-browser-smoke.sh):

- builds the backend and starts it on a fresh temporary database with the development seed
- starts the built SvelteKit server, plus a small Node proxy ([`smoke-proxy.mjs`](../../scripts/ci/smoke-proxy.mjs)) in place of Caddy
- waits for `/health`, signs in with the default password, and changes it, as a real owner would
- runs `public-api.spec.ts` first (before browser traffic uses up the anonymous rate-limit allowance), then `portfolio-portability.spec.ts`, then `admin-flows.spec.ts`, `booking-shortcodes.spec.ts`, `dependency-security.spec.ts` and `back-navigation.spec.ts`, then `project-experiences.spec.ts`

Production rate limits stay switched on during these runs. The other specs are run by developers locally.

## What's covered

| Spec                                                                                                                                                 | Covers                                                                           |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `public-api.spec.ts`                                                                                                                                 | Public endpoints, feeds (RSS, iCal), AI status endpoints                         |
| `seo-and-errors.spec.ts`                                                                                                                             | Sitemap, robots, canonical URLs, Open Graph, Twitter cards, JSON-LD, error pages |
| `admin-flows.spec.ts`                                                                                                                                | Owner sign-in, views, share-link lifecycle                                       |
| `security.spec.ts`                                                                                                                                   | XSS handling, path traversal, authentication requirements, input validation      |
| `media-management.spec.ts`, `media-replace.spec.ts`, `media-upload-guidance.spec.ts`                                                                 | Media listing, orphan detection and bulk delete, replacement, upload guidance    |
| `portfolio-portability.spec.ts`                                                                                                                      | Rich content (embeds, long links, code) rendering responsively, and export       |
| `project-experiences.spec.ts`                                                                                                                        | Project experiences in the editor and on public pages                            |
| `booking-shortcodes.spec.ts`                                                                                                                         | Booking embeds written as Markdown shortcodes                                    |
| `back-navigation.spec.ts`                                                                                                                            | Back-button journeys in the built app                                            |
| `live-preview.spec.ts`                                                                                                                               | The admin live preview                                                           |
| `testimonial-lists.spec.ts`                                                                                                                          | Testimonial lists                                                                |
| `dependency-security.spec.ts`                                                                                                                        | The guarded dependency forks                                                     |
| `backport-qa.spec.ts`                                                                                                                                | Accessibility, keyboard reordering, editor dialog and navigation checks          |
| `custom-css-contrast.spec.ts`, `text-color-aaa.spec.ts`, `accent-aaa-clamp.spec.ts`                                                                  | Colour contrast rules                                                            |
| `soft-premium-*.spec.ts`, `hero-rail-layout.spec.ts`, `footer-cta-toggle.spec.ts`, `admin-typography.spec.ts`, `admin-header-mobile-classes.spec.ts` | Design system, layouts and typography                                            |
| `health-route.spec.ts`, `hooks-server-retry.spec.ts`, `web-manifest.spec.ts`, `focus-after-remove.spec.ts`                                           | Helper and server-hook behaviour                                                 |

## Writing a test

Put new specs in this folder as `<topic>.spec.ts`. Use [`config.ts`](config.ts) for URLs and credentials and [`helpers.ts`](helpers.ts) for common steps (`loginAsAdmin`, `fetchFirstView`, `setSiteDesign`, `patchFirstProfile`).

A public API check:

```ts
import { test, expect } from '@playwright/test';
import { apiBaseURL } from './config';

test('public posts endpoint responds', async ({ request }) => {
	const response = await request.get(`${apiBaseURL}/api/posts`);
	expect(response.ok()).toBeTruthy();
});
```

A test that needs the owner:

```ts
import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';
import { apiBaseURL, adminEmail, adminPassword } from './config';

test.describe('my admin feature', () => {
	test.skip(!adminEmail || !adminPassword, 'Admin credentials required');

	test('does the thing', async ({ request }) => {
		const { token } = await loginAsAdmin(request);
		const response = await request.get(`${apiBaseURL}/api/export?format=json`, {
			headers: { Authorization: token }
		});
		expect(response.ok()).toBeTruthy();
	});
});
```

Good habits:

- Skip clearly with `test.skip()` when a prerequisite is missing, rather than failing obscurely.
- Clean up what you create, and restore any site setting you change.
- Test the failure path as well as the happy one.
- Keep URLs and credentials in `config.ts`, not in the spec.

## Troubleshooting

**Connection refused.** Check both servers are up: `curl http://localhost:5173` and `curl http://localhost:8090/api/health`.

**Lots of 429 responses.** The tests share the anonymous rate-limit allowance. Run the public API tests on their own, or restart the backend to reset the limits.

**Admin tests are skipped.** Set `ADMIN_EMAIL` and `ADMIN_PASSWORD`.

**Sign-in works in the browser but not in tests.** The tests sign in with the password you've set. Make sure `ADMIN_PASSWORD` matches, and that `ADMIN_EMAIL` is allowed by `ADMIN_EMAILS` if you've set that.

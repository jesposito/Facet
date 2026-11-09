# Playwright

Checked **2026-11-10**. `@playwright/test` is locked at **1.64.1**; latest is **1.65.0**. Facet runs source-unit, public-api and Chromium projects with one worker. After updating, install matching Chromium with `npx playwright install --with-deps chromium` and run `bash scripts/ci/run-browser-smoke.sh`.

Sources: [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/playwright.config.ts`](../frontend/playwright.config.ts), [`frontend/tests/README.md`](../frontend/tests/README.md), [npm registry latest](https://registry.npmjs.org/%40playwright%2Ftest/latest), checked 2026-11-10.

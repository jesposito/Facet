# Frontend test and lint tooling

Checked **2026-11-11**. Locked versions come from `frontend/package-lock.json`; latest versions come from official npm registry records on the same date.

| Package                  | Locked  | Latest  | Status                 |
| ------------------------ | ------- | ------- | ---------------------- |
| `@playwright/test`       | 1.64.1  | 1.65.0  | minor update available |
| `@axe-core/playwright`   | 4.14.0  | 4.14.0  | current                |
| `eslint`                 | 10.14.0 | 10.14.0 | current                |
| `@eslint/js`             | 10.0.1  | 10.0.1  | current                |
| `eslint-plugin-svelte`   | 3.24.0  | 3.24.1  | patch update available |
| `eslint-config-prettier` | 10.1.8  | 10.1.8  | current                |
| `globals`                | 17.14.0 | 17.14.0 | current                |
| `prettier`               | 3.9.10  | 3.9.10  | current                |
| `prettier-plugin-svelte` | 4.1.1   | 4.1.1   | current                |
| `svelte-check`           | 4.7.8   | 4.7.8   | current                |
| `typescript-eslint`      | 8.72.1  | 8.72.2  | patch update available |

The flat ESLint configuration combines recommended JavaScript, TypeScript and Svelte rules with Prettier compatibility. `npm run lint` checks both Prettier formatting and ESLint. Existing warnings are tracked separately in `Facet-9vt.70`; a clean dependency audit does not imply a warning-free lint run.

Playwright runs one worker because tests mutate shared owner state. CI uses two retries and retains traces, screenshots and videos on failure. The canonical full run is `bash scripts/ci/run-browser-smoke.sh`, not `make test`.

After a Playwright update, install the matching Chromium build with `npx playwright install --with-deps chromium` before running the smoke suite.

Registry sources: [Playwright](https://registry.npmjs.org/%40playwright%2Ftest/latest), [axe-core Playwright](https://registry.npmjs.org/%40axe-core%2Fplaywright/latest), [ESLint](https://registry.npmjs.org/eslint/latest), [@eslint/js](https://registry.npmjs.org/%40eslint%2Fjs/latest), [eslint-plugin-svelte](https://registry.npmjs.org/eslint-plugin-svelte/latest), [eslint-config-prettier](https://registry.npmjs.org/eslint-config-prettier/latest), [globals](https://registry.npmjs.org/globals/latest), [Prettier](https://registry.npmjs.org/prettier/latest), [prettier-plugin-svelte](https://registry.npmjs.org/prettier-plugin-svelte/latest), [svelte-check](https://registry.npmjs.org/svelte-check/latest), [typescript-eslint](https://registry.npmjs.org/typescript-eslint/latest). Repository sources: [`frontend/eslint.config.js`](../frontend/eslint.config.js), [`frontend/playwright.config.ts`](../frontend/playwright.config.ts), [`frontend/tests/README.md`](../frontend/tests/README.md), [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml). All checked 2026-11-11.

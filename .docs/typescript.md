# TypeScript

Checked **2026-11-10**.

| Package             | Locked  | Latest         | Status                                         |
| ------------------- | ------- | -------------- | ---------------------------------------------- |
| `typescript`        | 6.0.3   | 7.0.2          | new major intentionally blocked                |
| `typescript-eslint` | 8.72.1  | 8.72.2         | patch update available                         |
| `@types/node`       | 24.19.2 | 26.6.6 overall | aligned with Node 24; retain the runtime major |
| `@types/qrcode`     | 1.5.6   | 1.5.6          | current                                        |

Facet enables TypeScript strict mode and includes `src`, Vite configuration, Playwright configuration and tests. The normal static check is `cd frontend && npm run check`.

Dependabot ignores TypeScript 7 because `svelte-check` 4 does not support its compiler API. Keep TypeScript on 6 until the Svelte checking toolchain supports 7, then test the migration together with SvelteKit, typescript-eslint and IDE tooling. `Facet-ehg` tracks this planned compatibility work. The typescript-eslint 8.72.2 patch can be evaluated independently.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/tsconfig.json`](../frontend/tsconfig.json), [`.github/dependabot.yml`](../.github/dependabot.yml), [`CONTRIBUTING.md`](../CONTRIBUTING.md), npm registry latest endpoints for [TypeScript](https://registry.npmjs.org/typescript/latest), [typescript-eslint](https://registry.npmjs.org/typescript-eslint/latest), [@types/node](https://registry.npmjs.org/%40types%2Fnode/latest) and [@types/qrcode](https://registry.npmjs.org/%40types%2Fqrcode/latest), all checked 2026-11-10.

# Svelte and SvelteKit

Checked **2026-11-11**.

| Package                        | Locked | Latest | Status                 |
| ------------------------------ | ------ | ------ | ---------------------- |
| `svelte`                       | 5.57.3 | 5.57.3 | current                |
| `@sveltejs/kit`                | 3.1.1  | 3.2.0  | minor update available |
| `@sveltejs/adapter-node`       | 6.0.1  | 6.0.2  | patch update available |
| `@sveltejs/vite-plugin-svelte` | 7.3.1  | 7.3.1  | current                |
| `svelte-check`                 | 4.7.8  | 4.7.8  | current                |

Facet configures SvelteKit inline in `frontend/vite.config.ts` through the `sveltekit()` plugin. The Node adapter writes `build/`, enables precompression and uses an empty environment prefix. Production starts `node build/index.js` on port 3000.

The application uses Svelte 5 patterns, TypeScript strict mode and a generated `$app/tsconfig`. Run `npm run check` after frontend changes; it performs `svelte-kit sync` followed by `svelte-check`. A production `npm run build` also generates bundled-font licence notices before Vite builds the app.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/vite.config.ts`](../frontend/vite.config.ts), [`frontend/tsconfig.json`](../frontend/tsconfig.json), [`docker/start.sh`](../docker/start.sh), npm registry latest endpoints for [Svelte](https://registry.npmjs.org/svelte/latest), [SvelteKit](https://registry.npmjs.org/%40sveltejs%2Fkit/latest), [adapter-node](https://registry.npmjs.org/%40sveltejs%2Fadapter-node/latest), [vite-plugin-svelte](https://registry.npmjs.org/%40sveltejs%2Fvite-plugin-svelte/latest) and [svelte-check](https://registry.npmjs.org/svelte-check/latest), all checked 2026-11-11.

# Vite

Checked **2026-11-14**. Locked **8.3.5**; latest **8.3.8**. Status: **patch updates available**. The npm registry's latest entry was not marked deprecated.

Vite drives the SvelteKit development server and production build. Facet's configuration:

- registers SvelteKit, `vitePreprocess()` and the Node adapter;
- defines `__APP_VERSION__` from `FACET_VERSION` or `git describe`;
- copies the root changelog into frontend static assets during the build;
- proxies `/api` and `/_` to `POCKETBASE_URL`, defaulting to `http://127.0.0.1:8090`;
- rejects proxy targets with credentials, a path, query, fragment or a non-HTTP(S) scheme.

Vite 8 uses Rolldown/Oxc. Facet does not set `build.target` or legacy `build.rollupOptions` in its Vite configuration. Review the official Vite 8 migration guide before changing build inputs, browser targets or plugin hooks.

After updating to 8.3.8, run `npm run check`, `npm run build` and the canonical browser smoke script. The development proxy and server-side `POCKETBASE_URL` behaviour deserve a native and Docker development smoke check as well.

Sources: [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/vite.config.ts`](../frontend/vite.config.ts), [npm registry latest](https://registry.npmjs.org/vite/latest), [official Vite 8 migration guide](https://vite.dev/guide/migration), [Vite 8.3.8 release](https://github.com/vitejs/vite/releases/tag/v8.3.8), all checked 2026-11-14.

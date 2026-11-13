# npm

Checked **2026-11-14**.

The host's nvm default is Node 24.22.3 with **npm 11.19.1**, which is also the version bundled by the current Node 24 release. The npm registry reports **12.2.2** as latest overall. Facet uses lockfile version 3 and `npm ci` in CI and Docker builds. Keep npm aligned with the Node runtime unless a separate major migration is planned.

Sources: local `npm --version`, [`frontend/package-lock.json`](../frontend/package-lock.json), [`docker/Dockerfile`](../docker/Dockerfile), [Node release index](https://nodejs.org/dist/index.json), [npm registry latest](https://registry.npmjs.org/npm/latest), checked 2026-11-14.

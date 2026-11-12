# Node.js and npm

Checked **2026-11-12**.

- Facet selects **Node.js 24** in `.nvmrc`, requires `>=24 <25` in `frontend/package.json`, and uses `node:24-alpine` for the frontend build. The final image installs Node 24 from NodeSource. The host reports **24.22.1** with **npm 11.19.1**.
- The official Node release index reports **24.22.3** as the latest Node 24 release and **26.13.1** as the latest Node 26 LTS release. Both are security releases. Node 24.22.3 bundles npm 11.19.1. The npm registry reports **12.2.2** as the latest npm overall.
- Node 24 (Krypton) entered maintenance on 2026-10-20 and is supported until **2028-04-30**. Node 26 is the newer LTS line, supported until 2029-04-30. Facet deliberately ignores Node and `@types/node` major bumps in routine Dependabot updates.
- `@types/node` is locked at **24.19.2**. The latest overall is 26.6.6, but the types major should stay aligned with the selected Node runtime.
- Production base images are digest-pinned, but the NodeSource apt install does not select an exact Node patch. Development images and the Compose frontend use the floating `node:24-alpine` tag.

## Security action

Node 24.22.3, released 2026-11-11, fixes High HTTP/2 and TLS issues, including a remotely reachable TLS server crash (**CVE-2026-126051**). The host remains on 24.22.1, and v3.8.9 shipped Node 24.22.2. NodeSource publishes **24.22.3-1nodesource1** for both amd64 and arm64.

**Fixed in [PR #536](https://github.com/jesposito/Facet/pull/536):** the frontend builder digest selects the 24.22.3 image and the runtime rebuild installs NodeSource's fixed package. Publish the PR image, verify the packaged runtime reports 24.22.3, and update the host through its normal channel. Keep the runtime and types on major 24 until a deliberate migration; do not install npm 12 merely because it is the latest registry release.

Sources: [`.nvmrc`](../.nvmrc), [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`docker/Dockerfile`](../docker/Dockerfile), [`docker-compose.dev.yml`](../docker-compose.dev.yml), [official Node release index](https://nodejs.org/dist/index.json), [Node 24.22.3 release notes](https://nodejs.org/en/blog/release/v24.22.3), [Node 26.13.1 release notes](https://nodejs.org/en/blog/release/v26.13.1), [official Node support schedule](https://raw.githubusercontent.com/nodejs/Release/main/schedule.json), [npm registry latest](https://registry.npmjs.org/npm/latest), [`@types/node` registry latest](https://registry.npmjs.org/%40types%2Fnode/latest), NodeSource amd64 and arm64 package indexes under `https://deb.nodesource.com/node_24.x/dists/nodistro/main/`, all checked 2026-11-12.

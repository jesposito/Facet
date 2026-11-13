# PocketBase JavaScript SDK

Checked **2026-11-14**. Locked **0.28.2**; latest **0.29.0**. Status: **minor update available**. Facet's backend is PocketBase 0.40.6. New frontend code should use `pb.authStore.record`, as the current client does, rather than the older `model` property.

SDK 0.29.0 automatically clears `pb.authStore` when a token expires or is externally invalidated, including after a 401 response. `pb.autoClearAuthStore(false)` disables that behaviour. Review this change before updating.

After updating, run `npm run check` and the canonical browser smoke suite so login, expired-token, 401, realtime and record API paths are exercised together.

Sources: [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/src/lib/pocketbase.ts`](../frontend/src/lib/pocketbase.ts), [`docs/DEV.md`](../docs/DEV.md), [npm registry latest](https://registry.npmjs.org/pocketbase/latest), [official 0.29.0 package tarball and changelog](https://registry.npmjs.org/pocketbase/-/pocketbase-0.29.0.tgz), checked 2026-11-14.

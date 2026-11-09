# libheif / heif-convert

Checked **2026-11-10**.

The production runtime installs Debian's `libheif-examples` to provide `heif-convert` for HEIC image handling. The version is selected from Bookworm apt at build time rather than pinned independently. Include HEIC conversion and an image vulnerability scan in runtime refresh checks.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md), [Debian security information](https://www.debian.org/security/), checked 2026-11-10.

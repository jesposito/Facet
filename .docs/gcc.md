# GCC / cgo

Checked **2026-11-10**.

Facet needs a C compiler for go-fitz's cgo build. The host reports **GCC 13.3.0**; the production backend builder installs Debian's `gcc`, and the Alpine development image installs `gcc` plus `musl-dev`. go-fitz bundles MuPDF, so no separate MuPDF package is required.

Sources: local `gcc --version`, [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), [`docs/DEV.md`](../docs/DEV.md), checked 2026-11-10.

# go-fitz

Checked **2026-11-10**.

`github.com/gen2brain/go-fitz` is selected at **v1.28.2**, the latest Go module release. It provides PDF text extraction for résumé import, requires cgo and bundles MuPDF. Alpine development builds need `GOFLAGS=-tags=musl`; the Debian production build must not use that tag. A separate MuPDF installation is not required.

Sources: [`backend/go.mod`](../backend/go.mod), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), [`docs/DEV.md`](../docs/DEV.md), [Go module proxy latest](https://proxy.golang.org/github.com/gen2brain/go-fitz/@latest), checked 2026-11-10.

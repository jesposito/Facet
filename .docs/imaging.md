# disintegration/imaging

Checked **2026-11-10**.

`github.com/disintegration/imaging` is selected at **v1.6.2**, which remains the latest Go module release. GHSA-q7pp-wcgr-pffx covers a crafted-TIFF panic and has no patched upstream release. Facet disables TIFF thumbnail generation and its latest observed backend reachable vulnerability scan passed; that does not fix the upstream advisory. `Facet-aqv` tracks replacement or removal.

Sources: [`backend/go.mod`](../backend/go.mod), [Go module proxy latest](https://proxy.golang.org/github.com/disintegration/imaging/@latest), [GitHub advisory](https://github.com/advisories/GHSA-q7pp-wcgr-pffx), [`security-advisories.md`](security-advisories.md), checked 2026-11-10.

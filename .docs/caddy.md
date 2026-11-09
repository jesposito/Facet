# Caddy

Checked **2026-11-10**.

The `f3d7a14` baseline builds **Caddy 2.11.7** from `docker/caddy` with Go 1.27.3. The Go module proxy reports **2.11.9** as latest. Caddy 2.11.8 and 2.11.9 contain security and bug fixes covering reverse-proxy interim responses, host/path handling, file serving, FastCGI and TLS configuration merging. Facet's exact configuration does not use every affected feature, so applicability must be assessed per fix rather than assumed from the release count.

The baseline proxy module graph also selects `github.com/klauspost/compress` **1.20.1**. Scheduled CI run [42568077807](https://github.com/jesposito/Facet/actions/runs/42568077807) failed reachable `govulncheck` analysis through `huff0` and `fse` on 2026-11-09. GHSA-fpmh-frq8-7rj8 / GO-2026-6865 describes a crafted zstd frame that can panic decoding. **1.20.2 fixes it; 1.20.4 is latest.** Caddy 2.11.9's own `go.mod` still selects compress 1.20.1, so upgrading Caddy alone does not clear this finding. Follow-up: `Facet-cyi`.

[PR #528](https://github.com/jesposito/Facet/pull/528) was squash-merged as **`fde26de`**, selecting Caddy 2.11.9 and compress 1.20.2 while keeping smallstep/crypto 0.91.0 and the local go-jose replacement. Local proxy tests, vet, `govulncheck`, CGO-disabled build, Caddyfile validation and the go-jose suite passed. `go mod why` confirms v3 remains through `crypto/jose`. The full security workflow and both architecture image scan/smoke jobs passed. `Facet-cyi` is closed; ARM64 publication permission remains separate in `Facet-dcy`.

`go.step.sm/crypto` **0.92.0** is available and requires Go 1.26, which Facet's Go 1.27.3 satisfies. It selects go-jose/v4 4.1.9, but Caddy 2.11.9 and smallstep/certificates 0.30.2 still select go-jose/v3. A compatibility build on 2026-11-10 failed with certificates 0.30.2: `jwt.Builder.CompactSerialize` is undefined in `provisioner/aws.go`, and `jose.Expected.Audience` is missing in the Azure and OIDC provisioners. **Keep 0.91.0 plus the local v3 patch until smallstep/certificates supports the 0.92 API; do not take 0.92 as a routine update.**

The local go-jose v3.0.5 replacement backports the RSA bounds fix for GHSA-mgcc-cww5-fp2q only. It does not claim to fix GHSA-4w3g-jfj2-cw2q or GHSA-vvm8-gf78-28g4. See `security-advisories.md`.

After changes, run `go test ./...` and `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` in `docker/caddy`, then the full security workflow and packaged-image scan.

Sources: [`docker/caddy/go.mod`](../docker/caddy/go.mod), [`docker/caddy/third_party/go-jose-v3/FACET-PATCH.md`](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md), [Caddy module latest](https://proxy.golang.org/github.com/caddyserver/caddy/v2/@latest), [Caddy 2.11.8](https://github.com/caddyserver/caddy/releases/tag/v2.11.8), [Caddy 2.11.9](https://github.com/caddyserver/caddy/releases/tag/v2.11.9), [Caddy 2.11.9 go.mod](https://proxy.golang.org/github.com/caddyserver/caddy/v2/@v/v2.11.9.mod), [compress 1.20.2 fix](https://github.com/klauspost/compress/releases/tag/v1.20.2), [compress latest](https://proxy.golang.org/github.com/klauspost/compress/@latest), [smallstep/crypto 0.92.0 go.mod](https://proxy.golang.org/go.step.sm/crypto/@v/v0.92.0.mod), all checked 2026-11-10.

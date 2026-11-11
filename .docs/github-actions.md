# GitHub Actions

Checked **2026-11-11** against `.github/workflows/` and official GitHub releases.

| Action                            | Selected                | Latest | Note                      |
| --------------------------------- | ----------------------- | ------ | ------------------------- |
| `actions/checkout`                | v7                      | 7.0.1  | current major             |
| `actions/setup-go`                | v7                      | 7.0.0  | current major             |
| `actions/setup-node`              | v7                      | 7.1.0  | current major             |
| `actions/upload-artifact`         | v7                      | 7.0.2  | current major             |
| `actions/download-artifact`       | v8                      | 8.0.2  | current major             |
| `actions/github-script`           | v9                      | 9.0.0  | current major             |
| `actions/attest-build-provenance` | v4                      | 4.2.2  | current major             |
| `docker/setup-buildx-action`      | v4                      | 4.4.1  | current major             |
| `docker/login-action`             | v4                      | 4.8.0  | current major             |
| `docker/metadata-action`          | v6                      | 6.2.0  | current major             |
| `docker/build-push-action`        | v7                      | 7.4.0  | current major             |
| `sigstore/cosign-installer`       | v3                      | 4.1.3  | new major available       |
| `aquasecurity/trivy-action`       | SHA `ed142fd…` = 0.36.0 | 0.36.0 | current and commit-pinned |

`COSIGN_VERSION` selects cosign **2.4.1**; the latest cosign release is **3.1.4**. Review the installer and binary major upgrades together with signing and verification tests.

Except for Trivy, workflow actions use movable major-version tags rather than immutable commit SHAs. That is the current repository policy; this inventory does not silently treat those tags as exact minor versions. `ubuntu-latest` is also a moving runner image.

The `codeql.yml` workflow is only a placeholder because GitHub's default CodeQL setup owns that analysis. [PR #531](https://github.com/jesposito/Facet/pull/531), merged as `74c3e31` and released in v3.8.6, removed the retired GitHub Models call documented in `github-models.md`.

Current `main` runs `go test ./...` in the proxy security step. [PR #533](https://github.com/jesposito/Facet/pull/533) expands this to `go test ./... github.com/go-jose/go-jose/v3/...`, so the local replacement's regression tests run alongside Caddy's tests. The subsequent `govulncheck` command is unchanged.

Push run [42706080159](https://github.com/jesposito/Facet/actions/runs/42706080159) exposed intermittent GHCR denials during overlapping branch builds. Later releases published both architectures successfully, including [v3.8.7 run 42754619959](https://github.com/jesposito/Facet/actions/runs/42754619959). No configuration change was needed and `Facet-dcy` is closed. See `ghcr.md`.

Sources: [workflow definitions](../.github/workflows), [cosign-installer 4.1.3](https://github.com/sigstore/cosign-installer/releases/tag/v4.1.3), [cosign 3.1.4](https://github.com/sigstore/cosign/releases/tag/v3.1.4), [Trivy action 0.36.0](https://github.com/aquasecurity/trivy-action/releases/tag/v0.36.0), and each action's official `https://github.com/<owner>/<repo>/releases/tag/<tag>` release listed in the table, all checked 2026-11-11. Individual action notes are linked from `INDEX.md`.

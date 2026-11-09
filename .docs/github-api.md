# GitHub REST API

Checked **2026-11-10**.

Facet's GitHub import service uses `https://api.github.com` with a 30-second HTTP client. It reads repository metadata, languages, README content and latest-release information for project imports. This is separate from both GitHub Actions and the retired GitHub Models service.

The source selects the managed REST endpoint rather than pinning a client SDK version. No authenticated GitHub account or rate-limit state was inspected. Preserve the existing optional-token and error-handling behaviour when changing the service.

Sources: [`backend/services/github.go`](../backend/services/github.go), [official GitHub REST API documentation](https://docs.github.com/en/rest), checked 2026-11-10.

# GitHub REST API

Checked **2026-11-11**.

Facet's GitHub import service uses `https://api.github.com` with a 30-second HTTP client. It reads repository metadata, languages, README content and latest-release information for project imports. This is separate from both GitHub Actions and the retired GitHub Models service.

Repository and latest-release requests explicitly select API version `2022-11-28`; languages and README requests omit the header and currently default to that version. GitHub lists **2026-03-10** as current and supports `2022-11-28` until **2028-03-10**. Review and test the newer version across all import and update paths before migrating; there is a long support window.

The source selects the managed REST endpoint rather than pinning a client SDK version. No authenticated GitHub account or rate-limit state was inspected. Preserve the existing optional-token and error-handling behaviour when changing the service.

Sources: [`backend/services/github.go`](../backend/services/github.go), [official GitHub REST API versions](https://docs.github.com/en/rest/about-the-rest-api/api-versions), checked 2026-11-11.

# sigstore/cosign-installer

Checked **2026-11-10**. The Docker publish workflow selects the floating **v3** major tag; latest installer release is **v4.1.3**. Facet separately pins `COSIGN_VERSION` 2.4.1. Review installer and binary major upgrades together with signing and verification tests.

Sources: [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml), [official installer v4.1.3 release](https://github.com/sigstore/cosign-installer/releases/tag/v4.1.3), checked 2026-11-10.

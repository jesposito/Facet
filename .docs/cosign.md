# Cosign

Checked **2026-11-10**.

The Docker publish workflow sets `COSIGN_VERSION` to **v2.4.1**. Latest official Cosign release is **v3.1.4**. The installer is separately on major v3 with v4.1.3 available. Test signing, registry identity, verification and provenance together before changing these majors.

Sources: [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml), [official v3.1.4 release](https://github.com/sigstore/cosign/releases/tag/v3.1.4), checked 2026-11-10.

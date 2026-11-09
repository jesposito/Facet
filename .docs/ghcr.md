# GitHub Container Registry publishing

Checked **2026-11-10**.

Push run [42706080159](https://github.com/jesposito/Facet/actions/runs/42706080159) compiled the ARM64 image and then failed the upload with `permission_denied: write_package`. The workflow already has `packages: write`, and the GHCR login step succeeded. The package is public and linked to the `jesposito/Facet` repository; the workflow actor was `jesposito`.

The AMD64 job in the same push run passed its GHCR push, scanning and signing steps; only ARM64 failed the registry upload. This rules out a simple account-wide loss of write access. The corresponding pull-request image scans and packaged smoke tests passed on both architectures because they load images locally rather than requiring the failing ARM64 push. The remaining blocker is specific to publication permissions or policy. Follow-up: `Facet-dcy`; leave it open until both architecture digests and the signed manifest publish successfully.

Sources: [failed push run](https://github.com/jesposito/Facet/actions/runs/42706080159), [PR #528](https://github.com/jesposito/Facet/pull/528), [GitHub Packages permissions documentation](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility), checked 2026-11-10. The precise setting that restores publication had not been changed or verified when this note was written.

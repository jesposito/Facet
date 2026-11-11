# GitHub Container Registry publishing

Checked **2026-11-11**.

Push run [42706080159](https://github.com/jesposito/Facet/actions/runs/42706080159) compiled the ARM64 image and then failed the upload with `permission_denied: write_package`. The workflow already had `packages: write`, and the GHCR login step succeeded. In the same run, AMD64 pushed successfully. Other overlapping branch builds also saw intermittent denials, while later non-overlapping publishes succeeded.

The issue was transient and required no configuration change. Release run [42941554396](https://github.com/jesposito/Facet/actions/runs/42941554396) for **v3.8.8** passed both architecture builds, image scans, digest exports and signing, then pushed and verified the multi-architecture manifest. Earlier v3.8.4 through v3.8.7 releases also published both architectures. `Facet-dcy` is closed; reopen it if denials recur outside overlapping branch-build bursts. The observed correlation with concurrent publishes is a hypothesis, not a proven registry limitation.

Sources: [failed push run](https://github.com/jesposito/Facet/actions/runs/42706080159), [successful v3.8.8 release run](https://github.com/jesposito/Facet/actions/runs/42941554396), [PR #528](https://github.com/jesposito/Facet/pull/528), [GitHub Packages permissions documentation](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility), checked 2026-11-11. Direct GHCR inspection from the documentation host was blocked by token authorization; the successful GitHub release jobs are the publication evidence used here.

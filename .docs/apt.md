# APT

Checked **2026-11-11**.

CVE-2026-109518 describes an on-path package swap after verification that can install an attacker-controlled package as root. It requires redirects and non-zero `Acquire::Retries`; Bookworm defaults retries to 3. Debian's tracker lists **2.6.1** vulnerable and Bookworm security **2.6.2** fixed under DLA-4880-1.

Published-image scan 42568088543 reported exactly two HIGH findings, `apt` and `libapt-pkg6.0`, for this CVE. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade to the production Dockerfile; a disposable image check found both packages at **2.6.2**, and both architecture image scans and packaged smoke tests passed. The later [published-image scan 42724230846](https://github.com/jesposito/Facet/actions/runs/42724230846) passed, and subsequent releases published both architectures.

Sources: [Debian security tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [original published-image scan](https://github.com/jesposito/Facet/actions/runs/42568088543), [passing published-image scan](https://github.com/jesposito/Facet/actions/runs/42724230846), checked 2026-11-11.

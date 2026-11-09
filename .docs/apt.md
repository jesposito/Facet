# APT

Checked **2026-11-10**.

CVE-2026-109518 describes an on-path package swap after verification that can install an attacker-controlled package as root. It requires redirects and non-zero `Acquire::Retries`; Bookworm defaults retries to 3. Debian's tracker lists **2.6.1** vulnerable and Bookworm security **2.6.2** fixed under DLA-4880-1.

Published-image scan 42568088543 reported exactly two HIGH findings, `apt` and `libapt-pkg6.0`, for this CVE. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, refreshes the production base pins; a disposable pinned-base check found both packages at **2.6.2**, and both architecture image scans and packaged smoke tests passed. ARM64 publication is separately blocked by `Facet-dcy`.

Sources: [Debian security tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [published-image scan](https://github.com/jesposito/Facet/actions/runs/42568088543), checked 2026-11-10.

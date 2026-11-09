# LaTeX / TeX Live

Checked **2026-11-10**.

The production image installs `texlive-latex-base`, `texlive-latex-extra`, `texlive-xetex`, recommended fonts and `lmodern` from Debian Bookworm for AI résumé PDFs. Native development treats Pandoc and LaTeX as optional unless those paths are tested. Package versions follow the image build's apt state.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docs/DEV.md`](../docs/DEV.md), [official TeX Live documentation](https://tug.org/texlive/), checked 2026-11-10.

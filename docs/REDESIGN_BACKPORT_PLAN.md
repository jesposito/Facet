# Soft Premium Redesign Backport Plan (Cloud → Self-Hosted Facet)

> **Status:** IN PROGRESS · **Created:** 2026-06-23 · **Owner:** Jed
> **Source product:** Facet Cloud (`/home/jed/dev/facets-sh`, `jesposito/facetcloud`, ~v3.68.x)
> **Target product:** self-hosted Facet (`/home/jed/dev/Facet`, `jesposito/Facet`, ~v2.22.1)
> **Supersedes (for design scope only):** `BACKPORT_PLAN.md` (2026-05-15) — that doc covers a11y/component polish predating this redesign. Tier-2/Tier-3 API/webhook notes there remain valid.

This plan backports Cloud's **"Soft Premium"** redesign (shipped v3.61→v3.68: three campaigns — token/visual rebrand, AAA beauty pass, owner-UX/launch remediation) into self-hosted Facet **with zero regressions for existing self-hosted instances**.

---

## 0. How to use this document (resume protocol)

This file IS the session-persistent state. Any session resuming this work MUST:

1. Read this whole file. The **Status Dashboard** (§4) is the source of truth for what's done.
2. Read **Zero-Regression Doctrine** (§2) and **Hard Exclusions** (§3) before touching anything.
3. Pick the lowest-numbered `TODO` item whose dependencies are all `DONE`.
4. Follow that item's row (files / change / verify / rollback). Run the item's verification AND the **Global Verification Suite** (§8) before marking `DONE`.
5. Update the item's status inline and in §4, append a dated line to the **Progress Log** (§9), commit the plan doc with the work.
6. Mirror active items into beads at execution start (`bd create` per item, link this doc). Do not create beads items until execution begins (planning rule).

**Status values:** `TODO` · `IN_PROGRESS` · `DONE` · `BLOCKED` · `SKIPPED`. Never mark `DONE` without passing verification + a11y-lead review (§7).

---

## 1. Strategy: three independent tracks

| Track | Scope | Depends on | Rationale |
|-------|-------|-----------|-----------|
| **A — Fixes & hardening** | Correctness + a11y/UX bugfixes from the redesign | nothing | Highest ROI, lowest risk, several are data-safety. Ship as standalone PRs independent of any rebrand. |
| **B — Foundation + accent engine** | Token/foundation CSS layer + AAA-clamp accent | nothing (gates C) | One-time prerequisite. Shifts the whole look via shared semantic classes. **Highest regression surface — touches every existing site.** |
| **C — Public-surface visuals** | Hero, nav, sections, footer band, welcome | **B** | Per-surface MODERATE ports. Mind two regressions (§6). |

**Recommended order:** Track A first (banks value, no rebrand commitment) → Track B (gated, careful) → Track C (per-surface, incremental).

### Locked execution decisions (2026-06-23, user-approved)

1. **a11y always, look opt-in (hybrid).** AAA accent/focus contrast clamps (B2) apply to **every** instance — they only improve contrast, never regress it. The warm-stone/font *visual* rebrand (B1/B3/B4/B5 + all of Track C) is gated behind a new **`design` instance setting** (`classic` default | `soft-premium`). Existing sites render pixel-identical until the operator opts in.
2. **Scope:** full A + B + C.
3. **Certification:** automated (`make test`/`make lint`/`i18n:validate`/Go+Playwright) **and** live visual proof on a `SEED_DATA=dev` instance (320/768/1280px, both design modes, light+dark, incl. print/resume + stored-accent preservation), gated per item. CI does NOT gate tests — local responsibility.

### Opt-in `design` switch — verified wiring (new item **B0**, gates B1/B5/C)

- **Storage:** `design` SelectField on **`site_settings`** (`classic`|`soft-premium`, default `classic`). Migration copies `backend/migrations/1778830100_add_default_theme_mode.go`.
- **Backend:** `Design` field + `loadDesign` fallback in `backend/services/settings.go`; GET/PUT serialize+validate (whitelist) in `backend/hooks/settings.go`.
- **SSR FOUC-free:** extend the existing `transformPageChunk` `<html lang="en">` replace in `frontend/src/hooks.server.ts` to also emit `data-design="<design>"`. Server-rendered, no JS/flash.
- **CSS:** new warm tokens under `:root[data-design="soft-premium"]{…}` (+ `.dark`) in `app.css`, **additive** — never remove `--color-primary-*`.
- **Accent caveat:** `--color-primary-*` is inline on `<html>` (beats attribute selectors) → Soft Premium's accent default must change **through the accent pipeline** (different default palette when `design=soft-premium` and no operator accent), NOT a CSS override. Surface/radius/shadow/font tokens (not accent-owned) switch safely via the attribute.
- **Admin UI:** toggle wired to site-settings PUT (`/api/site-settings`), in `routes/admin/settings/site/`. Strings → all 5 locales.

Execution layer / approved plan: `~/.claude/plans/ethereal-scribbling-dove.md`.

---

## 2. Zero-Regression Doctrine (READ FIRST)

"Zero regressions for Facet users" = existing self-hosted instances upgrade with **no broken layout, no lost a11y affordance, no silently-overridden user settings, no broken resume/print export.** Self-hosted is MIT and distributed: many operators run their own instance with their own stored DB settings.

### 2.1 The defaults-vs-stored-state rule (the most important rule here)

Every visual change falls into one of two buckets. Treat them completely differently:

- **CODE DEFAULT change** (safe): changes what an *unconfigured* instance renders. Only affects operators who never set the value. Examples: `DEFAULT_ACCENT_COLOR`, default font pack, default layout. → Allowed, but **CHANGELOG it as a visible change** and gate behind "only when no stored value exists."
- **STORED-STATE re-derivation** (dangerous): changes what an instance that *did* configure the value renders. Examples: raising the accent contrast clamp re-derives every operator's chosen hue; renaming a stored enum value; changing how `custom_css` is parsed. → Requires explicit migration safety: **preserve the operator's intent (hue/choice), never overwrite their DB row, and document the visible delta.**

**Never** write a migration or runtime path that overwrites a stored `profile.accent_color`, `custom_css`, font pack, or hero color. Defaults apply only when the field is empty/unset (confirm the `else if (data.profile?.accent_color)` guard at `+layout.svelte:220` stays the gate).

### 2.2 Specific existing-user impacts to manage (not avoid — manage)

| Change | Who it touches | Handling |
|--------|---------------|----------|
| AAA accent clamp (`MIN_CONTRAST` → 7.0 in `generatePaletteFromHex`) | **Every** instance — re-derives the rendered 600/300 from the stored hue | Clamp only adjusts *lightness* to hit contrast; **hue is preserved**. Net effect is strictly *more* accessible, never less. Acceptable, but: CHANGELOG + release note "accent colors may render slightly deeper to meet AAA contrast." Add unit tests locking white-on-600 ≥7:1. Do NOT touch the stored hue. |
| Default accent `sky` → terracotta | Only instances with **no** stored `accent_color` | CODE DEFAULT change. Allowed. CHANGELOG. Verify a stored-`sky` instance still renders sky. |
| Default fonts Lora/Plus-Jakarta → Hanken/Newsreader | **Every** instance using the default pack (likely most) | High-visibility default change. Requires CSP/font-link update (§ B3) or fonts silently fail. CHANGELOG prominently. Consider keeping Lora available as a selectable pack so operators can pin the old look. |

### 2.3 Hard invariants (break = regression)

- **Resume/print export must stay intact.** Large `@media print` block at `app.css:438`. Re-verify ATS/PDF output after every Track B change (§8).
- **Luminance-aware hero text** (`DARK_TEXT_THRESHOLD = 0.179`, `hexLuminance`, colors.ts:393/409) must keep flipping text light/dark correctly when an operator set a custom hero bg.
- **i18n: all user-visible strings via `$t()`, 5 locales (en/de/elvish/klingon/lolcat), `npm run i18n:validate` must pass 100%.** Cloud components hardcode English — porting them verbatim VIOLATES this. Every ported string gets keys in all 5 files.
- **Preserve self-hosted a11y affordances Cloud dropped** (§6).
- **Don't churn load-bearing selectors** (§5).

---

## 3. Hard Exclusions (DO NOT PORT)

SaaS-only — out of scope entirely:

- Plan-gating padlocks / upsell screens / Starter-Pro-Creator capability matrix.
- AI-credit meter + 402-exhausted state + reserve-before-call metering (self-hosted is BYOK).
- Trial flow / free-trial-recycling ledger / Turnstile signup gating / `$3 Starter` pricing.
- Provisioning / fleet / systemic-breaker / multi-tenant chrome.
- **Footer "Powered by Facet" Shadow-DOM anti-tamper badge** (`onMount` MutationObserver block, ~95 lines, plan-forced) — self-hosted keeps its plain GitHub link.
- **AdminSidebar IA** — self-hosted's nav is *richer and better* for its product; Cloud collapsed to a flat SaaS IA (Money/members/plan-gating). Borrow visual treatment only; never straight-port.
- **Admin dashboard revenue/members/payout/AI-credit blocks** — welded to SaaS APIs that don't exist single-user.
- **SetupWizard persona system** — optional/borderline; default SKIP (adds 3 child components + a store + onboarding-funnel concept). Port only the animation/announce polish.
- Cloud's 17-locale i18n + multi-currency.

Cloud-only dirs never to mine (from `BACKPORT_PLAN.md §7`): `backend/hooks/course/`, `subscription*.go`, `quotas.go`, `newsletter_*.go`, `bundles.go`, `offers.go`, `member_*.go`, `services/connect.go|subscription.go`, `routes/admin/{billing,connect,domain,members}/`, `components/admin/courses/`, `course-learner/`, `MembershipGate.svelte`, `ContentGate.svelte`.

---

## 4. Status Dashboard

### Track A — Fixes & hardening (no dependencies)
| ID | Item | Risk | Status |
|----|------|------|--------|
> **Audit done 2026-06-23** (verified against current self-hosted code). Skip N/A + already-present; only the TODO rows are real work.
| A1 | Keyboard reorder commits on Tab/blur | — | **N/A** — self-hosted persists synchronously on click (no deferred drop to lose). |
| A2 | Keyed rows + move announce (wrong-record-delete) | low | **TODO (scoped)** — only `admin/courses/+page.svelte` curriculum `{#each}` (lines ~1405/1460) unkeyed + no ReorderAnnouncer. Everywhere else already correct. |
| A3 | Reorder autosave superset/subset tolerance | — | **N/A** — self-hosted has no permutation rejection; just persists the array. |
| A4 | Sparkline zero-fill 7 days | low | **DONE** — PR #449. `fillDailyViewCounts()` helper + 3 unit tests. |
| A5 | Shared `PageHeader` + responsive reflow (320px) | low-med | **TODO** — no PageHeader exists; admin list headers `flex justify-between` no wrap (e.g. experience:488, projects:624) overflow at 320px. |
| A6 | Setup-wizard no-nag + announced success | — | **ALREADY PRESENT** — `setupWizard.ts:201` gates; success is toast not fast-unmount. |
| A7 | Focus/announce discipline | low | **DONE** — PR #449. error/warning toasts persist (duration 0); success/info still 5s. |
| A8 | Touch-target 44px + `100vh`→`svh` | low | **DONE (scoped)** — PR #449. calc(100vh)→100dvh in AdminSidebar + ViewPreview. (Targets already 44px; broad `min-h-screen` swap left as optional follow-up.) |
| A9 | ICU plurals + microcopy | low | **TODO** — en.json count strings ("{count} lessons" etc., lines 932/934/1058/1407/2151) → ICU plural; propagate all 5 locales. |
| A10 | Remove joke 404 from admin | — | **DEFER/OPTIONAL** — self-hosted's joke 404 is intentional brand voice (`+error.svelte`, en.json:3086). Cloud removed it for SaaS trust; self-hosted may keep. Not porting unless requested. |
| A11 | `<title>` normalization `"{Page} \| {brand}"` | low | **TODO** — admin titles inconsistent (4 formats); standardize on `{$t(page)} \| {$brandName}`; optional route announcer in +layout. |
| A12 | Admin sans pin + tabular-nums | low | **TODO** — `app.css:70` global `h1,h2,h3 { font: var(--font-heading) }` leaks Lora into admin; add admin-scoped sans + tabular-nums on data cells. |

### Phase 0 — test harness prep
| ID | Item | Risk | Status |
|----|------|------|--------|
| P0 | Parametrize `backport-qa.spec.ts` host+creds → env, remove committed secret, local run path | low | **DONE** — PR #447 (`fix/backport-test-harness`). 13 tests collect; creds via env. History scrub + password rotation flagged to maintainer. |

### Track B — Foundation + accent engine (gates Track C)
| ID | Item | Risk | Status |
|----|------|------|--------|
| B0 | Opt-in `design` switch (site_settings field + backend + hooks.server `data-design` + admin toggle) | med | **DONE** — PR #448. Certified: go test, svelte-check 0/0, i18n 100%, accessibility-lead 16/16, `design-switch.spec.ts` 5/5 through real /admin. |
| B1 | Token/foundation layer: warm stone ramp + Soft Premium surface vocab + radius/shadow/motion/focus tokens (scoped to `[data-design=soft-premium]`) | **high** | TODO |
| B2 | AAA accent clamp in `generatePaletteFromHex` (MIN_CONTRAST 7.0, dual clamp) + unit tests | **high** | TODO |
| B3 | Default font pack → Hanken/Newsreader + font-link/CSP update (keep Lora selectable) | med-high | TODO |
| B4 | Default accent → terracotta (code default only) | low | TODO |
| B5 | `grain` / `font-accent` / `text-gradient` / `divider-editorial` utilities | med | TODO |

### Track C — Public-surface visuals (depend on B1/B5)
| ID | Item | Verdict | Status |
|----|------|---------|--------|
| C1 | ProfileNav: accent underline + `presence` prop + topmost-section scrollspy | MODERATE | TODO |
| C2 | PageHeader visual adoption across admin (builds on A5) | TRIVIAL | TODO |
| C3 | Footer CTA band (cherry-pick; NOT the badge machinery) | low | TODO |
| C4 | Resume sections: card plates + serif headings (Exp/Proj/Talks/Skills/Edu/Cert/Awards/Testimonials) | MODERATE ×~8 | TODO |
| C5 | ProfileHero: token reskin + `rail` layout + CTA slot | MODERATE | TODO |
| C6 | SiteNav: grain masthead + default-nav fallback (strip courses flag) | MOD-HEAVY | TODO |
| C7 | AccentPicker: merge Cloud ramp+collision model, RE-ADD i18n/reset/announce | HEAVY | TODO |
| C8 | WelcomePage editorial rewrite (self-hosted copy branch already exists in Cloud) | HEAVY-markup | TODO |
| C9 | AdminHeader: glass treatment + live-preview toggle (brand→"Facet") | low-med | TODO |
| C10 | Admin dashboard: adopt composition principles against self-hosted stat set (rebuild, not port) | HEAVY | TODO |

---

## 5. Load-bearing — preserve, do not rename/churn

Selectors and contracts other code depends on (grep before renaming):
`.dash-display-title`, `[data-opp-type]`, `#module-title`, `#page-title`, `#course-pricing`/`#course-trial` (+ their `tabindex="-1"`), the `opportunities[0]`-always-rendered invariant, `--accent-collision` flag, layout enums/props + `getWidthClass`/`isWidthValidForLayout`, the rail 2-col wrapper, `DARK_TEXT_THRESHOLD` hero-text logic, HMAC share tokens, JWT password views.
Self-hosted accent application path: `+layout.svelte` lines 118/128/140-141/148/220-221/231-236 (custom-CSS regex parse + mirror-500). Keep the empty-value gate intact.

---

## 6. Regression watchlist (self-hosted has wins Cloud dropped)

- **SkillsSection proficiency dots** — non-color WCAG signal (`proficiencyDots`) self-hosted has, Cloud REMOVED. When porting C4/Skills, keep the dots. Do not import the regression.
- **AccentPicker i18n** — Cloud hardcodes English (would fail `i18n:validate`). C7 must re-add self-hosted's `$t('admin.accent_picker.*')`, reset button, and persistent `aria-live` region.
- **Custom CSS override path** — operators can override individual `--color-primary-*`. Token-layer rename (B1/B5) must not break the `+layout.svelte` regex (`--color-primary-(50|...|950)`). If introducing new `--ink`/`--surface` tokens, ADD alongside; don't remove `--color-primary-*`.

---

## 7. Accessibility gate (mandatory, per project rule)

This is a web project; the project mandates `accessibility-agents:accessibility-lead` review for any UI-touching change. For **every** Track B and Track C item, and A2/A5/A6/A7/A8/A11/A12:
- Before writing UI code: delegate the item to `accessibility-agents:accessibility-lead` for an up-front plan.
- After implementing: re-run accessibility-lead review before marking `DONE`.
- Treat its findings as blocking. Record the review outcome in the Progress Log.

---

## 8. Global Verification Suite (run before every `DONE`)

From `Facet/frontend`:
1. `npm run build` — SSR build clean.
2. `npm run i18n:validate` — 100% across all 5 locales.
3. Lint/type: project lint + `tsc`/svelte-check clean (fix pre-existing diagnostics in touched files — `fix-preexisting-issues` rule).
4. **Print/resume:** load a profile, trigger ATS/PDF export, confirm `@media print` (app.css:438) output unbroken. **Mandatory after any Track B change.**
5. **Luminance:** set a custom hero bg (light AND dark), confirm hero text flips correctly.
6. **Accent preservation:** instance with stored `accent_color: sky` still renders sky; instance with custom hex still renders that hue (deeper after clamp is OK, different hue is NOT).
7. **Visual smoke (browse skill):** before/after screenshots of public profile + key admin pages at 320px / 768px / 1280px. Compare for unintended shifts.
8. **a11y:** axe/Playwright pass on touched surfaces; keyboard-only walkthrough of any reorder/dialog/nav touched.
9. Integration test on the Unraid container per house practice; do not run facets-sh cloud tests.

Per-PR: follow `Facet/CONTRIBUTING.md` 100% (commit types/scopes, branch prefix, CHANGELOG). Bundle related leaf items onto ONE branch/PR (`larger-prs-fewer-branches`). Each item independently revertable.

---

## 9. Phase 0 — anchors to re-verify before editing (don't trust this doc blindly)

Confirm these still hold (they did on 2026-06-23):
- `colors.ts:417 generatePaletteFromHex` has NO contrast clamp today (B2 adds it). Exports at 301/336/349/363/393/417/452.
- `DEFAULT_ACCENT_COLOR = 'sky'` (colors.ts:280); `ACCENT_COLOR_LIST` at 285.
- `app.css :root` primary ramp = sky (`--color-primary-500: #0ea5e9`, lines 11-35); fonts lines 37-39; `@media print` at 438.
- `app.html` font link line 9 (Lora/Plus Jakarta/Inter/JetBrains).
- `+layout.svelte` accent path lines 118-252; empty-value gate at 220.
- No `lucide-svelte`, no `culori`, no `@fontsource` in `frontend/package.json`.

---

## 10. Track B token mapping (B1) — old → new

Bring Cloud's `app-tokens.css` values. Current self-hosted is cool sky/gray; target is warm stone + Soft Premium surfaces. ADD new tokens alongside `--color-primary-*` (keep that for custom-CSS compat).

Warm stone ramp (OKLCH H≈53°, luminance held to old ramp so contrast survives):
`--stone-50 #fbf8f4 · 100 #faf4f0 · 200 #ece4de · 300 #ddd1ca · 400 #ada199 · 500 #7b706a · 600 #5a524d · 700 #463f3b · 800 #2a2522 · 900 #1a1613 · 950 #0d0a08` (emit `-rgb` triplets too for `<alpha-value>`).

Soft Premium semantic surfaces (additive):
`--bg #fbf8f4 · --surface #fff · --surface-2 #f4eee4 · --chip #f1eae0 · --ink #2a2522 · --muted #6f675f · --border #ebe3d6 · --border-interactive #c4b5a2`. Dark mode = warm browns (`--bg #1a1613 · --surface #221e1a · --ink #f4eee4`), NOT cool stone.
Radius `--r-1 4px…--r-5 16px · --r-full 9999px`. Shadows `--sh-1/2/3` (stone-950 base; black-based in dark). Motion `--t-fast 120 · --t-norm 160 · --t-slow 220 · --e-out cubic-bezier(0.16,1,0.3,1)` (→1ms under reduced-motion). Focus ring = 3px warm-stone halo + 2px stone-700 (SC 2.4.13), flips on dark.
Map these into `tailwind.config.js` (`rounded-token-*`, `shadow-token-*`, `duration-token-*`, `ease-token-out`, stone ramp) so utilities resolve.
Reference (read at implementation): `facets-sh/frontend/src/app-tokens.css`, `facets-sh/docs/design/2026-redesign/DESIGN_SPEC.md`, `.../02-composition.md`, `.../03-color-accent.md`.

---

## 11. Dependency decisions (resolve before Track B/C)

| Dep | Recommendation | Why |
|-----|---------------|-----|
| `lucide-svelte` | **Add it** | Unblocks all C-track ports; self-hosted already wants icon consistency (currently inline SVG via `lib/icons.ts`). Alternative is per-component SVG swaps forever. |
| `culori` | **Hand-port the ~50-line OKLCH clamp into existing `colors.ts`** | Avoids a dep; self-hosted's hand-rolled engine already works. Adopt only the binary-search clamp math. (Add culori only if the ramp math proves fiddly.) |
| Font system | **Adopt Soft Premium as new default pairing; keep Lora as a selectable pack.** Full 8-pack runtime switcher = optional later. | Single-user product; a full picker is lower value than for SaaS, but keeping Lora lets existing operators pin the old look (zero-regression escape hatch). |

---

## 11b. Live test environment (for visual/Playwright certification)

Docker compose dev is broken on this box (`dev-backend.sh: no such file`, and only `docker-compose` v1 is installed). Use **native**:
- Backend: `cd backend && ENCRYPTION_KEY=<32+ hex> SEED_DATA=dev APP_URL=http://localhost:5173 go run . serve --http=0.0.0.0:8090 --dir=<scratch pb_data>` (automigrate seeds the "Jedidiah Esposito" profile + views).
- Frontend: `cd frontend && POCKETBASE_URL=http://localhost:8090 npm run dev` (Vite :5173 proxies /api).
- App admin login (users collection): `admin@example.com`. Dev seed password is `changeme123` but the app forces a first-login password change; complete it once via `POST /api/auth/change-password` `{currentPassword,newPassword}` (sets `password_changed_from_default=true`). PB superuser for `/_/`: `admin@localhost.dev` / `admin123`.
- Playwright product tests take `PLAYWRIGHT_BASE_URL` + `ADMIN_EMAIL`/`ADMIN_PASSWORD` env. Reset `site_settings.design` to classic between manual experiments (superuser PATCH) — tests' afterEach restores it but manual flips can contaminate.
- `go` is 1.23.4 vs go.mod 1.25; tests/build run fine anyway.

## 12. Progress Log
- 2026-06-23 — **Track A PR #449** (`fix/track-a-backport-corrections`): A4 (analytics zero-fill +3 tests), A7 (toast persistence), A8 (dvh). Certified: go test green, svelte-check 0/0. Remaining Track A: A2 (courses keys), A9 (ICU plurals ×5 locales), A11 (title normalization), A12 (admin sans + tabular-nums). **Facet Cloud fix** (user request) delegated: admin vitest broken by @testing-library/svelte 5.4.1 under forced runes — agent applying dep/config fix + PR.
- 2026-06-23 — Plan created from 4-agent analysis (cloud redesign characterization, self-hosted UI map, non-aesthetic UX intent, component behavioral diff). No code changed. Status: all items TODO.
- 2026-06-23 — Execution approved. Decisions locked (hybrid opt-in, full A+B+C, automated+visual cert). Added opt-in `design`-switch architecture (B0) from settings/SSR exploration. Started **P0** on branch `fix/backport-test-harness`: parametrized `backport-qa.spec.ts` host/creds to env (`PLAYWRIGHT_BASE_URL`/`ADMIN_EMAIL`/`ADMIN_PASSWORD`), removed committed live credentials (history still contains them — flagged for optional scrub).
- 2026-06-23 — **P0 DONE** (PR #447). **Track A audited** against real code — A1/A3 N/A, A6 already present, A10 defer (intentional brand voice); rest are scoped TODOs (see dashboard). **B0 DONE** (PR #448, branch `feat/design-switch`): backend field+API+migration, SSR `data-design` injection (classic=no attribute=byte-identical), accessible admin radiogroup (accessibility-lead 16/16), Toast role fix, i18n ×5, `design-switch.spec.ts` 5/5 through real /admin. Live native dev instance stood up (see §11b). **Next ready:** B1 (warm tokens under `[data-design=soft-premium]` — makes the switch visible, gates C) then B2 (universal AAA clamp), and the scoped Track A TODOs (A4/A5/A7/A8/A9/A11/A12 + A2-courses). Also pending: audit/fix breakage in Facet Cloud (facets-sh) per user request.

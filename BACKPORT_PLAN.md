# Facet Cloud → Self-Hosted Backport Plan

> Generated: 2026-05-15
> Last mass backport: v2.19.0 (March 28, 2026)
> Self-hosted current: v2.21.11 (April 13)
> Cloud current: v3.24.0 (May 14)
> Window: ~6 weeks of cloud development

---

## 1. Executive Summary

**Facet Cloud** has diverged significantly with SaaS-specific infrastructure (multi-tenant provisioning, Stripe billing, course bundles, multi-currency, CF Workers). However, a substantial subset of changes are **general improvements** — a11y fixes, UX polish, component refactors, keyboard navigation, and SSR hardening — that can be backported to the self-hosted MIT repo with **zero risk of SaaS leakage**.

**Goal:** Backport *every* non-SaaS-specific improvement from Cloud to self-hosted — a11y, UX, performance, visual polish, admin productivity — with zero regression. SaaS-only features (Stripe, billing, multi-tenant, courses, CF Workers) are explicitly excluded. Everything else ships.

---

## 2. Distinction Analysis

### Cloud-Only (DO NOT backport)
| Category | Examples |
|----------|----------|
| **Multi-tenant infrastructure** | Fleet provisioning, Hetzner stock watcher, Coolify/Traefik ingress, blue-green deploys, watchdog |
| **Stripe billing** | Subscriptions, checkout, MRR dashboards, Connect, BNPL, coupons, order bumps, pricing tiers |
| **Course monetization** | Bundles, trials, Q&A, certificates, student progress tracking, enrollment dashboards |
| **Member accounts** | Member auth, claims, invitations, subscription bridge, lifecycle nudges, milestones |
| **Newsletter advanced** | SES cutover, Liquid personalization, sequences, broadcast scaling, sending reputation |
| **i18n edge** | 17 locales, multi-currency, CF Workers for landing i18n, Adaptive Pricing |
| **Ops/Admin** | Cart recovery, domain management, billing settings |

> **Note (May 2026):** API keys, webhooks, system alerts, and newsletter multi-list (segments) were **partially backported** as stripped-down single-tenant variants. See "Tier 2 + Tier 3" section below.

### Safe for Backport
| Category | Examples |
|----------|----------|
| **a11y improvements** | Keyboard nav on admin lists, focus traps, dialog fixes, ESC handling, ARIA labels |
| **UX polish** | Dirty-nav nudge, bottom sheets on mobile, confirm dialogs, conflict banners |
| **SSR/performance** | Favicon rendering, pressure gauge, latency budgets, middleware reordering |
| **Component hardening** | ConfirmDialog pre-empt fixes, DnD shadow filtering, focus restoration |
| **Admin productivity** | Keyboard shortcuts (j/k/e/x), section manager autosave, media edit modal |
| **Visual consistency** | CSS fixes, animation delays, color token alignment |

---

## 3. Backport Candidates by Priority

### P0 — High Confidence, High Value, Zero Regression Risk

#### 3.1 Dirty Navigation Nudge (v3.20.0 — `facets-fy56`) ⏳ DEFERRED
- **What:** Replaces browser `beforeunload` with a custom `<dialog>`-based nudge for in-app navigation when editor has unsaved changes
- **Status:** Deferred. Self-hosted lacks cloud's unified `ContentEditor.svelte` shell; would need per-page integration into every editor route (views, posts, talks, projects, newsletter, subscribers). Significant scope.
- **Files to port:**
  - `frontend/src/lib/components/editor/DirtyNavigationNudge.svelte` (new)
  - Integration into editor shell / routes
- **i18n keys needed:** `editor.shell.dirtyNav.*`
- **Risk:** Near-zero. Pure UI component using native `<dialog>`
- **Issues affected:** None directly, but prevents data loss

#### 3.2 ConfirmDialog Hardening (v3.0.91+ — `facets-sh-0ou`) ✅
- **What:** Fixes focus restoration when one confirm dialog pre-empts another (e.g., admin list `x` shortcut opens new dialog while old one is still showing). Adds `contentKey` for keyed remount, guards `previousActiveElement` with `wasOpen` state, checks `document.contains()` before restoring focus.
- **Files:** `frontend/src/components/shared/ConfirmDialog.svelte`
- **Risk:** Very low. Focus-management bugfix.
- **Issues affected:** #405 (focus trap reliability)

#### 3.3 SkillsCategoryManager Keyboard Reordering (v3.0.63+ related) ✅
- **What:** Adds `moveCategory()` function with up/down arrow buttons for keyboard-accessible category reordering. Also filters DnD shadow placeholders for visual consistency.
- **Files:** `frontend/src/components/admin/SkillsCategoryManager.svelte`
- **Risk:** Low. Enhances existing component, no schema changes.
- **Issues affected:** #259 (Change order of skill categories), #404 (keyboard DnD support)

#### 3.4 CSS/SSR Fixes ✅
- **v3.18.6:** Ensure trailing semicolon on inline-style CSS var concat ✅
- **v3.18.3:** Make floating cluster inert when nav pinned ✅
- **v3.20.1:** Chips mobile hamburger + CTA collision fix ⏳
- **Risk:** Near-zero. One-liner or small CSS changes.

---

### P1 — Good Value, Manageable Scope

#### 3.5 Keyboard Navigation Composables (v3.0.61–v3.0.67 — `facets-sh-5b4`, `facets-0aa8`)
- **What:** `createListShortcuts()` and `createTableShortcuts()` composables providing j/k/Enter/e/x//? keyboard navigation for admin lists. Includes roving tabindex, focus survival across deletes, ARIA keyshortcuts hints.
- **Files to port:**
  - `frontend/src/lib/admin/listShortcuts.svelte.ts` (new)
  - `frontend/src/lib/admin/tableShortcuts.svelte.ts` (new)
  - `frontend/src/lib/components/admin/KeyboardShortcutsHelp.svelte` (new)
  - Integration into existing admin list pages
- **Risk:** Low. Self-contained composables; pages that don't adopt them are unchanged.
- **Issues affected:** #405 (focus/keyboard), #406 (a11y)
- **Note:** Self-hosted has simpler admin list pages than cloud. Adoption would be per-page.

#### 3.6 ConfirmDiscardDialog + Dirty-State Shell (a11y P0)
- **What:** `<dialog>`-based confirmation when pressing Esc with dirty editor state. Three outcomes: discard, save, keep.
- **Files:**
  - `frontend/src/lib/components/editor/ConfirmDiscardDialog.svelte` (new)
- **Risk:** Low. Only affects editor routes.
- **Issues affected:** #405

#### 3.7 MediaEditModal (relates to #348)
- **What:** Modal for editing media item metadata (title, alt text, description, tags) inline in the media library.
- **Files:**
  - `frontend/src/components/admin/media/MediaEditModal.svelte` (new)
  - `frontend/src/components/admin/media/types.ts` (new)
- **Risk:** Medium-low. Requires integration with existing media library flow.
- **Issues affected:** #348 (Media library improvement: Add option to update images)

#### 3.8 AccentPicker a11y + Keyboard Nav
- **What:** Cloud has refactored `AccentPicker.svelte` with full keyboard navigation and test coverage.
- **Files:** `frontend/src/lib/accent/AccentPicker.svelte`, `derive.ts`, tests
- **Risk:** Low. Self-contained component.

#### 3.9 AutosaveIndicator + ConflictBanner (SectionManager wedge)
- **What:** Visual indicators for autosave status and conflict resolution when editing sections.
- **Files:**
  - `frontend/src/lib/components/admin/AutosaveIndicator.svelte`
  - `frontend/src/lib/components/admin/ConflictBanner.svelte`
- **Risk:** Low if integrated carefully.

---

### P2 — Nice to Have, Requires More Analysis

#### 3.10 FacetChipStrip + AssignToViewsPopover (v3.0.73 — `facets-sh-66o`)
- **What:** Drag-and-drop facet assignment chips for talks/projects/posts/courses. Could enable "assign to views" for testimonials.
- **Files:**
  - `frontend/src/lib/components/admin/FacetChipStrip.svelte`
  - `frontend/src/lib/components/admin/AssignToViewsPopover.svelte`
- **Risk:** Medium. New UI patterns, requires adoption.
- **Issues affected:** #283 (funnel testimonials to lists automatically — partial)

#### 3.11 Project Cover Images (cloud has this)
- **What:** Cloud's `ProjectForm.svelte` supports `cover_image` and `cover_image_library_url`.
- **Risk:** Medium. Requires schema migration (`projects` collection needs cover image fields).
- **Issues affected:** #400 (Cover image for talks — similar pattern, could extend to talks)

#### 3.12 TalksSection Visual Refresh
- **What:** Cloud has improved TalksSection with staggered animations, Icon component usage, better color tokens.
- **Files:** `frontend/src/components/public/TalksSection.svelte`
- **Risk:** Low. Visual-only changes.
- **Issues affected:** #400 (partial — visual improvements, not cover image)

#### 3.13 Drag-onto-card Facet Assignment (v3.0.68–69 — `facets-yc2t`)
- **What:** Additive drag + per-pill remove on admin posts listing.
- **Risk:** Medium. Self-hosted has different post list UI.

#### 3.14 SSR Pressure Gauge / Fast Recovery (v3.16.3 — `v8k`)
- **What:** SSR latency time-window, min-sample, middleware reorder, cascade gate.
- **Files:** `frontend/src/hooks.server.ts`, middleware
- **Risk:** Medium. SSR middleware changes need careful testing.

#### 3.15 Visible Focus Ring on Mobile Icon Switchers (v3.0.41)
- **What:** CSS focus ring improvements.
- **Risk:** Near-zero.

---

### P3 — Consider for Future, Not Immediate

#### 3.16 Dark Mode as Default Setting (#434)
- **What:** Neither repo currently has a "default dark mode" setting. Cloud enables dark mode via `dark` class. This would require:
  - New site setting field (`dark_mode_default` boolean)
  - Modify `app.html` script to read setting before paint
  - SSR pass of setting into layout data
- **Risk:** Low, but requires schema migration and new setting UI.
- **Issues affected:** #434
- **Recommendation:** Easy win, but not a cloud backport — it's a new feature request.

#### 3.17 Media Library Reordering (#407)
- **What:** Re-order images in attached media gallery.
- **Status:** Cloud does NOT have explicit gallery reordering. It has drag-onto-card facet assignment and media bulk tagging, but not gallery reorder.
- **Recommendation:** Not a cloud backport. Would need custom implementation.

#### 3.18 i18n Hardcoded Strings (#406)
- **What:** Cloud has done extensive i18n work, but it's coupled to the 17-locale system and machine-translation pipeline.
- **Risk:** Medium. Cloud's i18n keys have diverged significantly.
- **Recommendation:** Audit self-hosted components individually rather than bulk porting cloud i18n.

---

## 4. Open Issues Mapping

| Issue | Cloud Feature | Backportable? | Priority |
|-------|--------------|---------------|----------|
| #434 Darkmode as default | New setting | **Partial** (new impl) | P3 |
| #407 Re-order images | Not in cloud | **No** | — |
| #406 i18n hardcoded strings | Extensive i18n work | **Partial** (manual audit) | P2 |
| #405 Focus traps / keyboard dismissal | ConfirmDialog fixes, BottomSheet, DirtyNudge | **Yes** | P0 |
| #404 Keyboard DnD reordering | SkillsCategoryManager keyboard reorder | **Yes** | P0 |
| #403 Intelligent Visitor Counter | Bot detect (already in v2.19.0) | **Already done** | — |
| #400 Cover image for talks | Talks schema spike (v3.16.0) | **Partial** (visual only) | P2 |
| #375 Associate projects with experiences | Project improvements in cloud | **Partial** | P2 |
| #348 Media library: update images | MediaEditModal | **Yes** | P1 |
| #283 Funnel testimonials to lists | FacetChipStrip + AssignToViews | **Partial** | P2 |
| #281 Cloudflare Turnstile | Already exists in both | **Already done** | — |
| #259 Change order of skill categories | SkillsCategoryManager keyboard reorder | **Yes** | P0 |

---

## 5. Execution Plan

### Phase 1: P0 Quick Wins ✅ COMPLETE
1. ✅ Port ConfirmDialog hardening (focus restoration fix)
2. ✅ Port SkillsCategoryManager keyboard reordering + DnD shadow fixes
3. ✅ Port CSS/SSR micro-fixes (semicolon, floating cluster inert, chips collision)
4. ⏳ Port `DirtyNavigationNudge.svelte` — deferred until editor shell refactor

### Phase 2: P1 Safe Components ✅ COMPLETE
5. ✅ Port `AccentPicker.svelte` a11y + keyboard nav
6. ✅ Port MarkdownEditor inline dialog (replaces window.prompt/alert)
7. ✅ AdminSidebar span→button fix + restructuring
8. ⏳ Port `ConfirmDiscardDialog.svelte` — deferred until editor shell refactor
9. ⏳ Port `MediaEditModal.svelte` — self-hosted already has equivalent inline modal

### Phase 3: P1 Admin a11y + Keyboard Nav (current)
10. Port `createListShortcuts` + `createTableShortcuts` composables
11. Port `KeyboardShortcutsHelp.svelte`
12. Adopt keyboard nav on 3–4 highest-traffic admin list pages

### Phase 4: P2 Structural UI (explicitly requested)
13. Port sidebar facet display changes
14. Port homepage section layout options

### Phase 5: P2-P3 Visual Polish & Features
15. Port keyboard move buttons for remaining DnD zones (ViewSectionManager, NewViewSectionBuilder) — HomepageSectionManager ✅
16. Port AutosaveIndicator / ConflictBanner (if SectionManager is aligned)
17. Port TalksSection visual refresh
18. Assess FacetChipStrip for testimonials (#283)
19. Visible focus ring on mobile icon switchers
20. SSR pressure gauge / fast recovery

### Phase 6: P3 New Features (future)
18. Implement dark mode default setting (#434) — new work, not backport
19. Custom implementation of gallery reordering (#407) — new work

---

## 6. Risk Mitigation

- **No schema migrations from cloud:** All P0/P1 candidates are frontend-only or use existing collections.
- **No Stripe/cloud dependencies:** Every candidate has been vetted for SaaS coupling.
- **i18n safety:** New keys will be added to self-hosted's 5 locales (en/de/elvish/klingon/lolcat). English first, others via `fill-missing-i18n` or manual.
- **Testing:** Use the Unraid container for integration testing before push.
- **Incremental commits:** Each backport as atomic PR for easy revert.

---

## 7. Files to Ignore Entirely

Any file in these cloud-only directories/modules should NOT be examined for backporting:
- `backend/hooks/course/` (monetized course system)
- `backend/hooks/subscription*.go`, `backend/hooks/quotas.go`, `backend/hooks/starter_caps.go`
- `backend/hooks/newsletter_*.go` (advanced newsletter features)
- `backend/hooks/bundles.go`, `backend/hooks/offers.go`, `backend/hooks/cart_recovery.go`
- `backend/hooks/integrations.go`, `backend/hooks/webhooks.go`, `backend/hooks/api_*.go`
- `backend/hooks/member_*.go`
- `backend/services/connect.go`, `backend/services/subscription.go`, `backend/services/locale.go`
- `frontend/src/routes/admin/billing/`, `/connect/`, `/domain/`, `/members/`, `/integrations/`
- `frontend/src/components/admin/courses/` (course admin is cloud-only)
- `frontend/src/components/public/course-learner/` (course player is cloud-only)
- `frontend/src/components/public/MembershipGate.svelte`, `ContentGate.svelte`
- `services/`, `provisioning/`, `cli/`, `landing/` (cloud infrastructure)

---

## 8. Dependency Notes

Self-hosted is missing these deps that cloud uses:
- `culori` — used for color manipulation (accent hue → palette generation)
- `lucide-svelte` — cloud uses this extensively (self-hosted uses inline SVGs)
- `@axe-core/playwright` — a11y testing
- `vitest` + testing library — unit tests

If adopting cloud components that use `lucide-svelte`, either:
- (A) Add the dependency, or
- (B) Swap `Icon` component calls back to inline SVGs

Recommendation: Add `lucide-svelte` — it's lightweight and self-hosted already wants better icon consistency.

---

## 9. Conclusion

The **safest and highest-value** backports are:
1. **a11y hardening** (ConfirmDialog, dirty nav nudge, keyboard shortcuts)
2. **SkillsCategoryManager keyboard reordering** (closes #259, #404)
3. **MediaEditModal** (closes #348)
4. **CSS/SSR micro-fixes** (no-brainer)

These represent roughly **8–12 hours of focused work** and deliver measurable UX improvement with zero risk of SaaS feature leakage.

The **larger architectural backports** (multi-currency, i18n big-bang, SectionManager autosave, FacetChipStrip) should be evaluated individually after Phase 1 is complete and stable.

---

## 10. Tier 2 + Tier 3 Backports (Shipped May 2026)

Selected cloud "ops/admin" features were ported as **stripped-down single-tenant variants**. The cloud versions assume multi-tenant rate limiting, tier gating, credit reservation, and per-tenant scoping; the self-hosted variants drop all of that and operate on a single profile/database.

### Tier 2 — Read API + Integration Surface

- ✅ **API keys (read scopes)** — `read:profile`, `read:posts`, `read:projects`, `read:skills`, `read:experience`. Keys stored as SHA-256 hash, full key shown once, soft-revoke. Backed by new `api_keys` collection. `APIKeyMiddleware` enforces scope on `/api/v1/*`.
- ✅ **Webhooks dispatch infrastructure** — HMAC-SHA256 signing (`X-Facet-Signature`), SSRF-protected dialer, exponential backoff retry (6 attempts: 1m/5m/30m/2h/12h), auto-disable after 10 consecutive failures, per-webhook delivery log. Events: `post.published`, `comment.created`, `newsletter.subscribed`.
- ✅ **Webhook dispatch test event picker** — synthetic payloads to all active subscribers via the normal Dispatch pipeline.
- ✅ **AI resume parsing (BYO-key)** — PDF/DOCX upload, AI extracts structured data, smart deduplication. All AI features remain bring-your-own-key (no managed credits in self-hosted).

### Tier 3 — Write API + Ops Surfaces

- ✅ **API v1 writes** — `POST/PATCH/DELETE /api/v1/{posts,projects,skills,experience}` + `PATCH /api/v1/profile` (singleton, PATCH-only). New `write:*` scopes; read and write scopes are independent. Field allowlist per collection so callers can't sneak in protected fields.
- ✅ **System alerts inbox** — `/admin/alerts` with new `system_alerts` collection. Three severities (info/warning/critical), ack-one/ack-all, sidebar unread badge. `CreateSystemAlert` helper is lenient (logs and returns silently on failure).
- ✅ **Newsletter multi-list (segments)** — new `newsletter_lists` + `subscriber_list_memberships` collections. Per-list sender/reply-to/welcome email, atomic default swap, subscriber-count recompute hooks. Self-hosted has no list cap (`cap: 0`). Default-list backfill keeps existing instances unchanged.
- ✅ **Newsletter compose + lists UI** — `/admin/newsletter` (hub), `/admin/newsletter/lists`, `/admin/newsletter/compose`.
- ✅ **External media providers** — Loom, CodePen, Figma oEmbed (in addition to YouTube, Vimeo, SoundCloud). Brand icons.

### Explicitly Excluded from Self-Hosted

| Cloud Feature | Why Excluded |
|---------------|--------------|
| Tier-based rate limits / quotas | Self-hosted is single-user |
| Member tier gating on API keys | No member system in self-hosted |
| Newsletter list `cap` enforcement | No plan limits |
| Liquid personalization | Newsletter advanced — deferred |
| `webhook_events_log` (separate audit) | Per-webhook delivery log is sufficient |
| Managed AI credits / billing | All AI is BYO-key (OpenAI / Anthropic / Ollama) |
| `purchases.completed`, course events | No course system in self-hosted |

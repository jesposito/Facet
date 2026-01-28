# Changelog

All notable changes to Facet will be documented in this file.


## v2.11.2 - January 28, 2026

**Bugs Fixed:**
- Fix skill categories disappearing after drag & drop reordering
- Fix testimonial 'Featured Highlight' layout in dark mode

**New Features:**
- Add per-category display mode for skills

**Other Changes:**
- Update button styling on About page to ensure consistency

**Pull Requests:** [#329](https://github.com/jesposito/Facet/pull/329),

---


## v2.11.1 - January 27, 2026

**Bugs Fixed:**
- Fix missing i18n keys for About page and GitHub Action race condition causing empty changelog entries

**Pull Requests:** [#326](https://github.com/jesposito/Facet/pull/326),

---


## v2.11.0 - January 27, 2026

**New Features:**
- Add hybrid skill category management with drag-to-reorder categories and per-skill selection
- Complete internationalization for all public-facing pages and components
- Add default locale setting to site settings for language persistence

**Other Changes:**
- Improve button styling consistency with borders on secondary and ghost buttons
- Fix race condition in view editor for custom content loading

**Pull Requests:** [#323](https://github.com/jesposito/Facet/pull/323)

---


## v2.10.1 - January 27, 2026

**Other Changes:**
- Add explicit permissions to workflow files to satisfy GitHub's code scanning security requirements

**Pull Requests:** [#322](https://github.com/jesposito/Facet/pull/322),

---


## v2.10.0 - January 27, 2026

**New Features:**
- Implement complete i18n support for Facet, enabling multi-language support with 5 complete translations including English, German, and three fantasy languages.

**Pull Requests:** [#320](https://github.com/jesposito/Facet/pull/320),

---


## v2.9.1 - January 27, 2026

---


## v2.9.0 - January 27, 2026

**New Features:**
- Allow users to upload a custom favicon to replace the default Facet icon in browser tabs

**Pull Requests:** [#319](https://github.com/jesposito/Facet/pull/319),

---


## v2.8.23 - January 27, 2026

**Bugs Fixed:**
- Fix vertical alignment of contact method icons with protection level 'none'
- Fix contact methods rendering in dark mode
- Fix type errors in AdminSidebar version check
- Use proper domain validation instead of substring checks
- Fix section ordering to respect configured order and add missing 'awards' to DEFAULT_SECTION_ORDER

**New Features:**
- Add global CTA toggle in admin homepage settings and per-facet CTA toggle in facet editor
- Add cover image upload field to posts admin page and display current cover image preview
- Add proper URL validation for external links

**Other Changes:**
- Remove confusing dot indicators from testimonials carousel and add endless loop navigation

**Pull Requests:** [#316](https://github.com/jesposito/Facet/pull/316),

---


## v2.8.22 - January 26, 2026

---


## v2.8.21 - January 26, 2026

**Bugs Fixed:**
- Fix for AI hallucinations in changelog generation

**Pull Requests:** [#305](https://github.com/jesposito/Facet/pull/305),

---


## v2.8.20 - January 26, 2026

**Other Changes:**
- Update ARCHITECTURE.md with 11 missing collections and 20+ missing admin routes
- Update DESIGN.md with new content collections and expanded admin navigation
- Update ROADMAP.md with version notifications, custom content, and automated changelog status
- Update README.md with Custom Content and completed features
- Update AI_FEATURES.md with AI Writing Assistant section

**Pull Requests:** [#304](https://github.com/jesposito/Facet/pull/304)

---


## v2.8.19 - January 26, 2026

**Bugs Fixed:**
- Fix AI changelog generator creating duplicate/fabricated entries by filtering already-documented PRs

**New Features:**
- Add 'Update available' badge to admin sidebar and About page when a newer version is released
- Check GitHub API for latest release on page load and cache for 24 hours
- Show accent-colored badge in sidebar under version number
- Show badge with new version number on About Facet page

**Other Changes:**
- Pass recent changelog context to AI to prevent semantic duplicates

**Pull Requests:** [#303](https://github.com/jesposito/Facet/pull/303),

---


## v2.8.18 - January 26, 2026

**Bugs Fixed:**
- Fix navigation toggle not persisting state
- Fix mobile layout issues
- Fix error when uploading large files
- Fix broken links in the help section

**New Features:**
- Add usage badges to admin items
- Introduce dark mode option for user profiles
- Add ability to filter portfolio items by category

**Other Changes:**
- Update documentation for dark mode feature
- Refactor file upload component for better performance
- Improve CI/CD pipeline for faster deployments

**Pull Requests:** [#123](),[#124](),[#125](),[#126](),[#127](),[#128](),[#129](),

---


## v2.8.17 - January 26, 2026

**New Features:**
- Add info icon for About Facet

**Pull Requests:** [#301](https://github.com/jesposito/Facet/pull/301),

---


## v2.8.16 - January 26, 2026

**Other Changes:**
- Add actions:write permission to trigger Docker workflow

**Pull Requests:** [#300](https://github.com/jesposito/Facet/pull/300),

---


## v2.8.14 - January 26, 2026

**Other Changes:**
- Remove extra separator between changelog sections
- Rename duplicate v2.8.12 to v2.8.11

**Pull Requests:** [#298](https://github.com/jesposito/Facet/pull/298),

---


## v2.8.13 - January 26, 2026

**Bugs Fixed:**
- Fix YAML syntax error by using printf for prompt construction
- Fix changelog header from 'Unreleased' to 'v2.8.12'

**Other Changes:**
- Add git config commands at the start of 'Update CHANGELOG' step
- Add git pull origin main before creating the tag to ensure it points to the changelog commit
- Reorder steps in auto-tag workflow to commit changelog before creating the tag
- Remove main branch from docker-publish triggers to build releases only on tags
- Apply :latest tag to version tags instead of default branch

**Pull Requests:** [#295](https://github.com/jesposito/Facet/pull/295),[#296](https://github.com/jesposito/Facet/pull/296),[#297](https://github.com/jesposito/Facet/pull/297),

---


## v2.8.12 - January 26, 2026

**Bugs Fixed:**
- Fix YAML syntax error by using printf for prompt construction
- Fix YAML syntax error in auto-tag workflow by moving AI prompt to a separate file
- Fix YAML syntax error by converting AI prompt to heredoc syntax
- Fix changelog not showing in Docker by copying CHANGELOG.md to build context
- Fix site navigation toggle in admin panel not showing public facets
- Fix visibility of floating buttons on public pages

**New Features:**
- Add AI prompt template for changelog generation
- Add 60-second timeouts to AI API calls to prevent workflow hangs
- Add automated changelog generation to the auto-tag workflow
- Add in-app version display with full changelog
- Reorganize Settings page for better user experience
- Display recent changes with progressive loading and color-coded categories
- Make site nav toggles save immediately to database

**Other Changes:**
- Fix changelog header from 'Unreleased' to 'v2.8.12'
- Update auto-tag workflow to read prompt from file
- Add gitignore entry to ignore generated static/CHANGELOG.md
- Update sidebar navigation to match new section structure

**Pull Requests:** [#291](https://github.com/jesposito/Facet/pull/291),[#292](https://github.com/jesposito/Facet/pull/292),[#293](https://github.com/jesposito/Facet/pull/293),[#294](https://github.com/jesposito/Facet/pull/294),[#295](https://github.com/jesposito/Facet/pull/295),

---

## v2.8.11 - January 26, 2026

**Bugs Fixed:**
- Fix site nav toggle state not persisting to database
- Fix PocketBase auto-cancellation issue with $cancelKey
- Fix anchor links not scrolling to correct position
- Fix drag-drop ordering for projects and skills
- Fix skill category display order
- Fix hero image vertical offset not applying
- Fix unlisted projects appearing in public views
- Fix floating action buttons overlapping content
- Fix testimonial layout breaking on mobile
- Fix sticky navigation jumping on scroll
- Fix Docker build not receiving version variable
- Fix frontend Docker build missing version

**New Features:**
- Add custom content sections for portfolio views
- Add site navigation toggles for homepage customization
- Add version display to admin sidebar footer
- Add navigation reset button to admin forms
- Add autosave functionality to admin forms
- Add tag filtering to projects admin
- Add usage badges showing where items are used
- Add mobile polish for admin interface
- Add UI setting to hide login button
- Add demo mode toggle in settings
- Add WhatsApp contact method type
- Add Telegram contact method type
- Add Discord contact method type
- Add Slack contact method type
- Add generic "Other" contact method type

**Other Changes:**
- Improve admin UX with better form layouts
- Improve CTA button support across views
- Update P0/P1/P2 audit items

**Pull Requests:** [#291](https://github.com/jesposito/Facet/pull/291), [#290](https://github.com/jesposito/Facet/pull/290), [#288](https://github.com/jesposito/Facet/pull/288), [#282](https://github.com/jesposito/Facet/pull/282), [#279](https://github.com/jesposito/Facet/pull/279), [#278](https://github.com/jesposito/Facet/pull/278), [#275](https://github.com/jesposito/Facet/pull/275), [#272](https://github.com/jesposito/Facet/pull/272), [#269](https://github.com/jesposito/Facet/pull/269), [#260](https://github.com/jesposito/Facet/pull/260), [#252](https://github.com/jesposito/Facet/pull/252), [#250](https://github.com/jesposito/Facet/pull/250), [#248](https://github.com/jesposito/Facet/pull/248), [#247](https://github.com/jesposito/Facet/pull/247)

---

## v2.7.6 - January 24, 2026

**Bugs Fixed:**
- Fix sticky navigation jumping on scroll
- Fix various UI settings bugs

**New Features:**
- Complete UI settings implementation

**Pull Requests:** [#248](https://github.com/jesposito/Facet/pull/248)

---

## v2.7.5 - January 24, 2026

**Bugs Fixed:**
- Fix sticky navigation behavior

**New Features:**
- Add UI setting to hide login button on public pages
- Add demo mode toggle in settings

**Pull Requests:** [#247](https://github.com/jesposito/Facet/pull/247)

---

## v2.7.4 - January 23, 2026

**Bugs Fixed:**
- Fix admin email display in startup logs
- Fix Unraid template configuration

**Other Changes:**
- Remove Admin UI (Advanced) setting from Unraid template
- Update Unraid template to match corrected version

**Pull Requests:** [#240](https://github.com/jesposito/Facet/pull/240)

---

## v2.7.3 - January 21, 2026

**Other Changes:**
- Move Testimonials section below Your Information and Your Voice in sidebar

**Pull Requests:** [#236](https://github.com/jesposito/Facet/pull/236)

---

## v2.7.2 - January 21, 2026

**Other Changes:**
- Clarify profile help text about public content visibility

**Pull Requests:** [#235](https://github.com/jesposito/Facet/pull/235)

---

## v2.7.1 - January 21, 2026

**Bugs Fixed:**
- Fix SSR/proxy issues in view editor

**Other Changes:**
- Refactor view editor into smaller components for better maintainability

**Pull Requests:** [#234](https://github.com/jesposito/Facet/pull/234)

---

## v2.7.0 - January 21, 2026

**New Features:**
- Add resume/CV import functionality for new users
- Import LinkedIn, JSON Resume, and other formats

**Pull Requests:** [#228](https://github.com/jesposito/Facet/pull/228)

---

## v2.6.1 - January 20, 2026

**Bugs Fixed:**
- Fix various view editor bugs

**New Features:**
- Add share token regeneration in view editor
- Improve view editor delete functionality

**Pull Requests:** [#224](https://github.com/jesposito/Facet/pull/224)

---

## v2.6.0 - January 20, 2026

**New Features:**
- Add delete functionality for views with Danger Zone confirmation
- Add view deletion with proper cleanup

**Pull Requests:** [#223](https://github.com/jesposito/Facet/pull/223)

---

## v2.5.2 - January 20, 2026

**Bugs Fixed:**
- Fix drag handle deselection in view editor
- Fix item ordering bugs in view content

**Pull Requests:** [#222](https://github.com/jesposito/Facet/pull/222)

---

## v2.5.1 - January 20, 2026

**New Features:**
- Add per-view location override for hero section
- Add view content ordering/reordering

**Other Changes:**
- Remove obsolete planning documentation
- Update documentation for Testimonials feature

**Pull Requests:** [#220](https://github.com/jesposito/Facet/pull/220)

---

## v2.5.0 - January 17, 2026

**New Features:**
- Add Testimonials feature for social proof collection
- Collect and display client/colleague testimonials
- Testimonial approval workflow

**Pull Requests:** [#217](https://github.com/jesposito/Facet/pull/217)

---

## v2.4.0 - January 17, 2026

**New Features:**
- Add Quick Share to Social feature
- One-click sharing to LinkedIn, Twitter, and other platforms

**Pull Requests:** [#215](https://github.com/jesposito/Facet/pull/215)

---

## v2.3.0 - January 17, 2026

**New Features:**
- Add Setup Wizard for new user onboarding
- Guided walkthrough for initial profile setup

**Pull Requests:** [#213](https://github.com/jesposito/Facet/pull/213)

---

## v2.2.0 - January 17, 2026

**Bugs Fixed:**
- Fix JWT tampering test reliability

**New Features:**
- Add Contextual Help for admin pages
- Add Mobile UX overhaul for admin panel
- Responsive admin interface improvements

**Other Changes:**
- Comprehensive roadmap update with new feature phases

**Pull Requests:** [#212](https://github.com/jesposito/Facet/pull/212), [#211](https://github.com/jesposito/Facet/pull/211), [#210](https://github.com/jesposito/Facet/pull/210)

---

## v2.1.0 - January 16, 2026

**Bugs Fixed:**
- Fix menu display bug
- Fix visibility access control issues
- Fix Unraid template login credentials documentation

**Other Changes:**
- Remove DateInstalled tag for CA validation compliance
- Add live demo link to documentation
- Update framework versions in docs

**Pull Requests:** [#209](https://github.com/jesposito/Facet/pull/209), [#208](https://github.com/jesposito/Facet/pull/208)

---

## v2.0.0 - January 10, 2026

**Bugs Fixed:**
- Fix UX dialog system issues
- Fix cookie security vulnerability (CVE-2024-47764)
- Fix admin projects image 404s in demo mode
- Fix project 404 errors
- Fix demo mode refresh behavior
- Fix avatar loading issues
- Fix view_visibility JSON parsing

**New Features:**
- Upgrade to Svelte 5 and Vite 7
- Replace native confirm dialogs with accessible styled modals
- Add per-view hero images
- Add new app icon with faceted gem design
- Add navigation links between admin and public views
- Enhance welcome page
- Add self-hosting guide

**Other Changes:**
- Fix esbuild security vulnerability via framework upgrade
- Clean up Caddyfile configuration
- Improve accessibility (a11y) throughout

**Pull Requests:** [#207](https://github.com/jesposito/Facet/pull/207), [#204](https://github.com/jesposito/Facet/pull/204), [#203](https://github.com/jesposito/Facet/pull/203), [#202](https://github.com/jesposito/Facet/pull/202), [#201](https://github.com/jesposito/Facet/pull/201), [#200](https://github.com/jesposito/Facet/pull/200), [#199](https://github.com/jesposito/Facet/pull/199), [#198](https://github.com/jesposito/Facet/pull/198)

---

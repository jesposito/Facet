# Changelog

All notable changes to Facet will be documented in this file.


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

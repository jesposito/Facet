# Roadmap and known gaps

This page tracks what self-hosted Facet does today, what's only partly built, and ideas that haven't been built. It's written from the code in this repository. For what changed in each release, see the [changelog](../CHANGELOG.md). To suggest or discuss work, use [GitHub issues](https://github.com/jesposito/Facet/issues).

## Built

Each of these works end to end in the current code. Follow the links for details.

- **Profile and content**: experience, projects, education, certifications, awards, skills, posts, talks, custom content and contact methods, with drafts, ordering, duplication and bulk actions ([Features](FEATURES.md#your-content))
- **Views** with section and item selection, per-item overrides, hero and call-to-action settings, live preview, and a default view ([Features](FEATURES.md#views))
- **Visibility, share links and password views** ([Features](FEATURES.md#visibility))
- **Project experiences**: case studies, video courses, code walkthroughs ([Project experiences](experiences/README.md))
- **Media library** with uploads, external embeds, thumbnails, usage tracking and orphan cleanup ([Media](MEDIA.md))
- **Site design**: font packs, hero layouts, colours, dark mode, navigation options, custom CSS, favicon
- **Testimonials** with request links, email verification, approval and display layouts ([Features](FEATURES.md#testimonials))
- **Comments** with threading, spam heuristics, reports and moderation ([Features](FEATURES.md#comments))
- **Built-in analytics** ([Features](FEATURES.md#analytics))
- **GitHub import** with review and field locking, and **résumé import** with AI ([Features](FEATURES.md#importing-content))
- **AI writing assistant, GitHub enrichment and AI résumé generation**, bring your own provider ([AI features](AI_FEATURES.md))
- **Export** to JSON or YAML, **print styles**, **RSS**, **iCal** and **sitemap** ([Features](FEATURES.md#exporting-your-data))
- **Two-factor sign-in**, OAuth sign-in, and an audit log ([Security](SECURITY.md))
- **Scoped REST API and signed webhooks** ([API](API.md))
- **Built-in daily backups and restore** ([Upgrades, backups and recovery](UPGRADE.md#built-in-backups))
- **Five interface languages** with CI checks ([Translations](../CONTRIBUTING_TRANSLATIONS.md))
- **Demo mode** ([Features](FEATURES.md#demo-mode))
- **Multi-architecture Docker images** with signatures, provenance and vulnerability scanning ([Contributing](../CONTRIBUTING.md#releases-and-published-images))

## Partly built

These exist in the code but have a gap that affects how you can use them.

| Area | What works | What's missing |
|------|------------|----------------|
| Newsletter links | Personal placeholders, site and list placeholders in links | `{{ subscriber.* }}` placeholders are refused inside links ([details](AUDIENCE.md#placeholders)) |
| Payments | Stripe Checkout, emailed access links, partial and full refunds | Tested with signed fake Stripe events, not a live account. Partial refund amounts aren't stored ([details](AUDIENCE.md#payments-stripe)) |
| Courses | Public pages, enrolment, drip and sequential release, quizzes, certificates | Tested with a captured mailer, not live SMTP ([details](AUDIENCE.md#courses)) |
| System alerts | Settings links to recovery, backup, restore and retention alerts | SMTP and webhook failures use their operation status and logs ([details](API.md#system-alerts)) |
| Contact protection | Obfuscation and click-to-reveal | The CAPTCHA level hides the value from everyone, because no CAPTCHA check exists ([details](CONTACT_PROTECTION.md)) |
| Analytics | Visits, referrers, campaigns, share links | The countries panel is always empty |
| Google Analytics | Settings field and script injection | Blocked by the shipped Content-Security-Policy ([details](SECURITY.md#google-analytics)) |
| API keys | Read and write scopes, allowed origins, expiry, revocation | Scopes and origins can't be changed after a key is created. Allowed origins only limit browsers; they aren't authentication ([details](API.md#allowed-origins)) |
| Comment moderation | Spam heuristics, reports, moderation page | No control for "hold all comments for approval" |

## Not built

Ideas that have come up in earlier planning but aren't in the code. Listing them here isn't a promise they'll be built.

- Cloudflare Turnstile behind the CAPTCHA contact protection level
- Scheduled GitHub sync (refresh is manual today)
- GitHub, Twitter or LinkedIn verification for testimonials
- A LinkedIn importer
- QR codes for views and share links
- Newsletter drip sequences and full template personalisation
- Expiring signed URLs for uploaded files
- Offline support (PWA)
- An "Apply with Facet" button for sending a view to an employer
- Ready-made visual themes beyond the current design options
- A switch to turn off built-in visit logging, and automatic expiry of old logs

# Design principles

Why Facet is shaped the way it is. This page records the ideas behind decisions, so changes can be judged against them. For how the pieces fit together technically, see [Architecture](ARCHITECTURE.md). For what is and isn't built, see the [Roadmap](ROADMAP.md).

## What Facet is

A self-hosted, single-owner personal site for your professional life: the profile part of a professional network without the network, combined with a résumé, a portfolio and a place to write. Its distinguishing idea is **views**: one set of content, shown differently to different audiences.

It's meant to be something a thoughtful person runs for themselves, not a platform that runs people.

## Principles

| Principle | What it means in practice |
|-----------|---------------------------|
| **One owner** | One person owns an instance. There are no roles, teams or tenants. Signing in means full control. |
| **Self-hosted and simple to run** | One container, one exposed port, two folders to back up. No external database, queue or cache. |
| **Your data stays yours** | SQLite on your disk. Export it, back it up, or delete it whenever you like. |
| **Explicit sharing** | Nothing becomes more visible than you made it. Unlisted views need a link, password views need a password, private means private. |
| **Quiet** | No feeds, likes, follows or engagement nudges. Analytics exist for the owner's curiosity, not to drive behaviour. |
| **Powerful, but out of the way** | Rich features (views, AI, newsletter, API) stay hidden until you go looking for them. |
| **Review before publish** | Imports and AI output become proposals or previews you accept. Nothing is auto-published on your behalf. |

## Things Facet deliberately doesn't do

- engagement-driven social features: feeds, likes, follows, infinite scroll
- private messaging between visitors
- multi-tenant hosting, user roles, or team accounts
- advertising, or growth loops
- third-party tracking in the default setup

Some features sit close to these lines and are designed to stay on the right side of them. Comments are off until you enable them for an item, and you moderate them. Email exists for practical jobs (testimonial verification, notifications, newsletters you send), not as a messaging system. Built-in analytics stay on your server.

## Views: one source of truth, many lenses

Different audiences want different things. A recruiter wants employment history and skills. A conference organiser wants talks. A client wants case studies and testimonials. Your family wants the photos.

Without views you'd have three bad options: show everything to everyone, keep several profiles in sync by hand, or flip visibility switches back and forth. Views avoid all three. Content is written once, and each view selects, orders and frames it.

Some rules follow from that:

- **An empty selection means "everything visible".** A view with no hand-picked items in a section shows all eligible items, so adding new content doesn't mean editing every view.
- **Framing can change; facts can't.** A view can override how an item is described: a job title emphasis, a summary, bullets. It can't change facts such as company names, dates, institutions or issuers.
- **Identity is shared.** Your name, avatar and contact links come from the profile. A view can change the headline, summary, hero image, call to action and accent colour.
- **One default.** Exactly one view is the default for `/`. Facet clears the flag from others when you set it.

### Example: a career pivot

A product designer who also builds training materials can describe the same role two ways:

```text
Source: "Product Designer @ Acme Corp"

View "UX Designer"
  title:       Senior UX Designer
  description: Led user-centred design work…

View "Instructional Designer"
  title:       Learning Experience Designer
  description: Built scalable onboarding programmes…
```

Both views point at the same record. Change the dates once and both update.

## Addresses

- **Views live at the top level** (`/recruiter`, not `/v/recruiter`). Short, readable addresses look better on a CV or a slide. Old `/v/` links redirect.
- **Share links redirect to the clean address.** `/s/<token>` validates the link, stores it in a cookie and redirects, so the token doesn't sit in browser history or leak through the `Referer` header.
- **Reserved words** that would collide with Facet's own pages can't be used as view slugs ([list](../frontend/src/params.ts)).
- **Hidden things look absent.** An unlisted view without a link, or a private item, returns "not found", so its existence isn't confirmed.

## Imports are proposals

GitHub imports and AI-assisted changes produce a proposal you review field by field. Fields you've edited by hand can be locked, so refreshing from the source never overwrites your words. AI prompts tell the model not to invent facts or numbers, and to write plainly.

## The admin should feel calm

| Idea | In practice |
|------|-------------|
| Calm, not a CMS | Information density is earned. No dashboard for its own sake. |
| Empty isn't broken | First runs get welcoming guidance, not error states or rows of zeros |
| Tending, not managing | Editing your site should feel more like gardening than filling in forms |
| Accessible | Keyboard support, screen-reader labels and enough colour contrast, with browser tests guarding contrast rules |

Public pages aim to be readable, fast (server-rendered, light on JavaScript), accessible, printable, and well described to search engines and link previews.

## Accepted trade-offs

| Trade-off | Why it's acceptable |
|-----------|---------------------|
| SQLite only | One owner doesn't need horizontal scaling. Backups are file copies. |
| A single container | The simplest thing to run and the easiest to explain |
| Local file storage | No external storage service to configure or pay for |
| Password-view tokens can't be revoked early | They last an hour. Statelessness keeps the design simple. |
| PocketBase hasn't reached 1.0 | It's widely used, Facet pins its version, and dependency updates go through the full test workflow |
| One profile per instance | Family members run their own instances |
| No content version history | Backups are the safety net |
| Rate limits live in memory | One process, one owner. A restart resetting limits is fine. |

## Where to extend, and where not to

Good places to extend: new import sources (the `sources` collection and a service), new AI provider types (`ai_providers` and the AI service), new view sections, new design options, new webhook events for features that exist.

Places not to extend without a serious rethink: multiple owners or roles, social features, tracking, and anything that assumes a hosted multi-tenant service. Facet has no plugin system. Extensions are changes to the code.

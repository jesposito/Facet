# Newsletter, payments and courses

These features help you build an audience and sell to it. They're the youngest parts of self-hosted Facet. This page explains what each one does today and where its limits are.

| Feature | State in this version |
|---------|-----------------------|
| [Newsletter](#newsletter) | Double opt-in sign-ups, list targeting, idempotent sends, one-click and page unsubscribe. Link placeholders have limits. |
| [Payments (Stripe)](#payments-stripe) | Stripe Checkout, access links delivered by email, partial and full refunds |
| [Courses](#courses) | On by default. Public course pages, lessons, drip and sequential release, enrolment, quizzes and certificates. |

The automated tests for these features use a captured mailer and signed fake Stripe events. They haven't been run against a live SMTP server or a live Stripe account, so send yourself a real test before you rely on either.

## Newsletter

The newsletter sends email from your own SMTP server to people who sign up on your site. You'll find it in the sidebar under **Newsletter**.

### Before you start

1. **Set up email.** Configure SMTP in **Site Settings** or with `SMTP_*` variables ([Configuration](CONFIGURATION.md#email-smtp)), and send yourself a test.
2. **Set `APP_URL`.** Confirmation, unsubscribe and tracking links are built from PocketBase's application URL, which Facet sets from `APP_URL` at every start ([source](../backend/hooks/smtp.go)). Without it, PocketBase falls back to `http://localhost:8090` and your emails contain links nobody else can open.

### Sign-ups and consent

Sign-up is double opt-in. Someone who fills in the form gets a confirmation email, and only becomes an active subscriber after following its link (`/api/public/newsletter/confirm/<token>`). Repeat requests are held to a 24-hour confirmation cooldown.

- The form can name a list by id or slug. Without one, the default list is used.
- Only lists you've marked public can be joined from the site. Private, paused and unknown lists all get the same refusal, and nothing is stored.
- The list is joined when the subscriber confirms, not before. Someone already subscribed who asks for a second list confirms that one from their inbox too.
- A subscriber holds one pending list at a time.
- `GET /api/public/newsletter/lists` shows the public lists, without subscriber counts.

The form also has a hidden honeypot field to catch simple bots.

### Lists

A fresh install has one list, called "Newsletter". Under **Newsletter → Lists** you can add more, and for each one set a name, slug, description, sender name, reply-to address, welcome email and whether it's public. Lists can be paused, and one list is always the default.

### Writing and sending

Under **Newsletter → Compose**:

1. Write a subject and a body in Markdown, and pick a template.
2. Choose the audience: one list, or **All active subscribers**. Counts show active readers only.
3. Send a test to yourself and check it in a real inbox.
4. Send. Delivery runs in the background, up to 4 sends at a time (`NEWSLETTER_MAX_CONCURRENT_SENDS`).

Each send carries a request key, and the database refuses a reused key or a second send running at once ([source](../backend/hooks/newsletter.go)). In practice:

- A network retry with the same key gets the original send back and sends nothing new.
- The same key with different content is refused.
- The same newsletter to a different list counts as a new send.
- Sending the same newsletter again on purpose needs an explicit resend with a new key.
- Sends from before this version have no recorded audience, so they're matched on subject and body alone and still block an accidental repeat.

### Placeholders

`{{ site.url }}`, `{{ site.name }}` and `{{ list.name }}` work in links. `{{ subscriber.name }}` and `{{ subscriber.email }}` are refused inside links, on preview, test and send alike, because click tracking can only follow links stored in the body, and a per-reader link would hand the reader's details to the linked site. With personalisation off, any placeholder in a link is refused, because it would go out broken.

### Open and click tracking

Each newsletter includes a tracking image and routes its links through your server, so Facet can count opens and clicks per send. Many mail apps block tracking images, so open counts are approximate. If you have subscribers in places with strict privacy rules, mention the tracking in your sign-up text.

### Unsubscribing

Each newsletter offers two ways out:

- **The mail app's own unsubscribe button**, from the standard `List-Unsubscribe` headers with one-click support.
- **A link in the footer** to the `/unsubscribe/<token>` page on your site.

## Payments (Stripe)

Under **Commerce** in the sidebar you'll find:

- **Payments**: your Stripe secret key and webhook signing secret (both stored encrypted), with a connection test. `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` work as fallbacks.
- **Bulk Pricing**: prices on posts, talks, projects and courses.
- **Coupons**: percentage or fixed-amount codes with usage limits.

Point Stripe's webhook at `<APP_URL>/api/content/webhook`.

### How a purchase works

1. The buyer starts checkout (`POST /api/content/checkout`). Success and cancel addresses must match your configured `APP_URL` exactly: scheme, host and port. `Host` and `X-Forwarded-Host` headers are ignored. Plain HTTP is accepted only if `APP_URL` itself is HTTP, as on a home network. If no app URL is configured, checkout answers 503.
2. Stripe confirms payment through the signed webhook.
3. Facet emails an access link to the address the buyer paid with. Access is proven by that inbox, not by the browser that happened to finish checkout.
4. The access link issues a small token tied to the buyer's email. Every request checks the database for a completed purchase, so this works for buyers with more than 100 purchases, and a refund ends access straight away.

Free-coupon purchases never go through the buyer's inbox, so their link is limited to that single purchase.

### Refunds

Facet reads Stripe's running refund total. A partial refund keeps access; a full refund ends it. Once a purchase is refunded, nothing reactivates it: not repeated events, late partial-refund events, checkout webhook retries or the return page. Partial refund amounts are logged but not stored.

The lock that orders checkout and refund handling for one session works within a single process, which is how the container runs.

## Courses

Courses are **on** by default for self-hosted sites ([source](../backend/services/plan.go)). Set `FACET_FEATURE_COURSES=false` to turn them off; the repository's `docker-compose.yml` doesn't pass that variable, so add it to `environment:` ([Configuration](CONFIGURATION.md#features-and-tuning)). The two controls work at different levels:

- `FACET_FEATURE_COURSES=false` is read once at startup. Facet then doesn't register its course and quiz endpoints at all ([course.go](../backend/hooks/course.go), [quiz.go](../backend/hooks/quiz.go)), so those `/api` requests get `404` whatever the dashboard says. Course pages that depend on those endpoints stop working, for learners and for you.
- The **Settings → Features** switch is a site setting you can change at any time without restarting. Turning it off closes the catalogue, new enrolments and course checkout, while existing learners keep their access ([below](#turning-courses-off)).

Use the switch to stop taking new learners for a while. Use the variable when you don't want the course endpoints running at all.

In **Courses** in the sidebar you build courses from modules and lessons, add quizzes, issue completion certificates, and see enrolments and progress.

### What learners get

- A public outline at `/courses/<slug>` and lessons under `/courses/<slug>/learn`. Draft and private courses answer 404 to everyone except you.
- **Enrolment**: paid courses need a purchase. Free courses can require enrolment, which is confirmed by an emailed link (`/api/courses/<slug>/enroll/verify`).
- **Release rules**: lessons can be held back by drip timing or by sequential order. Preview lessons skip both.
- **Quizzes and progress**: one access check runs before a quiz is shown or submitted, a lesson is completed, or progress is read. A refused request stores nothing. Quiz routes answer 403 when there's no purchase or enrolment and 404 for hidden courses.
- **Certificates** on completion, with a public check at `/api/certificates/verify/<id>`.

### Turning courses off

Turning courses off closes the doors new learners would use: the public catalogue, new enrolments and course checkout ([feature gate](../backend/hooks/feature_gate.go)). People who already bought or enrolled keep their access. Access links, the Stripe webhook, enrolment verification and course pages stay open, and the course page tells newcomers it isn't taking new learners.

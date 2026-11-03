# Newsletter, payments and courses

These features help you build an audience and sell to it. They're the youngest parts of self-hosted Facet. This page explains what each one does today and where its limits are.

| Feature                               | State in this version                                                                                                    |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| [Newsletter](#newsletter)             | Double opt-in sign-ups, list targeting, idempotent sends, one campaign at a time with resume, signed one-click and page unsubscribe. Sent through your own SMTP server only. Link placeholders have limits. |
| [Payments (Stripe)](#payments-stripe) | Stripe Checkout, access links delivered by email, partial and full refunds                                               |
| [Courses](#courses)                   | On by default. Public course pages, lessons, drip and sequential release, enrolment, quizzes and certificates.           |

The automated tests for these features use a captured mailer and signed fake Stripe events. They haven't been run against a live SMTP server or a live Stripe account, so send yourself a real test before you rely on either.

## Newsletter

The newsletter sends email to people who sign up on your site, through an SMTP server you run or rent. You'll find it in the sidebar under **Newsletter**.

Self-hosted Facet sends every email, newsletters included, through that SMTP server and nothing else. There's no API-based provider, Facet-managed sender or local `sendmail` fallback. If SMTP isn't set up, nothing is sent.

### Before you start

1. **Set up SMTP.** Configure it in **Site Settings** or with `SMTP_*` variables ([Configuration](CONFIGURATION.md#email-smtp)), including a sender (From) address on a domain you control.
2. **Set `APP_URL`.** Confirmation, unsubscribe and tracking links are built from PocketBase's application URL, which Facet sets from `APP_URL` at every start ([source](../backend/hooks/smtp.go)). Without it, PocketBase falls back to `http://localhost:8090` and your emails contain links nobody else can open.
3. **Check readiness.** Newsletter pages ask the server whether sending is ready, not just whether a host is filled in. If SMTP is configured but the sender address or public site URL is missing, the page says so and the send buttons stay off.
4. **Set up your domain's mail records.** SPF, DKIM and DMARC for the sender's domain are yours to publish, usually following your SMTP provider's instructions. Facet doesn't create or check them. Without them, many inboxes will file your newsletter as spam or refuse it.

### Sign-ups and consent

Sign-up is double opt-in. Someone who fills in the form gets a confirmation email, and only becomes an active subscriber after following its link (`/api/public/newsletter/confirm/<token>`). Once a confirmation email has been accepted by your SMTP server, repeat requests for the same address wait 24 hours. If sending the confirmation fails, or the SMTP server's reply is lost, only a short cooldown applies, so the person can submit the form again soon; a new request replaces the earlier link. Facet never retries a confirmation on its own. A link works for seven days from when it was issued, so one that arrived despite a lost reply still confirms.

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
3. Send a test to yourself and check it in a real inbox. "Test sent" means your SMTP server accepted the message, not that it arrived: look in the inbox and the spam folder.
4. Send. Delivery runs in the background, one campaign at a time ([below](#delivery-pacing-and-resume)).

Each send carries a request key, and the database refuses a reused key or a second send running at once ([source](../backend/hooks/newsletter.go)). In practice:

- A network retry with the same key gets the original send back and sends nothing new.
- The same key with different content is refused.
- The same newsletter to a different list counts as a new send.
- Sending the same newsletter again on purpose needs an explicit resend with a new key. This also applies when an earlier send has only uncertain outcomes, because those messages may have been accepted.
- Sends from before this version have no recorded audience, so they're matched on subject and body alone and still block an accidental repeat.

### Delivery, pacing and resume

Facet sends one campaign at a time, one message after another, with a pause between messages set by `NEWSLETTER_SEND_INTERVAL_MS` (default 1000 ms, so at most about 3,600 messages an hour). While a campaign is sending, a new send or a resume is refused. The older `NEWSLETTER_MAX_CONCURRENT_SENDS` setting no longer allows parallel campaigns ([Configuration](CONFIGURATION.md#features-and-tuning)).

Connection, authentication and service-unavailable failures pause the campaign rather than attempting the whole list against a broken relay. Remaining recipients stay pending until the operator fixes SMTP and resumes.

Most SMTP providers limit messages per minute, hour or day. Set the interval so a full campaign stays under your provider's limits; Facet doesn't know them. A large list on a strict plan may need a long interval and hours to finish.

Each recipient's outcome is recorded as the campaign runs, and the send history shows them:

| Shown as | Meaning |
|----------|---------|
| **Accepted by SMTP** | Your SMTP server accepted the message. This is the `sent_count` figure. It doesn't mean the message reached an inbox. |
| **Failed** | Submission failed before acceptance, or the SMTP server explicitly refused it. |
| **Pending** | Not attempted yet, for example because the campaign stopped. |
| **Uncertain** | An attempt ended without a trustworthy recorded outcome, for example after a disconnect or crash. The message may or may not have been accepted. |
| **Skipped** | Deliberately not sent by the server, such as an address that was no longer an active subscriber when its turn came. |

A campaign ends as **Completed** (submitted messages were accepted, with skipped recipients counted separately), **Partially completed**, **Failed** or **Interrupted** (for example after a restart, an unavailable relay or an uncertain attempt). For a partial, failed or interrupted campaign, **Resume** in the send history retries only the pending and failed recipients. Accepted and uncertain recipients are never sent the campaign again, so a resume can't double-send to anyone whose outcome is unknown. The price is that some uncertain recipients may never get it.

Send logs from before per-recipient tracking keep their recorded counts as history, but there's no record of who is left, so they can't be resumed. If one was still marked as sending when Facet restarted, it's shown as interrupted with its counts unchanged.

Facet can't promise exactly-once delivery: if the server stops between the SMTP server accepting a message and Facet recording that, the campaign recipient is marked uncertain rather than retried. A failed SMTP connection or refused message never marks a valid subscriber as bounced or removes them from a list.

Facet doesn't receive bounce or complaint reports. It doesn't process delivery status notifications, feedback loops or spam complaints, and it doesn't suppress addresses because of them. Watch your SMTP provider's dashboard for bounces and complaints, and remove problem addresses under **Subscribers** yourself.

### Placeholders

`{{ site.url }}`, `{{ site.name }}` and `{{ list.name }}` work in links. `{{ subscriber.name }}` and `{{ subscriber.email }}` are refused inside links, on preview, test and send alike, because click tracking can only follow links stored in the body, and a per-reader link would hand the reader's details to the linked site. With personalisation off, any placeholder in a link is refused, because it would go out broken.

### Open and click tracking

Each newsletter includes a tracking image and routes its links through your server, so Facet can count opens and clicks per send. Many mail apps block tracking images, so open counts are approximate. If you have subscribers in places with strict privacy rules, mention the tracking in your sign-up text.

### Unsubscribing

Each newsletter offers two ways out:

- **The mail app's own unsubscribe button**, from the standard `List-Unsubscribe` headers with one-click support.
- **A link in the footer** to the `/unsubscribe/<token>` page on your site.

Both carry a signed token for that subscriber. An email address on its own can't unsubscribe anyone, so nobody can remove someone else by typing their address. Opening `/unsubscribe` without a valid token explains that the reader needs the link from one of their newsletter emails. Unsubscribing keeps working when the newsletter feature is switched off.

## Payments (Stripe)

Under **Commerce** in the sidebar you'll find:

- **Payments**: your Stripe secret key and webhook signing secret (both stored encrypted), with a connection test. `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` work as fallbacks.
- **Bulk Pricing**: prices on posts, talks and projects.
- **Coupons**: percentage or fixed-amount codes with usage limits.

Selling courses is only available on Facet Cloud. Self-hosted sites can publish courses for free ([below](#courses)), and other paid content works as described here.

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

### Course sales are Facet Cloud-only

Self-hosted Facet has courses, free authoring and the free learner experience, but it doesn't sell courses. The server refuses every course checkout with `403` and `"reason": "course_sales_unavailable"` before any coupon is claimed, purchase is stored or Stripe is called. That includes 100% coupons and forged requests, and no environment variable (`FACET_MANAGED` or `FACET_FEATURE_*`) turns it on. Coupons can't be scoped to courses, and a course can't be created paid, switched to paid or repriced. `GET /api/plan` reports `features.course_sales: false`, and course API responses carry `sales_available: false`.

A paid course imported from elsewhere keeps its tier and price: it is never turned free and its paid lessons stay locked. Its page says it isn't available to buy here. Learners who bought it earlier keep their access through their access links.

Legacy payments: if Stripe reports a course checkout session as paid that was started before this build (for example a delayed bank payment settling later), the webhook still records the purchase and emails the access link, because the buyer has already been charged. This can't be used to start a sale: this server never creates a course checkout session, and the webhook only accepts events signed with your webhook secret. Unpaid course sessions grant nothing, as for any other content.

### What learners get

- A public outline at `/courses/<slug>` and lessons under `/courses/<slug>/learn`. Draft and private courses answer 404 to everyone except you.
- **Enrolment**: imported paid courses open only to earlier buyers ([above](#course-sales-are-facet-cloud-only)). Free courses can require enrolment, which is confirmed by an emailed link (`/api/courses/<slug>/enroll/verify`).
- **Release rules**: lessons can be held back by drip timing or by sequential order. Preview lessons skip both.
- **Quizzes and progress**: one access check runs before a quiz is shown or submitted, a lesson is completed, or progress is read. A refused request stores nothing. Quiz routes answer 403 when there's no purchase or enrolment and 404 for hidden courses.
- **Certificates** on completion, with a public check at `/api/certificates/verify/<id>`.

### Turning courses off

Turning courses off closes the doors new learners would use: the public catalogue and new enrolments ([feature gate](../backend/hooks/feature_gate.go)). People who already bought or enrolled keep their access. Access links, the Stripe webhook, enrolment verification and course pages stay open, and the course page tells newcomers it isn't taking new learners.

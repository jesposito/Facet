# Contact methods and protection

Under **Contact Methods** you list the ways people can reach you: email, phone, LinkedIn, GitHub, Mastodon, Bluesky, WhatsApp, Telegram, Discord, a website and more, or a custom type. Each entry has a label, a value, an order, an optional "primary" badge, and two privacy controls:

- **Which views show it.** For example, recruiters see your phone number, conference visitors see only your social links.
- **A protection level**, which changes how the value is displayed to deter automated scrapers.

## Protection levels

| Level | What visitors see | What it's good for |
|-------|-------------------|--------------------|
| None | The value as a normal link | Things that are public anyway, like your GitHub profile |
| Obfuscation | A normal-looking link. In the page's HTML, the text is scrambled with hidden decoy characters and reversed text that CSS puts back in order ([component](../frontend/src/components/public/ObfuscatedLink.svelte)). Screen readers still read it correctly. | Stopping simple scrapers that read visible text |
| Click to reveal | A button. The value appears when the visitor clicks it. | Making casual harvesting a little more effort |
| CAPTCHA | A disabled button marked "CAPTCHA required". The value is removed from public page data, so nobody can reveal it. | Hiding an entry without deleting it. No CAPTCHA check (such as Cloudflare Turnstile) is built yet. |

## What protection doesn't do

None, Obfuscation and Click to reveal change **presentation only**. They're a deterrent against basic harvesters, not a secret: the full value is sent to every visitor's browser as part of the page data ([view data](../backend/hooks/view.go), [list component](../frontend/src/components/public/ContactMethodsList.svelte)). A scraper that reads page data rather than visible text, or anyone who opens the browser's developer tools, can still see it.

So protection reduces spam from basic harvesters, but it isn't secrecy. If a value must not be public:

- turn it off for every view a stranger can open, or
- show it only on an unlisted or password-protected view, or
- leave it out of Facet.

The CAPTCHA level is the exception: public routes strip the value before sending the page ([source](../backend/hooks/public_fields.go)). Until a CAPTCHA check exists, that means nobody can see it, so pick another level if you want people to reach you that way.

## Other protections nearby

- The robots file Facet serves asks known AI-training crawlers not to crawl your site ([Features](FEATURES.md#feeds-and-search-engines)). Well-behaved crawlers comply. Others ignore it.
- Public page data is rate limited per visitor ([Security](SECURITY.md#rate-limits)).

# SMTP

Checked **2026-11-10**.

Outgoing mail is owner-configured through PocketBase and enabled when `SMTP_HOST` is set. Facet replaces PocketBase's mailyak v3.6.2 transport with a local copy that applies a 30-second deadline to connection and exchange operations. No SMTP server or credentials were inspected for this inventory.

Sources: [`docker-compose.yml`](../docker-compose.yml), [`backend/third_party/mailyak-v3/FACET_PATCH.md`](../backend/third_party/mailyak-v3/FACET_PATCH.md), [`docs/CONFIGURATION.md`](../docs/CONFIGURATION.md), checked 2026-11-10. See `mailyak.md` for the patch's test command.

# mailyak SMTP transport

Checked **2026-11-10**.

PocketBase 0.40.6 uses `github.com/domodwyer/mailyak/v3` **v3.6.2**. Facet replaces it with `backend/third_party/mailyak-v3`, an Apache-2.0 local copy that adds a 30-second connection and exchange deadline to both SMTP senders.

Run `go test -race ./...` inside `backend/third_party/mailyak-v3` after changing the patch. Reassess the replacement whenever PocketBase changes its mail transport; remove it only when equivalent bounded sends exist upstream.

Sources: [`backend/go.mod`](../backend/go.mod), [`backend/third_party/mailyak-v3/FACET_PATCH.md`](../backend/third_party/mailyak-v3/FACET_PATCH.md), checked 2026-11-10.

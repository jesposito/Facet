# Local mailyak v3.6.2 with send deadlines

This directory is a copy of `github.com/domodwyer/mailyak/v3` at v3.6.2
(<https://github.com/domodwyer/mailyak>, Apache 2.0, see `LICENSE` and
`README.md`), the version PocketBase v0.40.6 uses to send mail. Facet
points at it with a `replace` line in `backend/go.mod`.

## Why

Upstream v3.6.2 dials with `net.Dial` and `tls.Dial` and never sets a
deadline. An SMTP server that accepts the connection and then goes quiet
(no greeting, a stuck TLS handshake, a stall mid-message) keeps
`Send` waiting forever, along with whatever Facet request or newsletter
send called it. PocketBase's `SMTPClient` has no timeout option to pass
through, and v3.6.2 is the newest upstream release.

## What changed

Only the two senders:

- `sender_starttls.go`: dials with a `net.Dialer` and sets one connection
  deadline before the SMTP exchange. The STARTTLS handshake runs on that
  connection, so the deadline covers it too.
- `sender_explicit_tls.go`: dials with `tls.DialWithDialer`, whose dialer
  deadline covers the TCP connect and the TLS handshake, then sets the same
  deadline on the connection for the exchange.

Each `Send` gets 30 seconds (`sendTimeout`) for the whole job. When it runs
out, the blocked read or write returns a timeout error and the connection is
closed; nothing is left running in the background.

TLS config, certificate checks, STARTTLS upgrade, AUTH and the MIME output
are unchanged. `sender_timeout_test.go` is new and checks that a silent
server (before the greeting, and during a TLS handshake) produces a timeout
error quickly and that the socket gets closed.

## Updating

If upstream adds its own timeout, drop this directory and the `replace`
line. Otherwise, to move to a newer release, copy the new version here,
reapply the changes in the two sender files and run:

    cd backend/third_party/mailyak-v3 && go test -race ./...

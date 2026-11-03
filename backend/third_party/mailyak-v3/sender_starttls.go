package mailyak

import (
	"bytes"
	"net"
	"time"
)

// sendTimeout bounds one Send: connecting, any TLS handshake, and the SMTP
// exchange together. Facet patch; upstream v3.6.2 has no timeout. Tests
// shorten it.
var sendTimeout = 30 * time.Second

// senderWithStartTLS connects to the remote SMTP server, upgrades the
// connection using STARTTLS if available, and sends the email.
type senderWithStartTLS struct {
	hostAndPort string
	hostname    string
	buf         *bytes.Buffer
}

func (s *senderWithStartTLS) Send(m sendableMail) error {
	// Facet patch: one deadline covers the dial, the STARTTLS handshake and
	// the whole SMTP conversation, so a stalled server can't hold Send open.
	deadline := time.Now().Add(sendTimeout)
	dialer := &net.Dialer{Deadline: deadline}
	conn, err := dialer.Dial("tcp", s.hostAndPort)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}

	return smtpExchange(m, conn, s.hostname, true)
}

func newSenderWithStartTLS(hostAndPort string) *senderWithStartTLS {
	hostName, _, err := net.SplitHostPort(hostAndPort)
	if err != nil {
		// Really this should be an error, but we can't return it from the New()
		// constructor without breaking compatability. Fortunately by the time
		// it gets to the dial() the user will get a pretty clear error as this
		// hostAndPort value is almost certainly invalid.
		//
		// This hostname must be split from the port so the correct value is
		// used when performing the SMTP AUTH as the Go SMTP implementation
		// refuses to send credentials over non-localhost plaintext connections,
		// and including the port messes this check up (and is probably the
		// wrong thing to be sending anyway).
		hostName = hostAndPort
	}

	return &senderWithStartTLS{
		hostAndPort: hostAndPort,
		hostname:    hostName,
		buf:         &bytes.Buffer{},
	}
}

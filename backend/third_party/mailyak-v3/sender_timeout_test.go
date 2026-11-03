package mailyak

// Facet patch tests: a server that accepts the connection and then says
// nothing must not hold Send open forever.

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

const stalledTestBudget = 200 * time.Millisecond

func withSendTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	old := sendTimeout
	sendTimeout = d
	t.Cleanup(func() { sendTimeout = old })
}

// stalledServer accepts one connection and never writes to it. It reports
// whether the client closed the connection after Send returned.
func stalledServer(t *testing.T) (addr string, clientClosed <-chan error) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	closed := make(chan error, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			closed <- err
			return
		}
		defer conn.Close()
		// Swallow whatever the client sends (a TLS ClientHello, say) until
		// it hangs up. The read deadline only stops a broken test hanging.
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, err = io.Copy(io.Discard, conn)
		closed <- err
	}()
	return l.Addr().String(), closed
}

func assertTimedOut(t *testing.T, err error, elapsed time.Duration, clientClosed <-chan error) {
	t.Helper()
	if err == nil {
		t.Fatal("Send returned nil from a server that never answered")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("Send error = %v, want a timeout", err)
	}
	if elapsed > stalledTestBudget+time.Second {
		t.Fatalf("Send took %v with a %v budget", elapsed, stalledTestBudget)
	}
	// io.Copy returns nil on EOF: the client closed its socket.
	select {
	case closeErr := <-clientClosed:
		if closeErr != nil {
			t.Fatalf("server didn't see the client close: %v", closeErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client left the connection open after Send returned")
	}
}

func TestSendTimesOutWaitingForGreeting(t *testing.T) {
	withSendTimeout(t, stalledTestBudget)
	addr, clientClosed := stalledServer(t)

	m := New(addr, nil)
	start := time.Now()
	err := m.sender.Send(m)
	assertTimedOut(t, err, time.Since(start), clientClosed)
}

func TestSendTimesOutInTLSHandshake(t *testing.T) {
	withSendTimeout(t, stalledTestBudget)
	addr, clientClosed := stalledServer(t)

	m, err := NewWithTLS(addr, nil, &tls.Config{ServerName: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	err = m.sender.Send(m)
	assertTimedOut(t, err, time.Since(start), clientClosed)
}

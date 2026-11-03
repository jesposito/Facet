package services

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// ClientIP returns the address of the client that sent r.
//
// In the Docker image, Caddy is the only thing that reaches the backend, and
// it connects over loopback. Caddy replaces X-Forwarded-For with the single
// client address it worked out (see docker/Caddyfile), so the backend believes
// that header only from a loopback peer, and only its rightmost entry: that's
// the one the nearest proxy wrote. A native install behind nginx on the same
// host works the same way, because nginx appends the real peer on the right.
//
// Headers a client can set for itself, such as CF-Connecting-IP, X-Real-IP
// and the left side of X-Forwarded-For, are never used. A request that
// reaches the backend directly from another host is keyed on that host.
// TRUST_PROXY=false ignores X-Forwarded-For even from loopback.
func ClientIP(r *http.Request) string {
	peer := remoteHost(r.RemoteAddr)
	if !isLoopback(peer) || os.Getenv("TRUST_PROXY") == "false" {
		return peer
	}
	if forwarded := rightmostForwardedFor(r.Header); forwarded != "" {
		return forwarded
	}
	return peer
}

// IsInternalRequest reports whether r is a server-side request from the
// SvelteKit frontend in the same container. Those carry X-Internal and skip
// Facet's per-IP limits, because every one of them comes from the frontend's
// own address. A request is only internal when it comes straight from
// loopback: anything that passed through Caddy carries X-Forwarded-For, so a
// visitor can't claim to be internal by sending the header.
func IsInternalRequest(r *http.Request) bool {
	if r.Header.Get("X-Internal") != "true" {
		return false
	}
	if !isLoopback(remoteHost(r.RemoteAddr)) {
		return false
	}
	return len(r.Header.Values("X-Forwarded-For")) == 0
}

func rightmostForwardedFor(h http.Header) string {
	values := h.Values("X-Forwarded-For")
	if len(values) == 0 {
		return ""
	}
	parts := strings.Split(values[len(values)-1], ",")
	candidate := strings.TrimSpace(parts[len(parts)-1])
	ip := net.ParseIP(candidate)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func isLoopback(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

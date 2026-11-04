package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	_ "github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

// These tests run the production Caddyfile through the compiled proxy. Only
// the listener, upstreams and log destinations change. sentinel stands in for
// a bearer token, and failure messages never repeat logged values.
const (
	sentinel   = "SENTINEL"
	caseHeader = "X-Test-Case"

	redirectCase   = "redirect"
	failureCase    = "upstream failure"
	visitProofCase = "visit proof response"
)

type logCase struct {
	name    string
	method  string
	uri     string
	want    string
	status  int
	headers map[string]string
}

type logEntry struct {
	Status      int                 `json:"status"`
	RespHeaders map[string][]string `json:"resp_headers"`
	Request     struct {
		Method  string              `json:"method"`
		URI     string              `json:"uri"`
		Headers map[string][]string `json:"headers"`
	} `json:"request"`
}

type forwardedRequest struct {
	uri    string
	header http.Header
}

func TestProductionCaddyfileKeepsBearersOutOfLogs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	accessLog := filepath.Join(dir, "access.log")
	defaultLog := filepath.Join(dir, "default.log")

	var mu sync.Mutex
	forwarded := map[string]forwardedRequest{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get(caseHeader)
		mu.Lock()
		forwarded[name] = forwardedRequest{uri: r.RequestURI, header: r.Header.Clone()}
		mu.Unlock()
		switch name {
		case failureCase:
			panic(http.ErrAbortHandler)
		case redirectCase:
			w.Header().Set("Location", "/s/"+sentinel)
			w.Header().Set("Set-Cookie", "me_share_token="+sentinel)
			w.WriteHeader(http.StatusFound)
		case visitProofCase:
			w.Header().Set("X-Share-Visit", sentinel)
			w.Header().Set("X-Share-Token", sentinel)
			w.Header().Set("X-Password-Token", sentinel)
		}
	}))
	defer upstream.Close()

	addr := freeAddr(t)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	config := testCaddyfile(t, port, upstream.Listener.Addr().String(), accessLog, defaultLog)
	configJSON, _, err := caddyconfig.GetAdapter("caddyfile").Adapt(config, map[string]any{"filename": "Caddyfile"})
	if err != nil {
		t.Fatalf("adapt production Caddyfile: %v", err)
	}
	if err := caddy.Load(configJSON, true); err != nil {
		t.Fatalf("load production Caddyfile: %v", err)
	}
	t.Cleanup(func() { _ = caddy.Stop() })

	cases := logCases()
	for i := range cases {
		c := &cases[i]
		if c.method == "" {
			c.method = http.MethodGet
		}
		if c.status == 0 {
			c.status = http.StatusOK
		}
		status, responseHeaders := send(t, addr, *c)
		if status != c.status {
			t.Errorf("%s: response status = %d, want %d", c.name, status, c.status)
		}
		if c.name == redirectCase && !strings.Contains(responseHeaders.Get("Location"), sentinel) {
			t.Errorf("%s: the client's Location header changed", c.name)
		}
		if c.name == visitProofCase {
			for _, header := range []string{"X-Share-Visit", "X-Share-Token", "X-Password-Token"} {
				if responseHeaders.Get(header) != sentinel {
					t.Errorf("%s: the client's %s header changed", c.name, header)
				}
			}
		}
	}

	// Stopping waits for in-flight handlers, so every entry is on disk.
	if err := caddy.Stop(); err != nil {
		t.Fatal(err)
	}
	access := readLog(t, accessLog)
	defaults := readLog(t, defaultLog)

	for _, c := range cases {
		entry, ok := access[c.name]
		if !ok {
			t.Errorf("%s: no access log entry", c.name)
			continue
		}
		checkEntry(t, c, entry)
		if entry.Status != c.status {
			t.Errorf("%s: logged status = %d, want %d", c.name, entry.Status, c.status)
		}
		if c.name == visitProofCase {
			for _, header := range []string{"X-Share-Visit", "X-Share-Token", "X-Password-Token"} {
				if _, ok := entry.RespHeaders[header]; ok {
					t.Errorf("%s: response %s was logged", c.name, header)
				}
			}
		}

		sent, ok := forwarded[c.name]
		if !ok {
			t.Errorf("%s: request never reached the upstream", c.name)
			continue
		}
		if strings.Contains(c.uri, sentinel) && !strings.Contains(sent.uri, sentinel) {
			t.Errorf("%s: the forwarded URI lost its token", c.name)
		}
		for name := range c.headers {
			if !strings.Contains(sent.header.Get(name), sentinel) {
				t.Errorf("%s: forwarded %s lost its token", c.name, name)
			}
		}
	}

	redirect := access[redirectCase]
	if _, ok := redirect.RespHeaders["Location"]; ok {
		t.Error("access log kept the Location response header")
	}
	if got := redirect.RespHeaders["Set-Cookie"]; len(got) != 1 || got[0] != "REDACTED" {
		t.Error("access log did not redact the Set-Cookie response header")
	}

	failure, ok := defaults[failureCase]
	if !ok {
		t.Fatal("default log has no entry for the failed request")
	}
	for _, c := range cases {
		if c.name == failureCase {
			checkEntry(t, c, failure)
		}
	}
}

func logCases() []logCase {
	tokenHeaders := map[string]string{
		"Authorization":    "Bearer " + sentinel,
		"Cookie":           "me_share_token=" + sentinel,
		"Referer":          "https://facet.test/s/" + sentinel,
		"X-Password-Token": sentinel,
		"X-Share-Token":    sentinel,
		"X-Share-Visit":    sentinel,
	}
	return []logCase{
		// Routes and queries without tokens stay readable.
		{name: "benign page", uri: "/pitch?tag=go&page=2", want: "/pitch?tag=go&page=2"},
		{name: "benign post", uri: "/posts/hello?from=pitch", want: "/posts/hello?from=pitch"},
		{name: "admin requests", uri: "/api/testimonials/requests?status=pending", want: "/api/testimonials/requests?status=pending"},
		{name: "unsubscribe form", method: http.MethodPost, uri: "/api/public/unsubscribe", want: "/api/public/unsubscribe"},

		// Bearers in the path.
		{name: "share", uri: "/s/" + sentinel + "?keep=benign", want: "/s/[REDACTED]?keep=benign"},
		{name: "share data", uri: "/s/" + sentinel + "/__data.json?keep=benign", want: "/s/[REDACTED]?keep=benign"},
		{name: "signed download", uri: "/api/downloads/" + sentinel + "?from=pitch", want: "/api/downloads/[REDACTED]?from=pitch"},
		{name: "purchase access", uri: "/api/content/access/" + sentinel, want: "/api/content/access/[REDACTED]"},
		{name: "newsletter confirm", uri: "/api/public/newsletter/confirm/" + sentinel, want: "/api/public/newsletter/confirm/[REDACTED]"},
		{name: "newsletter open", uri: "/api/public/newsletter/open/send1/" + sentinel, want: "/api/public/newsletter/open/[REDACTED]"},
		{name: "newsletter click", uri: "/api/public/newsletter/click/send1/" + sentinel + "/aHR0cHM?keep=benign", want: "/api/public/newsletter/click/[REDACTED]?keep=benign"},
		{name: "unsubscribe api", uri: "/api/public/unsubscribe/" + sentinel, want: "/api/public/unsubscribe/[REDACTED]"},
		{name: "unsubscribe one click", method: http.MethodPost, uri: "/api/public/unsubscribe/one-click/" + sentinel, want: "/api/public/unsubscribe/[REDACTED]"},
		{name: "unsubscribe page", uri: "/unsubscribe/" + sentinel, want: "/unsubscribe/[REDACTED]"},
		{name: "testimonial api", uri: "/api/testimonials/request/" + sentinel, want: "/api/testimonials/request/[REDACTED]"},
		{name: "testimonial verify api", uri: "/api/testimonials/verify/email/" + sentinel, want: "/api/testimonials/verify/email/[REDACTED]"},
		{name: "testimonial page", uri: "/testimonial/" + sentinel, want: "/testimonial/[REDACTED]"},
		{name: "testimonial verify page", uri: "/testimonial/verify/" + sentinel, want: "/testimonial/[REDACTED]"},

		// Bearers in the query.
		{name: "legacy share", uri: "/pitch?keep=benign&t=" + sentinel, want: "/pitch?keep=benign&t=[REDACTED]"},
		{name: "protected file", uri: "/api/files/c/r/a.pdf?thumb=100x100&token=" + sentinel, want: "/api/files/c/r/a.pdf?thumb=100x100&token=[REDACTED]"},
		{name: "course enroll verify", uri: "/api/courses/intro/enroll/verify?token=" + sentinel, want: "/api/courses/intro/enroll/verify?token=[REDACTED]"},
		{name: "oauth callback", uri: "/api/oauth2-redirect?code=" + sentinel + "&state=" + sentinel + "&keep=benign", want: "/api/oauth2-redirect?code=[REDACTED]&state=[REDACTED]&keep=benign"},
		{name: "checkout return", uri: "/checkout/return?type=post&session_id=" + sentinel, want: "/checkout/return?type=post&session_id=[REDACTED]"},

		// Spellings that still reach a token route or parameter.
		{name: "encoded share route", uri: "/%73/" + sentinel + "?keep=benign", want: "[REDACTED_PATH]?keep=benign"},
		{name: "encoded download route", uri: "/api/%64ownloads/" + sentinel + "?keep=benign", want: "[REDACTED_PATH]?keep=benign"},
		{name: "encoded slash", uri: "/s%2F" + sentinel + "?keep=benign", want: "[REDACTED_PATH]?keep=benign"},
		{name: "dot segment", uri: "/a/../s/" + sentinel + "?keep=benign", want: "[REDACTED_PATH]?keep=benign"},
		{name: "empty segment", uri: "//s/" + sentinel + "?keep=benign", want: "[REDACTED_PATH]?keep=benign"},
		{name: "encoded legacy key", uri: "/pitch?%74=" + sentinel + "&keep=benign", want: "/pitch?%74=[REDACTED]&keep=benign"},
		{name: "encoded token key", uri: "/pitch?to%6Ben=x%2D" + sentinel, want: "/pitch?to%6Ben=[REDACTED]"},
		{name: "repeated token", uri: "/pitch?t=" + sentinel + "&keep=benign&t=" + sentinel, want: "/pitch?t=[REDACTED]&keep=benign&t=[REDACTED]"},

		// Headers and the error log.
		{name: "token headers", uri: "/pitch?keep=benign", want: "/pitch?keep=benign", headers: tokenHeaders},
		{name: redirectCase, uri: "/pitch?keep=benign", want: "/pitch?keep=benign", status: http.StatusFound},
		{name: visitProofCase, uri: "/pitch?keep=benign", want: "/pitch?keep=benign"},
		{name: failureCase, uri: "/s/" + sentinel + "?keep=benign", want: "/s/[REDACTED]?keep=benign", status: http.StatusBadGateway, headers: tokenHeaders},
	}
}

func checkEntry(t *testing.T, c logCase, entry logEntry) {
	t.Helper()
	if entry.Request.URI != c.want {
		t.Errorf("%s: logged URI does not match %q", c.name, c.want)
	}
	if entry.Request.Method != c.method {
		t.Errorf("%s: logged method does not match %s", c.name, c.method)
	}
	for name := range c.headers {
		logged, ok := entry.Request.Headers[name]
		if name == "Authorization" || name == "Cookie" {
			if len(logged) != 1 || logged[0] != "REDACTED" {
				t.Errorf("%s: %s is not REDACTED", c.name, name)
			}
		} else if ok {
			t.Errorf("%s: %s was logged", c.name, name)
		}
	}
}

func testCaddyfile(t *testing.T, port, upstream, accessLog, defaultLog string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../Caddyfile")
	if err != nil {
		t.Fatal(err)
	}
	config := string(raw)
	for _, swap := range [][2]string{
		{"    servers {", "    admin off\n    persist_config off\n    servers {"},
		{"import token_safe_log_format json", "output file " + defaultLog + "\n        import token_safe_log_format json"},
		{"output stdout", "output file " + accessLog},
		{"localhost:8090", upstream},
		{"localhost:3000", upstream},
		{":8080 {", ":" + port + " {"},
	} {
		if !strings.Contains(config, swap[0]) {
			t.Fatalf("Caddyfile no longer contains %q", swap[0])
		}
		config = strings.ReplaceAll(config, swap[0], swap[1])
	}
	return []byte(config)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

// send writes the request line by hand so the URI reaches Caddy byte for byte.
func send(t *testing.T, addr string, c logCase) (int, http.Header) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}

	var request strings.Builder
	fmt.Fprintf(&request, "%s %s HTTP/1.1\r\nHost: facet.test\r\nConnection: close\r\n%s: %s\r\n", c.method, c.uri, caseHeader, c.name)
	for name, value := range c.headers {
		fmt.Fprintf(&request, "%s: %s\r\n", name, value)
	}
	request.WriteString("\r\n")
	if _, err := io.WriteString(conn, request.String()); err != nil {
		t.Fatal(err)
	}

	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("%s: read response: %v", c.name, err)
	}
	defer res.Body.Close()
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, res.Header.Clone()
}

// readLog indexes entries by test case. Both encoders put the fields in one
// JSON object: the whole line for json, the end of the line for console.
func readLog(t *testing.T, path string) map[string]logEntry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(sentinel)) {
		t.Errorf("%s contains a bearer sentinel", filepath.Base(path))
	}
	entries := map[string]logEntry{}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		start := bytes.IndexByte(line, '{')
		if start < 0 {
			continue
		}
		var entry logEntry
		if json.Unmarshal(line[start:], &entry) != nil {
			continue
		}
		if name := entry.Request.Headers[caseHeader]; len(name) == 1 {
			entries[name[0]] = entry
		}
	}
	return entries
}

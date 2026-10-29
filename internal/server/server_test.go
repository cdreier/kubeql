package server

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cdreier/kubeql/internal/kube"
	"github.com/coder/websocket"
)

func newTestHandler(t *testing.T, opts Options) http.Handler {
	t.Helper()
	h, err := NewRouter(kube.NewService(kube.NewFake()), opts)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func newTestServer(t *testing.T, opts Options) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(newTestHandler(t, opts))
	t.Cleanup(ts.Close)
	return ts
}

func TestHostCheck(t *testing.T) {
	ts := newTestServer(t, Options{AllowedHosts: []string{"mybox.lan"}})

	cases := []struct {
		host string
		want int
	}{
		{"localhost:8080", http.StatusOK},
		{"localhost", http.StatusOK},
		{"127.0.0.1:17687", http.StatusOK},
		{"[::1]:8080", http.StatusOK},
		{"LOCALHOST:8080", http.StatusOK},
		{"mybox.lan:8080", http.StatusOK}, // --allow-host
		{"evil.example:8080", http.StatusForbidden},
		{"127.0.0.1.evil.example", http.StatusForbidden},
		{"localhost.evil.example:8080", http.StatusForbidden},
		{"mybox.lan.:8080", http.StatusOK}, // trailing dot folds away
		{"127.0.0.1.nip.io", http.StatusForbidden},
	}
	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/healthz", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = tc.host
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Errorf("Host %q: status %d, want %d", tc.host, res.StatusCode, tc.want)
		}
	}
}

func TestWebSocketOriginCheck(t *testing.T) {
	ts := newTestServer(t, Options{})
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/query"

	cases := []struct {
		name   string
		origin string
		ok     bool
	}{
		{"no origin (non-browser client)", "", true},
		{"same origin (embedded UI, playground, app window)", ts.URL, true},
		{"vite dev server", "http://localhost:5173", true},
		{"loopback without port", "http://127.0.0.1", true},
		{"ipv6 loopback", "http://[::1]:5173", true},
		{"foreign website", "https://evil.example", false},
		{"lookalike host", "http://localhost.evil.example", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			h := http.Header{}
			if tc.origin != "" {
				h.Set("Origin", tc.origin)
			}
			conn, res, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
				HTTPHeader:   h,
				Subprotocols: []string{"graphql-transport-ws"},
			})
			if tc.ok {
				if err != nil {
					t.Fatalf("expected handshake to succeed: %v", err)
				}
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if err == nil {
				conn.Close(websocket.StatusNormalClosure, "")
				t.Fatal("expected handshake to be rejected")
			}
			if res == nil || res.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 403, got %v (err %v)", res, err)
			}
		})
	}
}

func TestOriginPatterns(t *testing.T) {
	got := strings.Join(originPatterns([]string{"localhost", "::1"}), " ")
	want := `localhost localhost:* \[::1\] \[::1\]:*`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeAllowedHosts(t *testing.T) {
	ok := []string{"MyBox.LAN", "mybox.lan.", "10.0.0.5", "::ffff:0:1"}
	got, err := normalizeAllowedHosts(ok[:2])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "mybox.lan mybox.lan" {
		t.Fatalf("normalized to %v", got)
	}
	if _, err := normalizeAllowedHosts([]string{"10.0.0.5"}); err != nil {
		t.Fatal(err)
	}
	bad := []string{"*.evil.com", "mybox.lan*", "mybox.lan:8080", "http://mybox.lan", "", "mybox.lan/query", "-bad.com", "::1", "2001:db8::1", "fe80::1"}
	for _, h := range bad {
		if _, err := normalizeAllowedHosts([]string{h}); err == nil {
			t.Errorf("expected %q to be rejected", h)
		}
	}
}

func TestHostCheckLoopbackPeer(t *testing.T) {
	h := newTestHandler(t, Options{AllowedHosts: []string{"mybox.lan", "10.1.2.3"}})
	cases := []struct {
		host   string
		remote string
		want   int
	}{
		{"localhost:8080", "127.0.0.1:1234", http.StatusOK},
		{"127.0.0.1:8080", "[::1]:1234", http.StatusOK},
		{"localhost:8080", "192.168.1.20:1234", http.StatusForbidden},
		{"127.0.0.1:8080", "10.0.0.8:9", http.StatusForbidden},
		{"mybox.lan:8080", "192.168.1.20:1234", http.StatusOK},
		{"10.1.2.3:8080", "192.168.1.20:1234", http.StatusOK},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/healthz", nil)
		req.Host = tc.host
		req.RemoteAddr = tc.remote
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("Host %q from %s: status %d, want %d", tc.host, tc.remote, rec.Code, tc.want)
		}
	}
}

func TestQueryOriginCheck(t *testing.T) {
	ts := newTestServer(t, Options{AllowedHosts: []string{"mybox.lan"}})
	body := []byte(`{"query":"{ __typename }"}`)
	cases := []struct {
		origin string
		want   int
	}{
		{"", http.StatusOK},
		{ts.URL, http.StatusOK},
		{"http://localhost:5173", http.StatusOK},
		{"http://127.0.0.1", http.StatusOK},
		{"http://[::1]:5173", http.StatusOK},
		{"http://mybox.lan:8080", http.StatusOK},
		{"http://MyBox.LAN.", http.StatusOK},
		{"https://evil.example", http.StatusForbidden},
		{"http://localhost.evil.example", http.StatusForbidden},
		{"null", http.StatusForbidden},
		{"http://", http.StatusForbidden},
	}
	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/query", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Sec-Fetch-Site", "cross-site")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Errorf("Origin %q: status %d, want %d", tc.origin, res.StatusCode, tc.want)
		}
	}
}

func TestFromLoopback(t *testing.T) {
	if !fromLoopback(&http.Request{RemoteAddr: "127.0.0.1:1"}) || !fromLoopback(&http.Request{RemoteAddr: "[::1]:1"}) {
		t.Fatal("loopback addresses rejected")
	}
	if fromLoopback(&http.Request{RemoteAddr: "192.168.0.1:1"}) || fromLoopback(&http.Request{RemoteAddr: ""}) {
		t.Fatal("non-loopback address accepted")
	}
	if ip := net.ParseIP("::ffff:127.0.0.1"); ip == nil || !ip.IsLoopback() {
		t.Fatal("mapped loopback should be loopback")
	}
}

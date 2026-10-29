package server

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/cdreier/kubeql/graph"
	"github.com/cdreier/kubeql/internal/kube"
	"github.com/cdreier/kubeql/web"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vektah/gqlparser/v2/ast"
)

// Options configures the HTTP router.
type Options struct {
	// AllowedHosts are extra hostnames (without port, scheme or wildcards)
	// accepted in the Host header and as an Origin host, in addition to
	// localhost, 127.0.0.1 and [::1]. Needed only when kubeql is reached via
	// a non-loopback name. Unlike the loopback names, these are accepted from
	// any client address — see hostCheck.
	AllowedHosts []string
}

// loopbackHosts are accepted only from a loopback client address.
var loopbackHosts = []string{"localhost", "127.0.0.1", "::1"}

// NewRouter builds the chi router with GraphQL, playground, and embedded SPA.
//
// kubeql exposes cluster data (including decoded Secret values) without
// authentication. Every request must therefore carry a loopback or explicitly
// allowed Host (loopback names only from a loopback client, so a public bind
// cannot be addressed as localhost), and /query — HTTP and WebSocket — only
// accepts a missing, same-origin, loopback or explicitly allowed Origin.
func NewRouter(svc *kube.Service, opts Options) (http.Handler, error) {
	extra, err := normalizeAllowedHosts(opts.AllowedHosts)
	if err != nil {
		return nil, err
	}
	hosts := append(append([]string{}, loopbackHosts...), extra...)

	resolver := graph.NewResolver(svc)
	es := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})

	srv := handler.New(es)
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		// Same-origin requests (embedded UI, playground, app window) are always
		// accepted by coder/websocket; OriginPatterns additionally allows the
		// Vite dev server (http://localhost:5173 proxying /query). hostCheck
		// already rejected a non-loopback client using a loopback Host.
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: websocket.AcceptOptions{OriginPatterns: originPatterns(hosts)},
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	r := chi.NewRouter()
	// Before RealIP: hostCheck must see the actual client, not X-Forwarded-For.
	r.Use(hostCheck(extra))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Timeout only for non-WebSocket requests (subscriptions stay open).
	r.Use(func(next http.Handler) http.Handler {
		timeout := middleware.Timeout(60 * time.Second)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
				next.ServeHTTP(w, req)
				return
			}
			timeout(next).ServeHTTP(w, req)
		})
	})

	// HTTP /query has no transport-level origin check (only the WebSocket
	// upgrade does). A foreign page can POST here with a spoofable Host.
	r.Handle("/query", originCheck(hosts)(srv))
	r.Handle("/playground", playground.Handler("kubeql", "/query"))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Embedded Vite SPA (production). In dev, use `cd web && npm run dev` instead.
	r.Handle("/*", spaHandler())

	return r, nil
}

// normalizeAllowedHosts rejects values that would widen path.Match origin
// patterns or never match a Host header (wildcards, ports, schemes, IPv6
// brackets). Names are stored lower-case and without a trailing dot.
func normalizeAllowedHosts(hosts []string) ([]string, error) {
	out := make([]string, 0, len(hosts))
	for _, raw := range hosts {
		h := strings.ToLower(strings.TrimSpace(raw))
		h = strings.TrimSuffix(h, ".")
		if h == "" {
			return nil, fmt.Errorf("empty --allow-host")
		}
		if strings.Contains(h, "://") || strings.ContainsAny(h, "/\\?#@") {
			return nil, fmt.Errorf("--allow-host %q must be a hostname, not a URL", raw)
		}
		if strings.Contains(h, ":") || strings.Contains(h, "[") || strings.Contains(h, "]") {
			return nil, fmt.Errorf("--allow-host %q must not include a port or brackets", raw)
		}
		if strings.ContainsAny(h, "*?[]") {
			return nil, fmt.Errorf("--allow-host %q must not contain wildcards", raw)
		}
		if ip := net.ParseIP(h); ip != nil {
			// An IPv6 Host header is bracketed ([::addr]:port). A bare address
			// never matches, and brackets are rejected above because they are
			// path.Match meta-characters. ::1 is already a built-in name.
			if ip.To4() == nil {
				return nil, fmt.Errorf("--allow-host %q is an IPv6 address; pass an IPv4 address or a hostname", raw)
			}
		} else if h != "localhost" && !validDNSName(h) {
			return nil, fmt.Errorf("--allow-host %q is not a hostname or IP", raw)
		}
		out = append(out, h)
	}
	return out, nil
}

func validDNSName(h string) bool {
	if len(h) > 253 {
		return false
	}
	for _, label := range strings.Split(h, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}
	return true
}

// spaHandler serves web/dist with index.html fallback for client-side routes.
func spaHandler() http.Handler {
	dist, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		panic("embed dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := dist.Open(path)
		if err != nil {
			// SPA fallback: unknown paths serve the app shell.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

// originPatterns turns hostnames into coder/websocket origin patterns
// (path.Match syntax) matching the host with and without a port.
func originPatterns(hosts []string) []string {
	out := make([]string, 0, 2*len(hosts))
	for _, h := range hosts {
		if strings.Contains(h, ":") { // IPv6 literal: brackets must be escaped for path.Match
			h = `\[` + h + `\]`
		}
		out = append(out, h, h+":*")
	}
	return out
}

// hostCheck rejects requests whose Host header is not an allowed hostname
// (any port).
//
// Loopback names are accepted only from a loopback client address. A remote
// client that sets Host: localhost (trivial once --addr is a public interface)
// is rejected; an explicitly allowed name is not, because that is the point
// of --allow-host. Must run before middleware that rewrites RemoteAddr.
func hostCheck(extra []string) func(http.Handler) http.Handler {
	loopback := make(map[string]struct{}, len(loopbackHosts)+len(extra))
	for _, h := range loopbackHosts {
		loopback[h] = struct{}{}
	}
	for _, h := range extra {
		if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
			loopback[h] = struct{}{}
		}
	}
	allowed := make(map[string]struct{}, len(loopback)+len(extra))
	for h := range loopback {
		allowed[h] = struct{}{}
	}
	for _, h := range extra {
		allowed[h] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := hostname(r.Host)
			if _, ok := allowed[host]; !ok {
				http.Error(w, "kubeql: host not allowed (use localhost or --allow-host)", http.StatusForbidden)
				return
			}
			if _, onlyLoopback := loopback[host]; onlyLoopback && !fromLoopback(r) {
				http.Error(w, "kubeql: loopback host is only accepted from a loopback client", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// originCheck rejects /query requests whose Origin is a foreign site.
// A missing Origin (curl, same-browser navigations) is allowed; same-origin,
// loopback and --allow-host origins are allowed. Browsers always send Origin
// on cross-site POST, which is how a page would read this unauthenticated API.
func originCheck(hosts []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		allowed[h] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			// "null" is what a sandboxed iframe / data: page sends. Only a non-browser
			// client omits Origin entirely (and has no Sec-Fetch-Site).
			if origin == "" || (origin == "null" && r.Header.Get("Sec-Fetch-Site") == "") {
				next.ServeHTTP(w, r)
				return
			}
			u, err := url.Parse(origin)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
				http.Error(w, "kubeql: origin not allowed", http.StatusForbidden)
				return
			}
			if strings.EqualFold(u.Host, r.Host) {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := allowed[hostname(u.Host)]; ok {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "kubeql: origin not allowed", http.StatusForbidden)
		})
	}
}

func fromLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// hostname strips an optional port, IPv6 brackets and a trailing dot from a
// Host header or origin host.
func hostname(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		h = hostport // no port
	}
	h = strings.ToLower(strings.Trim(h, "[]"))
	return strings.TrimSuffix(h, ".")
}

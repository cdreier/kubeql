package server

import (
	"io/fs"
	"net/http"
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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"
)

// Config holds HTTP server options.
type Config struct {
	Addr string
}

// NewRouter builds the chi router with GraphQL, playground, and embedded SPA.
func NewRouter(svc *kube.Service) http.Handler {
	resolver := graph.NewResolver(svc)
	es := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})

	srv := handler.New(es)
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
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

	r.Handle("/query", srv)
	r.Handle("/playground", playground.Handler("kubeql", "/query"))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Embedded Vite SPA (production). In dev, use `cd web && npm run dev` instead.
	r.Handle("/*", spaHandler())

	return r
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

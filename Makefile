.PHONY: generate generate-web build build-web install run run-app run-web test tidy snapshot

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

generate: generate-web
	go run github.com/99designs/gqlgen generate

# Merge GraphQL schema files and regenerate the gqty client.
generate-web:
	@cat graph/common.graphqls graph/context.graphqls graph/deployment.graphqls \
		graph/namespace.graphqls graph/pod.graphqls \
		graph/configmap.graphqls graph/secret.graphqls \
		graph/customresource.graphqls > web/schema.graphql
	cd web && npx gqty generate
	@# CLI rewrites index.ts — restore hand-maintained client (gotchas defaults).
	@git checkout -- web/src/gqty/index.ts 2>/dev/null || true
	@# If index.ts was untracked/new, ensure endpoint + suspense were not reset:
	@grep -q 'VITE_GQL_URL ?? "/query"' web/src/gqty/index.ts || \
	  (echo "error: web/src/gqty/index.ts lost hand-maintained fetcher; restore from git" && exit 1)

build-web:
	cd web && npm run build

build: build-web
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/kubeql ./cmd/kubeql

# Vite SPA + go install (GOBIN, else GOPATH/bin).
install: build-web
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" ./cmd/kubeql

run:
	go run ./cmd/kubeql serve

# Chrome app window on 127.0.0.1:17687; exits when the window closes.
run-app:
	go run ./cmd/kubeql serve --app

# Frontend dev server (proxies /query → :8080). Run backend separately.
run-web:
	cd web && npm run dev

test:
	go test -v ./...

tidy:
	go mod tidy

# Local GoReleaser snapshot (no GitHub publish). Needs goreleaser on PATH.
snapshot:
	goreleaser release --snapshot --clean

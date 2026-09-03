# kubeql

Read-only GraphQL API over your local kubeconfig.

## Install

Binaries with the embedded UI are published on [GitHub Releases](https://github.com/cdreier/kubeql/releases) for linux, macOS, and Windows (`amd64` + `arm64`).

```bash
# linux amd64 — other assets: darwin/windows × amd64/arm64
curl -fsSL -o kubeql.tar.gz \
  https://github.com/cdreier/kubeql/releases/latest/download/kubeql_linux_amd64.tar.gz
tar -xzf kubeql.tar.gz
sudo install -m 0755 kubeql /usr/local/bin/kubeql
kubeql serve
```

Or build from source (`make install` builds the Vite SPA before `go install`):

```bash
make install
kubeql serve
```

## Quick start

```bash
# GraphQL backend codegen (gqlgen) + frontend gqty client
make generate

# production binary (builds Vite SPA, embeds into Go)
make build
./bin/kubeql serve

# or install onto PATH (GOBIN / GOPATH/bin)
make install
kubeql serve

# or during development — two terminals:
make run          # API :8080  (UI if you already built web/dist)
cd web && npm run dev   # Vite :5173, proxies /query → :8080

# desktop-style window (Chrome --app on 127.0.0.1:17687; closes with the window)
kubeql serve --app
```

| URL | What |
|-----|------|
| http://localhost:8080/ | Embedded React UI |
| http://localhost:8080/playground | GraphQL Playground |
| http://localhost:8080/query | GraphQL HTTP/WS |
| http://localhost:5173/ | Vite dev UI (with proxy) |

Cluster clients are created **lazily per kubeconfig context**. The server starts even when no `current-context` is set.

## Example queries

Nest under contexts (one clientset per context name):

```graphql
query listing {
  contexts {
    name
    current
    namespaces {
      name
    }
  }
}
```

Root fields require an explicit `context` argument:

```graphql
query {
  namespaces(context: "my-ctx", filter: { hasDeployment: { nameContains: "api" } }) {
    name
    context
    deployments {
      name
      status
      restarts
      pods {
        name
        ready
        restarts
      }
    }
  }
}
```

Deployments filtered by owned pods:

```graphql
query {
  deployments(context: "my-ctx", namespace: "prod", filter: { hasPod: { ready: false } }) {
    name
    status
    pods(filter: { phase: "Running" }) {
      name
      ready
      containers { name state restartCount }
    }
  }
}
```

YAML + log subscription:

```graphql
query {
  pod(context: "my-ctx", namespace: "prod", name: "api-1") {
    phase
    yaml
  }
}

subscription {
  podLogs(context: "my-ctx", namespace: "prod", name: "api-1", tailLines: 50) {
    timestamp
    line
  }
}
```

## Layout

```
cmd/kubeql/          CLI entrypoint
graph/
  *.graphqls         schema split by domain (context, namespace, deployment, pod, …)
  *.resolvers.go     matching resolvers (gqlgen follow-schema)
  convert.go         domain → GraphQL model mapping
internal/kube/       ClusterReader, Registry (per-context clients), Service, fake
internal/server/     chi router, playground, SPA
web/                 Vite + React + TS + gqty (embedded via web/embed.go)
```

## Tests

```bash
make test
```

Unit tests use an in-memory fake cluster — no kubeconfig required.

## Releases

Push a semver tag to trigger GitHub Actions (Vite build, embed into Go, publish archives):

```bash
git tag v0.1.0
git push origin v0.1.0
```

Prerelease tags like `v0.1.0-rc.1` are marked as prereleases automatically.

Local dry-run (needs [GoReleaser](https://goreleaser.com/) on PATH):

```bash
make snapshot
```

## Notes

- **Read-only**: no mutations.
- **Multi-context**: nest under `contexts { ... }` or pass `context:` on root fields/subscriptions.
- **Metrics**: `cpuUsage` / `memoryUsage` come from metrics-server (null if missing). `cpuLimit` / `memoryLimit` are the summed container limits from the pod spec.
- **Status subscriptions** (`deploymentStatus`, `podStatus`) poll every 2s and include resource usage; can be swapped for informers later.

## License

[MIT](LICENSE)

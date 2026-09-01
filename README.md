# kubeql

Read-only GraphQL API over your local kubeconfig.

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

## Notes

- **Read-only**: no mutations.
- **Multi-context**: nest under `contexts { ... }` or pass `context:` on root fields/subscriptions.
- **Metrics**: `cpuUsage` / `memoryUsage` come from metrics-server (null if missing). `cpuLimit` / `memoryLimit` are the summed container limits from the pod spec.
- **Status subscriptions** (`deploymentStatus`, `podStatus`) poll every 2s and include resource usage; can be swapped for informers later.

import { useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import {
  ConfigSecretList,
  type ConfigOrSecret,
} from "../components/ConfigSecretList";
import { CopyKubectlButton } from "../components/CopyKubectlButton";
import { FavoriteStar } from "../components/FavoriteStar";
import { LogViewer } from "../components/LogViewer";
import { ResourceCell } from "../components/ResourceCell";
import { useFavoriteDeployments } from "../hooks/useFavoriteDeployments";
import { useQuery, useSubscription } from "../gqty";
import {
  kubectlExec,
  kubectlKillPod,
  kubectlRestartDeployment,
} from "../lib/kubectl";
import { podColor } from "../lib/podColor";
import type { Deployment } from "../gqty";
import { formatAbsolute, formatAge } from "../lib/age";
import { sumCpu, sumMem } from "../lib/resource";
import {
  containerStateClass,
  phaseClass,
  readyClass,
} from "../lib/status";
import "./ContextPage.css";
import "./DeploymentPage.css";

type PodView = {
  name: string;
  phase?: string | null;
  ready?: boolean | null;
  restarts?: number | null;
  lastRestartAt?: string | null;
  createdAt?: string | null;
  nodeName?: string | null;
  cpuUsage?: string | null;
  cpuLimit?: string | null;
  memoryUsage?: string | null;
  memoryLimit?: string | null;
  containers: Array<{
    name: string;
    image?: string | null;
    ready?: boolean | null;
    restartCount?: number | null;
    state?: string | null;
  }>;
};

function readConfigOrSecrets(
  list: Array<{
    name?: string | null;
    missing?: boolean | null;
    type?: string | null;
    refs?: Array<string | undefined>;
    keys?: Array<string | undefined>;
    data: Array<{ key?: string | null; value?: string | null }>;
  }>,
  withType = false
): ConfigOrSecret[] {
  return list
    .map((item) => ({
      name: item.name ?? "",
      missing: Boolean(item.missing),
      type: withType ? item.type ?? undefined : undefined,
      refs: (item.refs ?? []).filter((r): r is string => Boolean(r)),
      keys: (item.keys ?? []).filter((k): k is string => Boolean(k)),
      data: item.data
        .map((kv) => ({ key: kv.key ?? "", value: kv.value ?? "" }))
        .filter((kv) => Boolean(kv.key)),
    }))
    .filter((item) => Boolean(item.name));
}

function readPods(d: Deployment): PodView[] {
  return d
    .pods()
    .map((p) => ({
      name: p.name ?? "",
      phase: p.phase,
      ready: p.ready,
      restarts: p.restarts,
      lastRestartAt: p.lastRestartAt ?? undefined,
      createdAt: p.createdAt ?? undefined,
      nodeName: p.nodeName ?? undefined,
      cpuUsage: p.cpuUsage,
      cpuLimit: p.cpuLimit,
      memoryUsage: p.memoryUsage,
      memoryLimit: p.memoryLimit,
      containers: p.containers
        .map((c) => ({
          name: c.name ?? "",
          image: c.image,
          ready: c.ready,
          restartCount: c.restartCount,
          state: c.state,
        }))
        .filter((c) => Boolean(c.name)),
    }))
    .filter((p) => Boolean(p.name));
}

export function DeploymentPage() {
  const { contextName: rawCtx, nsName: rawNs, depName: rawDep } = useParams<{
    contextName: string;
    nsName: string;
    depName: string;
  }>();
  const contextName = rawCtx ? decodeURIComponent(rawCtx) : "";
  const nsName = rawNs ? decodeURIComponent(rawNs) : "";
  const depName = rawDep ? decodeURIComponent(rawDep) : "";

  const q = useQuery();
  const { isFavorite, toggleFavorite } = useFavoriteDeployments();
  // null = all pods in the log stream
  const [logPods, setLogPods] = useState<string[] | null>(null);
  const [subError, setSubError] = useState<string | null>(null);
  const sub = useSubscription({
    onError: (err) => setSubError(err.message),
  });
  const loadedRef = useRef(false);

  // Snapshot: labels + yaml stay on the query (not re-fetched every tick).
  // gqty accessors are always truthy proxies (see gqty-gotchas).
  const snap = q.deployment({
    context: contextName,
    namespace: nsName,
    name: depName,
  }) as Deployment;
  const snapName = snap.name;
  const snapStatus = snap.status;
  const snapReplicas = snap.replicas ?? 0;
  const snapReady = snap.readyReplicas ?? 0;
  const snapAvailable = snap.availableReplicas ?? 0;
  const snapRestarts = snap.restarts ?? 0;
  const yaml = snap.yaml;
  const labels = snap.labels
    .map((l) => ({ key: l.key, value: l.value }))
    .filter((l): l is { key: string; value: string } => Boolean(l.key))
    .sort((a, b) => a.key.localeCompare(b.key));
  const snapPods = readPods(snap);
  const configMaps = readConfigOrSecrets(snap.configMaps);
  const secrets = readConfigOrSecrets(snap.secrets, true);

  // Live stream: status + pod metrics (used/limit).
  const live = sub.deploymentStatus({
    context: contextName,
    namespace: nsName,
    name: depName,
  });
  const liveName = live.name;
  const liveStatus = live.status;
  const liveReplicas = live.replicas;
  const liveReady = live.readyReplicas;
  const liveAvailable = live.availableReplicas;
  const liveRestarts = live.restarts;
  const livePods = readPods(live);

  const name = liveName || snapName;
  const status = liveStatus ?? snapStatus;
  const replicas = liveReplicas ?? snapReplicas;
  const readyReplicas = liveReady ?? snapReady;
  const availableReplicas = liveAvailable ?? snapAvailable;
  const restarts = liveRestarts ?? snapRestarts;
  const pods = livePods.length > 0 ? livePods : snapPods;
  const liveOn = Boolean(liveName) && !subError;
  const allPodNames = pods.map((p) => p.name);
  const selectedLogPods =
    logPods === null
      ? allPodNames
      : logPods.filter((n) => allPodNames.includes(n));
  const activeLogPods =
    selectedLogPods.length > 0 ? selectedLogPods : allPodNames;
  const logFocus = logPods !== null && activeLogPods.length < allPodNames.length;

  const toggleLogPod = (name: string) => {
    setLogPods((prev) => {
      const current = prev ?? allPodNames;
      if (prev === null || current.length === allPodNames.length) {
        return [name];
      }
      if (current.includes(name)) {
        const next = current.filter((n) => n !== name);
        return next.length === 0 ? [name] : next;
      }
      return [...current, name];
    });
  };

  const cpuUsed = sumCpu(pods.map((p) => p.cpuUsage));
  const cpuLimit = sumCpu(pods.map((p) => p.cpuLimit));
  const memUsed = sumMem(pods.map((p) => p.memoryUsage));
  const memLimit = sumMem(pods.map((p) => p.memoryLimit));

  if (name) {
    loadedRef.current = true;
  }

  useEffect(() => {
    void q.$refetch(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount / key change
  }, [contextName, nsName, depName]);

  const showInitialLoading =
    !loadedRef.current && q.$state.isLoading && !name;

  if (!contextName || !nsName || !depName) {
    return <p className="error">Missing context, namespace, or deployment</p>;
  }

  if (showInitialLoading) {
    return <p className="muted">Loading deployment…</p>;
  }

  if (q.$state.error && !loadedRef.current) {
    return (
      <p className="error">Failed to load deployment: {q.$state.error.message}</p>
    );
  }

  if (!name && !q.$state.isLoading) {
    return <p className="muted">Deployment not found.</p>;
  }

  return (
    <div className="dep-page">
      <div className="page-heading">
        <FavoriteStar
          name={`${nsName}/${name ?? depName}`}
          favorited={isFavorite({
            context: contextName,
            namespace: nsName,
            name: name ?? depName,
          })}
          onToggle={() =>
            toggleFavorite({
              context: contextName,
              namespace: nsName,
              name: name ?? depName,
            })
          }
        />
        <h1>{name ?? depName}</h1>
        {liveOn ? (
          <span className="live-pill">Live</span>
        ) : subError ? (
          <span className="muted refresh-hint" title={subError}>
            Live updates unavailable
          </span>
        ) : (
          <span className="muted refresh-hint">Connecting…</span>
        )}
        <CopyKubectlButton
          label="Copy restart"
          command={kubectlRestartDeployment(contextName, nsName, name ?? depName)}
        />
      </div>

      <section className="dep-facts" aria-label="Deployment status">
        <div className="dep-fact">
          <span className="dep-fact-label">Status</span>
          <span
            className={`ready-pill ${readyClass(readyReplicas, replicas)}`}
          >
            {status ?? `${readyReplicas}/${replicas}`}
          </span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Ready</span>
          <span className="num">
            {readyReplicas}/{replicas}
          </span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Available</span>
          <span className="num">{availableReplicas}</span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Restarts</span>
          <span className="num">{restarts}</span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">CPU</span>
          <ResourceCell used={cpuUsed} limit={cpuLimit} kind="cpu" />
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Memory</span>
          <ResourceCell used={memUsed} limit={memLimit} kind="mem" />
        </div>
      </section>

      <section className="dep-section">
        <h2>Labels</h2>
        {labels.length === 0 ? (
          <p className="muted">No labels</p>
        ) : (
          <ul className="label-list">
            {labels.map((l) => (
              <li key={l.key} className="label-chip">
                <span className="label-key">{l.key}</span>
                <span className="label-eq">=</span>
                <span className="label-val">{l.value}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="dep-section">
        <h2>Pods ({pods.length})</h2>
        {pods.length === 0 ? (
          <p className="muted">No pods owned by this deployment.</p>
        ) : (
          <div className="dep-table-wrap ns-block">
            <table className="dep-table">
              <thead>
                <tr>
                  <th>Pod</th>
                  <th>Phase</th>
                  <th>Ready</th>
                  <th>Restarts</th>
                  <th>Age</th>
                  <th>Node</th>
                  <th>CPU used / limit</th>
                  <th>Memory used / limit</th>
                  <th>Containers</th>
                </tr>
              </thead>
              <tbody>
                {pods.map((p) => (
                  <tr key={p.name}>
                    <td className="dep-name">
                      <div className="pod-name-row">
                        <button
                          type="button"
                          className={
                            logFocus && activeLogPods.includes(p.name)
                              ? "pod-log-btn on"
                              : "pod-log-btn"
                          }
                          onClick={() => toggleLogPod(p.name)}
                          title="Toggle pod in log stream"
                          style={{ color: podColor(p.name) }}
                        >
                          {p.name}
                        </button>
                        <CopyKubectlButton
                          compact
                          tone="danger"
                          label="kill"
                          command={kubectlKillPod(
                            contextName,
                            nsName,
                            p.name
                          )}
                        />
                      </div>
                    </td>
                    <td>
                      <span className={`ready-pill ${phaseClass(p.phase)}`}>
                        {p.phase ?? "—"}
                      </span>
                    </td>
                    <td>
                      <span
                        className={`ready-pill ${p.ready ? "ready-ok" : "ready-bad"}`}
                      >
                        {p.ready ? "ready" : "not ready"}
                      </span>
                    </td>
                    <td
                      className="num"
                      title={
                        p.lastRestartAt
                          ? `Last restart: ${p.lastRestartAt}`
                          : undefined
                      }
                    >
                      {p.restarts ?? 0}
                    </td>
                    <td
                      className="num age-cell"
                      title={formatAbsolute(p.createdAt)}
                    >
                      {formatAge(p.createdAt)}
                    </td>
                    <td className="metric">{p.nodeName ?? "—"}</td>
                    <td>
                      <ResourceCell
                        used={p.cpuUsage}
                        limit={p.cpuLimit}
                        kind="cpu"
                      />
                    </td>
                    <td>
                      <ResourceCell
                        used={p.memoryUsage}
                        limit={p.memoryLimit}
                        kind="mem"
                      />
                    </td>
                    <td>
                      <ul className="container-list">
                        {p.containers.map((c) => (
                          <li key={c.name} title={c.image ?? undefined}>
                            <span
                              className={`ready-pill ${containerStateClass(c.state)}`}
                            >
                              {c.state ?? "—"}
                            </span>{" "}
                            <span className="container-name">{c.name}</span>
                            {c.restartCount ? (
                              <span className="muted">
                                {" "}
                                ({c.restartCount})
                              </span>
                            ) : null}{" "}
                            <CopyKubectlButton
                              compact
                              label="exec"
                              command={kubectlExec(
                                contextName,
                                nsName,
                                p.name,
                                c.name
                              )}
                            />
                          </li>
                        ))}
                      </ul>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <LogViewer
        context={contextName}
        namespace={nsName}
        pods={pods}
        selectedPods={activeLogPods}
        onTogglePod={toggleLogPod}
        onSelectAll={() => setLogPods(null)}
      />

      <ConfigSecretList
        title="ConfigMaps"
        kind="configmap"
        items={configMaps}
        context={contextName}
        namespace={nsName}
      />
      <ConfigSecretList
        title="Secrets"
        kind="secret"
        items={secrets}
        context={contextName}
        namespace={nsName}
      />

      {yaml ? (
        <details className="dep-section yaml-block">
          <summary>YAML</summary>
          <pre className="yaml-pre">
            <code>{yaml}</code>
          </pre>
        </details>
      ) : null}
    </div>
  );
}

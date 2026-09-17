import { useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import {
  ConfigSecretList,
  type ConfigOrSecret,
} from "../components/ConfigSecretList";
import { CopyKubectlButton } from "../components/CopyKubectlButton";
import { LogViewer } from "../components/LogViewer";
import { ResourceCell } from "../components/ResourceCell";
import { useQuery, useSubscription } from "../gqty";
import {
  kubectlDeleteJob,
  kubectlExec,
  kubectlKillPod,
  kubectlTriggerCronJob,
} from "../lib/kubectl";
import { podColor } from "../lib/podColor";
import type { CronJob } from "../gqty";
import { formatAbsolute, formatAge, formatDuration } from "../lib/age";
import { sumCpu, sumMem } from "../lib/resource";
import {
  containerStateClass,
  cronJobStatusClass,
  jobStatusClass,
  phaseClass,
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

type JobView = {
  name: string;
  status: string | null;
  completions: number;
  succeeded: number;
  failed: number;
  active: number;
  startTime: string | null;
  completionTime: string | null;
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

function readPods(cj: CronJob): PodView[] {
  return cj
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

function readJobs(cj: CronJob): JobView[] {
  return cj.jobs
    .map((j) => ({
      name: j.name ?? "",
      status: j.status ?? null,
      completions: j.completions ?? 0,
      succeeded: j.succeeded ?? 0,
      failed: j.failed ?? 0,
      active: j.active ?? 0,
      startTime: j.startTime ?? null,
      completionTime: j.completionTime ?? null,
    }))
    .filter((j) => Boolean(j.name));
}

export function CronJobPage() {
  const { contextName: rawCtx, nsName: rawNs, cronName: rawCron } = useParams<{
    contextName: string;
    nsName: string;
    cronName: string;
  }>();
  const contextName = rawCtx ? decodeURIComponent(rawCtx) : "";
  const nsName = rawNs ? decodeURIComponent(rawNs) : "";
  const cronName = rawCron ? decodeURIComponent(rawCron) : "";

  const q = useQuery();
  const [logPods, setLogPods] = useState<string[] | null>(null);
  const [subError, setSubError] = useState<string | null>(null);
  const sub = useSubscription({
    onError: (err) => setSubError(err.message),
  });
  const loadedRef = useRef(false);

  const snap = q.cronJob({
    context: contextName,
    namespace: nsName,
    name: cronName,
  }) as CronJob;
  const snapName = snap.name;
  const snapStatus = snap.status;
  const snapSchedule = snap.schedule;
  const snapTimeZone = snap.timeZone;
  const snapSuspend = snap.suspend;
  const snapConcurrency = snap.concurrencyPolicy;
  const snapActive = snap.active ?? 0;
  const snapLastSchedule = snap.lastScheduleTime;
  const snapLastSuccess = snap.lastSuccessfulTime;
  const yaml = snap.yaml;
  const labels = snap.labels
    .map((l) => ({ key: l.key, value: l.value }))
    .filter((l): l is { key: string; value: string } => Boolean(l.key))
    .sort((a, b) => a.key.localeCompare(b.key));
  const snapPods = readPods(snap);
  const snapJobs = readJobs(snap);
  const configMaps = readConfigOrSecrets(snap.configMaps);
  const secrets = readConfigOrSecrets(snap.secrets, true);

  const live = sub.cronJobStatus({
    context: contextName,
    namespace: nsName,
    name: cronName,
  });
  const liveName = live.name;
  const liveStatus = live.status;
  const liveSchedule = live.schedule;
  const liveTimeZone = live.timeZone;
  const liveSuspend = live.suspend;
  const liveConcurrency = live.concurrencyPolicy;
  const liveActive = live.active;
  const liveLastSchedule = live.lastScheduleTime;
  const liveLastSuccess = live.lastSuccessfulTime;
  const livePods = readPods(live);
  const liveJobs = readJobs(live);

  const name = liveName || snapName;
  const status = liveStatus ?? snapStatus;
  const schedule = liveSchedule ?? snapSchedule;
  const timeZone = liveTimeZone ?? snapTimeZone;
  const suspend = liveSuspend ?? snapSuspend;
  const concurrencyPolicy = liveConcurrency ?? snapConcurrency;
  const active = liveActive ?? snapActive;
  const lastScheduleTime = liveLastSchedule ?? snapLastSchedule;
  const lastSuccessfulTime = liveLastSuccess ?? snapLastSuccess;
  const liveOn = Boolean(liveName) && !subError;
  const pods = liveOn ? livePods : snapPods;
  const jobs = liveOn ? liveJobs : snapJobs;
  const allPodNames = pods.map((p) => p.name);
  const selectedLogPods =
    logPods === null
      ? allPodNames
      : logPods.filter((n) => allPodNames.includes(n));
  const activeLogPods =
    selectedLogPods.length > 0 ? selectedLogPods : allPodNames;
  const logFocus = logPods !== null && activeLogPods.length < allPodNames.length;

  const toggleLogPod = (podName: string) => {
    setLogPods((prev) => {
      const current = prev ?? allPodNames;
      if (prev === null || current.length === allPodNames.length) {
        return [podName];
      }
      if (current.includes(podName)) {
        const next = current.filter((n) => n !== podName);
        return next.length === 0 ? [podName] : next;
      }
      return [...current, podName];
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
  }, [contextName, nsName, cronName]);

  const showInitialLoading =
    !loadedRef.current && q.$state.isLoading && !name;

  if (!contextName || !nsName || !cronName) {
    return <p className="error">Missing context, namespace, or cron job</p>;
  }

  if (showInitialLoading) {
    return <p className="muted">Loading cron job…</p>;
  }

  if (q.$state.error && !loadedRef.current) {
    return (
      <p className="error">Failed to load cron job: {q.$state.error.message}</p>
    );
  }

  if (!name && !q.$state.isLoading) {
    return <p className="muted">CronJob not found.</p>;
  }

  return (
    <div className="dep-page">
      <div className="page-heading">
        <h1>{name ?? cronName}</h1>
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
          label="Copy trigger"
          command={kubectlTriggerCronJob(
            contextName,
            nsName,
            name ?? cronName
          )}
        />
      </div>

      <section className="dep-facts" aria-label="CronJob status">
        <div className="dep-fact">
          <span className="dep-fact-label">Status</span>
          <span className={`ready-pill ${cronJobStatusClass(status)}`}>
            {status ?? (suspend ? "Suspended" : "—")}
          </span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Schedule</span>
          <span className="sched">
            {schedule ?? "—"}
            {timeZone ? <span className="muted"> {timeZone}</span> : null}
          </span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Concurrency</span>
          <span>{concurrencyPolicy || "—"}</span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Active</span>
          <span className="num">{active}</span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Last schedule</span>
          <span className="num" title={formatAbsolute(lastScheduleTime)}>
            {formatAge(lastScheduleTime)}
          </span>
        </div>
        <div className="dep-fact">
          <span className="dep-fact-label">Last success</span>
          <span className="num" title={formatAbsolute(lastSuccessfulTime)}>
            {formatAge(lastSuccessfulTime)}
          </span>
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
        <h2>Jobs ({jobs.length})</h2>
        {jobs.length === 0 ? (
          <p className="muted">No jobs owned by this cron job.</p>
        ) : (
          <div className="dep-table-wrap ns-block">
            <table className="dep-table">
              <thead>
                <tr>
                  <th>Job</th>
                  <th>Status</th>
                  <th>Completions</th>
                  <th>Failed</th>
                  <th>Duration</th>
                  <th>Age</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((j) => (
                  <tr key={j.name}>
                    <td className="dep-name">
                      <div className="pod-name-row">
                        {j.name}
                        <CopyKubectlButton
                          compact
                          tone="danger"
                          label="delete"
                          command={kubectlDeleteJob(
                            contextName,
                            nsName,
                            j.name
                          )}
                        />
                      </div>
                    </td>
                    <td>
                      <span className={`ready-pill ${jobStatusClass(j.status)}`}>
                        {j.status ?? "—"}
                      </span>
                    </td>
                    <td className="num">
                      {j.succeeded}/{j.completions}
                    </td>
                    <td className="num">{j.failed}</td>
                    <td className="num">
                      {formatDuration(j.startTime, j.completionTime)}
                    </td>
                    <td
                      className="num age-cell"
                      title={formatAbsolute(j.startTime)}
                    >
                      {formatAge(j.startTime)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="dep-section">
        <h2>Pods ({pods.length})</h2>
        {pods.length === 0 ? (
          <p className="muted">No pods from owned jobs.</p>
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

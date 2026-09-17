import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "../gqty";
import { CustomResourceList } from "../components/CustomResourceList";
import { FavoriteStar } from "../components/FavoriteStar";
import { ResourceCell } from "../components/ResourceCell";
import { useFavoriteDeployments } from "../hooks/useFavoriteDeployments";
import { formatAbsolute, formatAge } from "../lib/age";
import { readDeploymentSummary } from "../lib/deploymentSummary";
import { cronJobStatusClass, readyClass } from "../lib/status";
import "./ContextPage.css";
import "./DeploymentPage.css";

export type DeploymentKey = string; // `${namespace}/${name}`

function depKey(namespace: string, name: string): DeploymentKey {
  return `${namespace}/${name}`;
}

export function ContextPage() {
  const { contextName: rawCtx, nsName: rawNs } = useParams<{
    contextName: string;
    nsName: string;
  }>();
  const contextName = rawCtx ? decodeURIComponent(rawCtx) : "";
  const nsName = rawNs ? decodeURIComponent(rawNs) : "";
  const [crOpen, setCrOpen] = useState(false);

  const q = useQuery({
    // Stable arg: remount page when context/ns changes (router key on parent).
  });
  const loadedRef = useRef(false);

  const [search, setSearch] = useState("");
  // Empty set = no filter (show all). Non-empty = only these deployments.
  const [activeKeys, setActiveKeys] = useState<Set<DeploymentKey>>(
    () => new Set()
  );

  // --- gqty selections: touch fields before any early return ---
  // gqty accessors are always truthy proxies (see gqty-gotchas).
  const { isFavorite, toggleFavorite } = useFavoriteDeployments();
  const ns = q.namespace({ context: contextName, name: nsName })!;
  const nsLoadedName = ns.name;
  const deployments = ns
    .deployments()
    .map((d) => {
      const summary = readDeploymentSummary(d, nsName);
      if (!summary) return null;
      return {
        ...summary,
        key: depKey(summary.namespace, summary.name),
      };
    })
    .filter((d): d is NonNullable<typeof d> => d != null);

  const cronJobs = ns
    .cronJobs()
    .map((cj) => {
      const name = cj.name;
      const namespace = cj.namespace ?? nsName;
      if (!name || !namespace) return null;
      return {
        name,
        namespace,
        schedule: cj.schedule ?? "",
        timeZone: cj.timeZone ?? null,
        lastScheduleTime: cj.lastScheduleTime ?? null,
        lastSuccessfulTime: cj.lastSuccessfulTime ?? null,
        active: cj.active ?? 0,
        status: cj.status ?? "",
      };
    })
    .filter((cj): cj is NonNullable<typeof cj> => cj != null);

  const sortedDeployments = useMemo(() => {
    return [...deployments].sort((a, b) => {
      const aFav = isFavorite({
        context: contextName,
        namespace: a.namespace,
        name: a.name,
      })
        ? 0
        : 1;
      const bFav = isFavorite({
        context: contextName,
        namespace: b.namespace,
        name: b.name,
      })
        ? 0
        : 1;
      return aFav - bFav;
    });
  }, [deployments, contextName, isFavorite]);

  const allDeployments = sortedDeployments;

  if (nsLoadedName || allDeployments.length > 0 || cronJobs.length > 0) {
    loadedRef.current = true;
  }

  useEffect(() => {
    void q.$refetch(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount / context+ns change
  }, [contextName, nsName]);

  // Reset filter when switching namespace.
  useEffect(() => {
    setSearch("");
    setActiveKeys(new Set());
  }, [contextName, nsName]);

  const searchLower = search.trim().toLowerCase();
  const pickerDeployments = useMemo(() => {
    if (!searchLower) return allDeployments;
    return allDeployments.filter(
      (d) =>
        d.name!.toLowerCase().includes(searchLower) ||
        d.namespace!.toLowerCase().includes(searchLower) ||
        d.key.toLowerCase().includes(searchLower)
    );
  }, [allDeployments, searchLower]);

  const filterActive = activeKeys.size > 0;

  const visibleDeployments = useMemo(() => {
    return allDeployments.filter((d) => !filterActive || activeKeys.has(d.key));
  }, [allDeployments, filterActive, activeKeys]);

  const showInitialLoading =
    !loadedRef.current &&
    q.$state.isLoading &&
    !nsLoadedName &&
    allDeployments.length === 0;

  const toggleKey = (key: DeploymentKey) => {
    setActiveKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const clearFilter = () => setActiveKeys(new Set());

  if (!contextName || !nsName) {
    return <p className="error">Missing context or namespace</p>;
  }

  if (showInitialLoading) {
    return <p className="muted">Loading deployments…</p>;
  }

  if (q.$state.error && !loadedRef.current) {
    return (
      <p className="error">
        Failed to load context: {q.$state.error.message}
      </p>
    );
  }

  return (
    <div className="context-page">
      <div className="page-heading">
        <h1>Deployments</h1>
        {q.$state.isLoading && loadedRef.current ? (
          <span className="muted refresh-hint">Refreshing…</span>
        ) : null}
      </div>

      <section className="filter-panel" aria-label="Deployment filter">
        <div className="filter-row">
          <input
            type="search"
            className="filter-search"
            placeholder="Search deployments…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-label="Search deployments"
          />
          {filterActive ? (
            <button type="button" className="btn-ghost" onClick={clearFilter}>
              Clear ({activeKeys.size})
            </button>
          ) : (
            <span className="muted filter-hint">
              Empty filter = show all. Click deployments to pin.
            </span>
          )}
        </div>
        <div className="filter-chips">
          {pickerDeployments.map((d) => {
            const on = activeKeys.has(d.key);
            return (
              <button
                key={d.key}
                type="button"
                className={on ? "chip on" : "chip"}
                onClick={() => toggleKey(d.key)}
                title={d.key}
              >
                <span className="chip-ns">{d.namespace}/</span>
                {d.name}
              </button>
            );
          })}
          {pickerDeployments.length === 0 && (
            <span className="muted">
              {allDeployments.length === 0
                ? "No deployments loaded"
                : "No match"}
            </span>
          )}
        </div>
      </section>

      <div className="ns-list">
        <section className="ns-block">
          <h2 className="ns-title">{nsName}</h2>
          <div className="dep-table-wrap">
            <table className="dep-table">
              <thead>
                <tr>
                  <th />
                  <th>Deployment</th>
                  <th>Ready</th>
                  <th>Pods</th>
                  <th>Restarts</th>
                  <th>CPU used / limit</th>
                  <th>Memory used / limit</th>
                </tr>
              </thead>
              <tbody>
                {visibleDeployments.map((d) => {
                  const ready = d.readyReplicas ?? 0;
                  const desired = d.replicas ?? 0;
                  return (
                    <tr key={d.key}>
                      <td className="fav-star-cell">
                        <FavoriteStar
                          name={`${d.namespace}/${d.name}`}
                          favorited={isFavorite({
                            context: contextName,
                            namespace: d.namespace,
                            name: d.name,
                          })}
                          onToggle={() =>
                            toggleFavorite({
                              context: contextName,
                              namespace: d.namespace,
                              name: d.name,
                            })
                          }
                        />
                      </td>
                      <td className="dep-name">
                        <Link
                          to={`/c/${encodeURIComponent(contextName)}/n/${encodeURIComponent(nsName)}/d/${encodeURIComponent(d.name)}`}
                          className="dep-link"
                        >
                          {d.name}
                        </Link>
                      </td>
                      <td>
                        <span
                          className={`ready-pill ${readyClass(ready, desired)}`}
                        >
                          {d.status ?? `${ready}/${desired}`}
                        </span>
                      </td>
                      <td className="num">
                        {d.podsTotal > 0
                          ? `${d.podsReady}/${d.podsTotal}`
                          : "—"}
                      </td>
                      <td className="num">{d.restarts ?? 0}</td>
                      <td>
                        <ResourceCell
                          used={d.cpuUsage}
                          limit={d.cpuLimit}
                          kind="cpu"
                        />
                      </td>
                      <td>
                        <ResourceCell
                          used={d.memoryUsage}
                          limit={d.memoryLimit}
                          kind="mem"
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
        {visibleDeployments.length === 0 && !q.$state.isLoading && (
          <p className="muted">
            {filterActive
              ? "No deployments match the active filter."
              : "No deployments in this namespace."}
          </p>
        )}

        <section className="ns-block">
          <h2 className="ns-title">CronJobs</h2>
          {cronJobs.length === 0 ? (
            <p className="muted ns-empty">
              No cron jobs in this namespace.
            </p>
          ) : (
            <div className="dep-table-wrap">
              <table className="dep-table">
                <thead>
                  <tr>
                    <th>CronJob</th>
                    <th>Schedule</th>
                    <th>Last schedule</th>
                    <th>Last success</th>
                    <th>Status</th>
                    <th>Active</th>
                  </tr>
                </thead>
                <tbody>
                  {cronJobs.map((cj) => (
                    <tr key={`${cj.namespace}/${cj.name}`}>
                      <td className="dep-name">
                        <Link
                          to={`/c/${encodeURIComponent(contextName)}/n/${encodeURIComponent(nsName)}/cj/${encodeURIComponent(cj.name)}`}
                          className="dep-link"
                        >
                          {cj.name}
                        </Link>
                      </td>
                      <td>
                        <span className="sched" title={cj.timeZone ?? undefined}>
                          {cj.schedule || "—"}
                          {cj.timeZone ? (
                            <span className="muted"> {cj.timeZone}</span>
                          ) : null}
                        </span>
                      </td>
                      <td
                        className="num age-cell"
                        title={formatAbsolute(cj.lastScheduleTime)}
                      >
                        {formatAge(cj.lastScheduleTime)}
                      </td>
                      <td
                        className="num age-cell"
                        title={formatAbsolute(cj.lastSuccessfulTime)}
                      >
                        {formatAge(cj.lastSuccessfulTime)}
                      </td>
                      <td>
                        <span
                          className={`ready-pill ${cronJobStatusClass(cj.status)}`}
                        >
                          {cj.status || "—"}
                        </span>
                      </td>
                      <td className="num">{cj.active}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      <details
        className="cr-fold"
        onToggle={(e) => setCrOpen(e.currentTarget.open)}
      >
        <summary>Custom resources</summary>
        {crOpen ? (
          <CustomResourceList context={contextName} namespace={nsName} />
        ) : (
          <p className="muted">Open to load CRDs in this namespace (Flux, cert-manager, …).</p>
        )}
      </details>
    </div>
  );
}

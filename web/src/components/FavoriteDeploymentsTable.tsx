import { Link } from "react-router-dom";
import { FavoriteStar } from "./FavoriteStar";
import { ResourceCell } from "./ResourceCell";
import { useSubscription } from "../gqty";
import type { Deployment, KubeContext } from "../gqty";
import {
  type FavDeployment,
  favDeploymentKey,
} from "../hooks/useFavoriteDeployments";
import { readDeploymentSummary } from "../lib/deploymentSummary";
import { readyClass } from "../lib/status";
import "./FavoriteDeploymentsTable.css";

type FavoriteDeploymentsTableProps = {
  favorites: FavDeployment[];
  contexts: KubeContext[];
  onToggle: (f: FavDeployment) => void;
};

type Row = FavDeployment & {
  cluster?: string;
  snapshot: ReturnType<typeof readDeploymentSummary>;
};

export function FavoriteDeploymentsTable({
  favorites,
  contexts,
  onToggle,
}: FavoriteDeploymentsTableProps) {
  if (favorites.length === 0) return null;

  const rows: Row[] = favorites.map((fav) => {
    const ctx = contexts.find((c) => c.name === fav.context);
    const cluster = ctx ? (ctx.cluster ?? undefined) : undefined;
    const d = ctx
      ? (ctx.deployment({
          namespace: fav.namespace,
          name: fav.name,
        }) as Deployment)
      : null;
    const snapshot = d ? readDeploymentSummary(d, fav.namespace) : null;
    return { ...fav, cluster, snapshot };
  });

  const groups: { context: string; cluster?: string; rows: Row[] }[] = [];
  for (const row of rows) {
    const last = groups[groups.length - 1];
    if (last && last.context === row.context) last.rows.push(row);
    else
      groups.push({ context: row.context, cluster: row.cluster, rows: [row] });
  }

  return (
    <section className="fav-deps" aria-label="Favorite deployments">
      <h1>Favorite deployments</h1>
      <p className="muted">Pinned deployments across contexts. Live via WebSocket.</p>
      <div className="dep-table-wrap ns-block">
        <table className="dep-table fav-dep-table">
          <thead>
            <tr>
              <th />
              <th>Context</th>
              <th>Deployment</th>
              <th>Ready</th>
              <th>Pods</th>
              <th>Restarts</th>
              <th>CPU used / limit</th>
              <th>Memory used / limit</th>
            </tr>
          </thead>
          <tbody>
            {groups.flatMap((g) =>
              g.rows.map((row, i) => (
                <LiveFavRow
                  key={favDeploymentKey(row)}
                  row={row}
                  showContext={i === 0}
                  contextRowSpan={g.rows.length}
                  onToggle={onToggle}
                />
              ))
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function LiveFavRow({
  row,
  showContext,
  contextRowSpan,
  onToggle,
}: {
  row: Row;
  showContext: boolean;
  contextRowSpan: number;
  onToggle: (f: FavDeployment) => void;
}) {
  const sub = useSubscription({
    onError: () => {
      // Keep the snapshot row if the stream drops.
    },
  });
  const live = sub.deploymentStatus({
    context: row.context,
    namespace: row.namespace,
    name: row.name,
  }) as Deployment;
  const liveSummary = readDeploymentSummary(live, row.namespace);
  const s = liveSummary ?? row.snapshot;
  const ready = s?.readyReplicas ?? 0;
  const desired = s?.replicas ?? 0;
  const href = `/c/${encodeURIComponent(row.context)}/n/${encodeURIComponent(row.namespace)}/d/${encodeURIComponent(row.name)}`;

  return (
    <tr>
      <td className="fav-star-cell">
        <FavoriteStar
          name={`${row.namespace}/${row.name}`}
          favorited
          onToggle={() => onToggle(row)}
        />
      </td>
      {showContext ? (
        <td className="fav-ctx-cell" rowSpan={contextRowSpan}>
          <span className="fav-ctx-name" title={row.context}>
            {row.context}
          </span>
          {/*{row.cluster ? (
            <span className="muted fav-ctx-cluster" title={row.cluster}>
              {row.cluster}
            </span>
          ) : null}*/}
        </td>
      ) : null}
      <td className="dep-name">
        <Link to={href} className="dep-link">
          <span className="muted">{row.namespace}/</span>
          {row.name}
        </Link>
      </td>
      <td>
        {s ? (
          <span className={`ready-pill ${readyClass(ready, desired)}`}>
            {s.status ?? `${ready}/${desired}`}
          </span>
        ) : (
          <span className="muted">—</span>
        )}
      </td>
      <td className="num">
        {s && s.podsTotal > 0 ? `${s.podsReady}/${s.podsTotal}` : "—"}
      </td>
      <td className="num">{s ? s.restarts : "—"}</td>
      <td>
        <ResourceCell used={s?.cpuUsage} limit={s?.cpuLimit} kind="cpu" />
      </td>
      <td>
        <ResourceCell used={s?.memoryUsage} limit={s?.memoryLimit} kind="mem" />
      </td>
    </tr>
  );
}

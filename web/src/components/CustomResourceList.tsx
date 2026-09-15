import { useEffect, useMemo, useRef, useState } from "react";
import { CopyKubectlButton } from "./CopyKubectlButton";
import { useQuery } from "../gqty";
import { formatAge } from "../lib/age";
import { kubectlGetCustom } from "../lib/kubectl";
import { crReadyClass } from "../lib/status";
import "./CustomResourceList.css";

export type CustomResourceRow = {
  name: string;
  kind: string;
  group: string;
  version: string;
  resource: string;
  namespace: string;
  ready: boolean | null;
  reason: string;
  message: string;
  createdAt: string | null;
};

type Group = {
  key: string;
  kind: string;
  group: string;
  items: CustomResourceRow[];
};

function readRows(
  raw: Array<{
    name?: string | null;
    kind?: string | null;
    group?: string | null;
    version?: string | null;
    resource?: string | null;
    namespace?: string | null;
    ready?: boolean | null;
    reason?: string | null;
    message?: string | null;
    createdAt?: string | null;
  }>,
  fallbackNs: string
): CustomResourceRow[] {
  return raw
    .map((cr) => ({
      name: cr.name ?? "",
      kind: cr.kind ?? "",
      group: cr.group ?? "",
      version: cr.version ?? "",
      resource: cr.resource ?? "",
      namespace: cr.namespace ?? fallbackNs,
      ready: cr.ready ?? null,
      reason: cr.reason ?? "",
      message: cr.message ?? "",
      createdAt: cr.createdAt ?? null,
    }))
    .filter((cr) => cr.name && cr.kind);
}

function groupRows(rows: CustomResourceRow[]): Group[] {
  const map = new Map<string, Group>();
  for (const row of rows) {
    const key = `${row.group}/${row.kind}`;
    let g = map.get(key);
    if (!g) {
      g = { key, kind: row.kind, group: row.group, items: [] };
      map.set(key, g);
    }
    g.items.push(row);
  }
  return [...map.values()].sort((a, b) => {
    const k = a.kind.localeCompare(b.kind);
    return k !== 0 ? k : a.group.localeCompare(b.group);
  });
}

type CustomResourceListProps = {
  context: string;
  namespace: string;
};

export function CustomResourceList({
  context,
  namespace,
}: CustomResourceListProps) {
  const q = useQuery();
  const loadedRef = useRef(false);
  const [search, setSearch] = useState("");

  const ns = q.namespace({ context, name: namespace })!;
  const rows = readRows(ns.customResources(), namespace);

  if (rows.length > 0 || !q.$state.isLoading) {
    loadedRef.current = true;
  }

  useEffect(() => {
    void q.$refetch(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount / ns change
  }, [context, namespace]);

  const searchLower = search.trim().toLowerCase();
  const visible = useMemo(() => {
    if (!searchLower) return rows;
    return rows.filter(
      (r) =>
        r.name.toLowerCase().includes(searchLower) ||
        r.kind.toLowerCase().includes(searchLower) ||
        r.group.toLowerCase().includes(searchLower) ||
        r.resource.toLowerCase().includes(searchLower)
    );
  }, [rows, searchLower]);

  const groups = useMemo(() => groupRows(visible), [visible]);

  if (!loadedRef.current && q.$state.isLoading) {
    return <p className="muted">Loading custom resources…</p>;
  }

  if (q.$state.error && !loadedRef.current) {
    return (
      <p className="error">
        Failed to load custom resources: {q.$state.error.message}
      </p>
    );
  }

  return (
    <div className="cr-list">
      <div className="cr-toolbar">
        <input
          type="search"
          className="filter-search"
          placeholder="Filter by kind, group, or name (flux, kustomization, …)"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Filter custom resources"
        />
        <span className="muted">
          {visible.length} object{visible.length === 1 ? "" : "s"}
        </span>
      </div>

      {groups.length === 0 ? (
        <p className="muted">
          {rows.length === 0
            ? "No custom resources in this namespace."
            : "No match."}
        </p>
      ) : (
        groups.map((g) => (
          <details key={g.key} className="cr-kind">
            <summary>
              {g.kind}{" "}
              <span className="muted cr-group">
                {g.group} ({g.items.length})
              </span>
            </summary>
            <div className="dep-table-wrap">
              <table className="dep-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Ready</th>
                    <th>Reason</th>
                    <th>Age</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {g.items.map((cr) => (
                    <CrRow
                      key={`${cr.resource}/${cr.name}`}
                      cr={cr}
                      context={context}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          </details>
        ))
      )}
    </div>
  );
}

function CrRow({
  cr,
  context,
}: {
  cr: CustomResourceRow;
  context: string;
}) {
  const [open, setOpen] = useState(false);
  const readyLabel =
    cr.ready == null ? "—" : cr.ready ? "Ready" : "Not ready";

  return (
    <>
      <tr>
        <td className="dep-name">{cr.name}</td>
        <td>
          <span className={`ready-pill ${crReadyClass(cr.ready)}`}>
            {readyLabel}
          </span>
        </td>
        <td className="cr-reason" title={cr.message || undefined}>
          {cr.reason || "—"}
        </td>
        <td className="num age-cell">{formatAge(cr.createdAt)}</td>
        <td>
          <div className="cr-actions">
            <CopyKubectlButton
              compact
              label="get"
              command={kubectlGetCustom(
                cr.resource,
                cr.group,
                context,
                cr.namespace,
                cr.name
              )}
            />
            <button
              type="button"
              className="btn-ghost kv-toggle"
              onClick={() => setOpen((v) => !v)}
            >
              {open ? "Hide YAML" : "YAML"}
            </button>
          </div>
        </td>
      </tr>
      {open ? (
        <tr className="cr-yaml-row">
          <td colSpan={5}>
            <CrYaml cr={cr} context={context} />
          </td>
        </tr>
      ) : null}
    </>
  );
}

function CrYaml({
  cr,
  context,
}: {
  cr: CustomResourceRow;
  context: string;
}) {
  const q = useQuery();
  const obj = q.customResource({
    context,
    group: cr.group,
    version: cr.version,
    resource: cr.resource,
    namespace: cr.namespace,
    name: cr.name,
  });
  const yaml = obj?.yaml;

  useEffect(() => {
    void q.$refetch(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [context, cr.group, cr.version, cr.resource, cr.namespace, cr.name]);

  if (!yaml) {
    return <p className="muted">Loading YAML…</p>;
  }
  return (
    <pre className="yaml-pre">
      <code>{yaml}</code>
    </pre>
  );
}

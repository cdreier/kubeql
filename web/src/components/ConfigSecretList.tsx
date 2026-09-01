import { useState } from "react";
import { CopyKubectlButton } from "./CopyKubectlButton";
import { kubectlGet } from "../lib/kubectl";
import "./ConfigSecretList.css";

export type KvEntry = { key: string; value: string };

export type ConfigOrSecret = {
  name: string;
  missing: boolean;
  refs: string[];
  keys: string[];
  data: KvEntry[];
  type?: string;
};

type ConfigSecretListProps = {
  title: string;
  kind: "configmap" | "secret";
  items: ConfigOrSecret[];
  context: string;
  namespace: string;
};

export function ConfigSecretList({
  title,
  kind,
  items,
  context,
  namespace,
}: ConfigSecretListProps) {
  return (
    <details className="dep-section kv-fold">
      <summary>
        {title} ({items.length})
      </summary>
      {items.length === 0 ? (
        <p className="muted">
          None referenced by this deployment’s pod template.
        </p>
      ) : (
        <ul className="kv-list">
          {items.map((item) => (
            <KvCard
              key={item.name}
              kind={kind}
              item={item}
              context={context}
              namespace={namespace}
            />
          ))}
        </ul>
      )}
    </details>
  );
}

function KvCard({
  kind,
  item,
  context,
  namespace,
}: {
  kind: "configmap" | "secret";
  item: ConfigOrSecret;
  context: string;
  namespace: string;
}) {
  const hideByDefault = kind === "secret";
  const [show, setShow] = useState(!hideByDefault);

  return (
    <li className={item.missing ? "kv-card missing" : "kv-card"}>
      <div className="kv-head">
        <span className="kv-name">{item.name}</span>
        {item.type ? <span className="muted kv-type">{item.type}</span> : null}
        {item.missing ? <span className="badge-missing">missing</span> : null}
        <CopyKubectlButton
          compact
          label="Copy get"
          command={kubectlGet(kind, context, namespace, item.name)}
        />
        {!item.missing && item.data.length > 0 ? (
          <button
            type="button"
            className="btn-ghost kv-toggle"
            onClick={() => setShow((v) => !v)}
          >
            {show ? "Hide values" : "Show values"}
          </button>
        ) : null}
      </div>
      {item.refs.length > 0 ? (
        <ul className="kv-refs">
          {item.refs.map((ref) => (
            <li key={ref} className="kv-ref">
              {ref}
            </li>
          ))}
        </ul>
      ) : null}
      {item.missing ? (
        <p className="muted">Referenced in the pod spec, but not found.</p>
      ) : show ? (
        item.data.length === 0 ? (
          <p className="muted">No data keys.</p>
        ) : (
          <table className="kv-table">
            <tbody>
              {item.data.map((row) => (
                <tr key={row.key}>
                  <th>{row.key}</th>
                  <td>
                    <pre>{row.value}</pre>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      ) : (
        <p className="muted">
          {item.keys.length} key{item.keys.length === 1 ? "" : "s"} hidden
        </p>
      )}
    </li>
  );
}

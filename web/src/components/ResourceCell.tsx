import { formatUsedLimit, usageRatio } from "../lib/resource";

type ResourceCellProps = {
  used?: string | null;
  limit?: string | null;
  kind: "cpu" | "mem";
};

export function ResourceCell({ used, limit, kind }: ResourceCellProps) {
  const ratio = usageRatio(used, limit, kind);
  return (
    <div className="res-cell">
      <span className="metric">{formatUsedLimit(used, limit, kind)}</span>
      {ratio != null ? (
        <div
          className="res-bar"
          title={`${Math.round(ratio * 100)}% of limit`}
        >
          <div
            className={`res-bar-fill${ratio >= 0.9 ? " hot" : ""}`}
            style={{ width: `${Math.round(ratio * 100)}%` }}
          />
        </div>
      ) : null}
    </div>
  );
}

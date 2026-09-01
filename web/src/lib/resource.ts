export function parseCpuMillis(value: string | null | undefined): number | null {
  if (!value) return null;
  const t = value.trim();
  const m = t.match(/^([0-9]*\.?[0-9]+)([num]?)$/);
  if (!m) return null;
  const n = Number(m[1]);
  if (!Number.isFinite(n)) return null;
  switch (m[2]) {
    case "n":
      return n / 1e6;
    case "u":
      return n / 1e3;
    case "m":
      return n;
    default:
      return n * 1000;
  }
}

export function parseMemBytes(value: string | null | undefined): number | null {
  if (!value) return null;
  const t = value.trim();
  const m = t.match(/^([0-9]*\.?[0-9]+)([KMGT]i?)?$/i);
  if (!m) return null;
  const n = Number(m[1]);
  if (!Number.isFinite(n)) return null;
  const unit = (m[2] ?? "").toLowerCase();
  const factor: Record<string, number> = {
    ki: 1024,
    mi: 1024 ** 2,
    gi: 1024 ** 3,
    ti: 1024 ** 4,
    k: 1e3,
    m: 1e6,
    g: 1e9,
    t: 1e12,
  };
  if (!unit) return n;
  const f = factor[unit];
  return f == null ? null : n * f;
}

export function usageRatio(
  used: string | null | undefined,
  limit: string | null | undefined,
  kind: "cpu" | "mem"
): number | null {
  const u = kind === "cpu" ? parseCpuMillis(used) : parseMemBytes(used);
  const l = kind === "cpu" ? parseCpuMillis(limit) : parseMemBytes(limit);
  if (u == null || l == null || l <= 0) return null;
  return Math.min(1, Math.max(0, u / l));
}

function prettyNum(n: number, maxDecimals: number): string {
  if (!Number.isFinite(n)) return "—";
  return n
    .toFixed(maxDecimals)
    .replace(/(\.\d*?)0+$/, "$1")
    .replace(/\.$/, "");
}

function formatCpuPair(usedMs: number | null, limitMs: number | null): string {
  const max = Math.max(usedMs ?? 0, limitMs ?? 0);
  const asCores = max >= 1000;
  const fmt = (n: number | null) => {
    if (n == null) return "—";
    if (asCores) return prettyNum(n / 1000, n >= 100 ? 2 : 3);
    return `${prettyNum(n, n >= 10 ? 0 : 1)}m`;
  };
  return `${fmt(usedMs)} / ${fmt(limitMs)}`;
}

function formatMemPair(usedB: number | null, limitB: number | null): string {
  const max = Math.max(usedB ?? 0, limitB ?? 0);
  const gi = 1024 ** 3;
  const mi = 1024 ** 2;
  const ki = 1024;
  const unit = max >= gi ? "Gi" : max >= mi ? "Mi" : max >= ki ? "Ki" : "B";
  const div = unit === "Gi" ? gi : unit === "Mi" ? mi : unit === "Ki" ? ki : 1;
  const decimals = unit === "Gi" ? 2 : unit === "Mi" && max < 10 * mi ? 1 : 0;
  const fmt = (n: number | null) => {
    if (n == null) return "—";
    return `${prettyNum(n / div, decimals)}${unit}`;
  };
  return `${fmt(usedB)} / ${fmt(limitB)}`;
}

export function formatUsedLimit(
  used: string | null | undefined,
  limit: string | null | undefined,
  kind: "cpu" | "mem"
): string {
  if (kind === "cpu") {
    const u = parseCpuMillis(used);
    const l = parseCpuMillis(limit);
    if (u == null && l == null) return "— / —";
    return formatCpuPair(u, l);
  }
  const u = parseMemBytes(used);
  const l = parseMemBytes(limit);
  if (u == null && l == null) return "— / —";
  return formatMemPair(u, l);
}

/** Sum CPU quantities; result is always millicores (`Nm`) for formatUsedLimit. */
export function sumCpu(values: Array<string | null | undefined>): string | null {
  let total = 0;
  let any = false;
  for (const v of values) {
    const n = parseCpuMillis(v);
    if (n == null) continue;
    total += n;
    any = true;
  }
  if (!any) return null;
  return `${total}m`;
}

/** Sum memory quantities; result is always bytes for formatUsedLimit. */
export function sumMem(values: Array<string | null | undefined>): string | null {
  let total = 0;
  let any = false;
  for (const v of values) {
    const n = parseMemBytes(v);
    if (n == null) continue;
    total += n;
    any = true;
  }
  if (!any) return null;
  return String(Math.round(total));
}

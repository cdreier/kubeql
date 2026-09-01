/** kubectl-style relative age: 12s, 5m, 3h, 2d, 1y */
export function formatAge(
  iso: string | null | undefined,
  now = Date.now()
): string {
  if (!iso) return "—";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "—";
  const sec = Math.max(0, Math.floor((now - t) / 1000));
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h`;
  const day = Math.floor(hr / 24);
  if (day < 365) return `${day}d`;
  return `${Math.floor(day / 365)}y`;
}

export function formatAbsolute(
  iso: string | null | undefined
): string | undefined {
  if (!iso) return undefined;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return undefined;
  return d.toLocaleString();
}

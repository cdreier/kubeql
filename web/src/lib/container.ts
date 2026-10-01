export type ContainerTerminationView = {
  reason?: string | null;
  exitCode?: number | null;
  message?: string | null;
  finishedAt?: string | null;
};

export type ContainerView = {
  name: string;
  image?: string | null;
  ready?: boolean | null;
  restartCount?: number | null;
  state?: string | null;
  reason?: string | null;
  message?: string | null;
  lastTerminated?: ContainerTerminationView | null;
};

/** Prefer waiting/terminated reason over the raw state name. */
export function containerDisplay(
  state?: string | null,
  reason?: string | null
): string {
  if (reason) return reason;
  return state ?? "—";
}

export function containerTitle(c: ContainerView): string | undefined {
  const parts: string[] = [];
  if (c.image) parts.push(c.image);
  if (c.message) parts.push(c.message);
  const last = c.lastTerminated;
  if (last?.reason) {
    const code = last.exitCode != null ? ` (${last.exitCode})` : "";
    parts.push(`last: ${last.reason}${code}`);
  }
  return parts.length ? parts.join(" — ") : undefined;
}

export function restartTitle(
  lastRestartReason?: string | null,
  lastRestartAtAbs?: string | null
): string | undefined {
  const bits: string[] = [];
  if (lastRestartReason) bits.push(lastRestartReason);
  if (lastRestartAtAbs) bits.push(`Last restart: ${lastRestartAtAbs}`);
  return bits.length ? bits.join(" · ") : undefined;
}

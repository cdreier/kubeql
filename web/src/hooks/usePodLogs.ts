import { useEffect, useRef, useState } from "react";
import { subscribe } from "../gqty";

export type LogEntry = {
  id: number;
  pod: string;
  timestamp: string | null;
  line: string;
  container: string | null;
};

export type LogTarget = {
  pod: string;
  container?: string | null;
};

const MAX_LINES = 3000;

type UsePodLogsOpts = {
  context: string;
  namespace: string;
  targets: LogTarget[];
  tailLines?: number;
  enabled?: boolean;
};

function targetKey(t: LogTarget): string {
  return `${t.pod}\0${t.container ?? ""}`;
}

function insertLine(prev: LogEntry[], entry: LogEntry): LogEntry[] {
  const next = [...prev, entry];
  if (entry.timestamp) {
    next.sort((a, b) => {
      if (!a.timestamp || !b.timestamp) return a.id - b.id;
      const cmp = a.timestamp.localeCompare(b.timestamp);
      return cmp !== 0 ? cmp : a.id - b.id;
    });
  }
  return next.length > MAX_LINES ? next.slice(-MAX_LINES) : next;
}

export function usePodLogs({
  context,
  namespace,
  targets,
  tailLines = 200,
  enabled = true,
}: UsePodLogsOpts) {
  const [lines, setLines] = useState<LogEntry[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [liveCount, setLiveCount] = useState(0);
  const seq = useRef(0);
  const key = targets.map(targetKey).join("|");

  useEffect(() => {
    if (!enabled || !context || !namespace || targets.length === 0) {
      setLiveCount(0);
      return;
    }

    setLines([]);
    setErrors({});
    setLiveCount(targets.length);
    let cancelled = false;
    const unsubs: Array<() => void> = [];

    for (const target of targets) {
      const iter = subscribe(
        ({ subscription }) => {
          const l = subscription.podLogs({
            context,
            namespace,
            name: target.pod,
            tailLines,
            ...(target.container ? { container: target.container } : {}),
          });
          return {
            timestamp: l.timestamp ?? null,
            line: l.line ?? "",
            container: l.container ?? null,
          };
        },
        {
          onError: (err) => {
            if (cancelled) return;
            const msg = err instanceof Error ? err.message : String(err);
            setErrors((prev) => ({ ...prev, [target.pod]: msg }));
            setLiveCount((n) => Math.max(0, n - 1));
          },
        }
      );
      unsubs.push(() => iter.unsubscribe());

      void (async () => {
        try {
          for await (const item of iter) {
            if (cancelled) break;
            if (!item || (item.line === "" && item.timestamp == null)) continue;
            const id = ++seq.current;
            setLines((prev) =>
              insertLine(prev, {
                id,
                pod: target.pod,
                timestamp: item.timestamp,
                line: item.line,
                container: item.container,
              })
            );
          }
        } catch (err) {
          if (!cancelled) {
            const msg = err instanceof Error ? err.message : String(err);
            setErrors((prev) => ({ ...prev, [target.pod]: msg }));
          }
        } finally {
          if (!cancelled) setLiveCount((n) => Math.max(0, n - 1));
        }
      })();
    }

    return () => {
      cancelled = true;
      for (const u of unsubs) u();
    };
    // key encodes targets
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [context, namespace, key, tailLines, enabled]);

  const errorList = Object.entries(errors).map(
    ([pod, message]) => `${pod}: ${message}`
  );

  return {
    lines,
    errors: errorList,
    connected: liveCount > 0,
    liveCount,
    clear: () => setLines([]),
  };
}

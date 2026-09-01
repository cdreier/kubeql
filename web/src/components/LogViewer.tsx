import { useEffect, useMemo, useRef, useState } from "react";
import { usePodLogs } from "../hooks/usePodLogs";
import { logLineTone } from "../lib/logTone";
import { podColor } from "../lib/podColor";
import { LogVolumeChart } from "./LogVolumeChart";
import "./LogViewer.css";

type ContainerOpt = { name: string };

export type LogPod = { name: string; containers: ContainerOpt[] };

type LogViewerProps = {
  context: string;
  namespace: string;
  pods: LogPod[];
  selectedPods: string[];
  onTogglePod: (name: string) => void;
  onSelectAll: () => void;
};

function formatLogTime(ts: string | null): string {
  if (!ts) return "";
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return ts;
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  const ss = String(d.getSeconds()).padStart(2, "0");
  const ms = String(d.getMilliseconds()).padStart(3, "0");
  return `${hh}:${mm}:${ss}.${ms}`;
}

function shortPodName(name: string): string {
  const dash = name.lastIndexOf("-");
  if (dash > 0 && name.length - dash <= 6) return name.slice(dash + 1);
  return name.length > 18 ? name.slice(-12) : name;
}

export function LogViewer({
  context,
  namespace,
  pods,
  selectedPods,
  onTogglePod,
  onSelectAll,
}: LogViewerProps) {
  const selectedSet = useMemo(() => new Set(selectedPods), [selectedPods]);
  const selected = pods.filter((p) => selectedSet.has(p.name));
  const allOn = pods.length > 0 && selected.length === pods.length;

  const containers = useMemo(() => {
    const names = new Set<string>();
    for (const p of selected) {
      for (const c of p.containers) names.add(c.name);
    }
    return [...names];
  }, [selected]);

  const [container, setContainer] = useState<string>("");
  const [filter, setFilter] = useState("");
  const scrollerRef = useRef<HTMLDivElement>(null);
  const stickRef = useRef(true);
  const firstContainer = containers[0] ?? "";

  useEffect(() => {
    if (container && containers.includes(container)) return;
    setContainer(firstContainer);
  }, [firstContainer, containers, container]);

  const targets = useMemo(
    () =>
      selected.map((p) => ({
        pod: p.name,
        container:
          (container && p.containers.some((c) => c.name === container)
            ? container
            : p.containers[0]?.name) || null,
      })),
    [selected, container]
  );

  const { lines, errors, connected, liveCount, clear } = usePodLogs({
    context,
    namespace,
    targets,
    tailLines: 200,
    enabled: selected.length > 0,
  });

  const [pickedRange, setPickedRange] = useState<{
    start: number;
    end: number;
  } | null>(null);

  const targetKey = targets.map((t) => `${t.pod}:${t.container ?? ""}`).join("|");
  useEffect(() => {
    setPickedRange(null);
  }, [context, namespace, targetKey]);

  const filterLower = filter.trim().toLowerCase();
  const visible = useMemo(() => {
    if (!filterLower) return lines;
    return lines.filter(
      (l) =>
        l.line.toLowerCase().includes(filterLower) ||
        l.pod.toLowerCase().includes(filterLower) ||
        (l.container ?? "").toLowerCase().includes(filterLower)
    );
  }, [lines, filterLower]);

  useEffect(() => {
    const el = scrollerRef.current;
    if (!el || !stickRef.current) return;
    el.scrollTop = el.scrollHeight;
  }, [visible.length]);

  const onScroll = () => {
    const el = scrollerRef.current;
    if (!el) return;
    stickRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 48;
  };

  const onPickRange = (start: number, end: number) => {
    setPickedRange({ start, end });
    const first = visible.find((l) => {
      if (!l.timestamp) return false;
      const t = Date.parse(l.timestamp);
      return !Number.isNaN(t) && t >= start && t < end;
    });
    if (!first) return;
    stickRef.current = false;
    requestAnimationFrame(() => {
      const el = scrollerRef.current?.querySelector(
        `[data-log-id="${first.id}"]`
      );
      el?.scrollIntoView({ block: "center" });
    });
  };

  const showPodName = selected.length !== 1;
  const showContainer = containers.length > 1;

  return (
    <section className="log-viewer" aria-label="Pod logs">
      <div className="log-toolbar">
        <h2>Logs</h2>
        {connected ? (
          <span className="live-pill">
            Live{selected.length > 1 ? ` · ${liveCount}` : ""}
          </span>
        ) : selected.length > 0 ? (
          <span className="muted refresh-hint">Disconnected</span>
        ) : null}
        {containers.length > 0 ? (
          <select
            className="log-select"
            value={container}
            onChange={(e) => setContainer(e.target.value)}
            aria-label="Container"
          >
            {containers.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        ) : null}
        <input
          type="search"
          className="log-filter"
          placeholder="Filter lines…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          aria-label="Filter log lines"
        />
        <span className="muted log-count">
          {filterLower
            ? `${visible.length} / ${lines.length}`
            : `${lines.length}`}
        </span>
        <button
          type="button"
          className="btn-ghost"
          onClick={() => {
            clear();
            setPickedRange(null);
          }}
        >
          Clear
        </button>
      </div>
      <div className="log-pods" role="group" aria-label="Pods in log stream">
        <button
          type="button"
          className={allOn ? "log-pod-chip on" : "log-pod-chip"}
          onClick={onSelectAll}
        >
          All
        </button>
        {pods.map((p) => {
          const on = selectedSet.has(p.name);
          const color = podColor(p.name);
          return (
            <button
              key={p.name}
              type="button"
              className={on ? "log-pod-chip on" : "log-pod-chip"}
              onClick={() => onTogglePod(p.name)}
              title={p.name}
              style={
                on
                  ? { borderColor: color, color }
                  : undefined
              }
            >
              <span className="log-pod-dot" style={{ background: color }} />
              {shortPodName(p.name)}
            </button>
          );
        })}
      </div>
      {errors.length > 0 ? (
        <p className="error log-error">Log stream: {errors.join(" · ")}</p>
      ) : null}
      <LogVolumeChart lines={visible} onPickRange={onPickRange} />
      <div
        ref={scrollerRef}
        className="log-scroller"
        onScroll={onScroll}
        role="log"
        aria-live="polite"
      >
        {visible.length === 0 ? (
          <p className="muted log-empty">
            {selected.length > 0
              ? filterLower
                ? "No lines match the filter."
                : "Waiting for log lines…"
              : "Select one or more pods to stream logs."}
          </p>
        ) : (
          <ol className="log-lines">
            {visible.map((l) => {
              const ts = l.timestamp ? Date.parse(l.timestamp) : NaN;
              const inRange =
                pickedRange != null &&
                !Number.isNaN(ts) &&
                ts >= pickedRange.start &&
                ts < pickedRange.end;
              return (
                <li
                  key={l.id}
                  data-log-id={l.id}
                  className={`log-line ${logLineTone(l.line)}${inRange ? " in-range" : ""}`}
                >
                  {l.timestamp ? (
                    <time className="log-ts" dateTime={l.timestamp}>
                      {formatLogTime(l.timestamp)}
                    </time>
                  ) : (
                    <span className="log-ts">—</span>
                  )}
                  {showPodName ? (
                    <span
                      className="log-pod"
                      style={{ color: podColor(l.pod) }}
                      title={l.pod}
                    >
                      {shortPodName(l.pod)}
                    </span>
                  ) : null}
                  {showContainer && l.container ? (
                    <span className="log-ctr">{l.container}</span>
                  ) : null}
                  <span className="log-text">{l.line}</span>
                </li>
              );
            })}
          </ol>
        )}
      </div>
    </section>
  );
}

import { useEffect, useMemo, useState } from "react";
import { logLineTone } from "../lib/logTone";
import "./LogVolumeChart.css";

export type VolumeLine = {
  timestamp: string | null;
  line: string;
};

type LogVolumeChartProps = {
  lines: VolumeLine[];
  onPickRange: (startMs: number, endMs: number) => void;
};

type Bucket = {
  start: number;
  end: number;
  info: number;
  warn: number;
  err: number;
};

const WINDOWS = [
  { label: "5m", ms: 5 * 60 * 1000 },
  { label: "15m", ms: 15 * 60 * 1000 },
  { label: "30m", ms: 30 * 60 * 1000 },
  { label: "1h", ms: 60 * 60 * 1000 },
] as const;

const BUCKET_COUNT = 60;

function pad(n: number): string {
  return String(n).padStart(2, "0");
}

function formatClock(ms: number, withSeconds = false): string {
  const d = new Date(ms);
  const clock = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  return withSeconds ? `${clock}:${pad(d.getSeconds())}` : clock;
}

function bucketTitle(b: Bucket, withSeconds: boolean): string {
  const range = `${formatClock(b.start, withSeconds)}–${formatClock(b.end, withSeconds)}`;
  const total = b.info + b.warn + b.err;
  if (total === 0) return `${range} · no lines`;
  const bits = [`${total} line${total === 1 ? "" : "s"}`];
  if (b.err) bits.push(`${b.err} error${b.err === 1 ? "" : "s"}`);
  if (b.warn) bits.push(`${b.warn} warning${b.warn === 1 ? "" : "s"}`);
  return `${range} · ${bits.join(" · ")}`;
}

function buildBuckets(
  lines: VolumeLine[],
  now: number,
  windowMs: number
): Bucket[] {
  const width = windowMs / BUCKET_COUNT;
  const end = Math.ceil(now / width) * width;
  const start = end - windowMs;
  const buckets: Bucket[] = Array.from({ length: BUCKET_COUNT }, (_, i) => {
    const s = start + i * width;
    return { start: s, end: s + width, info: 0, warn: 0, err: 0 };
  });
  for (const line of lines) {
    if (!line.timestamp) continue;
    const t = Date.parse(line.timestamp);
    if (Number.isNaN(t) || t < start || t > end) continue;
    const idx = Math.min(BUCKET_COUNT - 1, Math.floor((t - start) / width));
    const tone = logLineTone(line.line);
    if (tone === "err") buckets[idx].err += 1;
    else if (tone === "warn") buckets[idx].warn += 1;
    else buckets[idx].info += 1;
  }
  return buckets;
}

export function LogVolumeChart({ lines, onPickRange }: LogVolumeChartProps) {
  const [windowMs, setWindowMs] = useState(15 * 60 * 1000);
  const [now, setNow] = useState(() => Date.now());
  const [picked, setPicked] = useState<number | null>(null);

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const buckets = useMemo(
    () => buildBuckets(lines, now, windowMs),
    [lines, now, windowMs]
  );

  const max = Math.max(
    1,
    ...buckets.map((b) => b.info + b.warn + b.err)
  );
  const inWindow = buckets.reduce(
    (n, b) => n + b.info + b.warn + b.err,
    0
  );
  const windowLabel =
    WINDOWS.find((w) => w.ms === windowMs)?.label ?? "15m";
  const withSeconds = windowMs <= 15 * 60 * 1000;

  return (
    <div
      className="log-vol"
      role="img"
      aria-label={`Log volume over the last ${windowLabel}`}
    >
      <div className="log-vol-head">
        <span className="muted log-vol-meta">
          {inWindow} in last {windowLabel}
        </span>
        <div className="log-vol-windows" role="group" aria-label="Time window">
          {WINDOWS.map((w) => (
            <button
              key={w.label}
              type="button"
              className={w.ms === windowMs ? "log-vol-win on" : "log-vol-win"}
              onClick={() => {
                setWindowMs(w.ms);
                setPicked(null);
              }}
            >
              {w.label}
            </button>
          ))}
        </div>
      </div>
      <div className="log-vol-bars">
        {buckets.map((b) => {
          const total = b.info + b.warn + b.err;
          const title = bucketTitle(b, withSeconds);
          return (
            <button
              key={b.start}
              type="button"
              className={picked === b.start ? "log-vol-col on" : "log-vol-col"}
              title={title}
              aria-label={title}
              disabled={total === 0}
              onClick={() => {
                setPicked(b.start);
                onPickRange(b.start, b.end);
              }}
            >
              {b.info > 0 ? (
                <span
                  className="log-vol-seg info"
                  style={{ height: `${(b.info / max) * 100}%` }}
                />
              ) : null}
              {b.warn > 0 ? (
                <span
                  className="log-vol-seg warn"
                  style={{ height: `${(b.warn / max) * 100}%` }}
                />
              ) : null}
              {b.err > 0 ? (
                <span
                  className="log-vol-seg err"
                  style={{ height: `${(b.err / max) * 100}%` }}
                />
              ) : null}
            </button>
          );
        })}
      </div>
      <div className="log-vol-axis">
        <span>{formatClock(now - windowMs)}</span>
        <span>now</span>
      </div>
    </div>
  );
}

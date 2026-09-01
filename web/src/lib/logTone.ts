export type LogTone = "err" | "warn" | "";

export function logLineTone(line: string): LogTone {
  if (/\b(error|fatal|panic|emerg)\b/i.test(line)) return "err";
  if (/\b(warn|warning)\b/i.test(line)) return "warn";
  return "";
}

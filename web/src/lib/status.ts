export function formatMetric(value: string | null | undefined): string {
  if (value == null || value === "") return "—";
  return value;
}

export function readyClass(ready: number, desired: number): string {
  if (desired === 0) return "ready-unknown";
  if (ready >= desired) return "ready-ok";
  if (ready === 0) return "ready-bad";
  return "ready-warn";
}

export function phaseClass(phase: string | null | undefined): string {
  switch (phase) {
    case "Running":
    case "Succeeded":
      return "ready-ok";
    case "Pending":
      return "ready-warn";
    case "Failed":
      return "ready-bad";
    default:
      return "ready-unknown";
  }
}

export function crReadyClass(ready: boolean | null | undefined): string {
  if (ready == null) return "ready-unknown";
  return ready ? "ready-ok" : "ready-bad";
}

export function cronJobStatusClass(status: string | null | undefined): string {
  if (!status) return "ready-unknown";
  if (status === "Suspended") return "ready-unknown";
  if (status === "Idle") return "ready-ok";
  if (status.startsWith("Active")) return "ready-warn";
  return "ready-unknown";
}

export function jobStatusClass(status: string | null | undefined): string {
  switch (status) {
    case "Complete":
    case "Succeeded":
      return "ready-ok";
    case "Failed":
      return "ready-bad";
    case "Running":
      return "ready-warn";
    default:
      return "ready-unknown";
  }
}

export function containerStateClass(state: string | null | undefined): string {
  switch (state) {
    case "Running":
      return "ready-ok";
    case "Waiting":
      return "ready-warn";
    case "Terminated":
      return "ready-bad";
    default:
      return "ready-unknown";
  }
}

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

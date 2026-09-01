/** Quote a value for POSIX shells. Safe for ARNs, colons, slashes. */
export function shellQuote(value: string): string {
  if (value === "") return "''";
  if (/^[A-Za-z0-9_./:@+=,-]+$/.test(value)) return value;
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

function kubectl(args: string[]): string {
  return ["kubectl", ...args].map(shellQuote).join(" ");
}

export function kubectlRestartDeployment(
  context: string,
  namespace: string,
  name: string
): string {
  return kubectl([
    "rollout",
    "restart",
    `deployment/${name}`,
    "-n",
    namespace,
    "--context",
    context,
  ]);
}

export function kubectlGet(
  kind: "configmap" | "secret",
  context: string,
  namespace: string,
  name: string
): string {
  return kubectl([
    "get",
    kind,
    name,
    "-n",
    namespace,
    "--context",
    context,
    "-o",
    "yaml",
  ]);
}

export function kubectlExec(
  context: string,
  namespace: string,
  pod: string,
  container?: string | null
): string {
  const args = [
    "exec",
    "-it",
    pod,
    "-n",
    namespace,
    "--context",
    context,
  ];
  if (container) args.push("-c", container);
  args.push("--", "/bin/sh");
  return kubectl(args);
}

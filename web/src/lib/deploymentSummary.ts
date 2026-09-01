import type { Deployment } from "../gqty";
import { sumCpu, sumMem } from "./resource";

export type DeploymentSummary = {
  name: string;
  namespace: string;
  status: string | null;
  readyReplicas: number;
  replicas: number;
  restarts: number;
  podsReady: number;
  podsTotal: number;
  cpuUsage: string | null;
  cpuLimit: string | null;
  memoryUsage: string | null;
  memoryLimit: string | null;
};

export function readDeploymentSummary(
  d: Deployment,
  fallbackNs: string
): DeploymentSummary | null {
  const name = d.name;
  const namespace = d.namespace ?? fallbackNs;
  if (!name || !namespace) return null;
  const pods = d
    .pods()
    .map((p) => ({
      name: p.name,
      ready: p.ready,
      cpuUsage: p.cpuUsage,
      cpuLimit: p.cpuLimit,
      memoryUsage: p.memoryUsage,
      memoryLimit: p.memoryLimit,
    }))
    .filter((p) => Boolean(p.name));
  return {
    name,
    namespace,
    status: d.status ?? null,
    readyReplicas: d.readyReplicas ?? 0,
    replicas: d.replicas ?? 0,
    restarts: d.restarts ?? 0,
    podsReady: pods.filter((p) => p.ready).length,
    podsTotal: pods.length,
    cpuUsage: sumCpu(pods.map((p) => p.cpuUsage)),
    cpuLimit: sumCpu(pods.map((p) => p.cpuLimit)),
    memoryUsage: sumMem(pods.map((p) => p.memoryUsage)),
    memoryLimit: sumMem(pods.map((p) => p.memoryLimit)),
  };
}

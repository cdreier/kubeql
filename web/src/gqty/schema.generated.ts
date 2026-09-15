/**
 * GQty AUTO-GENERATED CODE: PLEASE DO NOT MODIFY MANUALLY
 */

import { type ScalarsEnumsHash } from "gqty";

export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = {
  [K in keyof T]: T[K];
};
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & {
  [SubKey in K]?: Maybe<T[SubKey]>;
};
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & {
  [SubKey in K]: Maybe<T[SubKey]>;
};
export type MakeEmpty<
  T extends { [key: string]: unknown },
  K extends keyof T
> = { [_ in K]?: never };
export type Incremental<T> =
  | T
  | {
      [P in keyof T]?: P extends " $fragmentName" | "__typename" ? T[P] : never;
    };
/** All built-in and custom scalars, mapped to their actual values */
export interface Scalars {
  ID: { input: string; output: string };
  String: { input: string; output: string };
  Boolean: { input: boolean; output: boolean };
  Int: { input: number; output: number };
  Float: { input: number; output: number };
  /** Kubernetes GraphQL viewer — read-only API. */
  Time: { input: string; output: string };
}

export interface CustomResourceFilter {
  group?: InputMaybe<Scalars["String"]["input"]>;
  kind?: InputMaybe<Scalars["String"]["input"]>;
  /** Case-insensitive substring on kind, resource, or API group (e.g. flux). */
  kindContains?: InputMaybe<Scalars["String"]["input"]>;
  /** Case-insensitive substring match on resource name. */
  nameContains?: InputMaybe<Scalars["String"]["input"]>;
  resource?: InputMaybe<Scalars["String"]["input"]>;
  version?: InputMaybe<Scalars["String"]["input"]>;
}

/** Filter deployments by name/labels and optionally by owned pods. */
export interface DeploymentFilter {
  /** Keep deployments that have at least one pod matching this filter. */
  hasPod?: InputMaybe<PodFilter>;
  /** Kubernetes label selector (e.g. "app=api,tier=frontend"). */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** Case-insensitive substring match on deployment name. */
  nameContains?: InputMaybe<Scalars["String"]["input"]>;
}

/**
 * Filter namespaces by name and/or by whether they contain matching resources.
 * Example: namespaces that have a deployment whose name contains "api".
 */
export interface NamespaceFilter {
  /** Keep namespaces that have at least one deployment matching this filter. */
  hasDeployment?: InputMaybe<DeploymentFilter>;
  /** Keep namespaces that have at least one pod matching this filter. */
  hasPod?: InputMaybe<PodFilter>;
  /** Case-insensitive substring match on namespace name. */
  nameContains?: InputMaybe<Scalars["String"]["input"]>;
}

/** Filter pods by name, labels, phase, and readiness. */
export interface PodFilter {
  /** Kubernetes label selector (e.g. "app=api"). */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** Case-insensitive substring match on pod name. */
  nameContains?: InputMaybe<Scalars["String"]["input"]>;
  /** Exact pod phase (Pending, Running, Succeeded, Failed, Unknown). */
  phase?: InputMaybe<Scalars["String"]["input"]>;
  /** When set, only pods with this overall ready state. */
  ready?: InputMaybe<Scalars["Boolean"]["input"]>;
}

export const scalarsEnumsHash: ScalarsEnumsHash = {
  Boolean: true,
  Int: true,
  String: true,
  Time: true,
};
export const generatedSchema = {
  ApiResource: {
    __typename: { __type: "String!" },
    group: { __type: "String!" },
    kind: { __type: "String!" },
    namespaced: { __type: "Boolean!" },
    resource: { __type: "String!" },
    version: { __type: "String!" },
  },
  ConfigMap: {
    __typename: { __type: "String!" },
    context: { __type: "String!" },
    data: { __type: "[KeyValue!]!" },
    keys: { __type: "[String!]!" },
    missing: { __type: "Boolean!" },
    name: { __type: "String!" },
    namespace: { __type: "String!" },
    refs: { __type: "[String!]!" },
    yaml: { __type: "String!" },
  },
  Container: {
    __typename: { __type: "String!" },
    image: { __type: "String!" },
    name: { __type: "String!" },
    ready: { __type: "Boolean!" },
    restartCount: { __type: "Int!" },
    state: { __type: "String!" },
  },
  CustomResource: {
    __typename: { __type: "String!" },
    apiVersion: { __type: "String!" },
    context: { __type: "String!" },
    createdAt: { __type: "Time" },
    group: { __type: "String!" },
    kind: { __type: "String!" },
    labels: { __type: "[Label!]!" },
    message: { __type: "String" },
    name: { __type: "String!" },
    namespace: { __type: "String" },
    ready: { __type: "Boolean" },
    reason: { __type: "String" },
    resource: { __type: "String!" },
    version: { __type: "String!" },
    yaml: { __type: "String!" },
  },
  CustomResourceFilter: {
    group: { __type: "String" },
    kind: { __type: "String" },
    kindContains: { __type: "String" },
    nameContains: { __type: "String" },
    resource: { __type: "String" },
    version: { __type: "String" },
  },
  Deployment: {
    __typename: { __type: "String!" },
    availableReplicas: { __type: "Int!" },
    configMaps: { __type: "[ConfigMap!]!" },
    context: { __type: "String!" },
    labels: { __type: "[Label!]!" },
    name: { __type: "String!" },
    namespace: { __type: "String!" },
    pods: { __type: "[Pod!]!", __args: { filter: "PodFilter" } },
    readyReplicas: { __type: "Int!" },
    replicas: { __type: "Int!" },
    restarts: { __type: "Int!" },
    secrets: { __type: "[Secret!]!" },
    status: { __type: "String!" },
    yaml: { __type: "String!" },
  },
  DeploymentFilter: {
    hasPod: { __type: "PodFilter" },
    labelSelector: { __type: "String" },
    nameContains: { __type: "String" },
  },
  KeyValue: {
    __typename: { __type: "String!" },
    key: { __type: "String!" },
    value: { __type: "String!" },
  },
  KubeContext: {
    __typename: { __type: "String!" },
    apiResources: { __type: "[ApiResource!]!" },
    cluster: { __type: "String!" },
    current: { __type: "Boolean!" },
    customResources: {
      __type: "[CustomResource!]!",
      __args: { filter: "CustomResourceFilter", namespace: "String" },
    },
    defaultNamespace: { __type: "String" },
    deployment: {
      __type: "Deployment",
      __args: { name: "String!", namespace: "String!" },
    },
    deployments: {
      __type: "[Deployment!]!",
      __args: { filter: "DeploymentFilter", namespace: "String" },
    },
    name: { __type: "String!" },
    namespace: { __type: "Namespace", __args: { name: "String!" } },
    namespaces: {
      __type: "[Namespace!]!",
      __args: { filter: "NamespaceFilter" },
    },
    pod: { __type: "Pod", __args: { name: "String!", namespace: "String!" } },
    pods: {
      __type: "[Pod!]!",
      __args: { filter: "PodFilter", namespace: "String" },
    },
    user: { __type: "String!" },
  },
  Label: {
    __typename: { __type: "String!" },
    key: { __type: "String!" },
    value: { __type: "String!" },
  },
  LogLine: {
    __typename: { __type: "String!" },
    container: { __type: "String" },
    line: { __type: "String!" },
    timestamp: { __type: "Time" },
  },
  Namespace: {
    __typename: { __type: "String!" },
    apiResources: { __type: "[ApiResource!]!" },
    context: { __type: "String!" },
    customResources: {
      __type: "[CustomResource!]!",
      __args: { filter: "CustomResourceFilter" },
    },
    deployments: {
      __type: "[Deployment!]!",
      __args: { filter: "DeploymentFilter" },
    },
    name: { __type: "String!" },
    pods: { __type: "[Pod!]!", __args: { filter: "PodFilter" } },
  },
  NamespaceFilter: {
    hasDeployment: { __type: "DeploymentFilter" },
    hasPod: { __type: "PodFilter" },
    nameContains: { __type: "String" },
  },
  Pod: {
    __typename: { __type: "String!" },
    containers: { __type: "[Container!]!" },
    context: { __type: "String!" },
    cpuLimit: { __type: "String" },
    cpuUsage: { __type: "String" },
    createdAt: { __type: "Time" },
    labels: { __type: "[Label!]!" },
    lastRestartAt: { __type: "Time" },
    memoryLimit: { __type: "String" },
    memoryUsage: { __type: "String" },
    name: { __type: "String!" },
    namespace: { __type: "String!" },
    nodeName: { __type: "String" },
    phase: { __type: "String!" },
    ready: { __type: "Boolean!" },
    restarts: { __type: "Int!" },
    yaml: { __type: "String!" },
  },
  PodFilter: {
    labelSelector: { __type: "String" },
    nameContains: { __type: "String" },
    phase: { __type: "String" },
    ready: { __type: "Boolean" },
  },
  Secret: {
    __typename: { __type: "String!" },
    context: { __type: "String!" },
    data: { __type: "[KeyValue!]!" },
    keys: { __type: "[String!]!" },
    missing: { __type: "Boolean!" },
    name: { __type: "String!" },
    namespace: { __type: "String!" },
    refs: { __type: "[String!]!" },
    type: { __type: "String!" },
    yaml: { __type: "String!" },
  },
  mutation: {},
  query: {
    __typename: { __type: "String!" },
    apiResources: { __type: "[ApiResource!]!", __args: { context: "String!" } },
    configMap: {
      __type: "ConfigMap",
      __args: { context: "String!", name: "String!", namespace: "String!" },
    },
    configMaps: {
      __type: "[ConfigMap!]!",
      __args: { context: "String!", namespace: "String!" },
    },
    contexts: { __type: "[KubeContext!]!" },
    customResource: {
      __type: "CustomResource",
      __args: {
        context: "String!",
        group: "String!",
        name: "String!",
        namespace: "String",
        resource: "String!",
        version: "String!",
      },
    },
    customResources: {
      __type: "[CustomResource!]!",
      __args: {
        context: "String!",
        filter: "CustomResourceFilter",
        namespace: "String",
      },
    },
    deployment: {
      __type: "Deployment",
      __args: { context: "String!", name: "String!", namespace: "String!" },
    },
    deployments: {
      __type: "[Deployment!]!",
      __args: {
        context: "String!",
        filter: "DeploymentFilter",
        namespace: "String",
      },
    },
    namespace: {
      __type: "Namespace",
      __args: { context: "String!", name: "String!" },
    },
    namespaces: {
      __type: "[Namespace!]!",
      __args: { context: "String!", filter: "NamespaceFilter" },
    },
    pod: {
      __type: "Pod",
      __args: { context: "String!", name: "String!", namespace: "String!" },
    },
    pods: {
      __type: "[Pod!]!",
      __args: { context: "String!", filter: "PodFilter", namespace: "String" },
    },
    secret: {
      __type: "Secret",
      __args: { context: "String!", name: "String!", namespace: "String!" },
    },
    secrets: {
      __type: "[Secret!]!",
      __args: { context: "String!", namespace: "String!" },
    },
  },
  subscription: {
    __typename: { __type: "String!" },
    deploymentStatus: {
      __type: "Deployment!",
      __args: { context: "String!", name: "String!", namespace: "String!" },
    },
    podLogs: {
      __type: "LogLine!",
      __args: {
        container: "String",
        context: "String!",
        name: "String!",
        namespace: "String!",
        tailLines: "Int",
      },
    },
    podStatus: {
      __type: "Pod!",
      __args: { context: "String!", name: "String!", namespace: "String!" },
    },
  },
} as const;

/**
 * A discovered API resource that is not a built-in Kubernetes kind (typically a CRD).
 */
export interface ApiResource {
  __typename?: "ApiResource";
  group?: Scalars["String"]["output"];
  kind?: Scalars["String"]["output"];
  namespaced?: Scalars["Boolean"]["output"];
  /**
   * Plural resource name, e.g. kustomizations.
   */
  resource?: Scalars["String"]["output"];
  version?: Scalars["String"]["output"];
}

export interface ConfigMap {
  __typename?: "ConfigMap";
  /**
   * Kubeconfig context this resource was loaded from.
   */
  context?: Scalars["String"]["output"];
  data: Array<KeyValue>;
  keys?: Array<Scalars["String"]["output"]>;
  /**
   * True when the object was referenced but not found in the cluster.
   */
  missing?: Scalars["Boolean"]["output"];
  name?: Scalars["String"]["output"];
  namespace?: Scalars["String"]["output"];
  /**
   * How a parent deployment references this object (env, envFrom, volume, …).
   */
  refs?: Array<Scalars["String"]["output"]>;
  /**
   * Full object as YAML.
   */
  yaml?: Scalars["String"]["output"];
}

export interface Container {
  __typename?: "Container";
  image?: Scalars["String"]["output"];
  name?: Scalars["String"]["output"];
  ready?: Scalars["Boolean"]["output"];
  restartCount?: Scalars["Int"]["output"];
  /**
   * e.g. Running, Waiting, Terminated.
   */
  state?: Scalars["String"]["output"];
}

/**
 * A generic custom resource instance (Flux Kustomization, cert-manager Certificate, …).
 */
export interface CustomResource {
  __typename?: "CustomResource";
  apiVersion?: Scalars["String"]["output"];
  /**
   * Kubeconfig context this resource was loaded from.
   */
  context?: Scalars["String"]["output"];
  createdAt?: Maybe<Scalars["Time"]["output"]>;
  group?: Scalars["String"]["output"];
  kind?: Scalars["String"]["output"];
  labels: Array<Label>;
  message?: Maybe<Scalars["String"]["output"]>;
  name?: Scalars["String"]["output"];
  /**
   * Null when the resource is cluster-scoped.
   */
  namespace?: Maybe<Scalars["String"]["output"]>;
  /**
   * status.conditions type=Ready (or Healthy/Available). Null when unknown.
   */
  ready?: Maybe<Scalars["Boolean"]["output"]>;
  reason?: Maybe<Scalars["String"]["output"]>;
  resource?: Scalars["String"]["output"];
  version?: Scalars["String"]["output"];
  /**
   * Full object as YAML.
   */
  yaml?: Scalars["String"]["output"];
}

export interface Deployment {
  __typename?: "Deployment";
  availableReplicas?: Scalars["Int"]["output"];
  /**
   * ConfigMaps referenced by the pod template (env, envFrom, volumes).
   */
  configMaps: Array<ConfigMap>;
  /**
   * Kubeconfig context this resource was loaded from.
   */
  context?: Scalars["String"]["output"];
  labels: Array<Label>;
  name?: Scalars["String"]["output"];
  namespace?: Scalars["String"]["output"];
  /**
   * Pods owned by this deployment (via label selector).
   */
  pods: (args?: { filter?: Maybe<PodFilter> }) => Array<Pod>;
  readyReplicas?: Scalars["Int"]["output"];
  replicas?: Scalars["Int"]["output"];
  /**
   * Sum of container restart counts across owned pods.
   */
  restarts?: Scalars["Int"]["output"];
  /**
   * Secrets referenced by the pod template (env, envFrom, volumes, imagePull).
   */
  secrets: Array<Secret>;
  /**
   * Human-readable summary, e.g. "3/3" or condition reason.
   */
  status?: Scalars["String"]["output"];
  /**
   * Full object as YAML.
   */
  yaml?: Scalars["String"]["output"];
}

/**
 * A string map entry (ConfigMap/Secret data, etc.).
 */
export interface KeyValue {
  __typename?: "KeyValue";
  key?: Scalars["String"]["output"];
  value?: Scalars["String"]["output"];
}

/**
 * A context entry from the loaded kubeconfig.
 * Cluster resources are nested here so one clientset is used per context.
 */
export interface KubeContext {
  __typename?: "KubeContext";
  /**
   * Discovered extension API resources (CRDs).
   */
  apiResources: Array<ApiResource>;
  cluster?: Scalars["String"]["output"];
  /**
   * True if this is the CLI/--context or kubeconfig current-context default.
   */
  current?: Scalars["Boolean"]["output"];
  /**
   * Custom resources. Namespace is optional (all namespaces when omitted).
   */
  customResources: (args?: {
    filter?: Maybe<CustomResourceFilter>;
    namespace?: Maybe<Scalars["String"]["input"]>;
  }) => Array<CustomResource>;
  /**
   * Default namespace from the kubeconfig context entry, if set.
   */
  defaultNamespace?: Maybe<Scalars["String"]["output"]>;
  /**
   * Get a single deployment in this context. Null if missing or cluster unreachable.
   */
  deployment: (args: {
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Maybe<Deployment>;
  /**
   * List deployments. Namespace is optional (all namespaces when omitted).
   * Unreachable clusters return [] (same best-effort behavior as namespaces).
   */
  deployments: (args?: {
    filter?: Maybe<DeploymentFilter>;
    namespace?: Maybe<Scalars["String"]["input"]>;
  }) => Array<Deployment>;
  name?: Scalars["String"]["output"];
  /**
   * Get a single namespace in this context. Null if missing or cluster unreachable.
   */
  namespace: (args: { name: Scalars["String"]["input"] }) => Maybe<Namespace>;
  /**
   * List namespaces in this context.
   * Unreachable/stale clusters log the error and return [] so other contexts still resolve.
   */
  namespaces: (args?: { filter?: Maybe<NamespaceFilter> }) => Array<Namespace>;
  /**
   * Get a single pod in this context. Null if missing or cluster unreachable.
   */
  pod: (args: {
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Maybe<Pod>;
  /**
   * List pods. Namespace is optional (all namespaces when omitted).
   * Unreachable clusters return [] (same best-effort behavior as namespaces).
   */
  pods: (args?: {
    filter?: Maybe<PodFilter>;
    namespace?: Maybe<Scalars["String"]["input"]>;
  }) => Array<Pod>;
  user?: Scalars["String"]["output"];
}

export interface Label {
  __typename?: "Label";
  key?: Scalars["String"]["output"];
  value?: Scalars["String"]["output"];
}

export interface LogLine {
  __typename?: "LogLine";
  container?: Maybe<Scalars["String"]["output"]>;
  line?: Scalars["String"]["output"];
  timestamp?: Maybe<Scalars["Time"]["output"]>;
}

export interface Namespace {
  __typename?: "Namespace";
  /**
   * Discovered namespaced extension API resources (CRDs).
   */
  apiResources: Array<ApiResource>;
  /**
   * Kubeconfig context this resource was loaded from.
   */
  context?: Scalars["String"]["output"];
  /**
   * Namespaced custom resources in this namespace (CRDs / extension APIs).
   */
  customResources: (args?: {
    filter?: Maybe<CustomResourceFilter>;
  }) => Array<CustomResource>;
  deployments: (args?: {
    filter?: Maybe<DeploymentFilter>;
  }) => Array<Deployment>;
  name?: Scalars["String"]["output"];
  pods: (args?: { filter?: Maybe<PodFilter> }) => Array<Pod>;
}

export interface Pod {
  __typename?: "Pod";
  containers: Array<Container>;
  /**
   * Kubeconfig context this resource was loaded from.
   */
  context?: Scalars["String"]["output"];
  /**
   * Sum of container CPU limits from the pod spec. Null when unset.
   */
  cpuLimit?: Maybe<Scalars["String"]["output"]>;
  /**
   * Optional; requires metrics-server. Null when unavailable.
   */
  cpuUsage?: Maybe<Scalars["String"]["output"]>;
  /**
   * Pod creation time (metadata.creationTimestamp).
   */
  createdAt?: Maybe<Scalars["Time"]["output"]>;
  labels: Array<Label>;
  lastRestartAt?: Maybe<Scalars["Time"]["output"]>;
  /**
   * Sum of container memory limits from the pod spec. Null when unset.
   */
  memoryLimit?: Maybe<Scalars["String"]["output"]>;
  /**
   * Optional; requires metrics-server. Null when unavailable.
   */
  memoryUsage?: Maybe<Scalars["String"]["output"]>;
  name?: Scalars["String"]["output"];
  namespace?: Scalars["String"]["output"];
  nodeName?: Maybe<Scalars["String"]["output"]>;
  phase?: Scalars["String"]["output"];
  ready?: Scalars["Boolean"]["output"];
  restarts?: Scalars["Int"]["output"];
  /**
   * Full object as YAML.
   */
  yaml?: Scalars["String"]["output"];
}

export interface Secret {
  __typename?: "Secret";
  /**
   * Kubeconfig context this resource was loaded from.
   */
  context?: Scalars["String"]["output"];
  /**
   * Decoded values when UTF-8; otherwise a binary placeholder.
   */
  data: Array<KeyValue>;
  keys?: Array<Scalars["String"]["output"]>;
  /**
   * True when the object was referenced but not found in the cluster.
   */
  missing?: Scalars["Boolean"]["output"];
  name?: Scalars["String"]["output"];
  namespace?: Scalars["String"]["output"];
  /**
   * How a parent deployment references this object (env, envFrom, volume, imagePull).
   */
  refs?: Array<Scalars["String"]["output"]>;
  /**
   * Kubernetes secret type, e.g. Opaque, kubernetes.io/tls.
   */
  type?: Scalars["String"]["output"];
  /**
   * Full object as YAML.
   */
  yaml?: Scalars["String"]["output"];
}

export interface Mutation {
  __typename?: "Mutation";
}

export interface Query {
  __typename?: "Query";
  /**
   * List discovered extension API resources (CRDs) in a kubeconfig context.
   */
  apiResources: (args: {
    context: Scalars["String"]["input"];
  }) => Array<ApiResource>;
  /**
   * Get a single ConfigMap. Null if missing.
   */
  configMap: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Maybe<ConfigMap>;
  /**
   * List ConfigMaps in a namespace.
   */
  configMaps: (args: {
    context: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Array<ConfigMap>;
  /**
   * List kubeconfig contexts. Nest cluster resources under a context, e.g.
   * contexts { name namespaces { name } }.
   */
  contexts: Array<KubeContext>;
  /**
   * Get a single custom resource. Null if missing.
   */
  customResource: (args: {
    context: Scalars["String"]["input"];
    group: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace?: Maybe<Scalars["String"]["input"]>;
    resource: Scalars["String"]["input"];
    version: Scalars["String"]["input"];
  }) => Maybe<CustomResource>;
  /**
   * List custom resource instances. Namespace is optional.
   */
  customResources: (args: {
    context: Scalars["String"]["input"];
    filter?: Maybe<CustomResourceFilter>;
    namespace?: Maybe<Scalars["String"]["input"]>;
  }) => Array<CustomResource>;
  /**
   * Get a single deployment.
   */
  deployment: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Maybe<Deployment>;
  /**
   * List deployments. Namespace is optional (all namespaces when omitted).
   */
  deployments: (args: {
    context: Scalars["String"]["input"];
    filter?: Maybe<DeploymentFilter>;
    namespace?: Maybe<Scalars["String"]["input"]>;
  }) => Array<Deployment>;
  /**
   * Get a single namespace in the given kubeconfig context.
   */
  namespace: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
  }) => Maybe<Namespace>;
  /**
   * List namespaces in the given kubeconfig context.
   */
  namespaces: (args: {
    context: Scalars["String"]["input"];
    filter?: Maybe<NamespaceFilter>;
  }) => Array<Namespace>;
  /**
   * Get a single pod.
   */
  pod: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Maybe<Pod>;
  /**
   * List pods. Namespace is optional (all namespaces when omitted).
   */
  pods: (args: {
    context: Scalars["String"]["input"];
    filter?: Maybe<PodFilter>;
    namespace?: Maybe<Scalars["String"]["input"]>;
  }) => Array<Pod>;
  /**
   * Get a single Secret. Null if missing.
   */
  secret: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Maybe<Secret>;
  /**
   * List Secrets in a namespace.
   */
  secrets: (args: {
    context: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Array<Secret>;
}

export interface Subscription {
  __typename?: "Subscription";
  /**
   * Push updates when deployment status or owned-pod metrics change.
   * MVP: poll about every 2s (informers later).
   */
  deploymentStatus: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Deployment;
  /**
   * Stream container logs for a pod. Prefer this over a one-shot query.
   */
  podLogs: (args: {
    container?: Maybe<Scalars["String"]["input"]>;
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
    tailLines?: Maybe<Scalars["Int"]["input"]>;
  }) => LogLine;
  /**
   * Push updates when pod status changes.
   */
  podStatus: (args: {
    context: Scalars["String"]["input"];
    name: Scalars["String"]["input"];
    namespace: Scalars["String"]["input"];
  }) => Pod;
}

export interface GeneratedSchema {
  query: Query;
  mutation: Mutation;
  subscription: Subscription;
}

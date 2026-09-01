/**
 * GQty client for kubeql.
 *
 * Conventions: see /gqty-gotchas.md (project root).
 * - suspense: false (never re-enable globally)
 * - maxAge: Infinity + hard $refetch(true) on mount for data screens
 * - only useQuery + q.$refetch / resolve — no useTransactionQuery / useRefetch
 *
 * Runtime endpoint: /query (same-origin; Vite proxies in dev).
 * After `gqty generate`, re-check this file if the CLI rewrote the fetcher.
 */

import { createReactClient } from "@gqty/react";
import {
  Cache,
  createClient,
  defaultResponseHandler,
  type QueryFetcher,
} from "gqty";
import { createClient as createWSClient } from "graphql-ws";
import {
  generatedSchema,
  scalarsEnumsHash,
  type GeneratedSchema,
} from "./schema.generated";

const GQL_ENDPOINT = import.meta.env.VITE_GQL_URL ?? "/query";

function graphqlWsUrl(): string {
  if (GQL_ENDPOINT.startsWith("ws://") || GQL_ENDPOINT.startsWith("wss://")) {
    return GQL_ENDPOINT;
  }
  if (GQL_ENDPOINT.startsWith("http://") || GQL_ENDPOINT.startsWith("https://")) {
    return GQL_ENDPOINT.replace(/^http/, "ws");
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  const path = GQL_ENDPOINT.startsWith("/") ? GQL_ENDPOINT : `/${GQL_ENDPOINT}`;
  return `${proto}//${window.location.host}${path}`;
}

const queryFetcher: QueryFetcher = async function (
  { query, variables, operationName },
  fetchOptions
) {
  const response = await fetch(GQL_ENDPOINT, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      query,
      variables,
      operationName,
    }),
    mode: "cors",
    ...fetchOptions,
  });

  return await defaultResponseHandler(response);
};

const cache = new Cache(undefined, {
  // Stale until hard refetch / hard navigation — see gqty-gotchas.md
  maxAge: Infinity,
  staleWhileRevalidate: 30 * 60 * 1000,
  normalization: true,
});

const subscriber = createWSClient({
  url: graphqlWsUrl,
});

export const client = createClient<GeneratedSchema>({
  schema: generatedSchema,
  scalars: scalarsEnumsHash,
  cache,
  fetchOptions: {
    fetcher: queryFetcher,
    subscriber,
  },
});

// Core: prefer resolve for one-shot imperative fetches (same fetcher path).
export const { resolve, subscribe, schema } = client;

// Legacy client surface (query proxy, mutate helpers). Prefer hooks in React.
export const {
  query,
  mutation,
  mutate,
  subscription,
  resolved,
  refetch,
  track,
} = client;

export const {
  graphql,
  useQuery,
  usePaginatedQuery,
  useLazyQuery,
  useMutation,
  useSubscription,
  useMetaState,
  prepareReactRender,
  useHydrateCache,
  prepareQuery,
  // Deliberately not exported (gotchas): useTransactionQuery, useRefetch
} = createReactClient<GeneratedSchema>(client, {
  defaults: {
    // NEVER re-enable globally — re-suspend after refetch tears down UI.
    suspense: false,
  },
});

export * from "./schema.generated";

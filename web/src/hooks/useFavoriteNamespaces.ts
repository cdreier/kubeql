import { useCallback, useState } from "react";

const STORAGE_KEY = "kubeql.favoriteNamespaces";

type Store = Record<string, string[]>;

function readStore(): Store {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    const out: Store = {};
    for (const [ctx, names] of Object.entries(parsed)) {
      if (!Array.isArray(names)) continue;
      out[ctx] = names.filter((n): n is string => typeof n === "string");
    }
    return out;
  } catch {
    return {};
  }
}

function writeStore(store: Store) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store));
  } catch {
    // quota / private mode — keep in-memory state
  }
}

export function useFavoriteNamespaces(contextName: string) {
  const [store, setStore] = useState<Store>(readStore);
  const favorites = store[contextName] ?? [];

  const toggleFavorite = useCallback(
    (name: string) => {
      if (!contextName) return;
      setStore((prev) => {
        const current = prev[contextName] ?? [];
        const nextNames = current.includes(name)
          ? current.filter((n) => n !== name)
          : [...current, name];
        const next: Store = { ...prev };
        if (nextNames.length === 0) delete next[contextName];
        else next[contextName] = nextNames;
        writeStore(next);
        return next;
      });
    },
    [contextName]
  );

  return { favorites, toggleFavorite };
}

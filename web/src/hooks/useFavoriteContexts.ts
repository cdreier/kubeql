import { useCallback, useState } from "react";

const STORAGE_KEY = "kubeql.favoriteContexts";

function readFavorites(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is string => typeof item === "string");
  } catch {
    return [];
  }
}

function writeFavorites(names: string[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(names));
  } catch {
    // quota / private mode — keep in-memory state
  }
}

export function useFavoriteContexts() {
  const [favorites, setFavorites] = useState<string[]>(readFavorites);

  const toggleFavorite = useCallback((name: string) => {
    setFavorites((prev) => {
      const next = prev.includes(name)
        ? prev.filter((n) => n !== name)
        : [...prev, name];
      writeFavorites(next);
      return next;
    });
  }, []);

  return { favorites, toggleFavorite };
}

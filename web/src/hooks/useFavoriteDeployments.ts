import { useCallback, useMemo, useState } from "react";

export type FavDeployment = {
  context: string;
  namespace: string;
  name: string;
};

export function favDeploymentKey(f: FavDeployment): string {
  return `${f.context}\0${f.namespace}\0${f.name}`;
}

const STORAGE_KEY = "kubeql.favoriteDeployments";

function isFav(item: unknown): item is FavDeployment {
  if (!item || typeof item !== "object") return false;
  const rec = item as Record<string, unknown>;
  return (
    typeof rec.context === "string" &&
    typeof rec.namespace === "string" &&
    typeof rec.name === "string"
  );
}

function readFavorites(): FavDeployment[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isFav);
  } catch {
    return [];
  }
}

function writeFavorites(items: FavDeployment[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
  } catch {
    // quota / private mode
  }
}

export function useFavoriteDeployments() {
  const [favorites, setFavorites] = useState<FavDeployment[]>(readFavorites);
  const keySet = useMemo(
    () => new Set(favorites.map(favDeploymentKey)),
    [favorites]
  );

  const isFavorite = useCallback(
    (f: FavDeployment) => keySet.has(favDeploymentKey(f)),
    [keySet]
  );

  const toggleFavorite = useCallback((f: FavDeployment) => {
    setFavorites((prev) => {
      const key = favDeploymentKey(f);
      const next = prev.some((p) => favDeploymentKey(p) === key)
        ? prev.filter((p) => favDeploymentKey(p) !== key)
        : [...prev, f];
      writeFavorites(next);
      return next;
    });
  }, []);

  return { favorites, isFavorite, toggleFavorite };
}

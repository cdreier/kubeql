import { useEffect, useMemo, useRef } from "react";
import { Link } from "react-router-dom";
import { FavoriteDeploymentsTable } from "../components/FavoriteDeploymentsTable";
import { FavoriteStar } from "../components/FavoriteStar";
import { useQuery } from "../gqty";
import type { Deployment } from "../gqty";
import { useFavoriteContexts } from "../hooks/useFavoriteContexts";
import { useFavoriteDeployments } from "../hooks/useFavoriteDeployments";
import { readDeploymentSummary } from "../lib/deploymentSummary";
import "./ContextListPage.css";
import "./ContextPage.css";
import "./DeploymentPage.css";

export function ContextListPage() {
  const q = useQuery();
  const loadedRef = useRef(false);
  const { favorites, toggleFavorite } = useFavoriteContexts();
  const favoriteSet = useMemo(() => new Set(favorites), [favorites]);
  const {
    favorites: favDeps,
    toggleFavorite: toggleFavDep,
  } = useFavoriteDeployments();

  // Touch selections before any early return.
  const contextProxies = q.contexts;
  const contexts = contextProxies
    .map((c) => ({
      name: c.name,
      cluster: c.cluster,
      user: c.user,
      current: c.current,
    }))
    .filter((c) => Boolean(c.name));

  // Touch favorite deployment fields before any early return.
  for (const fav of favDeps) {
    const ctx = contextProxies.find((c) => c.name === fav.context);
    if (!ctx) continue;
    void readDeploymentSummary(
      ctx.deployment({
        namespace: fav.namespace,
        name: fav.name,
      }) as Deployment,
      fav.namespace
    );
  }

  const sortedContexts = useMemo(() => {
    return [...contexts].sort((a, b) => {
      const aFav = favoriteSet.has(a.name!) ? 0 : 1;
      const bFav = favoriteSet.has(b.name!) ? 0 : 1;
      return aFav - bFav;
    });
  }, [contexts, favoriteSet]);

  if (contexts.length > 0) {
    loadedRef.current = true;
  }

  useEffect(() => {
    void q.$refetch(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount-only hard refetch
  }, []);

  const showInitialLoading =
    !loadedRef.current && q.$state.isLoading && contexts.length === 0;

  if (showInitialLoading) {
    return <p className="muted">Loading contexts…</p>;
  }

  if (q.$state.error && !loadedRef.current) {
    return (
      <p className="error">
        GraphQL error: {q.$state.error.message}
        <br />
        <span className="muted">
          Is the backend running on :8080? (Vite proxies /query)
        </span>
      </p>
    );
  }

  return (
    <div className="context-list-page">
      <FavoriteDeploymentsTable
        favorites={favDeps}
        contexts={contextProxies}
        onToggle={toggleFavDep}
      />
      <h1>Contexts</h1>
      <p className="muted">Pick a kubeconfig context to browse namespaces.</p>
      <ul className="context-list">
        {sortedContexts.map((c) => {
          const name = c.name!;
          const favorited = favoriteSet.has(name);
          return (
            <li key={name}>
              <div className="context-item">
                <FavoriteStar
                  name={name}
                  favorited={favorited}
                  onToggle={toggleFavorite}
                />
                <Link
                  to={`/c/${encodeURIComponent(name)}`}
                  className="context-row"
                >
                  <span className="context-name">
                    {name}
                    {c.current ? (
                      <span className="badge current">current</span>
                    ) : null}
                  </span>
                  <span className="muted context-meta">
                    {c.cluster}
                    {c.user ? ` · ${c.user}` : ""}
                  </span>
                </Link>
              </div>
            </li>
          );
        })}
        {contexts.length === 0 && !q.$state.isLoading && (
          <li className="muted empty">No contexts in kubeconfig</li>
        )}
      </ul>
    </div>
  );
}

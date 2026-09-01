import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { FavoriteStar } from "../components/FavoriteStar";
import { useQuery } from "../gqty";
import { useFavoriteNamespaces } from "../hooks/useFavoriteNamespaces";
import "./NamespaceListPage.css";

export function NamespaceListPage() {
  const { contextName: raw } = useParams<{ contextName: string }>();
  const contextName = raw ? decodeURIComponent(raw) : "";

  const q = useQuery();
  const loadedRef = useRef(false);
  const [search, setSearch] = useState("");
  const { favorites, toggleFavorite } = useFavoriteNamespaces(contextName);
  const favoriteSet = useMemo(() => new Set(favorites), [favorites]);

  // Touch selections before any early return. Names only — no deployments.
  const namespaces = q
    .namespaces({ context: contextName })
    .map((ns) => ns.name)
    .filter((name): name is string => Boolean(name));

  if (namespaces.length > 0) {
    loadedRef.current = true;
  }

  useEffect(() => {
    void q.$refetch(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount / context change
  }, [contextName]);

  useEffect(() => {
    setSearch("");
  }, [contextName]);

  const searchLower = search.trim().toLowerCase();
  const visible = useMemo(() => {
    const filtered = searchLower
      ? namespaces.filter((name) => name.toLowerCase().includes(searchLower))
      : namespaces;
    return [...filtered].sort((a, b) => {
      const aFav = favoriteSet.has(a) ? 0 : 1;
      const bFav = favoriteSet.has(b) ? 0 : 1;
      return aFav - bFav;
    });
  }, [namespaces, searchLower, favoriteSet]);

  const showInitialLoading =
    !loadedRef.current && q.$state.isLoading && namespaces.length === 0;

  if (!contextName) {
    return <p className="error">Missing context name</p>;
  }

  if (showInitialLoading) {
    return <p className="muted">Loading namespaces…</p>;
  }

  if (q.$state.error && !loadedRef.current) {
    return (
      <p className="error">Failed to load namespaces: {q.$state.error.message}</p>
    );
  }

  const contextPath = `/c/${encodeURIComponent(contextName)}`;

  return (
    <div className="ns-list-page">
      <div className="page-heading">
        <h1>Namespaces</h1>
        {q.$state.isLoading && loadedRef.current ? (
          <span className="muted refresh-hint">Refreshing…</span>
        ) : null}
      </div>
      <p className="muted">
        {searchLower
          ? `${visible.length} of ${namespaces.length} namespaces`
          : "Pick a namespace to browse deployments."}
      </p>
      <input
        type="search"
        className="ns-search"
        placeholder="Filter namespaces…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        aria-label="Filter namespaces by name"
        autoFocus
      />
      <ul className="ns-name-list">
        {visible.map((name) => {
          const favorited = favoriteSet.has(name);
          return (
            <li key={name}>
              <div className="ns-item">
                <FavoriteStar
                  name={name}
                  favorited={favorited}
                  onToggle={toggleFavorite}
                />
                <Link
                  to={`${contextPath}/n/${encodeURIComponent(name)}`}
                  className="ns-row"
                >
                  {name}
                </Link>
              </div>
            </li>
          );
        })}
        {visible.length === 0 && !q.$state.isLoading && (
          <li className="muted empty">
            {namespaces.length === 0
              ? "No namespaces (or cluster unreachable — check server logs)."
              : "No namespaces match the filter."}
          </li>
        )}
      </ul>
    </div>
  );
}

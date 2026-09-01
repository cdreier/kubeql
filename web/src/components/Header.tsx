import { Fragment } from "react";
import { Link } from "react-router-dom";
import "./Header.css";

export type HeaderCrumb = {
  label: string;
  to?: string;
};

type HeaderProps = {
  crumbs?: HeaderCrumb[];
};

export function Header({ crumbs = [] }: HeaderProps) {
  return (
    <header className="app-header">
      <Link to="/" className="brand">
        kubeql
      </Link>
      {crumbs.map((crumb) => (
        <Fragment key={`${crumb.to ?? ""}:${crumb.label}`}>
          <span className="sep" aria-hidden>
            /
          </span>
          {crumb.to ? (
            <Link to={crumb.to} className="subtitle crumb-link" title={crumb.label}>
              {crumb.label}
            </Link>
          ) : (
            <span className="subtitle" title={crumb.label}>
              {crumb.label}
            </span>
          )}
        </Fragment>
      ))}
      <nav className="header-nav">
        <Link to="/">Contexts</Link>
      </nav>
    </header>
  );
}

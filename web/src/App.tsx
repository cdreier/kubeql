import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
  useParams,
} from "react-router-dom";
import { Header } from "./components/Header";
import "./components/FavoriteStar.css";
import { ContextListPage } from "./pages/ContextListPage";
import { ContextPage } from "./pages/ContextPage";
import { CronJobPage } from "./pages/CronJobPage";
import { DeploymentPage } from "./pages/DeploymentPage";
import { NamespaceListPage } from "./pages/NamespaceListPage";
import "./App.css";

function decodeParam(value: string | undefined): string | undefined {
  return value ? decodeURIComponent(value) : undefined;
}

function ContextLayout() {
  const { contextName } = useParams<{ contextName: string }>();
  const context = decodeParam(contextName);
  return (
    <>
      <Header crumbs={context ? [{ label: context }] : []} />
      <main className="page">
        {/* key remounts query scope when context arg changes (gqty-gotchas) */}
        <NamespaceListPage key={contextName ?? ""} />
      </main>
    </>
  );
}

function NamespaceLayout() {
  const { contextName, nsName } = useParams<{
    contextName: string;
    nsName: string;
  }>();
  const context = decodeParam(contextName);
  const ns = decodeParam(nsName);
  return (
    <>
      <Header
        crumbs={[
          ...(context
            ? [
                {
                  label: context,
                  to: `/c/${encodeURIComponent(context)}`,
                },
              ]
            : []),
          ...(ns ? [{ label: ns }] : []),
        ]}
      />
      <main className="page">
        <ContextPage key={`${contextName ?? ""}/${nsName ?? ""}`} />
      </main>
    </>
  );
}

function DeploymentLayout() {
  const { contextName, nsName, depName } = useParams<{
    contextName: string;
    nsName: string;
    depName: string;
  }>();
  const context = decodeParam(contextName);
  const ns = decodeParam(nsName);
  const dep = decodeParam(depName);
  return (
    <>
      <Header
        crumbs={[
          ...(context
            ? [
                {
                  label: context,
                  to: `/c/${encodeURIComponent(context)}`,
                },
              ]
            : []),
          ...(context && ns
            ? [
                {
                  label: ns,
                  to: `/c/${encodeURIComponent(context)}/n/${encodeURIComponent(ns)}`,
                },
              ]
            : []),
          ...(dep ? [{ label: dep }] : []),
        ]}
      />
      <main className="page">
        <DeploymentPage
          key={`${contextName ?? ""}/${nsName ?? ""}/${depName ?? ""}`}
        />
      </main>
    </>
  );
}

function CronJobLayout() {
  const { contextName, nsName, cronName } = useParams<{
    contextName: string;
    nsName: string;
    cronName: string;
  }>();
  const context = decodeParam(contextName);
  const ns = decodeParam(nsName);
  const cron = decodeParam(cronName);
  return (
    <>
      <Header
        crumbs={[
          ...(context
            ? [
                {
                  label: context,
                  to: `/c/${encodeURIComponent(context)}`,
                },
              ]
            : []),
          ...(context && ns
            ? [
                {
                  label: ns,
                  to: `/c/${encodeURIComponent(context)}/n/${encodeURIComponent(ns)}`,
                },
              ]
            : []),
          ...(cron ? [{ label: cron }] : []),
        ]}
      />
      <main className="page">
        <CronJobPage
          key={`${contextName ?? ""}/${nsName ?? ""}/${cronName ?? ""}`}
        />
      </main>
    </>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <div className="app-shell">
        <Routes>
          <Route
            path="/"
            element={
              <>
                <Header />
                <main className="page">
                  <ContextListPage />
                </main>
              </>
            }
          />
          <Route path="/c/:contextName" element={<ContextLayout />} />
          <Route
            path="/c/:contextName/n/:nsName"
            element={<NamespaceLayout />}
          />
          <Route
            path="/c/:contextName/n/:nsName/d/:depName"
            element={<DeploymentLayout />}
          />
          <Route
            path="/c/:contextName/n/:nsName/cj/:cronName"
            element={<CronJobLayout />}
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

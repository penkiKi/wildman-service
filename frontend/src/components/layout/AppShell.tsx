import { Activity, AudioLines, LayoutDashboard, LogOut, MonitorSmartphone } from "lucide-react";
import { useState } from "react";
import { api, type User } from "../../lib/api";
import { ClientsPage } from "../clients/ClientsPage";
import { Dashboard } from "../dashboard/Dashboard";
import { OperationsPage } from "../operations/OperationsPage";

type AppShellProps = {
  user: User;
  csrfToken: string;
  onLogout: () => void;
};

export function AppShell({ user, csrfToken, onLogout }: AppShellProps) {
  const [view, setView] = useState<"overview" | "clients" | "operations">("overview");
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");

  async function logout() {
    setLoggingOut(true);
    setLogoutError("");
    try {
      await api.logout(csrfToken);
      onLogout();
    } catch (requestError) {
      setLogoutError(requestError instanceof Error ? requestError.message : "退出失败");
    } finally {
      setLoggingOut(false);
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="fixed inset-x-0 top-0 z-20 flex h-16 items-center border-b border-border bg-background/85 px-4 backdrop-blur-xl sm:px-6">
        <div className="flex items-center gap-3">
          <div className="grid size-9 place-items-center rounded-lg border border-primary/30 bg-primary/10 text-primary">
            <AudioLines size={18} aria-hidden="true" />
          </div>
          <span className="text-sm font-semibold tracking-[0.16em]">WILDMAN</span>
        </div>
        <div className="ml-auto flex items-center gap-3">
          <span className="max-w-32 truncate text-sm text-muted-foreground">{user.username}</span>
          <button
            aria-label="退出登录"
            className="grid size-9 place-items-center rounded-lg border border-border bg-surface text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-60"
            disabled={loggingOut}
            onClick={() => void logout()}
            type="button"
          >
            <LogOut size={16} aria-hidden="true" />
          </button>
        </div>
      </header>

      {logoutError ? (
        <p className="fixed right-4 top-20 z-30 rounded-lg border border-destructive/30 bg-surface px-3 py-2 text-xs text-destructive shadow-lg">
          {logoutError}
        </p>
      ) : null}

      <div className="flex min-h-screen pt-16">
        <aside className="fixed bottom-0 left-0 top-16 hidden w-56 border-r border-border bg-surface/35 p-3 sm:block">
          <nav aria-label="主导航" className="space-y-1">
            <button
              aria-current={view === "overview" ? "page" : undefined}
              className={view === "overview" ? "flex h-10 w-full items-center gap-3 rounded-lg bg-accent/10 px-3 text-sm font-medium text-accent" : "flex h-10 w-full items-center gap-3 rounded-lg px-3 text-sm text-muted-foreground transition hover:bg-muted hover:text-foreground"}
              onClick={() => setView("overview")}
              type="button"
            >
              <LayoutDashboard size={17} aria-hidden="true" />
              概览
            </button>
            <button aria-current={view === "operations" ? "page" : undefined} className={view === "operations" ? "flex h-10 w-full items-center gap-3 rounded-lg bg-accent/10 px-3 text-sm font-medium text-accent" : "flex h-10 w-full items-center gap-3 rounded-lg px-3 text-sm text-muted-foreground transition hover:bg-muted hover:text-foreground"} onClick={() => setView("operations")} type="button"><Activity size={17} aria-hidden="true" />运营</button>
            <button
              aria-current={view === "clients" ? "page" : undefined}
              className={view === "clients" ? "flex h-10 w-full items-center gap-3 rounded-lg bg-accent/10 px-3 text-sm font-medium text-accent" : "flex h-10 w-full items-center gap-3 rounded-lg px-3 text-sm text-muted-foreground transition hover:bg-muted hover:text-foreground"}
              onClick={() => setView("clients")}
              type="button"
            >
              <MonitorSmartphone size={17} aria-hidden="true" />
              客户端
            </button>
          </nav>
        </aside>

        <main className="w-full px-4 py-5 sm:ml-56 sm:px-8 sm:py-9" id={view}>
          <nav aria-label="主导航" className="mb-6 grid grid-cols-3 gap-2 sm:hidden">
            <button className={view === "overview" ? "flex h-10 items-center justify-center gap-2 rounded-lg bg-accent/10 text-sm font-medium text-accent" : "flex h-10 items-center justify-center gap-2 rounded-lg bg-surface text-sm text-muted-foreground"} onClick={() => setView("overview")} type="button">
              <LayoutDashboard size={16} aria-hidden="true" />概览
            </button>
            <button className={view === "clients" ? "flex h-10 items-center justify-center gap-2 rounded-lg bg-accent/10 text-sm font-medium text-accent" : "flex h-10 items-center justify-center gap-2 rounded-lg bg-surface text-sm text-muted-foreground"} onClick={() => setView("clients")} type="button">
              <MonitorSmartphone size={16} aria-hidden="true" />客户端
            </button>
            <button className={view === "operations" ? "flex h-10 items-center justify-center gap-2 rounded-lg bg-accent/10 text-sm font-medium text-accent" : "flex h-10 items-center justify-center gap-2 rounded-lg bg-surface text-sm text-muted-foreground"} onClick={() => setView("operations")} type="button"><Activity size={16} aria-hidden="true" />运营</button>
          </nav>
          {view === "overview" ? <Dashboard /> : view === "clients" ? <ClientsPage csrfToken={csrfToken} /> : <OperationsPage />}
        </main>
      </div>
    </div>
  );
}

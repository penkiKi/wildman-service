import { useCallback, useEffect, useState } from "react";
import { LoginPage } from "./components/auth/LoginPage";
import { AccountDevicePage } from "./components/account/AccountDevicePage";
import { SetupPage } from "./components/auth/SetupPage";
import { AppShell } from "./components/layout/AppShell";
import { BootstrapState } from "./components/system/BootstrapState";
import { APIError, api, csrfTokenFromCookie, type User } from "./lib/api";

type AppState =
  | { view: "loading" }
  | { view: "error"; message: string }
  | { view: "setup" }
  | { view: "login" }
  | { view: "app"; user: User; csrfToken: string };

export default function App() {
  if (window.location.pathname === "/account/device") {
    return <AccountDevicePage />;
  }
  return <OperatorApp />;
}

function OperatorApp() {
  const [state, setState] = useState<AppState>({ view: "loading" });

  const bootstrap = useCallback(async () => {
    setState({ view: "loading" });
    try {
      const setup = await api.setupStatus();
      if (setup.required) {
        setState({ view: "setup" });
        return;
      }

      try {
        const { user } = await api.me();
        setState({ view: "app", user, csrfToken: csrfTokenFromCookie() });
      } catch (requestError) {
        if (requestError instanceof APIError && requestError.status === 401) {
          setState({ view: "login" });
          return;
        }
        throw requestError;
      }
    } catch (requestError) {
      setState({
        view: "error",
        message: requestError instanceof Error ? requestError.message : "无法连接服务",
      });
    }
  }, []);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  if (state.view === "loading") {
    return <BootstrapState state="loading" />;
  }
  if (state.view === "error") {
    return <BootstrapState state="error" message={state.message} onRetry={() => void bootstrap()} />;
  }
  if (state.view === "setup") {
    return <SetupPage onComplete={() => setState({ view: "login" })} />;
  }
  if (state.view === "login") {
    return (
      <LoginPage
        onLogin={(user, csrfToken) => setState({ view: "app", user, csrfToken })}
      />
    );
  }

  return (
    <AppShell
      csrfToken={state.csrfToken}
      onLogout={() => setState({ view: "login" })}
      user={state.user}
    />
  );
}

import { useState, type FormEvent } from "react";
import { api, type User } from "../../lib/api";
import { AuthPanel } from "./AuthPanel";
import { FormField } from "./FormField";

type LoginPageProps = {
  onLogin: (user: User, csrfToken: string) => void;
};

export function LoginPage({ onLogin }: LoginPageProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const response = await api.login(username, password);
      onLogin(response.user, response.csrfToken);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "登录失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthPanel title="登录">
      <form className="grid gap-5" onSubmit={submit}>
        <FormField
          id="login-username"
          label="用户名"
          autoComplete="username"
          required
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />
        <FormField
          id="login-password"
          label="密码"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <button
          className="mt-1 h-11 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={submitting}
          type="submit"
        >
          {submitting ? "正在登录" : "登录"}
        </button>
      </form>
    </AuthPanel>
  );
}

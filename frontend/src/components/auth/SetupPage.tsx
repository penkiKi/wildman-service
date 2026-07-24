import { useState, type FormEvent } from "react";
import { api } from "../../lib/api";
import { AuthPanel } from "./AuthPanel";
import { FormField } from "./FormField";

type SetupPageProps = {
  onComplete: () => void;
};

export function SetupPage({ onComplete }: SetupPageProps) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (password !== confirmation) {
      setError("两次输入的密码不一致");
      return;
    }
    setSubmitting(true);
    try {
      await api.setupAdmin(username, password);
      onComplete();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "初始化失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthPanel title="初始化管理员">
      <form className="grid gap-5" onSubmit={submit}>
        <FormField
          id="setup-username"
          label="用户名"
          autoComplete="username"
          minLength={3}
          maxLength={32}
          required
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />
        <FormField
          id="setup-password"
          label="密码"
          type="password"
          autoComplete="new-password"
          minLength={6}
          required
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        <FormField
          id="setup-password-confirmation"
          label="确认密码"
          type="password"
          autoComplete="new-password"
          minLength={6}
          required
          value={confirmation}
          onChange={(event) => setConfirmation(event.target.value)}
        />
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <button
          className="mt-1 h-11 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={submitting}
          type="submit"
        >
          {submitting ? "正在初始化" : "创建管理员"}
        </button>
      </form>
    </AuthPanel>
  );
}

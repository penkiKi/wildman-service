import {
  AlertTriangle,
  Check,
  Clipboard,
  KeyRound,
  LoaderCircle,
  Plus,
  RefreshCw,
  ShieldOff,
  X,
} from "lucide-react";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { api, type ClientInstallation } from "../../lib/api";

type ClientsPageProps = {
  csrfToken: string;
};

function formatDate(value: string | null): string {
  if (!value) return "从未";
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

export function ClientsPage({ csrfToken }: ClientsPageProps) {
  const [clients, setClients] = useState<ClientInstallation[]>([]);
  const [name, setName] = useState("");
  const [issuedToken, setIssuedToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [revokingId, setRevokingId] = useState("");
  const [confirmingId, setConfirmingId] = useState("");
  const [error, setError] = useState("");

  const loadClients = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await api.clients();
      setClients(result.clients);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "无法读取客户端");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadClients();
  }, [loadClients]);

  async function createClient(event: FormEvent) {
    event.preventDefault();
    setCreating(true);
    setError("");
    try {
      const result = await api.createClient(name, csrfToken);
      setClients((current) => [result.client, ...current]);
      setIssuedToken(result.token);
      setCopied(false);
      setName("");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "创建客户端失败");
    } finally {
      setCreating(false);
    }
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(issuedToken);
      setCopied(true);
    } catch {
      setError("无法写入剪贴板，请手动复制 Token");
    }
  }

  async function revokeClient(clientId: string) {
    setRevokingId(clientId);
    setError("");
    try {
      const result = await api.revokeClient(clientId, csrfToken);
      setClients((current) =>
        current.map((client) => (client.id === clientId ? result.client : client)),
      );
      setConfirmingId("");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "撤销客户端失败");
    } finally {
      setRevokingId("");
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">客户端</h1>
          <p className="mt-1 text-sm text-muted-foreground">签发与撤销野人音乐安装凭证</p>
        </div>
        <button
          aria-label="刷新客户端列表"
          className="grid size-9 shrink-0 place-items-center rounded-lg border border-border bg-surface text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-60"
          disabled={loading}
          onClick={() => void loadClients()}
          type="button"
        >
          <RefreshCw className={loading ? "animate-spin" : ""} size={16} aria-hidden="true" />
        </button>
      </div>

      <form className="mt-8 flex flex-col gap-3 rounded-xl border border-border bg-surface p-5 sm:flex-row sm:items-end" onSubmit={createClient}>
        <label className="min-w-0 flex-1 text-sm font-medium text-foreground">
          客户端名称
          <input
            autoComplete="off"
            className="mt-2 h-10 w-full rounded-lg border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-accent"
            disabled={creating}
            maxLength={100}
            onChange={(event) => setName(event.target.value)}
            placeholder="例如：Alice NAS"
            required
            value={name}
          />
        </label>
        <button
          className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-primary-foreground transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={creating || name.trim().length === 0}
          type="submit"
        >
          {creating ? <LoaderCircle className="animate-spin" size={16} aria-hidden="true" /> : <Plus size={16} aria-hidden="true" />}
          创建客户端
        </button>
      </form>

      {issuedToken ? (
        <section className="mt-4 rounded-xl border border-success/40 bg-success/5 p-5" aria-live="polite">
          <div className="flex items-start gap-3">
            <KeyRound className="mt-0.5 shrink-0 text-success" size={18} aria-hidden="true" />
            <div className="min-w-0 flex-1">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="text-sm font-semibold text-foreground">客户端 Token 已签发</h2>
                  <p className="mt-1 text-xs leading-5 text-muted-foreground">请立即保存，关闭后无法再次查看。</p>
                </div>
                <button
                  aria-label="关闭一次性 Token"
                  className="grid size-8 shrink-0 place-items-center rounded-lg text-muted-foreground transition hover:bg-muted hover:text-foreground"
                  onClick={() => setIssuedToken("")}
                  type="button"
                >
                  <X size={16} aria-hidden="true" />
                </button>
              </div>
              <div className="mt-4 flex flex-col gap-2 sm:flex-row">
                <code className="min-w-0 flex-1 select-all overflow-x-auto rounded-lg border border-border bg-background px-3 py-2.5 text-xs text-foreground">
                  {issuedToken}
                </code>
                <button
                  className="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-lg border border-border bg-surface px-3 text-sm text-foreground transition hover:bg-muted"
                  onClick={() => void copyToken()}
                  type="button"
                >
                  {copied ? <Check size={16} aria-hidden="true" /> : <Clipboard size={16} aria-hidden="true" />}
                  {copied ? "已复制" : "复制"}
                </button>
              </div>
            </div>
          </div>
        </section>
      ) : null}

      {error ? (
        <div className="mt-4 flex items-start gap-2 rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive" role="alert">
          <AlertTriangle className="mt-0.5 shrink-0" size={16} aria-hidden="true" />
          <span>{error}</span>
        </div>
      ) : null}

      <section className="mt-6 overflow-hidden rounded-xl border border-border bg-surface" aria-label="客户端列表">
        {loading && clients.length === 0 ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground" role="status">
            <LoaderCircle className="mr-2 animate-spin text-accent" size={17} aria-hidden="true" />
            正在读取
          </div>
        ) : clients.length === 0 ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">尚未创建客户端</div>
        ) : (
          <div className="divide-y divide-border">
            {clients.map((client) => (
              <article className="grid gap-4 p-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center" key={client.id}>
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="truncate text-sm font-semibold text-foreground">{client.name}</h2>
                    <span className={client.status === "active" ? "rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success" : "rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground"}>
                      {client.status === "active" ? "有效" : "已撤销"}
                    </span>
                  </div>
                  <dl className="mt-3 grid gap-x-6 gap-y-2 text-xs sm:grid-cols-3">
                    <div>
                      <dt className="text-muted-foreground">Token 前缀</dt>
                      <dd className="mt-1 font-mono text-foreground">{client.tokenPrefix}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">最近使用</dt>
                      <dd className="mt-1 text-foreground">{formatDate(client.lastSeenAt)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">创建时间</dt>
                      <dd className="mt-1 text-foreground">{formatDate(client.createdAt)}</dd>
                    </div>
                  </dl>
                </div>
                {client.status === "active" ? (
                  confirmingId === client.id ? (
                    <div className="flex items-center gap-2">
                      <button className="h-9 rounded-lg px-3 text-sm text-muted-foreground transition hover:bg-muted hover:text-foreground" disabled={revokingId === client.id} onClick={() => setConfirmingId("")} type="button">取消</button>
                      <button className="inline-flex h-9 items-center gap-2 rounded-lg bg-destructive px-3 text-sm font-medium text-white transition hover:brightness-110 disabled:opacity-60" disabled={revokingId === client.id} onClick={() => void revokeClient(client.id)} type="button">
                        {revokingId === client.id ? <LoaderCircle className="animate-spin" size={15} aria-hidden="true" /> : null}
                        确认撤销
                      </button>
                    </div>
                  ) : (
                    <button className="inline-flex h-9 w-fit items-center gap-2 rounded-lg border border-border px-3 text-sm text-muted-foreground transition hover:border-destructive/40 hover:bg-destructive/5 hover:text-destructive" onClick={() => setConfirmingId(client.id)} type="button">
                      <ShieldOff size={15} aria-hidden="true" />
                      撤销
                    </button>
                  )
                ) : (
                  <span className="text-xs text-muted-foreground">撤销于 {formatDate(client.revokedAt)}</span>
                )}
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

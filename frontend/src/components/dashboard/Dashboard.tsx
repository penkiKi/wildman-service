import {
  AlertTriangle,
  CheckCircle2,
  Database,
  LoaderCircle,
  RefreshCw,
  type LucideIcon,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { api, type Readiness, type ReadinessCheck } from "../../lib/api";

type CheckDefinition = {
  checkKey: keyof Readiness["checks"];
  label: string;
  icon: LucideIcon;
};

const checkDefinitions: CheckDefinition[] = [
  { checkKey: "database", label: "数据库", icon: Database },
];

function CheckCard({ label, icon: Icon, check }: CheckDefinition & { check: ReadinessCheck }) {
  const healthy = check.status === "ok";

  return (
    <article className="rounded-xl border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="grid size-9 place-items-center rounded-lg bg-muted text-muted-foreground">
          <Icon size={18} aria-hidden="true" />
        </div>
        <span
          className={
            healthy
              ? "inline-flex items-center gap-1.5 text-xs font-medium text-success"
              : "inline-flex items-center gap-1.5 text-xs font-medium text-destructive"
          }
        >
          {healthy ? <CheckCircle2 size={14} aria-hidden="true" /> : <AlertTriangle size={14} aria-hidden="true" />}
          {healthy ? "正常" : "异常"}
        </span>
      </div>
      <h2 className="mt-5 text-sm font-medium text-foreground">{label}</h2>
      {check.message ? <p className="mt-2 break-words text-xs leading-5 text-muted-foreground">{check.message}</p> : null}
    </article>
  );
}

export function Dashboard() {
  const [readiness, setReadiness] = useState<Readiness | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const loadReadiness = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setReadiness(await api.readiness());
    } catch (requestError) {
      setReadiness(null);
      setError(requestError instanceof Error ? requestError.message : "无法读取服务状态");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadReadiness();
  }, [loadReadiness]);

  return (
    <div className="mx-auto w-full max-w-6xl">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">概览</h1>
          <p className="mt-1 text-sm text-muted-foreground">服务运行状态</p>
        </div>
        <button
          aria-label="刷新服务状态"
          className="grid size-9 place-items-center rounded-lg border border-border bg-surface text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-60"
          disabled={loading}
          onClick={() => void loadReadiness()}
          type="button"
        >
          <RefreshCw className={loading ? "animate-spin" : ""} size={16} aria-hidden="true" />
        </button>
      </div>

      {loading && !readiness ? (
        <div className="mt-8 flex h-40 items-center justify-center rounded-xl border border-border bg-surface text-sm text-muted-foreground" role="status">
          <LoaderCircle className="mr-2 animate-spin text-accent" size={17} aria-hidden="true" />
          正在检查
        </div>
      ) : error ? (
        <div className="mt-8 rounded-xl border border-destructive/30 bg-destructive/5 p-5">
          <div className="flex items-center gap-2 text-sm font-medium text-destructive">
            <AlertTriangle size={17} aria-hidden="true" />
            状态读取失败
          </div>
          <p className="mt-2 text-sm text-muted-foreground">{error}</p>
        </div>
      ) : readiness ? (
        <>
          <section className="mt-8 rounded-xl border border-border bg-surface p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <span className={readiness.status === "ready" ? "size-2 rounded-full bg-success shadow-[0_0_12px_var(--color-success)]" : "size-2 rounded-full bg-destructive"} />
                <p className="text-sm font-medium text-foreground">
                  {readiness.status === "ready" ? "服务已就绪" : "服务未就绪"}
                </p>
              </div>
              <time className="text-xs text-muted-foreground" dateTime={readiness.time}>
                {new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "medium" }).format(new Date(readiness.time))}
              </time>
            </div>
          </section>

          <section className="mt-4 grid gap-4 md:grid-cols-2" aria-label="服务检查项">
            {checkDefinitions.map((definition) => (
              <CheckCard
                key={definition.checkKey}
                {...definition}
                check={readiness.checks[definition.checkKey]}
              />
            ))}
          </section>
        </>
      ) : null}
    </div>
  );
}

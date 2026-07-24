import { AlertTriangle, DatabaseZap, LoaderCircle, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { api, type ProviderSummary, type ResolutionSummary } from "../../lib/api";

function formatDate(value: string | null) {
  return value ? new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value)) : "—";
}

export function OperationsPage() {
  const [resolutions, setResolutions] = useState<ResolutionSummary[]>([]);
  const [provider, setProvider] = useState<ProviderSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [resolutionResult, providerResult] = await Promise.all([api.operationResolutions(), api.operationProvider()]);
      setResolutions(resolutionResult.resolutions);
      setProvider(providerResult);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "无法读取运营状态");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const hitRate = provider && provider.metrics.cacheLookups > 0
    ? `${Math.round((provider.metrics.cacheHits / provider.metrics.cacheLookups) * 100)}%`
    : "—";

  return (
    <div className="mx-auto w-full max-w-6xl">
      <div className="flex items-start justify-between gap-4">
        <div><h1 className="text-xl font-semibold tracking-tight text-foreground">运营</h1><p className="mt-1 text-sm text-muted-foreground">解析任务与 Provider 状态</p></div>
        <button aria-label="刷新运营状态" className="grid size-9 place-items-center rounded-lg border border-border bg-surface text-muted-foreground hover:bg-muted disabled:opacity-60" disabled={loading} onClick={() => void load()} type="button">
          <RefreshCw className={loading ? "animate-spin" : ""} size={16} aria-hidden="true" />
        </button>
      </div>
      {error ? <div className="mt-6 flex gap-2 rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"><AlertTriangle size={16} />{error}</div> : null}
      {provider ? (
        <section className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4" aria-label="Provider 指标">
          <article className="rounded-xl border border-border bg-surface p-5"><p className="text-xs text-muted-foreground">Provider</p><p className="mt-2 text-sm font-semibold text-foreground">{provider.name}</p><p className={provider.configured ? "mt-2 text-xs text-success" : "mt-2 text-xs text-destructive"}>{provider.configured ? "已配置" : "未配置"}</p></article>
          <article className="rounded-xl border border-border bg-surface p-5"><p className="text-xs text-muted-foreground">缓存命中率</p><p className="mt-2 text-xl font-semibold text-foreground">{hitRate}</p><p className="mt-2 text-xs text-muted-foreground">{provider.metrics.cacheHits} / {provider.metrics.cacheLookups}</p></article>
          <article className="rounded-xl border border-border bg-surface p-5"><p className="text-xs text-muted-foreground">Provider 请求</p><p className="mt-2 text-xl font-semibold text-foreground">{provider.metrics.providerRequests}</p><p className="mt-2 text-xs text-muted-foreground">合并 {provider.metrics.coalescedRequests}</p></article>
          <article className="rounded-xl border border-border bg-surface p-5"><p className="text-xs text-muted-foreground">错误</p><p className="mt-2 text-xl font-semibold text-foreground">{Object.values(provider.metrics.errors).reduce((sum, count) => sum + count, 0)}</p><p className="mt-2 text-xs text-muted-foreground">负缓存命中 {provider.metrics.negativeHits}</p></article>
        </section>
      ) : null}
      <section className="mt-6 overflow-hidden rounded-xl border border-border bg-surface" aria-label="解析任务">
        <div className="flex items-center gap-2 border-b border-border px-5 py-4"><DatabaseZap size={17} className="text-accent" /><h2 className="text-sm font-semibold">最近解析任务</h2></div>
        {loading && resolutions.length === 0 ? <div className="flex h-40 items-center justify-center text-sm text-muted-foreground"><LoaderCircle className="mr-2 animate-spin" size={16} />正在读取</div> : resolutions.length === 0 ? <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">暂无任务</div> : (
          <div className="divide-y divide-border">{resolutions.map((item) => <article className="grid gap-3 p-5 text-sm sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-center" key={item.id}><div className="min-w-0"><p className="truncate font-mono text-xs text-foreground">{item.id}</p><p className="mt-1 text-xs text-muted-foreground">{item.clientName}</p></div><span className="w-fit rounded-full bg-muted px-2 py-1 text-xs text-accent">{item.status}</span><time className="text-xs text-muted-foreground" dateTime={item.createdAt}>{formatDate(item.createdAt)}</time></article>)}</div>
        )}
      </section>
    </div>
  );
}

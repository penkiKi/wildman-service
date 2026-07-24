import { AlertTriangle, AudioLines, LoaderCircle } from "lucide-react";

type BootstrapStateProps =
  | { state: "loading" }
  | { state: "error"; message: string; onRetry: () => void };

export function BootstrapState(props: BootstrapStateProps) {
  return (
    <main className="grid min-h-screen place-items-center bg-background px-5 text-foreground">
      <section className="w-full max-w-sm rounded-2xl border border-border bg-surface p-7 shadow-2xl shadow-black/30">
        <div className="mb-6 grid size-10 place-items-center rounded-xl border border-primary/30 bg-primary/10 text-primary">
          <AudioLines size={20} aria-hidden="true" />
        </div>
        {props.state === "loading" ? (
          <div className="flex items-center gap-3 text-sm text-muted-foreground" role="status">
            <LoaderCircle className="animate-spin text-accent" size={18} aria-hidden="true" />
            正在连接服务
          </div>
        ) : (
          <div>
            <div className="flex items-center gap-3 text-sm font-medium">
              <AlertTriangle className="text-destructive" size={18} aria-hidden="true" />
              无法载入服务
            </div>
            <p className="mt-3 text-sm leading-6 text-muted-foreground">{props.message}</p>
            <button
              className="mt-6 h-10 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition hover:bg-primary/90"
              onClick={props.onRetry}
              type="button"
            >
              重试
            </button>
          </div>
        )}
      </section>
    </main>
  );
}

import { AudioLines } from "lucide-react";
import type { ReactNode } from "react";

type AuthPanelProps = {
  title: string;
  children: ReactNode;
};

export function AuthPanel({ title, children }: AuthPanelProps) {
  return (
    <main className="relative grid min-h-screen place-items-center overflow-hidden bg-background px-5 py-10 text-foreground">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_10%,color-mix(in_srgb,var(--accent)_15%,transparent),transparent_34%),radial-gradient(circle_at_85%_85%,color-mix(in_srgb,var(--primary)_12%,transparent),transparent_32%)]" />
      <section className="relative w-full max-w-md rounded-2xl border border-border bg-surface/85 p-7 shadow-2xl shadow-black/30 backdrop-blur-xl sm:p-9">
        <div className="mb-8 flex items-center gap-3">
          <div className="grid size-10 place-items-center rounded-xl border border-primary/30 bg-primary/10 text-primary">
            <AudioLines size={20} aria-hidden="true" />
          </div>
          <div>
            <p className="text-sm font-semibold tracking-[0.18em]">WILDMAN</p>
            <p className="text-xs text-muted-foreground">曲库维护服务</p>
          </div>
        </div>
        <h1 className="mb-6 text-2xl font-semibold tracking-tight">{title}</h1>
        {children}
      </section>
    </main>
  );
}

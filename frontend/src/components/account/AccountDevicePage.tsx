import { AudioLines, CheckCircle2, LoaderCircle } from "lucide-react";
import { useState, type FormEvent } from "react";
import { api } from "../../lib/api";

export function AccountDevicePage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState("");
  const [userCode, setUserCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [approved, setApproved] = useState(false);

  async function authenticate(register: boolean) {
    setLoading(true); setError("");
    try {
      const result = register ? await api.registerAccount(email, password) : await api.loginAccount(email, password);
      setToken(result.token); setPassword("");
    } catch (requestError) { setError(requestError instanceof Error ? requestError.message : "账户请求失败"); }
    finally { setLoading(false); }
  }

  async function approve(event: FormEvent) {
    event.preventDefault(); setLoading(true); setError("");
    try { await api.approveDevice(userCode, token); setApproved(true); setToken(""); }
    catch (requestError) { setError(requestError instanceof Error ? requestError.message : "设备批准失败"); }
    finally { setLoading(false); }
  }

  return (
    <main className="grid min-h-screen place-items-center bg-background px-4 py-10 text-foreground">
      <section className="w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-2xl sm:p-8">
        <div className="flex items-center gap-3"><span className="grid size-10 place-items-center rounded-xl bg-primary/10 text-primary"><AudioLines size={19} /></span><h1 className="text-lg font-semibold">设备授权</h1></div>
        {approved ? <div className="mt-8 flex items-center gap-3 rounded-xl border border-success/30 bg-success/5 p-4 text-sm text-success"><CheckCircle2 size={18} />设备已批准，可以返回野人音乐。</div> : token ? (
          <form className="mt-8" onSubmit={approve}>
            <label className="text-sm font-medium">用户码<input autoFocus className="mt-2 h-11 w-full rounded-lg border border-border bg-background px-3 font-mono uppercase tracking-widest" maxLength={16} onChange={(event) => setUserCode(event.target.value)} required value={userCode} /></label>
            <button className="mt-4 inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-primary font-semibold text-primary-foreground disabled:opacity-60" disabled={loading} type="submit">{loading ? <LoaderCircle className="animate-spin" size={16} /> : null}批准设备</button>
          </form>
        ) : (
          <div className="mt-8 space-y-4">
            <label className="block text-sm font-medium">邮箱<input autoComplete="email" className="mt-2 h-11 w-full rounded-lg border border-border bg-background px-3" onChange={(event) => setEmail(event.target.value)} required type="email" value={email} /></label>
            <label className="block text-sm font-medium">密码<input autoComplete="current-password" className="mt-2 h-11 w-full rounded-lg border border-border bg-background px-3" onChange={(event) => setPassword(event.target.value)} required type="password" value={password} /></label>
            <div className="grid grid-cols-2 gap-3"><button className="h-11 rounded-lg border border-border text-sm font-medium hover:bg-muted disabled:opacity-60" disabled={loading || !email || !password} onClick={() => void authenticate(false)} type="button">登录</button><button className="h-11 rounded-lg bg-primary text-sm font-semibold text-primary-foreground disabled:opacity-60" disabled={loading || !email || !password} onClick={() => void authenticate(true)} type="button">注册</button></div>
          </div>
        )}
        {error ? <p className="mt-4 text-sm text-destructive" role="alert">{error}</p> : null}
      </section>
    </main>
  );
}

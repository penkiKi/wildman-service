export type User = {
  id: string;
  username: string;
  status: string;
};

export type SetupStatus = {
  required: boolean;
};

export type ReadinessCheck = {
  status: "ok" | "error";
  message?: string;
};

export type Readiness = {
  status: "ready" | "not_ready";
  checks: {
    database: ReadinessCheck;
    dataDirectory: ReadinessCheck;
  };
  time: string;
};

export type ClientInstallation = {
  id: string;
  name: string;
  tokenPrefix: string;
  status: "active" | "revoked";
  lastSeenAt: string | null;
  revokedAt: string | null;
  createdAt: string;
};

export type ResolutionSummary = {
  id: string;
  clientName: string;
  status: string;
  createdAt: string;
  finishedAt: string | null;
};

export type ProviderSummary = {
  name: string;
  configured: boolean;
  metrics: {
    cacheLookups: number;
    cacheHits: number;
    negativeHits: number;
    coalescedRequests: number;
    providerRequests: number;
    errors: Record<string, number>;
  };
};

type AccountAuthentication = { accountId: string; email: string; token: string };

type LoginResponse = {
  user: User;
  csrfToken: string;
};

type MeResponse = {
  user: User;
};

type APIErrorBody = {
  code?: string;
  message?: string;
};

type APIEnvelope<T> = {
  data: T;
  error: APIErrorBody | null;
  requestId: string;
};

export class APIError extends Error {
  status: number;
  code?: string;
  requestId?: string;

  constructor(status: number, error?: APIErrorBody, requestId?: string) {
    super(error?.message ?? "请求失败");
    this.name = "APIError";
    this.status = status;
    this.code = error?.code;
    this.requestId = requestId;
  }
}

async function request<T>(path: string, init?: RequestInit, acceptedStatuses: number[] = []): Promise<T> {
  const response = await fetch(path, {
    credentials: "same-origin",
    ...init,
    headers: {
      Accept: "application/json",
      ...init?.headers,
    },
  });
  if (response.status === 204) {
    return undefined as T;
  }
  let envelope: APIEnvelope<T> | undefined;
  try {
    envelope = (await response.json()) as APIEnvelope<T>;
  } catch {
    envelope = undefined;
  }
  if (!response.ok && !acceptedStatuses.includes(response.status)) {
    throw new APIError(response.status, envelope?.error ?? undefined, envelope?.requestId);
  }
  if (!envelope) {
    throw new APIError(response.status, { message: "服务返回了无效响应" });
  }
  return envelope.data;
}

function jsonRequest(body: unknown): RequestInit {
  return {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

export const api = {
  setupStatus: () => request<SetupStatus>("/api/v1/setup/status"),
  setupAdmin: (username: string, password: string) =>
    request<{ user: User }>("/api/v1/setup/admin", jsonRequest({ username, password })),
  login: (username: string, password: string) =>
    request<LoginResponse>("/api/v1/auth/login", jsonRequest({ username, password })),
  me: () => request<MeResponse>("/api/v1/auth/me"),
  logout: (csrfToken: string) =>
    request<void>("/api/v1/auth/logout", {
      method: "POST",
      headers: { "X-CSRF-Token": csrfToken },
    }),
  readiness: () => request<Readiness>("/api/v1/ready", undefined, [503]),
  clients: () => request<{ clients: ClientInstallation[] }>("/api/v1/clients"),
  createClient: (name: string, csrfToken: string) =>
    request<{ client: ClientInstallation; token: string }>("/api/v1/clients", {
      ...jsonRequest({ name }),
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
    }),
  revokeClient: (clientId: string, csrfToken: string) =>
    request<{ client: ClientInstallation }>(`/api/v1/clients/${encodeURIComponent(clientId)}/revoke`, {
      method: "POST",
      headers: { "X-CSRF-Token": csrfToken },
    }),
  operationResolutions: () => request<{ resolutions: ResolutionSummary[] }>("/api/v1/operations/resolutions"),
  operationProvider: () => request<ProviderSummary>("/api/v1/operations/provider"),
  registerAccount: (email: string, password: string) =>
    request<AccountAuthentication>("/api/v1/accounts/register", jsonRequest({ email, password })),
  loginAccount: (email: string, password: string) =>
    request<AccountAuthentication>("/api/v1/accounts/login", jsonRequest({ email, password })),
  approveDevice: (userCode: string, token: string) =>
    request<void>("/api/v1/account/device/approve", {
      ...jsonRequest({ userCode }),
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    }),
};

export function csrfTokenFromCookie(): string {
  const prefix = "wildman_csrf=";
  const value = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix));
  return value ? decodeURIComponent(value.slice(prefix.length)) : "";
}

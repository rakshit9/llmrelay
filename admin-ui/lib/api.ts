const BASE = process.env.NEXT_PUBLIC_ADMIN_API_URL ?? "http://localhost:8001";

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export type Project = { id: number; name: string; created_at: string };
export type APIKey = {
  id: number; name: string; project_id: number;
  rate_limit_rpm: number; budget_usd: string | null;
  spent_usd: string; is_active: boolean; created_at: string;
  key?: string;
};
export type UsageStats = {
  total_requests: number; total_tokens: number;
  total_cost_usd: string; cache_hit_rate: number;
  requests_by_model: Record<string, number>;
  requests_by_provider: Record<string, number>;
};

export const api = {
  projects: {
    list: () => req<Project[]>("/projects/"),
    create: (name: string) => req<Project>("/projects/", { method: "POST", body: JSON.stringify({ name }) }),
    delete: (id: number) => req<void>(`/projects/${id}`, { method: "DELETE" }),
  },
  keys: {
    list: (project_id?: number) => req<APIKey[]>(`/keys/${project_id ? `?project_id=${project_id}` : ""}`),
    create: (data: { name: string; project_id: number; rate_limit_rpm: number; budget_usd?: string }) =>
      req<APIKey>("/keys/", { method: "POST", body: JSON.stringify(data) }),
    revoke: (id: number) => req<APIKey>(`/keys/${id}/revoke`, { method: "PATCH" }),
    delete: (id: number) => req<void>(`/keys/${id}`, { method: "DELETE" }),
  },
  analytics: {
    stats: () => req<UsageStats>("/analytics/stats"),
  },
};

import { getAccessToken, setAccessToken } from "@/lib/auth";
export const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
type Opt = { method?: string; body?: unknown; auth?: boolean };
export async function apiFetch(
  path: string,
  opt: Opt = {},
  retried = false,
): Promise<any> {
  const headers: Record<string, string> = {};
  if (opt.body) headers["Content-Type"] = "application/json";
  if (opt.auth && getAccessToken())
    headers.Authorization = `Bearer ${getAccessToken()}`;
  const r = await fetch(API_BASE + path, {
    method: opt.method ?? "GET",
    headers,
    body: opt.body ? JSON.stringify(opt.body) : undefined,
    credentials: "include",
  });
  if (r.status === 401 && opt.auth && !retried) {
    const refresh = await fetch(API_BASE + "/client/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    if (refresh.ok) {
      setAccessToken((await refresh.json()).access_token);
      return apiFetch(path, opt, true);
    }
  }
  const data = r.status === 204 ? null : await r.json().catch(() => null);
  if (!r.ok) throw new Error(data?.error?.message ?? "خطا در ارتباط با سرور");
  return data;
}

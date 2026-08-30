import { apiFetch } from "@/lib/api";

export type User = {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  phone_number: string;
  national_id: string | null;
  account_type: "individual" | "company";
  status: "active" | "disabled";
  created_at: string;
};
export type ListParams = {
  first_name?: string;
  last_name?: string;
  phone?: string;
  status?: "active" | "disabled";
  created_from?: string;
  created_to?: string;
  page: number;
  page_size: number;
};
export type ListResult = {
  users: User[];
  total: number;
  page: number;
  page_size: number;
};
export type UserPayload = {
  first_name: string;
  last_name: string;
  email: string;
  national_id: string;
  account_type: "individual" | "company";
};
export type CreatePayload = UserPayload & {
  phone_number: string;
  password: string;
};
function qs(p: ListParams) {
  return Object.entries(p)
    .filter(([, v]) => v !== undefined && v !== "")
    .map(
      ([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`,
    )
    .join("&");
}
export const listUsers = (p: ListParams) =>
  apiFetch(`/admin/users?${qs(p)}`, { auth: true }) as Promise<ListResult>;
export const getUser = (id: number) =>
  apiFetch(`/admin/users/${id}`, { auth: true }) as Promise<{ user: User }>;
export const createUser = (body: CreatePayload) =>
  apiFetch("/admin/users", { method: "POST", auth: true, body }) as Promise<{
    user: User;
  }>;
export const updateUser = (id: number, body: UserPayload) =>
  apiFetch(`/admin/users/${id}`, {
    method: "PATCH",
    auth: true,
    body,
  }) as Promise<{ user: User }>;
export const setStatus = (id: number, status: User["status"]) =>
  apiFetch(`/admin/users/${id}/status`, {
    method: "POST",
    auth: true,
    body: { status },
  }) as Promise<{ user: User }>;
export const resetPassword = (id: number, new_password: string) =>
  apiFetch(`/admin/users/${id}/reset-password`, {
    method: "POST",
    auth: true,
    body: { new_password },
  });
export const deleteUser = (id: number) =>
  apiFetch(`/admin/users/${id}`, { method: "DELETE", auth: true });

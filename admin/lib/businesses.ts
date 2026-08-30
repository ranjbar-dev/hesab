import { apiFetch } from "@/lib/api";
export type Role = "owner" | "admin" | "accountant" | "viewer";
export const roleLabels: Record<Role, string> = {
  owner: "مالک",
  admin: "مدیر",
  accountant: "حسابدار",
  viewer: "ناظر",
};
export type Business = {
  id: number;
  name: string;
  owner_user_id: number;
  created_at: string;
};
export type Member = {
  user_id: number;
  first_name: string;
  last_name: string;
  phone_number: string;
  role: Role;
  created_at: string;
};
export type Owner = {
  id: number;
  first_name: string;
  last_name: string;
  phone_number: string;
};
export type BusinessRow = Business & { member_count: number; owner: Owner };
export type Owned = {
  id: number;
  name: string;
  member_count: number;
  created_at: string;
};
export type Joined = {
  id: number;
  name: string;
  role: Role;
  owner_name: string;
  created_at: string;
};
export const listBusinesses = (p: {
  name?: string;
  page: number;
  page_size: number;
}) =>
  apiFetch(
    `/admin/businesses?${new URLSearchParams(Object.entries(p).filter(([, v]) => v !== undefined && v !== "") as [string, string][]).toString()}`,
    { auth: true },
  ) as Promise<{
    businesses: BusinessRow[];
    total: number;
    page: number;
    page_size: number;
  }>;
export const createBusiness = (b: { name: string; owner_user_id: number }) =>
  apiFetch("/admin/businesses", {
    method: "POST",
    auth: true,
    body: b,
  }) as Promise<{ business: Business }>;
export const getBusiness = (id: number) =>
  apiFetch(`/admin/businesses/${id}`, { auth: true }) as Promise<{
    business: Business;
    owner: Owner;
    members: Member[];
  }>;
export const renameBusiness = (id: number, b: { name: string }) =>
  apiFetch(`/admin/businesses/${id}`, {
    method: "PATCH",
    auth: true,
    body: b,
  }) as Promise<{ business: Business }>;
export const deleteBusiness = (id: number) =>
  apiFetch(`/admin/businesses/${id}`, { method: "DELETE", auth: true });
export const addBusinessMember = (
  id: number,
  b: { phone_number: string; role: Exclude<Role, "owner"> },
) =>
  apiFetch(`/admin/businesses/${id}/members`, {
    method: "POST",
    auth: true,
    body: b,
  });
export const changeBusinessMemberRole = (
  id: number,
  userId: number,
  b: { role: Exclude<Role, "owner"> },
) =>
  apiFetch(`/admin/businesses/${id}/members/${userId}`, {
    method: "PATCH",
    auth: true,
    body: b,
  });
export const removeBusinessMember = (id: number, userId: number) =>
  apiFetch(`/admin/businesses/${id}/members/${userId}`, {
    method: "DELETE",
    auth: true,
  });
export const getUserBusinesses = (id: number) =>
  apiFetch(`/admin/users/${id}/businesses`, { auth: true }) as Promise<{
    owned: Owned[];
    joined: Joined[];
  }>;

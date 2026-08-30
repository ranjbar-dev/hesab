import { apiFetch } from "@/lib/api";
export type Role = "owner" | "admin" | "accountant" | "viewer";
export const roleLabels: Record<Role, string> = {
  owner: "مالک",
  admin: "مدیر",
  accountant: "حسابدار",
  viewer: "ناظر",
};
export const canManage = (r: Role) => r === "owner" || r === "admin";
export type Business = {
  id: number;
  name: string;
  owner_user_id: number;
  created_at: string;
};
export type BusinessRow = Business & { role: Role };
export type Member = {
  user_id: number;
  first_name: string;
  last_name: string;
  phone_number: string;
  role: Role;
  created_at: string;
};
export type Invite = {
  id: number;
  business_id: number;
  business_name: string;
  role: Role;
  invited_by_name: string;
  created_at: string;
};
export type OutgoingInvite = {
  id: number;
  business_id: number;
  user_id: number;
  first_name: string;
  last_name: string;
  phone_number: string;
  role: Role;
  created_at: string;
};
export const listBusinesses = () =>
  apiFetch("/client/businesses", { auth: true }) as Promise<{
    businesses: BusinessRow[];
  }>;
export const createBusiness = (b: { name: string }) =>
  apiFetch("/client/businesses", {
    method: "POST",
    auth: true,
    body: b,
  }) as Promise<{ business: Business }>;
export const getBusiness = (id: number) =>
  apiFetch(`/client/businesses/${id}`, { auth: true }) as Promise<{
    business: Business;
    role: Role;
  }>;
export const renameBusiness = (id: number, b: { name: string }) =>
  apiFetch(`/client/businesses/${id}`, {
    method: "PATCH",
    auth: true,
    body: b,
  }) as Promise<{ business: Business }>;
export const deleteBusiness = (id: number) =>
  apiFetch(`/client/businesses/${id}`, { method: "DELETE", auth: true });
export const listMembers = (id: number) =>
  apiFetch(`/client/businesses/${id}/members`, { auth: true }) as Promise<{
    members: Member[];
    role: Role;
  }>;
export const inviteMember = (
  id: number,
  b: { phone_number: string; role: Exclude<Role, "owner"> },
) =>
  apiFetch(`/client/businesses/${id}/members`, {
    method: "POST",
    auth: true,
    body: b,
  });
export const removeMember = (id: number, userId: number) =>
  apiFetch(`/client/businesses/${id}/members/${userId}`, {
    method: "DELETE",
    auth: true,
  });
export const changeMemberRole = (
  id: number,
  userId: number,
  b: { role: Exclude<Role, "owner"> },
) =>
  apiFetch(`/client/businesses/${id}/members/${userId}`, {
    method: "PATCH",
    auth: true,
    body: b,
  });
export const listOutgoingInvites = (id: number) =>
  apiFetch(`/client/businesses/${id}/invites`, { auth: true }) as Promise<{
    invites: OutgoingInvite[];
  }>;
export const cancelInvite = (id: number, i: number) =>
  apiFetch(`/client/businesses/${id}/invites/${i}`, {
    method: "DELETE",
    auth: true,
  });
export const listMyInvites = () =>
  apiFetch("/client/invites", { auth: true }) as Promise<{ invites: Invite[] }>;
export const acceptInvite = (id: number) =>
  apiFetch(`/client/invites/${id}/accept`, { method: "POST", auth: true });
export const rejectInvite = (id: number) =>
  apiFetch(`/client/invites/${id}/reject`, { method: "POST", auth: true });

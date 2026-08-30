import { apiFetch, API_BASE } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
export const updateProfile = (body: {
  first_name: string;
  last_name: string;
  email: string;
  phone_number: string;
  is_male: boolean;
}) => apiFetch("/admin/me", { method: "PATCH", auth: true, body });
export async function uploadAvatar(file: File) {
  const form = new FormData();
  form.append("file", file);
  const response = await fetch(`${API_BASE}/admin/me/avatar`, {
    method: "POST",
    headers: { Authorization: `Bearer ${getAccessToken()}` },
    body: form,
    credentials: "include",
  });
  const data = await response.json().catch(() => null);
  if (!response.ok)
    throw new Error(data?.error?.message ?? "خطا در بارگذاری تصویر");
  return data as { avatar_url: string };
}
export const deleteAvatar = () =>
  apiFetch("/admin/me/avatar", { method: "DELETE", auth: true });

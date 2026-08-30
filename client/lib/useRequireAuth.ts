"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
export type User = {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  phone_number: string;
  two_fa_enabled: boolean;
  created_at: string;
};
export function useRequireAuth() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const reload = async () => {
    try {
      setUser((await apiFetch("/client/me", { auth: true })).user);
    } catch {
      router.replace("/login");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void reload();
  }, []);
  return { user, loading, reload };
}

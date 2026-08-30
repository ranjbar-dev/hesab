"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useRequireAuth } from "@/lib/useRequireAuth";
export default function Dashboard() {
  const r = useRouter(),
    { loading } = useRequireAuth();
  useEffect(() => {
    if (!loading) r.replace("/select-business");
  }, [loading, r]);
  return (
    <main className="grid min-h-screen place-items-center text-brand-muted">
      در حال بارگذاری…
    </main>
  );
}

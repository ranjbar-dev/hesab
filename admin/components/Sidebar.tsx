"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { createContext, useContext, useEffect, useState } from "react";
import { API_BASE, apiFetch } from "@/lib/api";
import { clearSession } from "@/lib/auth";
import { useRequireAuth, type Admin } from "@/lib/useRequireAuth";

type AdminValue = { admin: Admin; reload: () => Promise<void> };
const AdminContext = createContext<AdminValue | null>(null);
export function useAdmin() {
  const value = useContext(AdminContext);
  if (!value) throw new Error("useAdmin must be used inside Sidebar");
  return value;
}
const initials = (a: Admin) =>
  `${a.first_name[0] ?? ""}${a.last_name[0] ?? ""}`;
const nav = [
  {
    href: "/users",
    label: "کاربران",
    icon: (
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
    ),
  },
  {
    href: "/businesses",
    label: "کسب‌وکارها",
    icon: <path d="M3 21h18M5 21V8l7-5 7 5v13M9 21v-6h6v6" />,
  },
  {
    href: "/profile",
    label: "پروفایل",
    icon: (
      <>
        <circle cx="12" cy="8" r="4" />
        <path d="M4 21a8 8 0 0 1 16 0" />
      </>
    ),
  },
];
function Icon({ children }: { children: React.ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5 shrink-0"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    >
      {children}
    </svg>
  );
}
export default function Sidebar({ children }: { children: React.ReactNode }) {
  const { admin, loading, reload } = useRequireAuth(),
    path = usePathname(),
    router = useRouter();
  const [collapsed, setCollapsed] = useState(false),
    [mobileOpen, setMobileOpen] = useState(false);
  useEffect(() => {
    if (typeof window !== "undefined")
      setCollapsed(localStorage.getItem("admin_sidebar_collapsed") === "true");
  }, []);
  const toggle = () =>
    setCollapsed((v) => {
      localStorage.setItem("admin_sidebar_collapsed", String(!v));
      return !v;
    });
  if (loading || !admin)
    return (
      <main className="grid min-h-screen place-items-center text-brand-muted">
        در حال بارگذاری…
      </main>
    );
  const slim = collapsed,
    sidebarClass = `${slim ? "w-16" : "w-64"} ${mobileOpen ? "max-sm:translate-x-0" : "max-sm:translate-x-full"}`;
  async function logout() {
    try {
      await apiFetch("/admin/auth/logout", { method: "POST" });
    } finally {
      clearSession();
      router.replace("/login");
    }
  }
  return (
    <AdminContext.Provider value={{ admin, reload }}>
      <aside
        className={`fixed right-0 top-0 z-30 flex h-screen flex-col border-l border-brand-border bg-brand-surface transition-[width,transform] duration-200 ${sidebarClass}`}
      >
        <div
          className={`flex h-20 items-center border-b border-brand-border p-3 ${slim ? "justify-center" : "justify-between"}`}
        >
          <Link
            href="/"
            title="داشبورد"
            className="text-lg font-bold text-brand-accent"
          >
            حساب
          </Link>
          <button
            type="button"
            aria-label="جمع کردن منو"
            title="جمع کردن منو"
            onClick={toggle}
            className={`${slim ? "hidden" : ""} cursor-pointer rounded-lg p-2 text-brand-muted hover:bg-brand-bg`}
          >
            <Icon>
              <path d="m15 18-6-6 6-6" />
            </Icon>
          </button>
        </div>
        <div
          className={`flex items-center gap-3 border-b border-brand-border p-3 ${slim ? "justify-center" : ""}`}
        >
          {admin.avatar_url ? (
            <img
              src={`${API_BASE}${admin.avatar_url}`}
              alt="تصویر پروفایل"
              className="h-10 w-10 rounded-full object-cover"
              onError={(e) => {
                e.currentTarget.style.display = "none";
              }}
            />
          ) : (
            <span className="grid h-10 w-10 place-items-center rounded-full bg-brand-accent font-bold text-brand-bg">
              {initials(admin)}
            </span>
          )}
          {!slim && (
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold">
                {admin.first_name} {admin.last_name}
              </p>
              <p dir="ltr" className="text-xs text-brand-muted">
                {admin.phone_number}
              </p>
            </div>
          )}
        </div>
        <nav className="flex-1 space-y-1 p-2">
          {nav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              title={slim ? item.label : undefined}
              className={`flex h-11 items-center gap-3 rounded-lg px-3 text-sm transition-colors ${path === item.href || path.startsWith(`${item.href}/`) ? "bg-brand-accent text-brand-bg" : "text-brand-muted hover:bg-brand-bg hover:text-brand-text"} ${slim ? "justify-center px-0" : ""}`}
            >
              <Icon>{item.icon}</Icon>
              {!slim && item.label}
            </Link>
          ))}
        </nav>
        <div className="space-y-1 border-t border-brand-border p-2">
          <Link
            href="/settings/security"
            title={slim ? "تنظیمات امنیتی" : undefined}
            className={`flex h-11 items-center gap-3 rounded-lg px-3 text-sm ${path.startsWith("/settings") ? "bg-brand-bg text-brand-text" : "text-brand-muted hover:bg-brand-bg"} ${slim ? "justify-center px-0" : ""}`}
          >
            <Icon>
              <path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06-2 2-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V20h-2.82v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06-2-2 .06-.06A1.65 1.65 0 0 0 7.6 15a1.65 1.65 0 0 0-1.51-1H6v-2.82h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06 2-2 .06.06a1.65 1.65 0 0 0 1.82.33 1.65 1.65 0 0 0 1-1.51V5h2.82v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06 2 2-.06.06a1.65 1.65 0 0 0-.33 1.82 1.65 1.65 0 0 0 1.51 1H20v2.82h-.09a1.65 1.65 0 0 0-.51 1.09Z" />
            </Icon>
            {!slim && "تنظیمات امنیتی"}
          </Link>
          <button
            onClick={logout}
            title={slim ? "خروج" : undefined}
            className={`flex h-11 w-full cursor-pointer items-center gap-3 rounded-lg px-3 text-sm text-brand-muted hover:bg-brand-bg hover:text-brand-text ${slim ? "justify-center px-0" : ""}`}
          >
            <Icon>
              <path d="M10 17l5-5-5-5M15 12H3M21 19V5a2 2 0 0 0-2-2h-6" />
            </Icon>
            {!slim && "خروج"}
          </button>
          {slim && (
            <button
              onClick={toggle}
              aria-label="باز کردن منو"
              className="flex h-11 w-full cursor-pointer items-center justify-center rounded-lg text-brand-muted hover:bg-brand-bg"
            >
              <Icon>
                <path d="m9 18 6-6-6-6" />
              </Icon>
            </button>
          )}
        </div>
      </aside>
      <button
        onClick={() => setMobileOpen((v) => !v)}
        aria-label="منو"
        className="fixed right-3 top-3 z-20 hidden h-11 w-11 cursor-pointer place-items-center rounded-lg border border-brand-border bg-brand-surface text-brand-text max-sm:grid"
      >
        <Icon>
          <path d="M4 6h16M4 12h16M4 18h16" />
        </Icon>
      </button>
      <main
        className={`min-h-screen transition-[margin] duration-200 ${slim ? "mr-16" : "mr-64"} max-sm:mr-0`}
      >
        {children}
      </main>
    </AdminContext.Provider>
  );
}

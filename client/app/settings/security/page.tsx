"use client";
import { useState, type FormEvent } from "react";
import { apiFetch } from "@/lib/api";
import { useRequireAuth } from "@/lib/useRequireAuth";
import { Button, Field } from "../../login/page";
export default function Security() {
  const { user, loading, reload } = useRequireAuth(),
    [secret, setSecret] = useState(""),
    [url, setURL] = useState(""),
    [code, setCode] = useState(""),
    [password, setPassword] = useState(""),
    [message, setMessage] = useState("");
  if (loading)
    return (
      <main className="grid min-h-screen place-items-center">
        در حال بارگذاری…
      </main>
    );
  async function setup() {
    const d = await apiFetch("/client/2fa/setup", {
      method: "POST",
      auth: true,
    });
    setSecret(d.secret);
    setURL(d.otpauth_url);
  }
  async function activate(e: FormEvent) {
    e.preventDefault();
    await apiFetch("/client/2fa/activate", {
      method: "POST",
      auth: true,
      body: { secret, code },
    });
    setMessage("ورود دو مرحله‌ای فعال شد");
    await reload();
  }
  async function disable(e: FormEvent) {
    e.preventDefault();
    await apiFetch("/client/2fa/disable", {
      method: "POST",
      auth: true,
      body: { password },
    });
    setMessage("ورود دو مرحله‌ای غیرفعال شد");
    await reload();
  }
  return (
    <main className="mx-auto max-w-xl p-6 sm:p-12">
      <h1 className="text-2xl font-bold">تنظیمات امنیتی</h1>
      <section className="mt-8 rounded-2xl border border-brand-border bg-brand-surface p-6">
        {message && <p className="mb-4 text-green-700">{message}</p>}
        {!user?.two_fa_enabled && !secret && (
          <button
            onClick={setup}
            className="h-11 w-full cursor-pointer rounded-lg bg-brand-accent font-semibold text-white"
          >
            فعال‌سازی ورود دو مرحله‌ای
          </button>
        )}
        {!user?.two_fa_enabled && secret && (
          <form onSubmit={activate} className="space-y-4">
            <p className="text-sm text-brand-muted">
              این کلید را در برنامه احراز هویت وارد کنید.
            </p>
            <code className="block overflow-x-auto rounded bg-brand-bg p-3 text-brand-accent">
              {secret}
            </code>
            <a
              className="block break-all text-sm text-brand-accent underline"
              href={url}
            >
              {url}
            </a>
            <Field label="کد تأیید" value={code} set={setCode} />
            <Button>فعال‌سازی</Button>
          </form>
        )}
        {user?.two_fa_enabled && (
          <form onSubmit={disable} className="space-y-4">
            <p className="text-brand-muted">ورود دو مرحله‌ای فعال است.</p>
            <Field
              label="رمز عبور"
              value={password}
              set={setPassword}
              type="password"
            />
            <Button>غیرفعال‌سازی</Button>
          </form>
        )}
      </section>
    </main>
  );
}

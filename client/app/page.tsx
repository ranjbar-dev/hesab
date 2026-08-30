"use client";

import { useState, type FormEvent, type ReactNode } from "react";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    // ponytail: dummy submit, not wired to the API yet
    setLoading(true);
    setTimeout(() => setLoading(false), 900);
  }

  return (
    <main className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-sm">
        <div className="flex flex-col items-center text-center mb-8">
          <LogoMark />
          <h1 className="mt-4 text-2xl font-bold">خوش آمدید</h1>
          <p className="mt-1 text-sm text-brand-muted">
            برای مدیریت حساب‌داری خود وارد شوید.
          </p>
        </div>

        <div className="rounded-xl bg-brand-surface border border-brand-border p-6 sm:p-8">
          <form onSubmit={handleSubmit} className="space-y-5" noValidate>
            <Field
              id="email"
              label="ایمیل"
              type="email"
              value={email}
              onChange={setEmail}
              placeholder="you@example.com"
              autoComplete="email"
              icon={<MailIcon />}
            />
            <Field
              id="password"
              label="رمز عبور"
              type="password"
              value={password}
              onChange={setPassword}
              placeholder="••••••••"
              autoComplete="current-password"
              icon={<LockIcon />}
            />

            <div className="flex items-center justify-between text-sm">
              <label className="flex items-center gap-2 text-brand-muted cursor-pointer">
                <input
                  type="checkbox"
                  className="size-4 rounded border-brand-border accent-brand-accent"
                />
                مرا به خاطر بسپار
              </label>
              <a
                href="#"
                className="text-brand-accent hover:text-brand-accent-hover transition-colors"
              >
                فراموشی رمز عبور
              </a>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full h-11 rounded-lg bg-brand-accent text-white font-semibold hover:bg-brand-accent-hover transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-accent"
            >
              {loading ? "در حال ورود…" : "ورود"}
            </button>
          </form>
        </div>

        <p className="mt-6 text-center text-sm text-brand-muted">
          حساب کاربری ندارید؟{" "}
          <a
            href="#"
            className="text-brand-accent hover:text-brand-accent-hover transition-colors font-medium"
          >
            ثبت‌نام کنید
          </a>
        </p>
      </div>
    </main>
  );
}

function Field({
  id,
  label,
  type,
  value,
  onChange,
  placeholder,
  autoComplete,
  icon,
}: {
  id: string;
  label: string;
  type: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  autoComplete?: string;
  icon: ReactNode;
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm mb-2">
        {label}
      </label>
      <div className="relative">
        <span className="absolute inset-y-0 right-3 flex items-center text-brand-muted pointer-events-none">
          {icon}
        </span>
        <input
          id={id}
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          autoComplete={autoComplete}
          required
          className="w-full h-11 rounded-lg bg-brand-bg border border-brand-border pr-10 pl-3 text-sm placeholder:text-brand-muted/60 outline-none focus:border-brand-accent focus:ring-2 focus:ring-brand-accent/20 transition-colors"
        />
      </div>
    </div>
  );
}

function LogoMark() {
  return (
    <span className="inline-flex size-11 items-center justify-center rounded-xl bg-brand-accent text-white">
      <svg
        className="size-6"
        fill="none"
        viewBox="0 0 24 24"
        strokeWidth={2}
        stroke="currentColor"
        aria-hidden="true"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M3 13.5 8.25 8.25l3.75 3.75L21 3m0 0h-5.25M21 3v5.25"
        />
      </svg>
    </span>
  );
}

function MailIcon() {
  return (
    <svg
      className="size-5"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L3.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75"
      />
    </svg>
  );
}

function LockIcon() {
  return (
    <svg
      className="size-5"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={1.5}
      stroke="currentColor"
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"
      />
    </svg>
  );
}

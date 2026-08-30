"use client";
import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { setAccessToken, takePending } from "@/lib/auth";
import { Auth, Button, Field } from "../page";
export default function TwoFA() {
  const r = useRouter(),
    [pending, setPending] = useState(""),
    [code, setCode] = useState(""),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(false);
  useEffect(() => {
    const p = takePending();
    if (!p) r.replace("/login");
    else setPending(p);
  }, [r]);
  async function submit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      const d = await apiFetch("/client/auth/login/2fa", {
        method: "POST",
        body: { pending_token: pending, code },
      });
      setAccessToken(d.access_token);
      r.replace("/select-business");
    } catch (e) {
      setError(e instanceof Error ? e.message : "کد نامعتبر است");
    } finally {
      setLoading(false);
    }
  }
  return (
    <Auth title="تأیید ورود" text="کد شش‌رقمی برنامه احراز هویت را وارد کنید.">
      <form onSubmit={submit} className="space-y-5">
        <Field
          label="کد تأیید دو مرحله‌ای"
          value={code}
          set={setCode}
          auto="one-time-code"
        />
        {error && <p className="text-sm text-red-600">{error}</p>}
        <Button loading={loading}>تأیید و ورود</Button>
      </form>
    </Auth>
  );
}

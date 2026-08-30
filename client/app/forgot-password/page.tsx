"use client";
import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { Auth, Button, Field } from "../login/page";
export default function Forgot() {
  const r = useRouter(),
    [phone, setPhone] = useState(""),
    [error, setError] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    const p = phone.replace(/[\s()\-]/g, "").replace(/^\+98|^0098|^0/, "");
    if (!/^9\d{9}$/.test(p)) {
      setError("شماره موبایل معتبر نیست");
      return;
    }
    try {
      await apiFetch("/client/auth/forgot-password", {
        method: "POST",
        body: { phone_number: p },
      });
      r.push(`/reset-password?phone=${p}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "خطا");
    }
  }
  return (
    <Auth
      title="بازیابی رمز عبور"
      text="شماره موبایل حساب کاربری را وارد کنید."
    >
      <form onSubmit={submit} className="space-y-5">
        <Field label="شماره موبایل" value={phone} set={setPhone} />
        {error && <p className="text-sm text-red-600">{error}</p>}
        <Button>ارسال کد</Button>
      </form>
    </Auth>
  );
}

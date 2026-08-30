"use client";
import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { apiFetch } from "@/lib/api";
import { Auth, Button, Field } from "../login/page";
export default function Forgot() {
  const r = useRouter();
  const [phone, setPhone] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    const p = phone.replace(/[\s()\-]/g, "").replace(/^\+98|^0098|^0/, "");
    if (!/^9\d{9}$/.test(p)) {
      toast.error("شماره موبایل معتبر نیست");
      return;
    }
    try {
      await apiFetch("/admin/auth/forgot-password", {
        method: "POST",
        body: { phone_number: p },
      });
      toast.success("کد بازیابی پیامک شد");
      r.push(`/reset-password?phone=${p}`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  return (
    <Auth title="بازیابی رمز عبور" text="شماره موبایل مدیر را وارد کنید.">
      <form onSubmit={submit} className="space-y-5">
        <Field label="شماره موبایل" value={phone} set={setPhone} />
        <Button>ارسال کد</Button>
      </form>
    </Auth>
  );
}

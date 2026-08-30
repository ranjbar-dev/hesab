"use client";
import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useState, type FormEvent } from "react";
import { toast } from "sonner";
import { apiFetch } from "@/lib/api";
import { Auth, Button, Field } from "../login/page";
export default function Reset() {
  return (
    <Suspense
      fallback={
        <main className="grid min-h-screen place-items-center">
          در حال بارگذاری…
        </main>
      }
    >
      <ResetForm />
    </Suspense>
  );
}
function ResetForm() {
  const q = useSearchParams(),
    r = useRouter();
  const [phone, setPhone] = useState(q.get("phone") ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (password !== confirm) {
      toast.error("تکرار رمز عبور یکسان نیست");
      return;
    }
    try {
      await apiFetch("/admin/auth/reset-password", {
        method: "POST",
        body: { phone_number: phone, code, new_password: password },
      });
      toast.success("رمز عبور تغییر کرد");
      setTimeout(() => r.replace("/login"), 700);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  return (
    <Auth
      title="تنظیم رمز عبور"
      text="کد پیامک‌شده و رمز عبور جدید را وارد کنید."
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label="شماره موبایل" value={phone} set={setPhone} />
        <Field label="کد تأیید" value={code} set={setCode} />
        <Field
          label="رمز عبور جدید"
          value={password}
          set={setPassword}
          type="password"
        />
        <Field
          label="تکرار رمز عبور"
          value={confirm}
          set={setConfirm}
          type="password"
        />
        <Button>ثبت رمز جدید</Button>
      </form>
    </Auth>
  );
}

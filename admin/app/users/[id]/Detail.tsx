"use client";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { StatusBadge } from "@/app/users/page";
import { isoToJalaliLabel } from "@/lib/jalali";
import { deleteUser, getUser, resetPassword, setStatus, updateUser, type User } from "@/lib/users";
import { useRequireAuth } from "@/lib/useRequireAuth";

const input = "h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3 focus-visible:ring-2 focus-visible:ring-brand-accent/30";
const button = "h-11 cursor-pointer rounded-lg border border-brand-border px-4 transition-colors duration-200 hover:bg-brand-bg focus-visible:ring-2 focus-visible:ring-brand-accent/30";

function Info({ l, children }: { l: string; children: React.ReactNode }) {
  return <div><dt className="text-brand-muted">{l}</dt><dd>{children}</dd></div>;
}

export default function Detail() {
  const id = Number(useParams<{ id: string }>().id);
  const r = useRouter();
  const { loading: auth } = useRequireAuth();
  const [u, setU] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [p, setP] = useState("");

  useEffect(() => {
    if (!Number.isInteger(id) || id < 1) { r.replace("/users"); return; }
    getUser(id).then(x => setU(x.user)).catch(() => { toast.error("کاربر یافت نشد"); r.replace("/users"); }).finally(() => setLoading(false));
  }, [id, r]);

  if (auth || loading) return <main className="grid min-h-screen place-items-center text-brand-muted">در حال بارگذاری…</main>;
  if (!u) return null;

  const edit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    try {
      setU((await updateUser(u.id, {
        first_name: String(f.get("first_name")), last_name: String(f.get("last_name")),
        email: String(f.get("email")), national_id: String(f.get("national_id")),
        account_type: String(f.get("account_type")) as User["account_type"],
      })).user);
      toast.success("ذخیره شد");
    } catch (e) { toast.error(e instanceof Error ? e.message : "خطا"); }
  };
  const toggle = async () => {
    try {
      const status = u.status === "active" ? "disabled" : "active";
      setU((await setStatus(u.id, status)).user);
      toast.success("ذخیره شد");
    } catch (e) { toast.error(e instanceof Error ? e.message : "خطا"); }
  };
  const reset = async (e: React.FormEvent) => {
    e.preventDefault();
    try { await resetPassword(u.id, p); setP(""); toast.success("رمز عبور تغییر کرد"); }
    catch (e) { toast.error(e instanceof Error ? e.message : "خطا"); }
  };
  const remove = async () => {
    if (!window.confirm("این کاربر حذف شود؟ این کار قابل بازگشت نیست.")) return;
    try { await deleteUser(u.id); toast.success("کاربر حذف شد"); r.push("/users"); }
    catch (e) { toast.error(e instanceof Error ? e.message : "خطا"); }
  };

  return (
    <main className="mx-auto max-w-2xl p-6 sm:p-12">
      <Link href="/users" className="rounded-lg text-sm text-brand-muted focus-visible:ring-2 focus-visible:ring-brand-accent/30">→ بازگشت به فهرست</Link>
      <section className="mt-6 rounded-2xl border border-brand-border bg-brand-surface p-6">
        <div className="flex items-center gap-3"><h1 className="text-2xl font-bold">{u.first_name} {u.last_name}</h1><StatusBadge status={u.status} /></div>
        <dl className="mt-5 grid gap-3 text-sm sm:grid-cols-2">
          <Info l="موبایل"><span dir="ltr">{u.phone_number}</span><small className="block text-brand-muted">شماره موبایل قابل تغییر نیست</small></Info>
          <Info l="ایمیل">{u.email || "—"}</Info>
          <Info l="کد ملی">{u.national_id || "—"}</Info>
          <Info l="نوع حساب">{u.account_type === "company" ? "حقوقی" : "حقیقی"}</Info>
          <Info l="تاریخ ایجاد">{isoToJalaliLabel(u.created_at)}</Info>
        </dl>
      </section>

      <form onSubmit={edit} className="mt-5 space-y-3 rounded-2xl border border-brand-border bg-brand-surface p-6">
        <h2 className="font-bold">ویرایش مشخصات</h2>
        <input name="first_name" defaultValue={u.first_name} required className={input} aria-label="نام" />
        <input name="last_name" defaultValue={u.last_name} required className={input} aria-label="نام خانوادگی" />
        <input name="email" type="email" defaultValue={u.email} className={input} aria-label="ایمیل" />
        <input name="national_id" defaultValue={u.national_id ?? ""} className={input} aria-label="کد ملی" />
        <select name="account_type" defaultValue={u.account_type} className={input} aria-label="نوع حساب">
          <option value="individual">حقیقی</option><option value="company">حقوقی</option>
        </select>
        <button className="h-11 cursor-pointer rounded-lg bg-brand-accent px-5 font-semibold text-brand-bg transition-colors duration-200 hover:bg-brand-accent-hover focus-visible:ring-2 focus-visible:ring-brand-accent/30">ذخیره تغییرات</button>
      </form>

      <section className="mt-5 rounded-2xl border border-brand-border bg-brand-surface p-6">
        <h2 className="font-bold">عملیات</h2>
        <div className="mt-4 space-y-5">
          <button onClick={toggle} className={button}>{u.status === "active" ? "غیرفعال‌سازی حساب" : "فعال‌سازی حساب"}</button>
          <form onSubmit={reset} className="flex gap-3">
            <input required type="password" value={p} onChange={e => setP(e.target.value)} placeholder="رمز عبور جدید" className={input} aria-label="رمز عبور جدید" />
            <button className={button}>تغییر رمز عبور</button>
          </form>
          <button onClick={remove} className="cursor-pointer text-red-400 focus-visible:ring-2 focus-visible:ring-brand-accent/30">حذف کاربر</button>
        </div>
      </section>
    </main>
  );
}

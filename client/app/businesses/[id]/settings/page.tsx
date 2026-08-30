"use client";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { useBusiness } from "../Shell";
import { canManage, deleteBusiness, removeMember, renameBusiness } from "@/lib/businesses";
import { useRequireAuth } from "@/lib/useRequireAuth";

export default function Settings() {
  const router = useRouter();
  const { business, role, reload } = useBusiness();
  const { user } = useRequireAuth();
  const [name, setName] = useState(business.name);
  async function save(event: React.FormEvent) { event.preventDefault(); try { await renameBusiness(business.id, { name }); await reload(); toast.success("تنظیمات ذخیره شد"); } catch (error) { toast.error(error instanceof Error ? error.message : "خطا"); } }
  async function leave() { if (!confirm("از این کسب‌وکار خارج شوید؟")) return; try { await removeMember(business.id, user!.id); toast.success("از کسب‌وکار خارج شدید"); router.replace("/select-business"); } catch (error) { toast.error(error instanceof Error ? error.message : "خطا"); } }
  async function removeBusiness() { if (!confirm("این کسب‌وکار حذف شود؟ قابل بازگشت نیست.")) return; try { await deleteBusiness(business.id); toast.success("کسب‌وکار حذف شد"); router.replace("/select-business"); } catch (error) { toast.error(error instanceof Error ? error.message : "خطا"); } }
  return <section className="max-w-xl"><p className="text-sm text-brand-muted">مدیریت کسب‌وکار</p><h1 className="text-2xl font-bold">تنظیمات</h1>{canManage(role) && <form onSubmit={save} className="mt-6 rounded-2xl border border-brand-border bg-brand-surface p-6"><label className="block text-sm">نام کسب‌وکار<input value={name} onChange={event => setName(event.target.value)} className="mt-2 h-11 w-full rounded-lg border border-brand-border bg-brand-bg px-3" /></label><button className="mt-4 cursor-pointer rounded-lg bg-brand-accent px-4 py-2 font-bold text-white">ذخیره</button></form>}<section className="mt-5 rounded-2xl border border-red-200 bg-brand-surface p-6"><h2 className="font-bold text-red-700">ناحیه خطر</h2>{role === "owner" ? <button onClick={removeBusiness} className="mt-4 cursor-pointer rounded-lg border border-red-300 px-4 py-2 text-red-700">حذف کسب‌وکار</button> : <button onClick={leave} className="mt-4 cursor-pointer rounded-lg border border-red-300 px-4 py-2 text-red-700">خروج از کسب‌وکار</button>}</section></section>;
}

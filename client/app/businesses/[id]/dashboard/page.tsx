"use client";
import { useBusiness } from "../Shell";
export default function Dashboard() {
  const { business } = useBusiness();
  return (
    <section className="rounded-2xl border border-brand-border bg-brand-surface p-8">
      <p className="text-brand-muted">داشبورد حسابداری</p>
      <h1 className="mt-2 text-3xl font-bold">خوش آمدید به {business.name}</h1>
      <p className="mt-4 text-brand-muted">
        اطلاعات مالی این کسب‌وکار به‌زودی در اینجا نمایش داده می‌شود.
      </p>
    </section>
  );
}

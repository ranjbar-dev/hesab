"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  acceptInvite,
  listBusinesses,
  listMyInvites,
  rejectInvite,
  roleLabels,
  type BusinessRow,
  type Invite,
} from "@/lib/businesses";
import { useRequireAuth } from "@/lib/useRequireAuth";
export default function SelectBusiness() {
  const r = useRouter(),
    { loading: auth } = useRequireAuth(),
    [bs, setBs] = useState<BusinessRow[]>([]),
    [inv, setInv] = useState<Invite[]>([]),
    [loading, setLoading] = useState(true);
  const load = async () => {
    try {
      const [b, i] = await Promise.all([listBusinesses(), listMyInvites()]);
      setBs(b.businesses);
      setInv(i.invites);
      if (b.businesses.length === 1 && !i.invites.length)
        r.replace(`/businesses/${b.businesses[0].id}/dashboard`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  if (auth || loading)
    return (
      <main className="grid min-h-screen place-items-center text-brand-muted">
        در حال بارگذاری…
      </main>
    );
  async function respond(i: Invite, yes: boolean) {
    try {
      if (yes) await acceptInvite(i.id);
      else await rejectInvite(i.id);
      toast.success(yes ? "دعوت پذیرفته شد" : "دعوت رد شد");
      load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  return (
    <main className="mx-auto max-w-4xl p-6 sm:p-12">
      <header className="flex items-center justify-between">
        <div>
          <p className="text-sm text-brand-muted">حساب</p>
          <h1 className="text-3xl font-bold">انتخاب کسب‌وکار</h1>
        </div>
        <button
          onClick={() => r.push("/businesses/new")}
          className="cursor-pointer rounded-lg bg-brand-accent px-4 py-2 font-bold text-white"
        >
          ساخت کسب‌وکار جدید
        </button>
      </header>
      {!bs.length && !inv.length ? (
        <section className="mt-10 rounded-2xl border border-brand-border bg-brand-surface p-10 text-center">
          <h2 className="text-xl font-bold">شما هنوز کسب‌وکاری ندارید</h2>
          <button
            onClick={() => r.push("/businesses/new")}
            className="mt-5 cursor-pointer rounded-lg bg-brand-accent px-5 py-3 font-bold text-white"
          >
            ساخت کسب‌وکار
          </button>
        </section>
      ) : (
        <>
          <section className="mt-8 grid gap-3 sm:grid-cols-2">
            {bs.map((b) => (
              <button
                key={b.id}
                onClick={() => r.push(`/businesses/${b.id}/dashboard`)}
                className="cursor-pointer rounded-2xl border border-brand-border bg-brand-surface p-5 text-right transition-colors hover:border-brand-accent"
              >
                <h2 className="font-bold">{b.name}</h2>
                <p className="mt-2 text-sm text-brand-muted">
                  نقش شما: {roleLabels[b.role]}
                </p>
              </button>
            ))}
          </section>
          {inv.length > 0 && (
            <section className="mt-8 rounded-2xl border border-brand-border bg-brand-surface p-6">
              <h2 className="font-bold">دعوت‌ها</h2>
              {inv.map((i) => (
                <div
                  key={i.id}
                  className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-brand-border p-4"
                >
                  <span>
                    <b>{i.business_name}</b>
                    <small className="mr-2 text-brand-muted">
                      {roleLabels[i.role]} · دعوت از {i.invited_by_name}
                    </small>
                  </span>
                  <span className="flex gap-2">
                    <button
                      onClick={() => respond(i, true)}
                      className="cursor-pointer rounded-lg bg-brand-accent px-3 py-2 text-sm text-white"
                    >
                      پذیرفتن
                    </button>
                    <button
                      onClick={() => respond(i, false)}
                      className="cursor-pointer rounded-lg border border-brand-border px-3 py-2 text-sm"
                    >
                      رد کردن
                    </button>
                  </span>
                </div>
              ))}
            </section>
          )}
        </>
      )}
    </main>
  );
}

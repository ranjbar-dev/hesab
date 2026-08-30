"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAdmin } from "@/components/Sidebar";
import Modal from "@/components/Modal";
import { isoToJalaliLabel } from "@/lib/jalali";
import {
  createBusiness,
  listBusinesses,
  type BusinessRow,
} from "@/lib/businesses";
import { listUsers, type User } from "@/lib/users";
const field =
  "h-10 w-full rounded-lg border border-brand-border bg-brand-bg px-3 text-sm focus-visible:ring-2 focus-visible:ring-brand-accent/30";
function useDebounce(v: string) {
  const [d, setD] = useState(v);
  useEffect(() => {
    const t = setTimeout(() => setD(v), 300);
    return () => clearTimeout(t);
  }, [v]);
  return d;
}
function Create({
  open,
  close,
  done,
}: {
  open: boolean;
  close: () => void;
  done: () => void;
}) {
  const [name, setName] = useState(""),
    [phone, setPhone] = useState(""),
    [matches, setMatches] = useState<User[]>([]),
    [owner, setOwner] = useState<number>(),
    [busy, setBusy] = useState(false);
  async function search() {
    try {
      const r = await listUsers({ phone, page: 1, page_size: 5 });
      setMatches(r.users);
      if (r.users.length === 1) setOwner(r.users[0].id);
      else if (!r.users.length) toast.error("کاربری با این شماره یافت نشد");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!owner) {
      toast.error("مالک کسب‌وکار را انتخاب کنید");
      return;
    }
    setBusy(true);
    try {
      await createBusiness({ name, owner_user_id: owner });
      toast.success("کسب‌وکار ساخته شد");
      close();
      setName("");
      setPhone("");
      setMatches([]);
      setOwner(undefined);
      done();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open={open} onClose={close} title="کسب‌وکار جدید">
      <form onSubmit={submit} className="space-y-4">
        <label className="block text-sm">
          نام کسب‌وکار
          <input
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="مثال: فروشگاه رضایی"
            className={`mt-2 ${field}`}
          />
        </label>
        <div>
          <label className="block text-sm">
            شماره مالک
            <input
              required
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="۰۹۱۲۱۲۳۴۵۶۷"
              className={`mt-2 ${field}`}
              dir="ltr"
            />
          </label>
          <button
            type="button"
            onClick={search}
            className="mt-2 cursor-pointer text-sm text-brand-accent"
          >
            جست‌وجوی کاربر
          </button>
          {matches.length > 1 && (
            <div className="mt-2 space-y-1">
              {matches.map((u) => (
                <label
                  key={u.id}
                  className="flex cursor-pointer items-center gap-2 rounded-lg border border-brand-border p-2"
                >
                  <input
                    type="radio"
                    checked={owner === u.id}
                    onChange={() => setOwner(u.id)}
                  />
                  {u.first_name} {u.last_name}{" "}
                  <span dir="ltr" className="text-brand-muted">
                    {u.phone_number}
                  </span>
                </label>
              ))}
            </div>
          )}
        </div>
        <button
          disabled={busy}
          className="h-11 w-full cursor-pointer rounded-lg bg-brand-accent font-bold text-brand-bg disabled:opacity-60"
        >
          {busy ? "در حال انجام…" : "ساخت کسب‌وکار"}
        </button>
      </form>
    </Modal>
  );
}
export default function Businesses() {
  useAdmin();
  const r = useRouter(),
    [name, setName] = useState(""),
    d = useDebounce(name),
    [data, setData] = useState<{ businesses: BusinessRow[]; total: number }>({
      businesses: [],
      total: 0,
    }),
    [page, setPage] = useState(1),
    [loading, setLoading] = useState(true),
    [reload, setReload] = useState(0),
    [open, setOpen] = useState(false);
  useEffect(() => setPage(1), [d]);
  useEffect(() => {
    setLoading(true);
    listBusinesses({ name: d, page, page_size: 20 })
      .then(setData)
      .catch((e) => toast.error(e instanceof Error ? e.message : "خطا"))
      .finally(() => setLoading(false));
  }, [d, page, reload]);
  const pages = Math.max(1, Math.ceil(data.total / 20));
  return (
    <main className="mx-auto max-w-7xl p-6 sm:p-12">
      <header className="mb-8 flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-sm text-brand-muted">پنل مدیریت</p>
          <h1 className="text-3xl font-bold">کسب‌وکارها</h1>
          <Link
            href="/users"
            className="mt-2 inline-block text-sm text-brand-accent"
          >
            مدیریت کاربران
          </Link>
        </div>
        <button
          onClick={() => setOpen(true)}
          className="h-11 cursor-pointer rounded-lg bg-brand-accent px-4 font-bold text-brand-bg"
        >
          کسب‌وکار جدید
        </button>
      </header>
      <label className="mb-4 block max-w-sm text-sm text-brand-muted">
        جست‌وجو بر اساس نام
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className={`mt-2 ${field}`}
          placeholder="نام کسب‌وکار"
        />
      </label>
      <div
        className={`overflow-x-auto rounded-2xl border border-brand-border ${loading ? "opacity-60" : ""}`}
      >
        <table className="w-full min-w-[48rem] text-right text-sm">
          <thead className="bg-brand-surface text-brand-muted">
            <tr>
              <th className="p-4">نام</th>
              <th className="p-4">مالک</th>
              <th className="p-4">تعداد اعضا</th>
              <th className="p-4">تاریخ ایجاد</th>
            </tr>
          </thead>
          <tbody>
            {data.businesses.length ? (
              data.businesses.map((b) => (
                <tr
                  key={b.id}
                  onClick={() => r.push(`/businesses/${b.id}`)}
                  className="cursor-pointer border-t border-brand-border transition-colors hover:bg-brand-surface"
                >
                  <td className="p-4 font-medium">{b.name}</td>
                  <td className="p-4">
                    <Link
                      onClick={(e) => e.stopPropagation()}
                      href={`/users/${b.owner.id}`}
                      className="hover:text-brand-accent"
                    >
                      {b.owner.first_name} {b.owner.last_name}
                      <small dir="ltr" className="mr-2 text-brand-muted">
                        {b.owner.phone_number}
                      </small>
                    </Link>
                  </td>
                  <td className="p-4">
                    {b.member_count.toLocaleString("fa-IR")}
                  </td>
                  <td className="p-4">{isoToJalaliLabel(b.created_at)}</td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={4} className="p-10 text-center text-brand-muted">
                  کسب‌وکاری یافت نشد
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <footer className="mt-5 flex justify-end gap-3">
        <button
          disabled={page === 1}
          onClick={() => setPage((v) => v - 1)}
          className="cursor-pointer rounded-lg border border-brand-border px-4 py-2 disabled:opacity-40"
        >
          قبلی
        </button>
        <span className="p-2 text-sm text-brand-muted">
          صفحه {page} از {pages}
        </span>
        <button
          disabled={page === pages}
          onClick={() => setPage((v) => v + 1)}
          className="cursor-pointer rounded-lg border border-brand-border px-4 py-2 disabled:opacity-40"
        >
          بعدی
        </button>
      </footer>
      <Create
        open={open}
        close={() => setOpen(false)}
        done={() => setReload((x) => x + 1)}
      />
    </main>
  );
}

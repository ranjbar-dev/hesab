"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { isoToJalaliLabel } from "@/lib/jalali";
import {
  createUser,
  listUsers,
  type CreatePayload,
  type ListResult,
  type User,
} from "@/lib/users";
import Modal from "@/components/Modal";
import Select from "@/components/Select";
import { DateRangePicker } from "@/components/DatePicker";
import FormField from "@/components/FormField";
const field =
  "h-9 w-full rounded-lg border border-brand-border bg-brand-bg px-2 text-sm transition-colors duration-200 focus-visible:ring-2 focus-visible:ring-brand-accent/30";
type Filters = {
  first_name: string;
  last_name: string;
  phone: string;
  status: "" | User["status"];
  range: { from: string; to: string } | null;
};
function useDebounced<T>(v: T, ms: number) {
  const [d, setD] = useState(v);
  useEffect(() => {
    const t = setTimeout(() => setD(v), ms);
    return () => clearTimeout(t);
  }, [v, ms]);
  return d;
}
export function StatusBadge({ status }: { status: User["status"] }) {
  const on = status === "active";
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${on ? "bg-emerald-500/15 text-emerald-400" : "bg-brand-border text-brand-muted"}`}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {on ? "فعال" : "غیرفعال"}
    </span>
  );
}
const empty: CreatePayload = {
  first_name: "",
  last_name: "",
  phone_number: "",
  email: "",
  national_id: "",
  account_type: "individual",
  password: "",
};
function Create({
  open,
  close,
  done,
}: {
  open: boolean;
  close: () => void;
  done: () => void;
}) {
  const [f, setF] = useState(empty),
    [busy, setBusy] = useState(false),
    set = (k: keyof CreatePayload, v: string) =>
      setF((x) => ({ ...x, [k]: v }));
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await createUser(f);
      toast.success("کاربر ساخته شد");
      setF(empty);
      close();
      done();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open={open} onClose={close} title="کاربر جدید">
      <form onSubmit={submit} className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <FormField
            label="نام"
            value={f.first_name}
            onChange={(v) => set("first_name", v)}
            placeholder="مثال: احمد"
          />
          <FormField
            label="نام خانوادگی"
            value={f.last_name}
            onChange={(v) => set("last_name", v)}
            placeholder="مثال: رضایی"
          />
        </div>
        <FormField
          label="موبایل"
          type="tel"
          value={f.phone_number}
          onChange={(v) => set("phone_number", v)}
          placeholder="۰۹۱۲۱۲۳۴۵۶۷"
        />
        <FormField
          label="ایمیل"
          type="email"
          required={false}
          value={f.email}
          onChange={(v) => set("email", v)}
          placeholder="example@mail.com"
        />
        <FormField
          label="کد ملی"
          required={false}
          value={f.national_id}
          onChange={(v) => set("national_id", v)}
          placeholder="۰۰۱۲۳۴۵۶۷۸"
        />
        <label className="block text-sm">
          نوع حساب
          <Select
            value={{
              value: f.account_type,
              label: f.account_type === "company" ? "حقوقی" : "حقیقی",
            }}
            onChange={(v) => set("account_type", v?.value ?? "individual")}
            options={[
              { value: "individual", label: "حقیقی" },
              { value: "company", label: "حقوقی" },
            ]}
          />
        </label>
        <FormField
          label="رمز عبور"
          type="password"
          value={f.password}
          onChange={(v) => set("password", v)}
          placeholder="حداقل ۸ نویسه"
        />
        <button
          disabled={busy}
          className="h-11 w-full cursor-pointer rounded-lg bg-brand-accent font-bold text-brand-bg"
        >
          {busy ? "در حال انجام…" : "ساخت کاربر"}
        </button>
      </form>
    </Modal>
  );
}
export default function Users() {
  const router = useRouter(),
    [f, setF] = useState<Filters>({
      first_name: "",
      last_name: "",
      phone: "",
      status: "",
      range: null,
    }),
    d = useDebounced(f, 300),
    [page, setPage] = useState(1),
    [data, setData] = useState<ListResult>({
      users: [],
      total: 0,
      page: 1,
      page_size: 20,
    }),
    [loading, setLoading] = useState(true),
    [reload, setReload] = useState(0),
    [open, setOpen] = useState(false);
  useEffect(() => setPage(1), [d]);
  useEffect(() => {
    let alive = true;
    setLoading(true);
    listUsers({
      first_name: d.first_name,
      last_name: d.last_name,
      phone: d.phone,
      status: d.status || undefined,
      created_from: d.range?.from,
      created_to: d.range?.to,
      page,
      page_size: 20,
    })
      .then((x) => alive && setData(x))
      .catch((e) => toast.error(e instanceof Error ? e.message : "خطا"))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [d, page, reload]);
  const update = (k: keyof Filters, v: Filters[keyof Filters]) =>
    setF((x) => ({ ...x, [k]: v }));
  return (
    <main className="mx-auto max-w-7xl p-6 sm:p-12">
      <header className="mb-8 flex items-center justify-between">
        <h1 className="text-3xl font-bold">کاربران</h1>
        <button
          onClick={() => setOpen(true)}
          className="h-11 cursor-pointer rounded-lg bg-brand-accent px-4 font-bold text-brand-bg"
        >
          کاربر جدید
        </button>
      </header>
      <div className={loading ? "opacity-60" : ""}>
        <table className="w-full min-w-[55rem] border border-brand-border text-sm">
          <thead>
            <tr className="bg-brand-surface text-brand-muted">
              <th className="p-4">نام</th>
              <th className="p-4">نام خانوادگی</th>
              <th className="p-4">موبایل</th>
              <th className="p-4">نوع</th>
              <th className="p-4">وضعیت</th>
              <th className="p-4">تاریخ</th>
            </tr>
            <tr>
              <th>
                <input
                  placeholder="نام"
                  value={f.first_name}
                  onChange={(e) => update("first_name", e.target.value)}
                  className={field}
                />
              </th>
              <th>
                <input
                  placeholder="نام خانوادگی"
                  value={f.last_name}
                  onChange={(e) => update("last_name", e.target.value)}
                  className={field}
                />
              </th>
              <th>
                <input
                  placeholder="۰۹۱۲۱۲۳۴۵۶۷"
                  value={f.phone}
                  onChange={(e) => update("phone", e.target.value)}
                  className={field}
                />
              </th>
              <th />
              <th>
                <Select
                  value={
                    f.status
                      ? {
                          value: f.status,
                          label: f.status === "active" ? "فعال" : "غیرفعال",
                        }
                      : null
                  }
                  onChange={(v) =>
                    update("status", (v?.value ?? "") as Filters["status"])
                  }
                  options={[
                    { value: "", label: "همه" },
                    { value: "active", label: "فعال" },
                    { value: "disabled", label: "غیرفعال" },
                  ]}
                />
              </th>
              <th>
                <DateRangePicker
                  value={f.range}
                  onChange={(v) => update("range", v)}
                />
              </th>
            </tr>
          </thead>
          <tbody>
            {data.users.map((u) => (
              <tr
                key={u.id}
                onClick={() => router.push(`/users/${u.id}`)}
                className="cursor-pointer border-t border-brand-border"
              >
                <td className="p-4">{u.first_name}</td>
                <td className="p-4">{u.last_name}</td>
                <td className="p-4" dir="ltr">
                  {u.phone_number}
                </td>
                <td className="p-4">
                  {u.account_type === "company" ? "حقوقی" : "حقیقی"}
                </td>
                <td className="p-4">
                  <StatusBadge status={u.status} />
                </td>
                <td className="p-4">{isoToJalaliLabel(u.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Create
        open={open}
        close={() => setOpen(false)}
        done={() => setReload((v) => v + 1)}
      />
    </main>
  );
}

"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import Select from "@/components/Select";
import { useBusiness } from "../Shell";
import {
  canManage,
  changeMemberRole,
  inviteMember,
  listMembers,
  listOutgoingInvites,
  removeMember,
  roleLabels,
  type Member,
  type OutgoingInvite,
  type Role,
  cancelInvite,
} from "@/lib/businesses";
import { useRequireAuth } from "@/lib/useRequireAuth";
const input = "h-10 rounded-lg border border-brand-border bg-brand-bg px-3";
const roles: [Exclude<Role, "owner">, string][] = [
  ["admin", "مدیر"],
  ["accountant", "حسابدار"],
  ["viewer", "ناظر"],
];
export default function Members() {
  const r = useRouter(),
    { business, role } = useBusiness(),
    bid = business.id,
    { user } = useRequireAuth(),
    [members, setMembers] = useState<Member[]>([]),
    [invites, setInvites] = useState<OutgoingInvite[]>([]),
    [phone, setPhone] = useState(""),
    [inviteRole, setInviteRole] = useState<Exclude<Role, "owner">>("viewer");
  const roleOptions = roles.map(([value, label]) => ({ value, label }));
  const load = async () => {
    try {
      const m = await listMembers(bid);
      setMembers(m.members);
      if (canManage(role)) setInvites((await listOutgoingInvites(bid)).invites);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  };
  useEffect(() => {
    void load();
  }, [bid, role]);
  async function invite(e: React.FormEvent) {
    e.preventDefault();
    try {
      await inviteMember(bid, { phone_number: phone, role: inviteRole });
      setPhone("");
      toast.success("دعوت ارسال شد");
      load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  async function change(m: Member, v: Exclude<Role, "owner">) {
    try {
      await changeMemberRole(bid, m.user_id, { role: v });
      toast.success("نقش عضو تغییر کرد");
      load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  async function remove(id: number) {
    if (!confirm("عضو حذف شود؟")) return;
    try {
      await removeMember(bid, id);
      toast.success("عضو حذف شد");
      if (id === user?.id) r.replace("/select-business");
      else load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  async function cancel(i: number) {
    try {
      await cancelInvite(bid, i);
      toast.success("دعوت لغو شد");
      load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "خطا");
    }
  }
  return (
    <section>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-brand-muted">دسترسی تیم</p>
          <h1 className="text-2xl font-bold">اعضا</h1>
        </div>
        {role !== "owner" && (
          <button
            onClick={() => remove(user!.id)}
            className="cursor-pointer rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600"
          >
            خروج از کسب‌وکار
          </button>
        )}
      </div>
      <div className="mt-6 overflow-x-auto rounded-2xl border border-brand-border bg-brand-surface">
        <table className="w-full min-w-[42rem] text-right text-sm">
          <thead className="text-brand-muted">
            <tr>
              <th className="p-4">نام</th>
              <th className="p-4">موبایل</th>
              <th className="p-4">نقش</th>
              <th className="p-4">تاریخ عضویت</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <tr key={m.user_id} className="border-t border-brand-border">
                <td className="p-4">
                  {m.first_name} {m.last_name}
                </td>
                <td dir="ltr" className="p-4">
                  {m.phone_number}
                </td>
                <td className="p-4">
                  {canManage(role) &&
                  m.role !== "owner" &&
                  m.user_id !== user?.id ? (
                    <Select
                      value={{ value: m.role, label: roleLabels[m.role] }}
                      onChange={(v) =>
                        change(
                          m,
                          (v?.value ?? m.role) as Exclude<Role, "owner">,
                        )
                      }
                      options={roleOptions}
                    />
                  ) : (
                    roleLabels[m.role]
                  )}
                </td>
                <td className="p-4 text-brand-muted">
                  {new Date(m.created_at).toLocaleDateString("fa-IR")}
                </td>
                <td className="p-4">
                  {canManage(role) && m.role !== "owner" && (
                    <button
                      onClick={() => remove(m.user_id)}
                      className="cursor-pointer text-red-600"
                    >
                      حذف
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {canManage(role) && (
        <>
          <form
            onSubmit={invite}
            className="mt-5 flex flex-wrap gap-2 rounded-2xl border border-brand-border bg-brand-surface p-5"
          >
            <input
              required
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="شماره موبایل"
              dir="ltr"
              className={`min-w-48 flex-1 ${input}`}
            />
            <Select
              value={{ value: inviteRole, label: roleLabels[inviteRole] }}
              onChange={(v) =>
                setInviteRole(
                  (v?.value ?? inviteRole) as Exclude<Role, "owner">,
                )
              }
              options={roleOptions}
            />
            <button className="cursor-pointer rounded-lg bg-brand-accent px-4 font-bold text-white">
              دعوت عضو
            </button>
          </form>
          <section className="mt-5 rounded-2xl border border-brand-border bg-brand-surface p-5">
            <h2 className="font-bold">دعوت‌های در انتظار</h2>
            <div className="mt-3 space-y-2">
              {invites.length ? (
                invites.map((i) => (
                  <div
                    key={i.id}
                    className="flex items-center justify-between rounded-lg border border-brand-border p-3 text-sm"
                  >
                    <span>
                      {i.first_name} {i.last_name}{" "}
                      <small dir="ltr" className="mr-2 text-brand-muted">
                        {i.phone_number}
                      </small>{" "}
                      · {roleLabels[i.role]}
                    </span>
                    <button
                      onClick={() => cancel(i.id)}
                      className="cursor-pointer text-red-600"
                    >
                      لغو دعوت
                    </button>
                  </div>
                ))
              ) : (
                <p className="text-sm text-brand-muted">
                  دعوت فعالی وجود ندارد
                </p>
              )}
            </div>
          </section>
        </>
      )}
    </section>
  );
}

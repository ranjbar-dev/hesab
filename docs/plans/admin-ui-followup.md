# Follow-up: finish the control swap (review found gaps)

`docs/plans/admin-ui.md` was implemented but section 7/9 (native control swap)
was incomplete. Fix these specific files — don't touch anything else:

1. `admin/app/(dash)/businesses/page.tsx` — its `Create` component still
   renders a raw `<dialog ref={dialog}>` / `showModal()`/`.close()`, the
   exact bug the user reported (not centered, no X, backdrop click does
   nothing). Rewrite it to use `@/components/Modal` exactly like
   `admin/app/(dash)/users/page.tsx`'s `Create` component does (open boolean
   state owned by the parent, `<Modal open={open} onClose={...}
   title="کسب‌وکار جدید">`). Add placeholders: name input
   `placeholder="مثال: فروشگاه رضایی"`, owner-phone input
   `placeholder="۰۹۱۲۱۲۳۴۵۶۷"`. Also replace the page's own
   `useRequireAuth()` call with `useAdmin()` from `@/components/Sidebar`
   (the `(dash)` layout already auth-guards — same change already applied
   in `users/page.tsx`, mirror it here).

2. `admin/app/(dash)/businesses/[id]/Detail.tsx` — two native `<select>`s
   remain: the per-row member role select in the members table, and the
   role select in the "add member" form. Replace both with
   `@/components/Select` (same `{value,label}` option-object contract used
   in `admin/app/(dash)/users/page.tsx`'s account_type select). Add a
   placeholder to the rename-name `<input>`. Replace `useRequireAuth()` with
   `useAdmin()` from `@/components/Sidebar`.

3. `admin/app/(dash)/users/[id]/Detail.tsx` line ~92 — the `account_type`
   `<select>` is still native. Replace with `@/components/Select`. Replace
   `useRequireAuth()` with `useAdmin()` from `@/components/Sidebar`.

4. `admin/app/(dash)/settings/security/page.tsx` — replace
   `useRequireAuth()` with `useAdmin()` from `@/components/Sidebar` (it
   needs `admin` and `reload`, both already exposed by that context).

5. `client/app/businesses/[id]/members/page.tsx` — two native `<select>`s
   remain: the per-row member role select and the invite-role select.
   Replace both with `@/components/Select` (client's copy of the wrapper,
   same contract as admin's).

After all five: `cd admin && npm run build` and `cd client && npm run build`
must both still pass with 0 errors. Grep to confirm no `<select` or
`<dialog` remains under `admin/app/(dash)/**/*.tsx` or
`client/app/businesses/**/*.tsx` (aside from `Select.tsx`/`Modal.tsx`
themselves, which legitimately render one internally).

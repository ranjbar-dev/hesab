import type { Metadata } from "next";
import type { ReactNode } from "react";
import "./globals.css";
import { Toaster } from "sonner";

export const metadata: Metadata = {
  title: "ورود | حساب",
  description: "ورود به حساب کاربری برای مدیریت داده‌های حسابداری",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="fa" dir="rtl">
      <body className="antialiased">{children}<Toaster position="top-center" richColors /></body>
    </html>
  );
}

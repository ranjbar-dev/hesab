import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Toaster } from "sonner";
import "./globals.css";

export const metadata: Metadata = {
  title: "ورود | حساب",
  description: "ورود به حساب کاربری برای مدیریت داده‌های حسابداری",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="fa" dir="rtl">
      <body className="antialiased">
        {children}
        <Toaster
          dir="rtl"
          theme="light"
          richColors
          position="top-center"
          toastOptions={{ style: { fontFamily: "inherit" } }}
        />
      </body>
    </html>
  );
}

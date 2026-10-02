import type { Metadata } from "next";
import { SessionProvider } from "@/components/auth/session-provider";
import { AppShell } from "@/components/layout/app-shell";
import "./globals.css";
export const metadata: Metadata = {
  title: { default: "تونل زمان | حساب مرکزی", template: "%s | تونل زمان" },
  description: "مدیریت حساب کاربری و دسترسی به سامانه‌های زنجیره تأمین بهشت",
  icons: { icon: "/images/logo/accounts.png" },
};
export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="fa" dir="rtl" suppressHydrationWarning>
      <body>
        <SessionProvider>
          <AppShell>{children}</AppShell>
        </SessionProvider>
      </body>
    </html>
  );
}

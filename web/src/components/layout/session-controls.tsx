"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useSession } from "@/components/auth/session-provider";
import { message } from "@/lib/api";
export function AccountMenu() {
  const { user } = useSession();
  if (!user) return null;
  return (
    <details className="px-5 py-4 text-[#868B90]">
      <summary className="cursor-pointer truncate">{user.name}</summary>
      <nav
        aria-label="حساب کاربری"
        className="mt-4 flex flex-col gap-5 text-sm"
      >
        <Link href="/home">پیشخوان</Link>
        <Link href="/account">حساب کاربری</Link>
        <Link href="/personal-info">اطلاعات شخصی</Link>
        <Link href="/change-password">تغییر رمز عبور</Link>
      </nav>
    </details>
  );
}
export function SessionControl() {
  const { user, loading, logout } = useSession();
  const router = useRouter();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function signOut() {
    setBusy(true);
    try {
      await logout();
      router.replace("/login");
    } catch (error) {
      setError(message(error));
    } finally {
      setBusy(false);
    }
  }
  if (loading) return <span className="text-xs text-gray-500">…</span>;
  return (
    <div>
      {user ? (
        <button
          title="خروج"
          disabled={busy}
          className="session-button rounded-[10px] bg-red-600 p-2 text-white"
          onClick={signOut}
        >
          {busy ? "…" : "خروج"}
        </button>
      ) : (
        <Link
          title="ورود"
          href="/login"
          className="session-button block rounded-[10px] bg-[#2667FF] p-2 text-center text-white dark:bg-[#FFC700] dark:text-black"
        >
          ورود
        </Link>
      )}
      {error && (
        <p role="alert" className="text-xs text-red-600">
          {error}
        </p>
      )}
    </div>
  );
}

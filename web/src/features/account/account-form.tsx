"use client";
import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "@/components/auth/session-provider";
import { RequireSession } from "@/components/auth/require-session";
import { Field } from "@/components/ui/field";
import { api, ApiError, message, type User } from "@/lib/api";
export function AccountForm({ password = false }: { password?: boolean }) {
  const { user, setUser } = useSession();
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    setBusy(true);
    setError("");
    setNotice("");
    setErrors({});
    try {
      const result = await api<{ data?: User; message?: string }>(
        password ? "/change-password" : "/account",
        {
          method: "PUT",
          body: JSON.stringify(Object.fromEntries(new FormData(form))),
        },
      );
      if (result.data) {
        setUser(result.data);
        if (!result.data.email_verified_at) router.replace("/email/verify");
      } else form.reset();
      setNotice(result.message ?? "اطلاعات حساب ذخیره شد.");
    } catch (error) {
      setError(message(error));
      if (error instanceof ApiError) setErrors(error.fields);
    } finally {
      setBusy(false);
    }
  }
  return (
    <RequireSession>
      <section className="panel mx-auto max-w-2xl">
        <p className="muted">تنظیمات حساب</p>
        <h1 className="mt-2 text-2xl font-bold">
          {password ? "تغییر رمز عبور" : "اطلاعات حساب کاربری"}
        </h1>
        <p className="muted mt-3">
          {password
            ? "پس از تغییر رمز، نشست‌های دستگاه‌های دیگر بسته می‌شوند."
            : "پس از تغییر ایمیل، تأیید نشانی جدید لازم است."}
        </p>
        <form
          className="mt-7 space-y-5"
          onSubmit={submit}
          aria-busy={busy}
          key={user?.id}
        >
          {password ? (
            <>
              <Field
                name="current_password"
                label="رمز فعلی"
                type="password"
                dir="ltr"
                autoComplete="current-password"
                error={errors.current_password}
              />
              <Field
                name="password"
                label="رمز جدید"
                type="password"
                dir="ltr"
                autoComplete="new-password"
                minLength={8}
                maxLength={72}
                required
                error={errors.password}
              />
              <p className="muted !text-xs">
                حداقل ۸ نویسه و حداکثر ۷۲ بایت، شامل حروف بزرگ و کوچک، عدد و نماد.
              </p>
              <Field
                name="password_confirmation"
                label="تکرار رمز جدید"
                type="password"
                dir="ltr"
                autoComplete="new-password"
                required
                error={errors.password_confirmation}
              />
            </>
          ) : (
            <>
              <Field
                name="name"
                label="نام"
                defaultValue={user?.name}
                autoComplete="name"
                maxLength={255}
                required
                error={errors.name}
              />
              <Field
                name="email"
                label="ایمیل"
                type="email"
                dir="ltr"
                defaultValue={user?.email}
                autoComplete="email"
                maxLength={255}
                required
                error={errors.email}
              />
            </>
          )}
          {error && (
            <p role="alert" className="text-sm text-red-600">
              {error}
            </p>
          )}
          {notice && (
            <p
              role="status"
              className="text-sm text-green-700 dark:text-green-400"
            >
              {notice}
            </p>
          )}
          <button className="primary w-full sm:w-auto" disabled={busy}>
            {busy ? "در حال ذخیره…" : "ذخیره تغییرات"}
          </button>
        </form>
      </section>
    </RequireSession>
  );
}

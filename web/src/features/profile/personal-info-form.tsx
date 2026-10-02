"use client";
import { useEffect, useState, type FormEvent } from "react";
import { RequireSession } from "@/components/auth/require-session";
import { Field } from "@/components/ui/field";
import { api, apiForm, ApiError, message, type PersonalInfo } from "@/lib/api";

type Docs = Record<string, boolean>;

export function PersonalInfoForm() {
  const [info, setInfo] = useState<PersonalInfo | null>(null);
  const [docs, setDocs] = useState<Docs>({});
  const [isCompany, setIsCompany] = useState(false);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    let active = true;
    api<{ data: PersonalInfo; documents: Docs }>("/personal-info")
      .then((result) => {
        if (!active) return;
        setInfo(result.data);
        setDocs(result.documents ?? {});
        setIsCompany(Boolean(result.data.is_company));
      })
      .catch((err) => {
        if (!active) return;
        // Empty profile returns defaults; only surface real errors.
        if (err instanceof ApiError && err.status === 404) return;
        setError(message(err));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    // Checkbox without value posts "on"; normalize to true/false strings.
    form.set("is_company", isCompany ? "true" : "false");
    setBusy(true);
    setError("");
    setNotice("");
    setErrors({});
    try {
      const result = await apiForm<{ data: PersonalInfo; documents: Docs; message?: string }>(
        "/personal-info",
        form,
        "PUT",
      );
      setInfo(result.data);
      setDocs(result.documents ?? {});
      setIsCompany(Boolean(result.data.is_company));
      setNotice(result.message ?? "اطلاعات با موفقیت ثبت شد.");
    } catch (err) {
      setError(message(err));
      if (err instanceof ApiError) setErrors(err.fields);
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <p role="status">در حال بارگذاری…</p>;

  return (
    <RequireSession>
      <section className="panel mx-auto max-w-2xl">
        <p className="muted">اطلاعات هویتی</p>
        <h1 className="mt-2 text-2xl font-bold">اطلاعات شخصی / شرکتی</h1>
        <p className="muted mt-3">
          مدارک jpg، png یا webp تا ۱ مگابایت. پس از بار اول، ویرایش متن بدون ارسال مجدد مدارک ممکن است.
        </p>
        <form className="mt-7 space-y-5" onSubmit={submit} aria-busy={busy}>
          <label className="flex items-center gap-2 text-sm font-bold">
            <input
              type="checkbox"
              checked={isCompany}
              onChange={(e) => setIsCompany(e.target.checked)}
              className="h-4 w-4 accent-[#2667ff] dark:accent-[#ffc700]"
            />
            حساب شرکتی است
          </label>
          {errors.is_company && <p className="text-xs text-red-600 dark:text-red-400">{errors.is_company}</p>}
          <div className="grid gap-5 sm:grid-cols-2">
            <Field name="first_name" label="نام" defaultValue={info?.first_name} maxLength={255} required error={errors.first_name} />
            <Field name="last_name" label="نام خانوادگی" defaultValue={info?.last_name} maxLength={255} required error={errors.last_name} />
          </div>
          <div className="grid gap-5 sm:grid-cols-2">
            <Field name="mobile" label="موبایل (09…)" dir="ltr" defaultValue={info?.mobile} maxLength={11} required error={errors.mobile} />
            <Field name="telephone" label="تلفن ثابت با کد شهر" dir="ltr" defaultValue={info?.telephone} maxLength={11} required error={errors.telephone} />
          </div>
          <Field name="national_code" label="کد ملی" dir="ltr" defaultValue={info?.national_code} maxLength={10} required error={errors.national_code} />
          <Field name="address" label="نشانی" defaultValue={info?.address} maxLength={255} required error={errors.address} />
          {isCompany && (
            <div className="space-y-5">
              <Field name="company_name" label="نام شرکت" defaultValue={info?.company_name} maxLength={255} required error={errors.company_name} />
              <Field name="company_address" label="نشانی شرکت" defaultValue={info?.company_address} maxLength={255} required error={errors.company_address} />
              <div className="grid gap-5 sm:grid-cols-2">
                <Field name="company_registration_number" label="شماره ثبت" defaultValue={info?.company_registration_number} maxLength={255} required error={errors.company_registration_number} />
                <Field name="company_national_number" label="شناسه ملی شرکت" defaultValue={info?.company_national_number} maxLength={255} required error={errors.company_national_number} />
              </div>
              <div className="grid gap-5 sm:grid-cols-2">
                <Field name="company_tax_number" label="شماره مالیاتی" defaultValue={info?.company_tax_number} maxLength={255} required error={errors.company_tax_number} />
                <Field name="company_executive_name" label="نام مدیرعامل" defaultValue={info?.company_executive_name} maxLength={255} required error={errors.company_executive_name} />
              </div>
            </div>
          )}
          <div className="space-y-3">
            {(
              [
                ["melli_card_scan", "اسکن کارت ملی"],
                ["certificate_scan", "اسکن شناسنامه / روزنامه رسمی"],
                ["bank_card_scan", "اسکن کارت بانکی"],
              ] as const
            ).map(([name, label]) => (
              <div key={name} className="space-y-1">
                <label htmlFor={name} className="block text-sm font-bold">
                  {label} {docs[name] ? "✓ ثبت شده" : "(الزامی)"}
                </label>
                <input id={name} name={name} type="file" accept=".jpg,.jpeg,.png,.webp" className="field file:mr-4 file:rounded-lg file:border-0 file:bg-gray-100 file:px-4 file:py-2 file:text-sm file:font-bold dark:file:bg-neutral-800 dark:file:text-white" />
                {errors[name] && <p className="text-xs text-red-600 dark:text-red-400">{errors[name]}</p>}
              </div>
            ))}
          </div>
          {info?.is_verified && <p className="text-sm text-green-700 dark:text-green-400">وضعیت: تأیید شده</p>}
          {error && <p role="alert" className="text-sm text-red-600 dark:text-red-400">{error}</p>}
          {notice && <p role="status" className="text-sm text-green-700 dark:text-green-400">{notice}</p>}
          <button className="primary w-full sm:w-auto" disabled={busy}>
            {busy ? "در حال ذخیره…" : "ذخیره اطلاعات"}
          </button>
        </form>
      </section>
    </RequireSession>
  );
}

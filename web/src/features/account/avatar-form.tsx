"use client";
import { useEffect, useState, type FormEvent } from "react";
import { apiForm, ApiError, message } from "@/lib/api";

export function AvatarForm() {
  const [preview, setPreview] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function load() {
    try {
      const res = await fetch("/api/account/avatar", { credentials: "same-origin", cache: "no-store" });
      if (!res.ok) return;
      const blob = await res.blob();
      setPreview((old) => {
        if (old) URL.revokeObjectURL(old);
        return URL.createObjectURL(blob);
      });
    } catch {
      // No avatar yet.
    }
  }

  useEffect(() => {
    load();
    return () => {
      setPreview((old) => {
        if (old) URL.revokeObjectURL(old);
        return null;
      });
    };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await apiForm("/account/avatar", form, "PUT");
      setNotice("آواتار به‌روزرسانی شد.");
      await load();
    } catch (err) {
      setError(message(err));
      if (err instanceof ApiError && err.fields.avatar) setError(err.fields.avatar);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel mx-auto mt-6 max-w-2xl">
      <h2 className="text-lg font-bold">آواتار</h2>
      <p className="muted mt-2 text-sm">jpg، png یا webp تا ۱ مگابایت.</p>
      {preview && <img src={preview} alt="آواتار فعلی" className="mt-4 h-20 w-20 rounded-full object-cover ring-1 ring-black/10 dark:ring-white/20" />}
      <form className="mt-4 space-y-4" onSubmit={submit} aria-busy={busy}>
        <input name="avatar" type="file" accept=".jpg,.jpeg,.png,.webp" className="field file:mr-4 file:rounded-lg file:border-0 file:bg-gray-100 file:px-4 file:py-2 file:text-sm file:font-bold dark:file:bg-neutral-800 dark:file:text-white" required />
        {error && <p role="alert" className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        {notice && <p role="status" className="text-sm text-green-700 dark:text-green-400">{notice}</p>}
        <button className="primary w-full sm:w-auto" disabled={busy}>
          {busy ? "در حال بارگذاری…" : "بارگذاری آواتار"}
        </button>
      </form>
    </section>
  );
}

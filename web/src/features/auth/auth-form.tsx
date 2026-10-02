"use client";
// @refresh reset
// Rebuild the form after development edits instead of retaining stale keyed DOM.
import Link from "next/link";
import { oauthReturnTo } from "@/lib/navigation";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState, type FormEvent, type MouseEvent } from "react";
import { api, ApiError, message, type User } from "@/lib/api";
import { useSession } from "@/components/auth/session-provider";
import { Field } from "@/components/ui/field";
import { PasswordChecklist } from "./password-checklist";
import { WalletOptions } from "./wallet-options";
import {
  registrationErrors,
  loginErrors,
  type RegistrationValues,
} from "./registration-validation";

type Mode = "login" | "register" | "forgot" | "reset";
const titles: Record<Mode, string> = {
  login: "ورود",
  register: "ثبت نام",
  forgot: "بازیابی دسترسی",
  reset: "یک رمز تازه انتخاب کنید",
};
const buttons: Record<Mode, string> = {
  login: "ورود به حساب",
  register: "ایجاد حساب کاربری",
  forgot: "ارسال پیوند بازیابی",
  reset: "ذخیره رمز جدید",
};
const paths: Record<Mode, string> = {
  login: "/login",
  register: "/register",
  forgot: "/password/email",
  reset: "/password/reset",
};
export function AuthForm({ mode: initialMode }: { mode: Mode }) {
  const pathname = usePathname();
  const mode: Mode =
    pathname === "/login"
      ? "login"
      : pathname === "/register"
        ? "register"
        : initialMode;
  const router = useRouter();
  const params = useSearchParams();
  const { setUser } = useSession();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [values, setValues] = useState<RegistrationValues>({
    username: "",
    name: "",
    email: params.get("email") ?? "",
    password: "",
    password_confirmation: "",
    referral: params.get("referral") ?? "",
  });
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [showRequirements, setShowRequirements] = useState(false);
  const validation =
    mode === "register"
      ? registrationErrors(values)
      : mode === "login"
        ? loginErrors(values)
        : {};
  const entryInvalid =
    (mode === "register" || mode === "login") &&
    (Object.keys(validation).length > 0 || Object.keys(fields).length > 0);
  function fieldError(name: keyof RegistrationValues) {
    return fields[name] || (touched[name] ? validation[name] : undefined);
  }
  function inputProps(name: keyof RegistrationValues) {
    return {
      value: values[name],
      disabled: busy,
      onChange: (event: React.ChangeEvent<HTMLInputElement>) => {
        setValues((current) => ({ ...current, [name]: event.target.value }));
        setTouched((current) => ({ ...current, [name]: true }));
        setFields((current) => {
          const next = { ...current };
          delete next[name];
          return next;
        });
      },
      onBlur: () => setTouched((current) => ({ ...current, [name]: true })),
      onFocus: () => {
        if (name === "password" || name === "password_confirmation")
          setShowRequirements(true);
      },
    };
  }
  useEffect(() => {
    setError("");
    setNotice("");
    setFields({});
    setValues({
      username: "",
      name: "",
      email: params.get("email") ?? "",
      password: "",
      password_confirmation: "",
      referral: params.get("referral") ?? "",
    });
    setTouched({});
    setShowRequirements(false);
    if (mode === "login" || mode === "register") {
      document.title = `${mode === "login" ? "ورود" : "ثبت‌نام"} | تونل زمان`;
    }
  }, [mode]);
  function switchEntry(
    event: MouseEvent<HTMLAnchorElement>,
    next: "login" | "register",
  ) {
    if (
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    )
      return;
    if (mode !== "login" && mode !== "register") return;
    event.preventDefault();
    if (busy || mode === next) return;
    // Next synchronizes native history with usePathname without fetching a new page.
    const query = new URLSearchParams();
    for (const key of ["email", "referral", "return_to", "client_id", "redirect_uri", "back_url"]) {
      const value = params.get(key);
      if (value) query.set(key, value);
    }
    window.history.pushState(
      null,
      "",
      `/${next}${query.size ? `?${query}` : ""}`,
    );
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    if (
      (mode === "register" || mode === "login") &&
      Object.keys(validation).length > 0
    ) {
      setTouched({
        username: true,
        name: true,
        email: true,
        password: true,
        password_confirmation: true,
        referral: true,
      });
      return;
    }
    setBusy(true);
    setError("");
    setFields({});
    setNotice("");
    const payload: Record<string, unknown> = Object.fromEntries(new FormData(event.currentTarget));
 if (mode==="register") {
 for(const key of ["redirect_uri","back_url"]){const value=params.get(key);if(value)payload[key]=value;}
 const client=params.get("client_id");if(client && /^\d+$/.test(client))payload.client_id=Number(client);
 }
    if (mode === "login") {
      payload.remember = payload.remember === "on";
 payload.login = payload.email;
      delete payload.email;
    }
    if (mode === "reset") payload.token = params.get("token") ?? "";
    try {
      const result = await api<{
        data?: User;
        message: string;
        verification_sent?: boolean;
      }>(paths[mode], { method: "POST", body: JSON.stringify(payload) });
      if (mode === "login") {
        const account = await api<{ data: User }>("/account");
        result.data = account.data;
      }
      if (result.data) {
        setUser(result.data);
        const intended=oauthReturnTo(params.get("return_to"));
 if(intended && !result.data.email_verified_at)sessionStorage.setItem("sso:return_to",intended);
 if(result.data.email_verified_at && intended)window.location.assign(intended);
 else router.replace(result.data.email_verified_at ? "/home" : "/email/verify");
      } else {
        setNotice(result.message);
      }
    } catch (error) {
      setError(message(error));
      if (error instanceof ApiError) {
        const errors = { ...error.fields };
        if (mode === "login" && errors.login) {
          errors.email = errors.login;
          delete errors.login;
        }
        setFields(errors);
      }
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="auth-stage">
      <div className="auth-intro">
        <span className="auth-eyebrow">
          <span /> حساب مرکزی شما
        </span>
        <h1>زیرساخت های فعال زنجیره تامین بهشت</h1>
        <p>
          با ثبت نام در این صفحه شما میتوانید به تمامی سامانه های تحت پوشش
          هلدینگ زنجیره تامین بهشت دسترسی مستقیم داشته باشید.
        </p>
      </div>
      <section className="auth-card" aria-labelledby="auth-title">
        <div className="auth-card-heading">
          <div className="auth-symbol" aria-hidden="true">
            <svg
              width="26"
              height="26"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <rect x="5" y="10" width="14" height="11" rx="3" />
              <path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3" />
            </svg>
          </div>
          <h2 id="auth-title">{titles[mode]}</h2>
          {(mode === "forgot" || mode === "reset") && (
            <p>با چند قدم ساده، دوباره به حساب خود دسترسی پیدا کنید.</p>
          )}
        </div>
        {(mode === "login" || mode === "register") && (
          <nav className="auth-tabs" aria-label="ورود و ثبت‌نام">
            <Link
              href="/login"
              onClick={(event) => switchEntry(event, "login")}
              aria-current={mode === "login" ? "page" : undefined}
            >
              ورود به حساب
            </Link>
            <Link
              href="/register"
              onClick={(event) => switchEntry(event, "register")}
              aria-current={mode === "register" ? "page" : undefined}
            >
              ساخت حساب
            </Link>
          </nav>
        )}
        {(mode === "login" || mode === "register") && (
          <WalletOptions key={`wallet-${mode}`} mode={mode} busy={busy} />
        )}
        <form
          key={`form-${mode}`}
          className="mt-7 space-y-5"
          onSubmit={submit}
          aria-busy={busy}
          noValidate={mode === "register" || mode === "login"}
        >
          {mode === "register" && (
            <Field
              label="نام"
              name="name"
              autoComplete="name"
              maxLength={50}
              required
              error={fieldError("name")}
              {...inputProps("name")}
            />
          )}
          {mode === "register" && (
            <Field
              label="نام کاربری"
              name="username"
              dir="ltr"
              autoComplete="username"
              autoCapitalize="none"
              spellCheck={false}
              maxLength={30}
              required
              error={fieldError("username")}
              {...inputProps("username")}
            />
          )}
          <Field
            label={mode === "login" ? "نام کاربری یا ایمیل" : "ایمیل"}
            name="email"
            type={mode === "login" ? "text" : "email"}
            dir="ltr"
            autoComplete={mode === "login" ? "username" : "email"}
            autoCapitalize="none"
            spellCheck={false}
            maxLength={255}
            required
            error={fieldError("email")}
            {...inputProps("email")}
          />
          {mode !== "forgot" && (
            <Field
              label={mode === "reset" ? "رمز جدید" : "رمز عبور"}
              name="password"
              type="password"
              dir="ltr"
              autoComplete={
                mode === "login" ? "current-password" : "new-password"
              }
              minLength={mode === "login" ? undefined : 8}
              maxLength={mode === "login" ? undefined : mode === "register" ? 40 : 72}
              required
              error={fieldError("password")}
              {...inputProps("password")}
              aria-describedby={
                mode === "register" && showRequirements
                  ? "password-requirements"
                  : undefined
              }
            />
          )}
          {(mode === "register" || mode === "reset") && (
            <>
              {mode === "reset" && (
                <p className="muted !text-xs">
                  حداقل ۸ نویسه و حداکثر ۷۲ بایت.
                </p>
              )}
              {mode === "register" && showRequirements && (
                <PasswordChecklist
                  password={values.password}
                  confirmation={values.password_confirmation}
                />
              )}
              <Field
                label="تکرار رمز عبور"
                name="password_confirmation"
                type="password"
                dir="ltr"
                autoComplete="new-password"
                required
                error={fieldError("password_confirmation")}
                {...inputProps("password_confirmation")}
                aria-describedby={
                  mode === "register" && showRequirements
                    ? "password-requirements"
                    : undefined
                }
              />
            </>
          )}
          {mode === "register" && (
            <Field
              label="کد معرف (اختیاری)"
              name="referral"
              dir="ltr"
              error={fieldError("referral")}
              {...inputProps("referral")}
            />
          )}
          {mode === "login" && (
            <Link className="auth-link block text-sm" href="/password/email">
              رمز عبور را فراموش کرده‌اید؟
            </Link>
          )}
          {error && (
            <p
              role="alert"
              className="rounded-xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950 dark:text-red-200"
            >
              {error}
            </p>
          )}
          {notice && (
            <p
              role="status"
              className="rounded-xl bg-green-50 p-4 text-sm text-green-800 dark:bg-green-950 dark:text-green-200"
            >
              {notice}
            </p>
          )}
          <button
            disabled={busy || entryInvalid}
            aria-busy={busy}
            className="primary auth-submit w-full"
            type="submit"
          >
            {busy ? (
              <span className="submit-loading" role="status">
                <span className="submit-spinner" aria-hidden="true" />
                در حال ارسال…
              </span>
            ) : (
              buttons[mode]
            )}
          </button>
          {mode === "login" && <label className="flex items-center gap-2"><input type="checkbox" name="remember" disabled={busy} />مرا به خاطر بسپار</label>}
        </form>
        <p className="muted mt-6 text-center">
          {mode === "login" ? "هنوز حساب ندارید؟ " : "حساب کاربری دارید؟ "}
          <Link
            className="auth-link font-bold"
            href={mode === "login" ? "/register" : "/login"}
            onClick={(event) =>
              switchEntry(event, mode === "login" ? "register" : "login")
            }
          >
            {mode === "login" ? "ثبت‌نام" : "ورود"}
          </Link>
        </p>
      </section>
      <p className="auth-footnote">
        فضایی برای ارتباط، با دسترسی ساده و یکپارچه
      </p>
    </div>
  );
}

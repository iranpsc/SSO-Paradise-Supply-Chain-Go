"use client";
import Link from "next/link";
import { useSession } from "@/components/auth/session-provider";
import { RequireSession } from "@/components/auth/require-session";
const services = [
  [
    "متارنگ",
    "metargb.irpsc.png",
    "https://metarang.com",
    "ورود به دنیای متاورس",
  ],
  [
    "سه‌بعدی متا",
    "3d.irpsc.png",
    "https://3d.irpsc.com",
    "محصولات و محتوای سه‌بعدی",
  ],
  ["فروشگاه ملی", "shop.irpsc.png", "https://shop.irpsc.com", "کالاها و خدمات"],
  ["انجمن حم", "faq.irpsc.png", "https://faqhub.ir", "گفت‌وگو و پرسش و پاسخ"],
  [
    "دانشگاه متاورس",
    "uni.irpsc.png",
    "https://uni.irpsc.com",
    "یادگیری و آموزش",
  ],
  [
    "مرکز آموزش ویدئویی",
    "video.irpsc.png",
    "https://video.irpsc.com",
    "آموزش‌های تصویری",
  ],
];
export function Dashboard() {
  const { user } = useSession();
  return (
    <RequireSession>
      <div className="space-y-7">
        <section className="panel flex flex-col justify-between gap-6 sm:flex-row sm:items-center">
          <div>
            <p className="muted">پیشخوان حساب مرکزی</p>
            <h1 className="mt-2 text-2xl font-bold sm:text-3xl">
              {user?.name}، خوش آمدید
            </h1>
            <p className="muted mt-3">
              حساب و مسیر دسترسی به سامانه‌ها، در یک نگاه.
            </p>
          </div>
          <Link className="primary shrink-0" href="/account">
            مدیریت حساب
          </Link>
        </section>
        <div className="grid gap-4 sm:grid-cols-3">
          {[
            ["وضعیت ایمیل", "تأیید شده"],
            ["شناسهٔ عضویت", user?.code ?? "—"],
            ["امنیت حساب", "مدیریت رمز عبور"],
          ].map(([label, value]) => (
            <div className="panel !p-6" key={label}>
              <p className="muted">{label}</p>
              {label === "امنیت حساب" ? (
                <Link
                  href="/change-password"
                  className="mt-3 block font-bold text-blue-600 dark:text-yellow-400"
                >
                  {value} ←
                </Link>
              ) : (
                <p
                  className="mt-3 font-bold"
                  dir={label === "شناسهٔ عضویت" ? "ltr" : "rtl"}
                >
                  {value}
                </p>
              )}
            </div>
          ))}
        </div>
        <section>
          <div className="mb-5">
            <h2 className="text-xl font-bold">سامانه‌های زنجیره تأمین بهشت</h2>
            <p className="muted mt-2">سرویس مورد نظر خود را انتخاب کنید.</p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {services.map(([name, icon, href, description]) => (
              <a
                className="panel group flex items-center gap-4 !p-6 transition hover:border-blue-400"
                href={href}
                key={href}
                target="_blank"
                rel="noopener noreferrer"
              >
                <img
                  src={`/images/logo/${icon}`}
                  alt=""
                  width={44}
                  height={44}
                />
                <div className="min-w-0">
                  <h3 className="font-bold group-hover:text-blue-600 dark:group-hover:text-yellow-400">
                    {name}
                  </h3>
                  <p className="muted mt-1 !text-xs">{description}</p>
                </div>
                <span className="mr-auto" aria-hidden>
                  ↗
                </span>
              </a>
            ))}
          </div>
        </section>
      </div>
    </RequireSession>
  );
}

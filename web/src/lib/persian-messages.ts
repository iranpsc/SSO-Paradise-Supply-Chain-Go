const fallback = "خطایی رخ داد؛ دوباره تلاش کنید.";
const known: Record<string, string> = {
  "Invalid credentials": "اطلاعات ورود معتبر نیست.",
  "Unauthenticated": "برای ادامه وارد حساب کاربری شوید.",
  "Unauthenticated.": "برای ادامه وارد حساب کاربری شوید.",
  "Too Many Attempts.": "تعداد تلاش‌ها بیش از حد مجاز است؛ کمی صبر کنید و دوباره تلاش کنید.",
  invalid_request: "درخواست ارسال‌شده معتبر نیست.",
  invalid_client: "سامانهٔ درخواست‌کننده معتبر نیست.",
  invalid_grant: "مجوز ورود نامعتبر یا منقضی شده است.",
  invalid_scope: "سطح دسترسی درخواست‌شده معتبر نیست.",
  access_denied: "درخواست دسترسی تأیید نشد.",
  unsupported_grant_type: "روش دریافت مجوز پشتیبانی نمی‌شود.",
  unsupported_response_type: "نوع پاسخ درخواست‌شده پشتیبانی نمی‌شود.",
};

export function persianMessage(value: unknown, defaultMessage = fallback): string {
  if (typeof value !== "string") return defaultMessage;
  if (known[value]) return known[value];
  // External services and wallet providers can return arbitrary English text.
  // Do not expose it (or a raw exception) to the user.
  return /[\u0600-\u06ff]/u.test(value) && !/[a-z]/i.test(value)
    ? value
    : defaultMessage;
}

export function errorMessage(error: unknown): string {
  if (error && typeof error === "object") {
    const value = error as { code?: unknown; message?: unknown; cause?: unknown };
    const code = Number(value.code);
    if (code === 4001 || code === 5000) return "درخواست اتصال یا امضا در کیف پول لغو شد.";
    if (code === -32002) return "یک درخواست در کیف پول منتظر پاسخ است؛ پنجرهٔ کیف پول را باز کنید.";
    if (code === 4100) return "دسترسی به حساب کیف پول تأیید نشده است؛ اتصال را دوباره تأیید کنید.";
    if (code === 4200) return "کیف پول انتخاب‌شده از این عملیات پشتیبانی نمی‌کند.";
    if (code === 4900 || code === 4901) return "ارتباط کیف پول با شبکه قطع است؛ اتصال را بررسی کنید.";
    if (error instanceof TypeError) return "ارتباط با سامانه برقرار نشد؛ اتصال اینترنت را بررسی کنید و دوباره تلاش کنید.";
    return persianMessage(value.message);
  }
  return fallback;
}

export function responseMessage(status: number): string {
  switch (status) {
    case 400: return "دادهٔ درخواست معتبر نیست.";
    case 401: return "اطلاعات ورود معتبر نیست؛ دوباره وارد حساب شوید.";
    case 403: return "اجازهٔ انجام این عملیات را ندارید.";
    case 404: return "اطلاعات یا صفحهٔ درخواستی یافت نشد.";
    case 405: return "روش ارسال این درخواست مجاز نیست.";
    case 413: return "حجم فایل یا دادهٔ ارسال‌شده بیش از حد مجاز است.";
    case 415: return "نوع دادهٔ ارسال‌شده پشتیبانی نمی‌شود.";
    case 422: return "اطلاعات واردشده معتبر نیست.";
    case 429: return "تعداد تلاش‌ها بیش از حد مجاز است؛ کمی صبر کنید و دوباره تلاش کنید.";
    default: return "درخواست انجام نشد؛ دوباره تلاش کنید.";
  }
}

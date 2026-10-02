export type RegistrationValues = {
  username: string;
  name: string;
  email: string;
  password: string;
  password_confirmation: string;
  referral: string;
};

export function passwordRules(password: string, confirmation: string) {
  const length = Array.from(password).length;
  return [
    {
      id: "length",
      label: "بین ۸ تا ۴۰ نویسه",
      passed: length >= 8 && length <= 40,
    },
    {
      id: "upper",
      label: "حداقل یک حرف بزرگ (مانند A)",
      passed: new RegExp("\\p{Lu}", "u").test(password),
    },
    {
      id: "lower",
      label: "حداقل یک حرف کوچک (مانند a)",
      passed: new RegExp("\\p{Ll}", "u").test(password),
    },
    {
      id: "number",
      label: "حداقل یک عدد",
      passed: new RegExp("\\p{Nd}", "u").test(password),
    },
    {
      id: "symbol",
      label: "حداقل یک نماد (مانند ! یا @)",
      passed: new RegExp("[\\p{P}\\p{S}]", "u").test(password),
    },
    {
      id: "bytes",
      label: "حداکثر ۷۲ بایت برای نویسه‌های چندبایتی",
      passed:
        password.length > 0 && new TextEncoder().encode(password).length <= 72,
    },
    {
      id: "confirmation",
      label: "یکسان بودن رمز و تکرار آن",
      passed: password.length > 0 && password === confirmation,
    },
  ];
}

export function emailError(value: string): string | undefined {
  const email = value.trim();
  if (!email) return "ایمیل را وارد کنید.";
  if (
    new TextEncoder().encode(email).length > 255 ||
    !/^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(email)
  )
    return "یک ایمیل معتبر وارد کنید.";
}

export function loginErrors(
  values: Pick<RegistrationValues, "email" | "password">,
): Record<string, string> {
  const errors: Record<string, string> = {};
  const login = values.email.trim();
  if (!login) errors.email = "نام کاربری یا ایمیل را وارد کنید.";
  else if (login.includes("@")) {
    const invalidEmail = emailError(login);
    if (invalidEmail) errors.email = invalidEmail;
  } else if (!/^[a-z][a-z0-9_]{2,29}$/i.test(login))
    errors.email = "نام کاربری معتبر وارد کنید.";
  if (!values.password) errors.password = "رمز عبور را وارد کنید.";
  return errors;
}

export function registrationErrors(
  values: RegistrationValues,
): Record<string, string> {
  const errors: Record<string, string> = {};
  const name = values.name.trim();
  if (!/^[a-z][a-z0-9_]{2,29}$/i.test(values.username.trim()))
    errors.username =
      "نام کاربری باید ۳ تا ۳۰ نویسه، با حرف انگلیسی آغاز و فقط شامل حروف انگلیسی، عدد و زیرخط باشد.";
  if (!name) errors.name = "نام را وارد کنید.";
  else if (Array.from(name).length > 50)
    errors.name = "نام حداکثر ۵۰ نویسه باشد.";
  else if (/^hm-/i.test(name)) errors.name = "نام نباید با hm- شروع شود.";
  const invalidEmail = emailError(values.email);
  if (invalidEmail) errors.email = invalidEmail;
  const rules = passwordRules(values.password, values.password_confirmation);
  if (!values.password) errors.password = "رمز عبور را وارد کنید.";
  else if (rules.some((rule) => rule.id !== "confirmation" && !rule.passed))
    errors.password = "رمز عبور باید همهٔ شرایط زیر را داشته باشد.";
  if (!values.password_confirmation)
    errors.password_confirmation = "رمز عبور را دوباره وارد کنید.";
  else if (values.password_confirmation !== values.password)
    errors.password_confirmation = "تکرار رمز با رمز عبور مطابقت ندارد.";
  return errors;
}

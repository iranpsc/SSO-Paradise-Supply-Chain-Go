import { errorMessage, persianMessage, responseMessage } from "./persian-messages";
export type User = {
  wallet_address: string | null;
  id: number;
  username: string;
  name: string;
  email: string;
  code: string | null;
  email_verified_at: string | null;
  created_at: string;
};
export type PersonalInfo = {
  user_id: number;
  is_company: boolean | null;
  first_name: string;
  last_name: string;
  mobile: string;
  telephone: string;
  national_code: string;
  address: string;
  company_name: string;
  company_address: string;
  company_registration_number: string;
  company_national_number: string;
  company_tax_number: string;
  company_executive_name: string;
  is_verified: boolean;
  verification_messages: string;
};
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public fields: Record<string, string> = {},
  ) {
    super(message);
  }
}
export async function apiForm<T>(path: string, form: FormData, method = "PUT"): Promise<T> {
  const response = await fetch(`/api${path}`, {
    method,
    body: form,
    credentials: "same-origin",
    cache: "no-store",
  });
  let body;
  try {
    body = await response.json();
  } catch {
    throw new ApiError(response.status, "ارتباط با سرور برقرار نشد. دوباره تلاش کنید.");
  }
  if (!response.ok)
    throw new ApiError(
      response.status,
      persianMessage(body.message ?? body.error_description ?? body.error, responseMessage(response.status)),
      Object.fromEntries(
        Object.entries(body.errors ?? {}).map(([field, value]) => [
          field,
          persianMessage(Array.isArray(value) ? value.join(" ") : value, "مقدار واردشده برای این فیلد معتبر نیست."),
        ]),
      ),
    );
  if (typeof body.message === "string") body.message = persianMessage(body.message, "عملیات با موفقیت انجام شد.");
  return body as T;
}
export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(`/api${path}`, {
    ...options,
    credentials: "same-origin",
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  let body;
  try {
    body = await response.json();
  } catch {
    throw new ApiError(
      response.status,
      "ارتباط با سرور برقرار نشد. دوباره تلاش کنید.",
    );
  }
  if (!response.ok)
    throw new ApiError(
      response.status,
      persianMessage(body.message ?? body.error_description ?? body.error, responseMessage(response.status)),
      Object.fromEntries(
        Object.entries(body.errors ?? {}).map(([field, value]) => [
          field,
          persianMessage(Array.isArray(value) ? value.join(" ") : value, "مقدار واردشده برای این فیلد معتبر نیست."),
        ]),
      ),
    );
  if (typeof body.message === "string") body.message = persianMessage(body.message, "عملیات با موفقیت انجام شد.");
  return body as T;
}
export function message(error: unknown) {
  return errorMessage(error);
}

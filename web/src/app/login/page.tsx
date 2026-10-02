import { Suspense } from "react";
import { AuthForm } from "@/features/auth/auth-form";
export const metadata = { title: "ورود" };
export default function Page() {
  return (
    <Suspense fallback={<p>در حال بارگذاری…</p>}>
      <AuthForm mode="login" />
    </Suspense>
  );
}

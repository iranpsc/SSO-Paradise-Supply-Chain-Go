import { Suspense } from "react";
import { AuthForm } from "@/features/auth/auth-form";
export default function Page() {
  return (
    <Suspense fallback={<p role="status">در حال بارگذاری…</p>}>
      <AuthForm mode="reset" />
    </Suspense>
  );
}

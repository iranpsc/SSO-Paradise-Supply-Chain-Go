import { Suspense } from "react";
import { VerifyEmail } from "@/features/auth/verify-email";
export default function Page() {
  return (
    <Suspense fallback={<p role="status">در حال بارگذاری…</p>}>
      <VerifyEmail />
    </Suspense>
  );
}

import { Suspense } from "react";
import { AccountForm } from "@/features/account/account-form";
export default function Page() {
  return (
    <Suspense fallback={<p role="status">در حال بارگذاری…</p>}>
      <AccountForm password />
    </Suspense>
  );
}

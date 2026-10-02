import { Suspense } from "react";
import { Dashboard } from "@/features/account/dashboard";
export default function Page() {
  return (
    <Suspense fallback={<p role="status">در حال بارگذاری…</p>}>
      <Dashboard />
    </Suspense>
  );
}

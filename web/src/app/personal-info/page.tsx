import { Suspense } from "react";
import { PersonalInfoForm } from "@/features/profile/personal-info-form";

export default function Page() {
  return (
    <Suspense fallback={<p role="status">در حال بارگذاری…</p>}>
      <PersonalInfoForm />
    </Suspense>
  );
}

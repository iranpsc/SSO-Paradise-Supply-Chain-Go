import { WalletLink } from "@/features/account/wallet-link";
import { Suspense } from "react";
import { AccountForm } from "@/features/account/account-form";
import { AvatarForm } from "@/features/account/avatar-form";
export default function Page() {
  return (
    <Suspense fallback={<p role="status">در حال بارگذاری…</p>}>
      <AccountForm />
      <AvatarForm />
      <WalletLink />
    </Suspense>
  );
}

"use client";
import { useEffect, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "./session-provider";
export function RequireSession({
  children,
  verified = true,
}: {
  children: ReactNode;
  verified?: boolean;
}) {
  const { user, loading, error, refresh } = useSession();
  const router = useRouter();
  useEffect(() => {
    if (!loading && !error) {
      if (!user) router.replace("/login");
      else if (verified && !user.email_verified_at)
        router.replace("/email/verify");
    }
  }, [loading, error, user, verified, router]);
  if (error)
    return (
      <div role="alert" className="panel space-y-5">
        <p>{error}</p>
        <button className="primary" onClick={() => void refresh()}>
          تلاش دوباره
        </button>
      </div>
    );
  if (loading || !user || (verified && !user.email_verified_at))
    return (
      <div role="status" className="panel animate-pulse">
        در حال دریافت اطلاعات حساب…
      </div>
    );
  return children;
}

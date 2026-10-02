import { Suspense } from "react";
import { ConfirmPassword } from "@/features/auth/confirm-password";
export default function Page(){return <Suspense fallback={<p role="status">در حال بارگذاری…</p>}><ConfirmPassword/></Suspense>;}

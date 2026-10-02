"use client";
import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { useSession } from "@/components/auth/session-provider";
import { authenticateWallet, type WalletKind } from "@/lib/wallet";
import { message } from "@/lib/api";
import { oauthReturnTo } from "@/lib/navigation";
const wallets = [
 { id: "metamask", name: "MetaMask", logo: "/images/wallets/metamask.svg" },
 { id: "walletconnect", name: "WalletConnect", logo: "/images/wallets/walletconnect.svg" },
] as const;
export function WalletOptions({ mode, busy }: { mode: "login" | "register"; busy: boolean }) {
 const [pending, setPending] = useState(false);
 const [notice, setNotice] = useState("");
 const { refresh } = useSession();
 const params = useSearchParams();
 const action = mode === "login" ? "ورود" : "ثبت نام";
 async function connect(kind: WalletKind) {
  setPending(true); setNotice("");
  try {
   const result = await authenticateWallet(kind);
   const user = await refresh();
   const intended = oauthReturnTo(params.get("return_to"));
   if (intended && !user?.email_verified_at) sessionStorage.setItem("sso:return_to", intended);
   window.location.assign(user?.email_verified_at && intended ? intended : new URL(result.redirect, window.location.origin).pathname);
  } catch (error) { setNotice(message(error)); }
  finally { setPending(false); }
 }
 return (
  <section className="wallet-options" aria-label={`${action} با کیف پول`}>
   <div className="wallet-options-heading"><span>با کیف پول خود ادامه دهید</span></div>
   <div className="wallet-buttons">
    {wallets.map(wallet => <button type="button" key={wallet.id} disabled={busy || pending} className={`wallet-button wallet-${wallet.id}`} onClick={() => void connect(wallet.id)}>
     <span className="wallet-logo"><img src={wallet.logo} width={34} height={34} alt="" /></span>
     <span className="wallet-button-label">{action} با <bdi lang="en">{wallet.name}</bdi></span><span className="wallet-arrow" aria-hidden="true">↗</span>
    </button>)}
   </div>
   {pending && <p className="wallet-notice" role="status">در انتظار اتصال و امضای پیام در کیف پول…</p>}
   {notice && <p className="wallet-notice" role="alert">{notice}</p>}
   <div className="auth-divider"><span />یا با حساب کاربری<span /></div>
  </section>
 );
}

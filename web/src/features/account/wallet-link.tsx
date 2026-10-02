"use client";
import { useState } from "react";
import { useSession } from "@/components/auth/session-provider";
import { RequireSession } from "@/components/auth/require-session";
import { authenticateWallet, type WalletKind } from "@/lib/wallet";
import { message } from "@/lib/api";
export function WalletLink() {
 const { user, refresh } = useSession();
 const [busy, setBusy] = useState(false);
 const [notice, setNotice] = useState("");
 async function link(kind: WalletKind) {
  setBusy(true); setNotice("");
  try { await authenticateWallet(kind, true); await refresh(); setNotice("کیف پول با موفقیت متصل شد."); }
  catch (error) { setNotice(message(error)); }
  finally { setBusy(false); }
 }
 return <RequireSession><section className="panel mx-auto mt-6 max-w-2xl"><h2 className="text-xl font-bold">کیف پول حساب</h2>
  {user?.wallet_address ? <p className="mt-4 break-all" dir="ltr">{user.wallet_address}</p> : <div className="mt-4 flex gap-3"><button className="primary" disabled={busy} onClick={() => void link("metamask")}>اتصال MetaMask</button><button className="secondary" disabled={busy} onClick={() => void link("walletconnect")}>اتصال WalletConnect</button></div>}
  {notice && <p className="mt-4" role="status">{notice}</p>}
 </section></RequireSession>;
}

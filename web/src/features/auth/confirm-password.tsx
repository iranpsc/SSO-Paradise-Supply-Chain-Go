"use client";
import { useState, type FormEvent } from "react";
import { useSearchParams } from "next/navigation";
import { RequireSession } from "@/components/auth/require-session";
import { Field } from "@/components/ui/field";
import { api, message } from "@/lib/api";
import { oauthReturnTo } from "@/lib/navigation";
export function ConfirmPassword() {
 const params=useSearchParams();const [busy,setBusy]=useState(false);const [error,setError]=useState("");
 async function submit(event:FormEvent<HTMLFormElement>){
  event.preventDefault();const password=new FormData(event.currentTarget).get("password");setBusy(true);setError("");
  try{await api("/password/confirm",{method:"POST",body:JSON.stringify({password})});window.location.assign(oauthReturnTo(params.get("return_to"))??"/home");}
  catch(error){setError(message(error));}finally{setBusy(false);}
 }
 return <RequireSession verified={false}><section className="panel mx-auto max-w-xl"><h1 className="text-2xl font-bold">تأیید رمز عبور</h1><form className="mt-6 space-y-4" onSubmit={submit}>
 <Field label="رمز عبور" name="password" type="password" autoComplete="current-password" required disabled={busy}/>
 <button className="primary" type="submit" disabled={busy}>تأیید و ادامه</button>{error&&<p role="alert">{error}</p>}
 </form></section></RequireSession>;
}

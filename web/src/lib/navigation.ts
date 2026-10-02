export function oauthReturnTo(value: string | null): string | null {
 if (!value || !value.startsWith("/") || value.startsWith("//")) return null;
 try {
  const target = new URL(value, window.location.origin);
  if (target.origin !== window.location.origin || target.pathname !== "/oauth/authorize") return null;
  return target.pathname + target.search;
 } catch { return null; }
}
export function verificationTarget(value?: string): string {
 const intended = oauthReturnTo(sessionStorage.getItem("sso:return_to"));
 if (intended) { sessionStorage.removeItem("sso:return_to"); return intended; }
 if (value) {
  try { const target = new URL(value, window.location.origin);
   if (target.origin === window.location.origin || (target.protocol === "https:" && target.host === "metarang.com" && !target.username && !target.password)) return target.href;
  } catch { /* Fall back to the account home. */ }
 }
 return "/home";
}

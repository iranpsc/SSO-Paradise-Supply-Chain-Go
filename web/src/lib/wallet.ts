import { api } from "@/lib/api";
export type WalletKind = "metamask" | "walletconnect";
type Provider = { request: (input: { method: string; params?: unknown[] }) => Promise<unknown> };
export async function authenticateWallet(kind: WalletKind, link = false) {
 let provider: Provider;
 if (kind === "metamask") {
  const injected = (window as unknown as { ethereum?: Provider }).ethereum;
  if (!injected) throw new Error("کیف پول وب۳ پیدا نشد. لطفاً متامسک یا یک کیف پول سازگار را نصب کنید.");
  provider = injected;
 } else {
  const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID;
  if (!projectId) throw new Error("اتصال والت‌کانکت برای این سامانه آماده نیست.");
  const { default: EthereumProvider } = await import("@walletconnect/ethereum-provider");
  const connection = await EthereumProvider.init({ projectId, chains: [1], showQrModal: true });
  await connection.connect();
  provider = connection;
 }
 const accounts = await provider.request({ method: "eth_requestAccounts" });
 if (!Array.isArray(accounts) || typeof accounts[0] !== "string") throw new Error("آدرس کیف پول دریافت نشد.");
 const address = accounts[0];
 const path = link ? "/web3/link" : "/web3";
 const { nonce } = await api<{ nonce: string }>(`${path}/nonce?address=${encodeURIComponent(address)}`);
 const hex = "0x" + Array.from(new TextEncoder().encode(nonce), byte => byte.toString(16).padStart(2, "0")).join("");
 const signature = await provider.request({ method: "personal_sign", params: [hex, address] });
 if (typeof signature !== "string") throw new Error("امضای کیف پول دریافت نشد.");
 return api<{ message: string; redirect: string }>(link ? "/web3/link" : "/web3/verify", { method: "POST", body: JSON.stringify({ address, signature }) });
}

import { test, expect } from "@playwright/test";
test("MetaMask connects, signs the server nonce and opens the authenticated account",async({page})=>{
 const address="0x"+"a".repeat(40);const signature="0x"+"b".repeat(130);const nonce="Sign in to Laravel at localhost.\n\nWallet: "+address+"\nNonce: test";
 let loggedIn=false;let signed="";
 await page.addInitScript(({address,signature})=>{Object.defineProperty(window,"ethereum",{value:{request:async(input:{method:string;params?:string[]})=>{
 if(input.method==="eth_requestAccounts")return[address];
 if(input.method==="personal_sign"){sessionStorage.setItem("test:signed",input.params?.[0]??"");return signature;}
 throw new Error("Unexpected wallet method");
 }}});},{address,signature});
 await page.route("**/api/account",route=>route.fulfill(loggedIn?{json:{data:{id:1,name:"Wallet member",username:"",email:"",code:"hm-2000001",wallet_address:address,email_verified_at:new Date().toISOString(),created_at:new Date().toISOString()}}}:{status:401,json:{message:"Unauthenticated"}}));
 await page.route("**/api/web3/nonce?*",route=>{expect(new URL(route.request().url()).searchParams.get("address")).toBe(address);return route.fulfill({json:{nonce}});});
 await page.route("**/api/web3/verify",async route=>{expect(route.request().postDataJSON()).toEqual({address,signature});signed=await page.evaluate(()=>sessionStorage.getItem("test:signed")??"");loggedIn=true;await route.fulfill({json:{message:"Authenticated successfully",redirect:"http://localhost:3100/home"}});});
 await page.goto("/login");await page.getByRole("button",{name:"ورود با MetaMask"}).click();await expect(page).toHaveURL(/\/home$/);
 expect(signed).toBe("0x"+Buffer.from(nonce,"utf8").toString("hex"));
});

import { execFileSync } from "node:child_process";
export default async function setup() {
 return async () => { execFileSync("go", ["run", "../cmd/testdb", "--drop"], { env: process.env, stdio: "inherit" }); };
}

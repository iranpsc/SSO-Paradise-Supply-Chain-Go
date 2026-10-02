# Manual frontend test data

From the project root, with the local WAMP `.env` loaded:

```powershell
go run ./cmd/seed-local
# Validate username/email login and real signatures for the two existing wallets:
go run ./cmd/seed-local --check
```

The command only supports development mode, a loopback MySQL server and a separate target database. It creates five demo accounts and generates five random secp256k1 wallet keys. Re-running preserves the same keys and existing accounts; it does not reset passwords, verification status or wallet changes made while testing.

Open `var/test-data/frontend-test-guide.md` for the actual accounts, addresses and importable private keys. `var/test-data/credentials.json` is the machine-readable manifest. These artifacts are Git-ignored. All initial passwords are `TestPass!2026`.

| Account | Email | Initial behavior |
|---|---|---|
| demo_verified | verified@example.test | Login using username or email; opens home |
| demo_pending | pending@example.test | Login opens verification notice |
| demo_wallet | wallet@example.test | Verified account with an existing wallet |
| demo_wallet_pending | wallet.pending@example.test | Unverified account with an existing wallet |
| demo_member | member@example.test | Verified member; wallet attaches through simulated Metarang member code |

Register `new_member` / `new.member@example.test` using the frontend. This account is deliberately not seeded. Use a fresh suffix for later registrations. Use an existing demo username/email to check duplicate validation, and an incorrect password to check rejected login; after five failures for the same identifier/IP, wait one minute. Password reset and verification mail are written to `var/mail`; open the URL from the text file in the same browser session where required.

Wallet keys can be imported into separate MetaMask test accounts. The same Ethereum account can be used through WalletConnect with a compatible mobile wallet. These keys are exclusively for local testing; do not fund them. In the generated guide, `wallet_login` and `wallet_unverified` are already linked. `wallet_signup` is unregistered and creates an account on its first successful signature. `wallet_link` is unused and can be attached after logging in as `demo_verified`. `metarang_member` initially has no local wallet owner; the fixture lookup maps it to `demo_member` by member code. Logout between login/registration cases.

Try connecting `wallet_login` from `demo_verified` to check ownership conflict. Repeat a link to check the already-connected state. A rejected signature and an expired nonce must fail independently of the fixture lookup. Wallet-only registration leads to the verification screen, matching the original application flow.

For isolated local membership lookup, set `METARANG_LOCAL_FIXTURES=var/test-data/metarang.json` in the private local `.env` and restart API. This emulates only Metarang's registration lookup: unknown wallets are treated as fresh, and the one fixture member resolves to its demo account. Cryptographic signatures, single-use nonces, ownership and MySQL persistence use the normal code. Production/staging startup rejects this setting. Clear it and restart to restore the real configured Metarang lookup.

This seed data exercises the local frontend and backend. Actual WalletConnect pairing and a live Metarang integration still depend on the wallet client and network.

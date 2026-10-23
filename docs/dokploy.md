# Independent Dokploy deployments

Use a separate Compose application for each repository. In **both**, set Compose Path to `./docker-compose.yml`. The backend `compose.yaml` is an optional backend-only stack with managed MySQL/Redis; no Compose file in this repository builds the frontend. Do not add `compose.dokploy.yaml` to the independent `docker-compose.yml`.

## Backend

Select this backend repository and the branch containing `docker-compose.yml`. Set its environment in Dokploy; Dokploy writes it to `.env`. The file starts only `migrate`, `api`, and `worker`. It does not create MySQL or Redis and does not need MYSQL_ROOT_PASSWORD or the frontend repository. Keep the actual application/SMTP/database secrets in Dokploy's environment.

Set MYSQL_HOST to the reachable MySQL service hostname and REDIS_ADDR to the reachable Redis service hostname plus port. Docker's `127.0.0.1` refers to the application container, not the server or another container. For Dokploy-managed databases, use their internal hostnames and ensure they share `dokploy-network`. Use their actual credentials; an empty REDIS_PASSWORD is supported only if your external Redis does not require authentication. Existing database creation/grants are not performed by this Compose file; the migration command creates application tables in the configured target database.

Provision the existing Laravel OAuth RSA private key on the deployment server at `/etc/dokploy/secrets/sso-go/oauth-private.key`. It must be readable by container UID 10001. This directory is mounted read-only into all backend services, and Compose overrides OAUTH_PRIVATE_KEY_PATH to `/run/sso-secrets/oauth-private.key`. To use another server directory, set SSO_SECRETS_DIR to that absolute path. Keep it outside the git checkout so redeploys preserve it. Never commit the private key.

Add the backend domain `apidev-accounts.irpsc.com` in Dokploy, targeting service **api**, container port **8080**, with HTTPS. Set PUBLIC_URL to `https://dev-accounts.irpsc.com` (the browser-facing frontend origin). Set TRUSTED_PROXY_CIDRS to the actual trusted reverse proxy's network/IP range; do not blindly use the combined stack's subnet. API and worker start only after migrations succeed. If startup fails, inspect migrate first, then api/worker logs.

## Frontend

Select the frontend repository separately and use its `./docker-compose.yml`. Set these in the Dokploy Environment panel:

```dotenv
NODE_ENV=production
API_ORIGIN=https://apidev-accounts.irpsc.com
PUBLIC_URL=https://dev-accounts.irpsc.com
NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID=YOUR_PUBLIC_PROJECT_ID
```

The Compose file forwards the three public configuration values as build arguments automatically. No backend secrets belong in the frontend environment. Add domain `dev-accounts.irpsc.com`, targeting service **web**, container port **3000**, with HTTPS. Deploy the backend first, then the frontend. Redeploy/rebuild the frontend when API_ORIGIN or the public project ID changes.

Both applications require Dokploy's existing external `dokploy-network`. Neither publishes host ports; Dokploy's reverse proxy reaches the container ports through that network.

## HTTP-only production

Keep `APP_ENV=production`; explicitly opt into HTTP on the backend:

```dotenv
APP_ENV=production
ALLOW_INSECURE_HTTP=true
PUBLIC_URL=http://dev-accounts.irpsc.com
```

Keep the existing APP_KEY, OAuth private key, SMTP TLS and other production settings. The opt-in only relaxes the public URL's HTTPS requirement. HTTP public URLs produce cookies without Secure and no application HSTS header. HTTPS URLs retain Secure cookies and HSTS, even when the opt-in is enabled. Passwords and session credentials sent over HTTP are unencrypted.

Set the frontend build environment:

```dotenv
NODE_ENV=production
PUBLIC_URL=http://dev-accounts.irpsc.com
API_ORIGIN=http://apidev-accounts.irpsc.com
```

Keep NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID unchanged. Rebuild/redeploy both applications. In Dokploy, use HTTP for both domains and disable forced HTTPS redirects or proxy-added HSTS. Configure the connected systems to use the HTTP SSO URL; OAuth callback URLs must exactly match each client's registered redirect URI. Existing browser HSTS (including inherited parent-domain policies) can still force HTTPS; removing application headers does not clear it. A parent HSTS policy must be resolved at that domain before its subdomains can use HTTP. Changing web transport does not disable SMTP TLS or HTTPS calls to external services.

# Independent Dokploy deployments

Use a separate Compose application for each repository. In **both**, set Compose Path to `./docker-compose.yml`. The existing `compose.yaml` is the optional combined stack with managed MySQL/Redis and the sibling frontend checkout; do not use it for independent deployments. Do not add `compose.dokploy.yaml` to these independent files.

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

# Troubleshooting

## Database Connection Failed

```
Error: failed to connect to database
```

Ensure PostgreSQL is running and `DATABASE_URL` is correct:

```bash
make db-up
export DATABASE_URL="postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable"
```

## OIDC Discovery Failed

```
Error: failed to get provider: Get ".../.well-known/openid-configuration": dial tcp: lookup ... no such host
```

Verify `OIDC_ISSUER` is correct and reachable. For local development:

```bash
make services-up  # Starts mock OIDC server
```

## Session Secret Too Short

```
Warning: SESSION_SECRET is less than 32 characters
```

Generate a secure secret:

```bash
export SESSION_SECRET=$(openssl rand -base64 32)
```

## Port Already in Use

```
Error: listen tcp :3000: bind: address already in use
```

Change the port or stop the conflicting process:

```bash
export SERVER_ADDR=:3001
# or
lsof -i :3000 | grep LISTEN | awk '{print $2}' | xargs kill
```

## Links Not Resolving

- Verify the link status is `approved`
- Confirm the scope matches the user's organization
- Check resolution priority: personal > org > global
- For org links, ensure the user belongs to the correct organization

## OIDC Claims Not Populating

- Enable development mode for debug logging: `ENV=development`
- Verify the OIDC provider returns the expected claims
- Check that `OIDC_ORG_CLAIM` matches the actual claim name from the provider
- GoLinks fetches claims from both the ID token and the userinfo endpoint

## No Request Logs Visible

If you see application startup messages but no per-request log lines, the Fiber request logger may be writing to a stream your log collector is not capturing.

GoLinks writes **all** logs (application + request) to **stderr**. Ensure your log collector captures stderr. You can also increase verbosity:

```bash
LOG_LEVEL=debug
```

## 500 Errors on Static Files

Static file requests returning 500 errors typically indicate middleware ordering or session issues. The error handler logs every error with structured context (status, method, path, IP, error message) to stderr.

1. **Check pod logs for error details:**
   ```bash
   kubectl logs -l app=golinks
   oc logs -l app=golinks
   ```
   Look for lines like `level=ERROR msg="request error" path=/static/...`.

2. **Enable debug logging** to see middleware registration and request details:
   ```bash
   LOG_LEVEL=debug
   ```

3. **Verify the static directory exists** inside the container:
   ```bash
   kubectl exec deploy/golinks -- ls -la /app/static/
   ```

## 500 Errors on OpenShift / Kubernetes

When deploying to OpenShift or Kubernetes, 500 errors can indicate:

1. **Check pod logs first:**
   ```bash
   kubectl logs -l app=golinks
   oc logs -l app=golinks
   ```

2. **Verify ConfigMap rendered correctly:**
   ```bash
   helm template golinks ./chart/golinks --values your-values.yaml | grep -A 20 'kind: ConfigMap'
   ```
   Ensure all environment variables have values (not empty strings). Common issues:
   - Wrong value paths in Helm template (e.g., `config.siteTitle` → `config.branding.title`)
   - Missing required secrets (`existingSecret` not set)

3. **Confirm image declares non-root USER:**
   The Dockerfile must include `USER 1001` and set group-0 ownership for OpenShift compatibility:
   ```dockerfile
   RUN chown -R 0:0 /app && chmod -R g+rX /app
   USER 1001
   ```

4. **Check probe paths:**
   Liveness and readiness probes must point to `/healthz` and `/readyz` (not `/`), as the root path requires authentication and will redirect to OIDC login.

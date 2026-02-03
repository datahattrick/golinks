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

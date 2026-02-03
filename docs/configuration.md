# Configuration

All settings are configured via environment variables. Copy `.env.example` to `.env` for local development.

## Core Settings

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ENV` | Environment mode (`development`, `production`) | `development` | No |
| `SERVER_ADDR` | Server listen address | `:3000` | No |
| `BASE_URL` | Public base URL for redirects | `http://localhost:3000` | No |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://localhost:5432/golinks` | Yes |
| `SESSION_SECRET` | Cookie signing secret (min 32 chars) | - | Yes (production) |

## OIDC Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OIDC_ISSUER` | OIDC provider URL | (disabled if empty) | No |
| `OIDC_CLIENT_ID` | OIDC client ID | - | If OIDC enabled |
| `OIDC_CLIENT_SECRET` | OIDC client secret | - | If OIDC enabled |
| `OIDC_REDIRECT_URL` | OAuth callback URL | `http://localhost:3000/auth/callback` | If OIDC enabled |
| `OIDC_ORG_CLAIM` | Claim name for organization extraction | `organisation` | No |

### Provider Examples

**Google Workspace:**
```bash
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id.apps.googleusercontent.com
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=hd  # Google uses 'hd' for hosted domain
```

**Microsoft Entra ID (Azure AD):**
```bash
OIDC_ISSUER=https://login.microsoftonline.com/YOUR_TENANT_ID/v2.0
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=tid  # Tenant ID
```

**Okta:**
```bash
OIDC_ISSUER=https://your-domain.okta.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=organization
```

**Keycloak:**
```bash
OIDC_ISSUER=https://keycloak.example.com/realms/your-realm
OIDC_CLIENT_ID=golinks
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=organisation
```

**Development (Mock Server):**
```bash
make dev  # Starts mock OIDC server on port 8080
```
The mock server provides interactive login — enter any username/password.

## Site Branding

| Variable | Description | Default |
|----------|-------------|---------|
| `SITE_TITLE` | Site title displayed in header | `GoLinks` |
| `SITE_TAGLINE` | Tagline displayed on home page | `Fast URL shortcuts for your team` |
| `SITE_FOOTER` | Footer text | `GoLinks - Fast URL shortcuts for your team` |
| `SITE_LOGO_URL` | URL to logo image | (text only if empty) |

Example:
```bash
SITE_TITLE=MyCompany Links
SITE_TAGLINE=Internal URL shortcuts for the team
SITE_LOGO_URL=https://example.com/logo.png
SITE_FOOTER=Powered by GoLinks
```

## Feature Flags

| Variable | Description | Default |
|----------|-------------|---------|
| `ENABLE_RANDOM_KEYWORDS` | Enable "I'm Feeling Lucky" feature | `false` |
| `ENABLE_PERSONAL_LINKS` | Enable personal link scopes | `true` |
| `ENABLE_ORG_LINKS` | Enable organization link scopes | `true` |
| `CORS_ORIGINS` | Comma-separated allowed CORS origins | (none) |

**Simple Mode**: When both `ENABLE_PERSONAL_LINKS` and `ENABLE_ORG_LINKS` are `false`, only global links are available and `/go/:keyword` does not require authentication.

## Organization Fallbacks

| Variable | Description | Default |
|----------|-------------|---------|
| `ORG_FALLBACKS` | Per-org fallback redirect URLs | (none) |

Format: `org_slug=fallback_url,org2_slug=fallback_url2`

```bash
ORG_FALLBACKS=acme=https://other-golinks.acme.com/go/,corp=https://corp-links.example.com/go/
```

When a keyword is not found, users in the specified org are redirected to the fallback URL with the keyword appended.

## TLS / mTLS

| Variable | Description | Default |
|----------|-------------|---------|
| `TLS_ENABLED` | Enable TLS | (disabled if empty) |
| `TLS_CERT_FILE` | Path to TLS certificate | - |
| `TLS_KEY_FILE` | Path to TLS private key | - |
| `TLS_CA_FILE` | CA file for client cert verification (mTLS) | - |
| `CLIENT_CERT_HEADER` | Header containing client cert CN (for ingress-terminated TLS) | - |

```bash
# Basic TLS
TLS_ENABLED=true
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem

# Mutual TLS
TLS_CA_FILE=/path/to/ca.pem

# Ingress-terminated with cert forwarding
CLIENT_CERT_HEADER=X-Client-CN
```

## Email Notifications (SMTP)

| Variable | Description | Default |
|----------|-------------|---------|
| `SMTP_ENABLED` | Enable email notifications | (disabled if empty) |
| `SMTP_HOST` | SMTP server hostname | - |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USERNAME` | SMTP authentication username | - |
| `SMTP_PASSWORD` | SMTP authentication password | - |
| `SMTP_FROM` | Sender email address | - |
| `SMTP_FROM_NAME` | Sender display name | `GoLinks` |
| `SMTP_TLS` | TLS mode: `none`, `starttls`, `tls` | `starttls` |

### Notification Controls

| Variable | Description | Default |
|----------|-------------|---------|
| `EMAIL_NOTIFY_MODS_ON_SUBMIT` | Notify moderators on new submissions | `true` |
| `EMAIL_NOTIFY_USER_ON_APPROVAL` | Notify users when links are approved | `true` |
| `EMAIL_NOTIFY_USER_ON_REJECTION` | Notify users when links are rejected | `true` |
| `EMAIL_NOTIFY_USER_ON_DELETION` | Notify users when links are deleted | `true` |
| `EMAIL_NOTIFY_MODS_ON_HEALTH_FAILURE` | Notify moderators on health check failures | `true` |

| Event | Recipients |
|-------|------------|
| Link Submitted | Moderators |
| Link Approved | Submitter |
| Link Rejected | Submitter |
| Link Deleted | Creator |
| Health Check Failed | Moderators |

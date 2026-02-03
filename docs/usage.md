# Usage

## Creating Links

1. Log in via the Login button
2. Use the form at the top of the page
3. Fill in:
   - **Keyword** — the short name (e.g., `docs`, `wiki`, `hr`)
   - **URL** — the full destination URL
   - **Description** — (optional) context about the link
   - **Scope** — Global, Organization, or Personal

The submit button changes label based on scope and your role: it shows "Create Link" if you have permission to bypass approval, or "Submit Link" if the link will enter the pending moderation queue.

## Link Scopes

| Scope | Visibility | Use Case |
|-------|------------|----------|
| **Global** | All users | Company-wide resources |
| **Organization** | Organization members | Team-specific links |
| **Personal** | Only you | Private shortcuts |

## Resolution Priority

When a keyword is requested, GoLinks resolves in this order:

1. **Personal links** (highest priority)
2. **Organization links**
3. **Global links** (lowest priority)

Organization links shadow global links with the same keyword, and personal links shadow both.

## Searching

Type in the search box for instant results via HTMX. Search matches against keywords, URLs, and descriptions using PostgreSQL trigram matching for fuzzy results.

## Redirects

Links are accessible via two URL patterns:

| Pattern | Example |
|---------|---------|
| `/go/:keyword` | `http://go.example.com/go/docs` |
| `/:keyword` | `http://go.example.com/docs` |

Both patterns redirect (HTTP 302) to the same destination. The JSON API endpoint `/api/v1/resolve/:keyword` returns the URL without redirecting.

## Click Tracking

Every redirect increments the link's click count. The home page displays the top-used links with 24-hour sparkline graphs showing hourly click activity.

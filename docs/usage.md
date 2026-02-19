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

Links are accessed via the `/go/:keyword` URL pattern:

| Pattern | Example |
|---------|---------|
| `/go/:keyword` | `http://go.example.com/go/docs` |

This redirects (HTTP 302) to the destination URL. The JSON API endpoint `/api/v1/resolve/:keyword` returns the URL without redirecting.

## Sharing Links

You can share personal links with other users from the **My Links** page:

1. Scroll to the **Share a Link** section (or click the **Share** button on any existing personal link to pre-fill the form)
2. Search for recipients by name or email — select one or more users
3. Fill in the keyword, URL, and optional description
4. Click **Share**

The recipient sees the shared link under **Shared With You** on their My Links page. They can:

- **Accept** — copies the link into their personal links
- **Decline** — removes the offer

As the sender, you can **Withdraw** any pending outgoing share.

### Anti-Spam Limits

| Constraint | Limit |
|------------|-------|
| Pending outgoing shares per user | 5 |
| Pending incoming shares per user | 5 |
| Self-sharing | Blocked |
| Duplicate (same sender + recipient + keyword) | Blocked |

## "Did You Mean?" Suggestions

When navigating to a keyword that doesn't exist, GoLinks shows a "Not Found" page with:

- **Fuzzy suggestions** — similar keywords ranked by trigram similarity (using the `pg_trgm` extension), displayed as clickable cards
- **Browse link** — links to `/browse?q=<keyword>` with the attempted keyword pre-filled in the filter
- **Go Home link**

If the user has a fallback redirect configured (see below), they are redirected to the fallback URL instead of seeing the not-found page.

## Fallback Redirects

Admins can configure named fallback redirect options per organization at `/admin/fallback-redirects`. Users choose a fallback in their profile page. When a keyword is not found, users with a fallback selected are redirected to that URL with the keyword appended (e.g., `https://other-golinks.example.com/go/<keyword>`).

Fallback options can also be seeded from the `REDIRECT_FALLBACKS` environment variable on startup.

## Click Tracking

Every redirect increments the link's click count. The home page displays the top-used links with 24-hour sparkline graphs showing hourly click activity.

# Administration

## User Roles

| Role | Permissions |
|------|-------------|
| `user` | Create personal links, view approved links |
| `org_mod` | Moderate links within their organization |
| `global_mod` | Moderate all links (global + all organizations) |
| `admin` | Full access including user and org management |

## Moderation Workflow

Links may require approval before becoming active:

1. User submits a new link
2. Link enters `pending` status
3. Moderator reviews and approves or rejects
4. Approved links become active immediately

The moderation queue is accessible from the navigation menu. Global moderators and admins can moderate all pending links, including organization-specific ones.

The submit button on the create form adapts to context: it reads "Create Link" when the user has the permissions to bypass approval, and "Submit Link" when the link will enter the pending queue.

## User Management

Administrators can manage users at `/admin/users`:

- View all users
- Assign organizations
- Change user roles
- Delete users

## Organization Management

Organizations are automatically created when:
- A user authenticates with a new organization claim (controlled by `OIDC_ORG_CLAIM`)
- Fallback URLs are configured via `ORG_FALLBACKS`

Organizations support:
- **Fallback redirect URL** — when a keyword isn't found, redirect to another GoLinks instance
- **Per-org colored badges** on the manage page for quick visual identification
- **Moderator scoping** — org mods only see and manage links within their organization

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

### Automatic Role Assignment (OIDC Groups)

Roles can be automatically derived from OIDC group claims, eliminating manual role management:

```bash
OIDC_GROUPS_CLAIM=groups
OIDC_ADMIN_GROUPS=golinks-admin
OIDC_MODERATOR_GROUPS=golinks-moderator
```

**Role Resolution (on every login):**
- Users in `OIDC_ADMIN_GROUPS` → `admin`
- Users in `OIDC_MODERATOR_GROUPS` with an org → `org_mod` (scoped to their org)
- Users in `OIDC_MODERATOR_GROUPS` without an org → `global_mod`
- Others → `user`

**Note:** When group-to-role mapping is enabled, roles are updated on every login. Manual role changes will be overridden the next time the user logs in. If you need manual control, leave `OIDC_ADMIN_GROUPS` and `OIDC_MODERATOR_GROUPS` empty.

## Organization Management

Organizations are automatically created when:
- A user authenticates with a new organization claim (controlled by `OIDC_ORG_CLAIM`)
- Fallback redirects are configured via `REDIRECT_FALLBACKS`

Organizations support:
- **Fallback redirects** — admins manage named fallback options per org at `/admin/fallback-redirects`; users choose one in their profile (default: none). When a keyword isn't found, users with a fallback selected are redirected to that URL with the keyword appended.
- **Per-org colored badges** on the manage page for quick visual identification
- **Moderator scoping** — org mods only see and manage links within their organization

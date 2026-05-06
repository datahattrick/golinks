# Feature Requests

## Passthrough (Sub-keyword) Links

**Request:** Allow a keyword to act as a passthrough to another golinks instance or external URL shortener. For example, `go/bin/google` would resolve the `bin` keyword and redirect to `https://bin.io/go/google` — appending the sub-keyword to the base URL.

**Use case:** Teams that run multiple golinks instances or have a preferred external shortener can bridge between them transparently.

**Design notes:**
- Add `is_passthrough boolean NOT NULL DEFAULT false` to the `links` table
- New route `/go/:keyword/*` captures the tail after the keyword
- Resolution: when a passthrough link matches, append the validated tail to the base URL and redirect
- The base URL should end with `/` or have a clear append point
- The tail must be validated before appending: reject `..`, leading `/`, and absolute/protocol-relative URLs; percent-encode the result
- Passthrough is an explicit per-link opt-in so operators control which links expose this behaviour
- UI needs a checkbox on the create/edit forms

**Security considerations:**
- Path traversal: reject tails containing `..` or starting with `/`
- URL injection: strip any protocol or authority components from the tail
- Treat the tail as untrusted user input; percent-encode before appending
- The existing keyword validator (alphanumeric + hyphens + slashes) can be extended to cover allowed tail characters

package email

import (
	"fmt"
	"html"
	"strings"

	"golinks/internal/config"
	"golinks/internal/models"
)

// Templates provides email template generation.
type Templates struct {
	cfg *config.Config
}

// NewTemplates creates a new templates instance.
func NewTemplates(cfg *config.Config) *Templates {
	return &Templates{cfg: cfg}
}

// baseHTML wraps content in a basic HTML email template.
func (t *Templates) baseHTML(title, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2563eb; color: white; padding: 20px; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; border-top: none; }
        .footer { background: #f3f4f6; padding: 15px; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 8px 8px; font-size: 12px; color: #6b7280; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 10px 0; }
        .button:hover { background: #1d4ed8; }
        .link-details { background: white; padding: 15px; border-radius: 6px; margin: 15px 0; border: 1px solid #e5e7eb; }
        .link-details dt { font-weight: 600; color: #374151; margin-top: 10px; }
        .link-details dd { margin: 5px 0 0 0; color: #6b7280; }
        .status-approved { color: #059669; font-weight: 600; }
        .status-rejected { color: #dc2626; font-weight: 600; }
        .status-pending { color: #d97706; font-weight: 600; }
        .warning { background: #fef3c7; border: 1px solid #f59e0b; padding: 10px; border-radius: 6px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s</h1>
    </div>
    <div class="content">
        %s
    </div>
    <div class="footer">
        <p>This is an automated message from %s.</p>
        <p>%s</p>
    </div>
</body>
</html>`, html.EscapeString(title), html.EscapeString(t.cfg.SiteTitle), content, html.EscapeString(t.cfg.SiteTitle), html.EscapeString(t.cfg.BaseURL))
}

// LinkSubmittedForReview generates an email for moderators when a new link is submitted.
func (t *Templates) LinkSubmittedForReview(link *models.Link, submitter *models.User) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] New link pending review: %s", t.cfg.SiteTitle, link.Keyword)

	scope := "Global"
	if link.Scope == models.ScopeOrg {
		scope = "Organization"
	}

	content := fmt.Sprintf(`
        <p>A new link has been submitted and requires your review.</p>
        <dl class="link-details">
            <dt>Keyword</dt>
            <dd><strong>%s</strong></dd>
            <dt>URL</dt>
            <dd><a href="%s">%s</a></dd>
            <dt>Scope</dt>
            <dd>%s</dd>
            <dt>Description</dt>
            <dd>%s</dd>
            <dt>Submitted by</dt>
            <dd>%s (%s)</dd>
        </dl>
        <p>
            <a href="%s/moderation" class="button">Review Link</a>
        </p>
    `, html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		html.EscapeString(link.URL),
		scope,
		html.EscapeString(link.Description),
		html.EscapeString(submitter.Name),
		html.EscapeString(submitter.Email),
		t.cfg.BaseURL)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`New Link Pending Review

A new link has been submitted and requires your review.

Keyword: %s
URL: %s
Scope: %s
Description: %s
Submitted by: %s (%s)

Review the link at: %s/moderation

--
%s
%s
`, link.Keyword, link.URL, scope, link.Description, submitter.Name, submitter.Email, t.cfg.BaseURL, t.cfg.SiteTitle, t.cfg.BaseURL)

	return
}

// LinkApproved generates an email for users when their link is approved.
func (t *Templates) LinkApproved(link *models.Link, approver *models.User) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Your link '%s' has been approved", t.cfg.SiteTitle, link.Keyword)

	content := fmt.Sprintf(`
        <p>Great news! Your link has been <span class="status-approved">approved</span> and is now active.</p>
        <dl class="link-details">
            <dt>Keyword</dt>
            <dd><strong>%s</strong></dd>
            <dt>URL</dt>
            <dd><a href="%s">%s</a></dd>
            <dt>Short Link</dt>
            <dd><a href="%s/go/%s">%s/go/%s</a></dd>
        </dl>
        <p>You can now use your short link to redirect to the destination URL.</p>
        <p>
            <a href="%s" class="button">Go to %s</a>
        </p>
    `, html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		html.EscapeString(link.URL),
		t.cfg.BaseURL, html.EscapeString(link.Keyword),
		t.cfg.BaseURL, html.EscapeString(link.Keyword),
		t.cfg.BaseURL, t.cfg.SiteTitle)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Link Approved

Great news! Your link has been approved and is now active.

Keyword: %s
URL: %s
Short Link: %s/go/%s

You can now use your short link to redirect to the destination URL.

--
%s
%s
`, link.Keyword, link.URL, t.cfg.BaseURL, link.Keyword, t.cfg.SiteTitle, t.cfg.BaseURL)

	return
}

// LinkRejected generates an email for users when their link is rejected.
func (t *Templates) LinkRejected(link *models.Link, reason string) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Your link '%s' was not approved", t.cfg.SiteTitle, link.Keyword)

	reasonHTML := ""
	reasonText := ""
	if reason != "" {
		reasonHTML = fmt.Sprintf(`
            <dt>Reason</dt>
            <dd>%s</dd>
        `, html.EscapeString(reason))
		reasonText = fmt.Sprintf("Reason: %s\n", reason)
	}

	content := fmt.Sprintf(`
        <p>Unfortunately, your link was <span class="status-rejected">not approved</span> by a moderator.</p>
        <dl class="link-details">
            <dt>Keyword</dt>
            <dd><strong>%s</strong></dd>
            <dt>URL</dt>
            <dd>%s</dd>
            %s
        </dl>
        <p>If you believe this was an error, please contact a moderator or submit a new link with any necessary corrections.</p>
        <p>
            <a href="%s/new" class="button">Submit New Link</a>
        </p>
    `, html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		reasonHTML,
		t.cfg.BaseURL)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Link Not Approved

Unfortunately, your link was not approved by a moderator.

Keyword: %s
URL: %s
%s
If you believe this was an error, please contact a moderator or submit a new link with any necessary corrections.

Submit a new link at: %s/new

--
%s
%s
`, link.Keyword, link.URL, reasonText, t.cfg.BaseURL, t.cfg.SiteTitle, t.cfg.BaseURL)

	return
}

// LinkDeleted generates an email for users when their link is deleted.
func (t *Templates) LinkDeleted(link *models.Link, reason string) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Your link '%s' has been removed", t.cfg.SiteTitle, link.Keyword)

	reasonHTML := ""
	reasonText := ""
	if reason != "" {
		reasonHTML = fmt.Sprintf(`
            <dt>Reason</dt>
            <dd>%s</dd>
        `, html.EscapeString(reason))
		reasonText = fmt.Sprintf("Reason: %s\n", reason)
	}

	content := fmt.Sprintf(`
        <p>Your link has been removed from the system.</p>
        <dl class="link-details">
            <dt>Keyword</dt>
            <dd><strong>%s</strong></dd>
            <dt>URL</dt>
            <dd>%s</dd>
            %s
        </dl>
        <p>If you have questions about this removal, please contact an administrator.</p>
    `, html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		reasonHTML)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Link Removed

Your link has been removed from the system.

Keyword: %s
URL: %s
%s
If you have questions about this removal, please contact an administrator.

--
%s
%s
`, link.Keyword, link.URL, reasonText, t.cfg.SiteTitle, t.cfg.BaseURL)

	return
}

// HealthCheckFailed generates an email for moderators when a link's health check fails.
func (t *Templates) HealthCheckFailed(links []models.Link) (subject, htmlBody, textBody string) {
	count := len(links)
	subject = fmt.Sprintf("[%s] %d link(s) failing health checks", t.cfg.SiteTitle, count)

	var linkListHTML strings.Builder
	var linkListText strings.Builder

	for _, link := range links {
		errMsg := "Unknown error"
		if link.HealthError != nil {
			errMsg = *link.HealthError
		}

		linkListHTML.WriteString(fmt.Sprintf(`
            <div class="link-details">
                <dt>Keyword</dt>
                <dd><strong>%s</strong></dd>
                <dt>URL</dt>
                <dd><a href="%s">%s</a></dd>
                <dt>Error</dt>
                <dd class="status-rejected">%s</dd>
            </div>
        `, html.EscapeString(link.Keyword),
			html.EscapeString(link.URL),
			html.EscapeString(link.URL),
			html.EscapeString(errMsg)))

		linkListText.WriteString(fmt.Sprintf("- %s (%s): %s\n", link.Keyword, link.URL, errMsg))
	}

	content := fmt.Sprintf(`
        <div class="warning">
            <strong>⚠️ Health Check Alert</strong>
        </div>
        <p>The following %d link(s) are currently failing health checks:</p>
        %s
        <p>
            <a href="%s/manage?filter=unhealthy" class="button">View Unhealthy Links</a>
        </p>
    `, count, linkListHTML.String(), t.cfg.BaseURL)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Health Check Alert

The following %d link(s) are currently failing health checks:

%s
View unhealthy links at: %s/manage?filter=unhealthy

--
%s
%s
`, count, linkListText.String(), t.cfg.BaseURL, t.cfg.SiteTitle, t.cfg.BaseURL)

	return
}

// WelcomeUser generates a welcome email for new users.
func (t *Templates) WelcomeUser(user *models.User) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("Welcome to %s!", t.cfg.SiteTitle)

	content := fmt.Sprintf(`
        <p>Hi %s,</p>
        <p>Welcome to %s! Your account has been created successfully.</p>
        <p>You can now:</p>
        <ul>
            <li>Create short links for quick access to your favorite URLs</li>
            <li>Search existing links to find what you need</li>
            <li>Manage your personal link collection</li>
        </ul>
        <p>
            <a href="%s" class="button">Get Started</a>
        </p>
    `, html.EscapeString(user.Name), html.EscapeString(t.cfg.SiteTitle), t.cfg.BaseURL)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Welcome to %s!

Hi %s,

Welcome to %s! Your account has been created successfully.

You can now:
- Create short links for quick access to your favorite URLs
- Search existing links to find what you need
- Manage your personal link collection

Get started at: %s

--
%s
%s
`, t.cfg.SiteTitle, user.Name, t.cfg.SiteTitle, t.cfg.BaseURL, t.cfg.SiteTitle, t.cfg.BaseURL)

	return
}

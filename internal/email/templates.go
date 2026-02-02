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

// baseHTML wraps content in a consistent HTML email template.
func (t *Templates) baseHTML(title, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2563eb; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
        .footer { background: #f3f4f6; padding: 15px; text-align: center; font-size: 12px; color: #6b7280; border-radius: 0 0 8px 8px; border: 1px solid #e5e7eb; border-top: none; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 10px 0; }
        .button:hover { background: #1d4ed8; }
        .info-box { background: white; border: 1px solid #e5e7eb; border-radius: 6px; padding: 15px; margin: 15px 0; }
        .label { font-weight: 600; color: #374151; }
        .value { color: #6b7280; }
        .success { color: #059669; }
        .warning { color: #d97706; }
        .error { color: #dc2626; }
        code { background: #e5e7eb; padding: 2px 6px; border-radius: 4px; font-family: monospace; }
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
        <p>This email was sent by %s</p>
        <p><a href="%s">%s</a></p>
    </div>
</body>
</html>`, html.EscapeString(title), html.EscapeString(t.cfg.SiteTitle), content, html.EscapeString(t.cfg.SiteTitle), t.cfg.BaseURL, t.cfg.BaseURL)
}

// LinkSubmittedForReview generates email for moderators when a link needs review.
func (t *Templates) LinkSubmittedForReview(link *models.Link, submitter *models.User) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] New link pending review: %s", t.cfg.SiteTitle, link.Keyword)

	scope := "Global"
	if link.Scope == models.ScopeOrg {
		scope = "Organization"
	}

	content := fmt.Sprintf(`
        <p>A new link has been submitted and requires your review.</p>

        <div class="info-box">
            <p><span class="label">Keyword:</span> <code>%s</code></p>
            <p><span class="label">URL:</span> <a href="%s">%s</a></p>
            <p><span class="label">Scope:</span> %s</p>
            <p><span class="label">Description:</span> %s</p>
            <p><span class="label">Submitted by:</span> %s (%s)</p>
        </div>

        <p style="text-align: center;">
            <a href="%s/moderation" class="button">Review in Dashboard</a>
        </p>
    `,
		html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		html.EscapeString(link.URL),
		scope,
		html.EscapeString(link.Description),
		html.EscapeString(submitter.Name),
		html.EscapeString(submitter.Email),
		t.cfg.BaseURL,
	)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`New link pending review

Keyword: %s
URL: %s
Scope: %s
Description: %s
Submitted by: %s (%s)

Review at: %s/moderation

--
%s
%s`,
		link.Keyword,
		link.URL,
		scope,
		link.Description,
		submitter.Name,
		submitter.Email,
		t.cfg.BaseURL,
		t.cfg.SiteTitle,
		t.cfg.BaseURL,
	)

	return
}

// LinkApproved generates email for user when their link is approved.
func (t *Templates) LinkApproved(link *models.Link, approver *models.User) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Your link '%s' has been approved!", t.cfg.SiteTitle, link.Keyword)

	content := fmt.Sprintf(`
        <p>Great news! Your link has been approved and is now active.</p>

        <div class="info-box">
            <p><span class="label">Keyword:</span> <code>%s</code></p>
            <p><span class="label">URL:</span> <a href="%s">%s</a></p>
            <p><span class="label">Status:</span> <span class="success">Approved</span></p>
            <p><span class="label">Approved by:</span> %s</p>
        </div>

        <p>You can now use your link:</p>
        <p style="text-align: center;">
            <a href="%s/go/%s" class="button">%s/go/%s</a>
        </p>
    `,
		html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		html.EscapeString(link.URL),
		html.EscapeString(approver.Name),
		t.cfg.BaseURL,
		html.EscapeString(link.Keyword),
		t.cfg.BaseURL,
		html.EscapeString(link.Keyword),
	)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Your link has been approved!

Keyword: %s
URL: %s
Status: Approved
Approved by: %s

Your link is now active at: %s/go/%s

--
%s
%s`,
		link.Keyword,
		link.URL,
		approver.Name,
		t.cfg.BaseURL,
		link.Keyword,
		t.cfg.SiteTitle,
		t.cfg.BaseURL,
	)

	return
}

// LinkRejected generates email for user when their link is rejected.
func (t *Templates) LinkRejected(link *models.Link, rejector *models.User, reason string) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Your link '%s' was not approved", t.cfg.SiteTitle, link.Keyword)

	reasonHTML := ""
	reasonText := ""
	if reason != "" {
		reasonHTML = fmt.Sprintf(`<p><span class="label">Reason:</span> %s</p>`, html.EscapeString(reason))
		reasonText = fmt.Sprintf("\nReason: %s", reason)
	}

	content := fmt.Sprintf(`
        <p>Unfortunately, your link submission was not approved.</p>

        <div class="info-box">
            <p><span class="label">Keyword:</span> <code>%s</code></p>
            <p><span class="label">URL:</span> %s</p>
            <p><span class="label">Status:</span> <span class="error">Rejected</span></p>
            <p><span class="label">Reviewed by:</span> %s</p>
            %s
        </div>

        <p>If you believe this was a mistake, please contact a moderator or submit a new link with appropriate modifications.</p>

        <p style="text-align: center;">
            <a href="%s/new" class="button">Submit New Link</a>
        </p>
    `,
		html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		html.EscapeString(rejector.Name),
		reasonHTML,
		t.cfg.BaseURL,
	)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Your link was not approved

Keyword: %s
URL: %s
Status: Rejected
Reviewed by: %s%s

If you believe this was a mistake, please contact a moderator or submit a new link.

Submit new link: %s/new

--
%s
%s`,
		link.Keyword,
		link.URL,
		rejector.Name,
		reasonText,
		t.cfg.BaseURL,
		t.cfg.SiteTitle,
		t.cfg.BaseURL,
	)

	return
}

// LinkDeleted generates email for user when their link is deleted.
func (t *Templates) LinkDeleted(link *models.Link, deletedBy *models.User, reason string) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Your link '%s' has been removed", t.cfg.SiteTitle, link.Keyword)

	reasonHTML := ""
	reasonText := ""
	if reason != "" {
		reasonHTML = fmt.Sprintf(`<p><span class="label">Reason:</span> %s</p>`, html.EscapeString(reason))
		reasonText = fmt.Sprintf("\nReason: %s", reason)
	}

	content := fmt.Sprintf(`
        <p>Your link has been removed by a moderator.</p>

        <div class="info-box">
            <p><span class="label">Keyword:</span> <code>%s</code></p>
            <p><span class="label">URL:</span> %s</p>
            <p><span class="label">Removed by:</span> %s</p>
            %s
        </div>

        <p>If you have questions about this action, please contact a moderator.</p>
    `,
		html.EscapeString(link.Keyword),
		html.EscapeString(link.URL),
		html.EscapeString(deletedBy.Name),
		reasonHTML,
	)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Your link has been removed

Keyword: %s
URL: %s
Removed by: %s%s

If you have questions about this action, please contact a moderator.

--
%s
%s`,
		link.Keyword,
		link.URL,
		deletedBy.Name,
		reasonText,
		t.cfg.SiteTitle,
		t.cfg.BaseURL,
	)

	return
}

// HealthCheckFailed generates email for moderators when health checks fail.
func (t *Templates) HealthCheckFailed(links []models.Link) (subject, htmlBody, textBody string) {
	count := len(links)
	subject = fmt.Sprintf("[%s] %d link(s) failed health check", t.cfg.SiteTitle, count)

	var linksHTML strings.Builder
	var linksText strings.Builder

	for _, link := range links {
		errorMsg := "Unknown error"
		if link.HealthError != nil {
			errorMsg = *link.HealthError
		}

		linksHTML.WriteString(fmt.Sprintf(`
            <div class="info-box">
                <p><span class="label">Keyword:</span> <code>%s</code></p>
                <p><span class="label">URL:</span> <a href="%s">%s</a></p>
                <p><span class="label">Error:</span> <span class="error">%s</span></p>
            </div>
        `,
			html.EscapeString(link.Keyword),
			html.EscapeString(link.URL),
			html.EscapeString(link.URL),
			html.EscapeString(errorMsg),
		))

		linksText.WriteString(fmt.Sprintf("\n- %s: %s\n  Error: %s\n",
			link.Keyword,
			link.URL,
			errorMsg,
		))
	}

	content := fmt.Sprintf(`
        <p>The following %d link(s) failed their health check and may be broken:</p>
        %s
        <p style="text-align: center;">
            <a href="%s/manage?filter=unhealthy" class="button">Review Unhealthy Links</a>
        </p>
    `,
		count,
		linksHTML.String(),
		t.cfg.BaseURL,
	)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Health Check Alert

%d link(s) failed their health check:
%s
Review at: %s/manage?filter=unhealthy

--
%s
%s`,
		count,
		linksText.String(),
		t.cfg.BaseURL,
		t.cfg.SiteTitle,
		t.cfg.BaseURL,
	)

	return
}

// WeeklyDigest generates a weekly summary email for moderators.
func (t *Templates) WeeklyDigest(stats DigestStats) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("[%s] Weekly Digest", t.cfg.SiteTitle)

	content := fmt.Sprintf(`
        <p>Here's your weekly summary:</p>

        <div class="info-box">
            <p><span class="label">New links created:</span> %d</p>
            <p><span class="label">Links pending review:</span> %d</p>
            <p><span class="label">Links approved:</span> %d</p>
            <p><span class="label">Links rejected:</span> %d</p>
            <p><span class="label">Total clicks this week:</span> %d</p>
            <p><span class="label">Unhealthy links:</span> %d</p>
        </div>

        <p style="text-align: center;">
            <a href="%s" class="button">Go to Dashboard</a>
        </p>
    `,
		stats.NewLinks,
		stats.PendingReview,
		stats.Approved,
		stats.Rejected,
		stats.TotalClicks,
		stats.UnhealthyLinks,
		t.cfg.BaseURL,
	)

	htmlBody = t.baseHTML(subject, content)

	textBody = fmt.Sprintf(`Weekly Digest

New links created: %d
Links pending review: %d
Links approved: %d
Links rejected: %d
Total clicks this week: %d
Unhealthy links: %d

Dashboard: %s

--
%s
%s`,
		stats.NewLinks,
		stats.PendingReview,
		stats.Approved,
		stats.Rejected,
		stats.TotalClicks,
		stats.UnhealthyLinks,
		t.cfg.BaseURL,
		t.cfg.SiteTitle,
		t.cfg.BaseURL,
	)

	return
}

// DigestStats holds statistics for weekly digest emails.
type DigestStats struct {
	NewLinks       int
	PendingReview  int
	Approved       int
	Rejected       int
	TotalClicks    int
	UnhealthyLinks int
}

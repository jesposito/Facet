package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// TestimonialEmailData holds template data for the notification email.
type TestimonialEmailData struct {
	SiteName       string
	AuthorName     string
	AuthorTitle    string
	AuthorCompany  string
	Relationship   string
	ContentPreview string
	SubmittedAt    string
	ReviewURL      string
	RequestLabel   string
	// Translated strings
	T EmailStrings
}

// FindAdminEmails returns the list of admin email addresses.
// Checks ADMIN_EMAILS env var first, then falls back to querying users collection.
func FindAdminEmails(app core.App) []string {
	// Check env var first (same as auth.go)
	if envEmails := os.Getenv("ADMIN_EMAILS"); envEmails != "" {
		var emails []string
		for _, e := range strings.Split(envEmails, ",") {
			if trimmed := strings.TrimSpace(e); trimmed != "" {
				emails = append(emails, trimmed)
			}
		}
		if len(emails) > 0 {
			return emails
		}
	}

	// Fallback: query users with is_admin=true
	records, err := app.FindRecordsByFilter("users", "is_admin = true", "", 10, 0)
	if err != nil {
		// Last resort: get all users (single-user app)
		records, err = app.FindRecordsByFilter("users", "1=1", "", 10, 0)
		if err != nil {
			return nil
		}
	}

	var emails []string
	for _, r := range records {
		if email := r.Email(); email != "" {
			emails = append(emails, email)
		}
	}
	return emails
}

// SendTestimonialNotification sends an email notification to admins about a new testimonial.
// This is designed to be called in a goroutine — it logs errors but never returns them
// to avoid blocking the testimonial submission.
func SendTestimonialNotification(app core.App, record *core.Record, requestRecord *core.Record, baseURL string) {
	if !app.Settings().SMTP.Enabled {
		app.Logger().Info("Email notification: SMTP not configured, skipping testimonial notification")
		return
	}

	adminEmails := FindAdminEmails(app)
	if len(adminEmails) == 0 {
		app.Logger().Warn("Email notification: no admin emails found, skipping testimonial notification")
		return
	}

	locale := GetSiteLocale(app)
	t := GetEmailStrings(locale)

	// Build template data
	data := TestimonialEmailData{
		SiteName:       app.Settings().Meta.SenderName,
		AuthorName:     record.GetString("author_name"),
		AuthorTitle:    record.GetString("author_title"),
		AuthorCompany:  record.GetString("author_company"),
		Relationship:   record.GetString("relationship"),
		ContentPreview: truncate(record.GetString("content"), 300),
		SubmittedAt:    time.Now().Format("Jan 2, 2006 at 3:04 PM"),
		T:              t,
	}

	if requestRecord != nil {
		data.RequestLabel = requestRecord.GetString("label")
	}

	// Build review URL from the resolved base URL (from request headers)
	if baseURL != "" {
		data.ReviewURL = strings.TrimRight(baseURL, "/") + "/admin/testimonials"
	}

	// Render HTML
	html, err := renderTestimonialEmail(data)
	if err != nil {
		app.Logger().Error("Email notification: failed to render template", "error", err)
		return
	}

	// Build recipient list
	var toAddresses []mail.Address
	for _, email := range adminEmails {
		toAddresses = append(toAddresses, mail.Address{Address: email})
	}

	subject := t.NotifSubjectFrom + data.AuthorName
	if data.AuthorName == "" {
		subject = t.NotifSubjectFallback
	}

	message := &mailer.Message{
		From: mail.Address{
			Name:    app.Settings().Meta.SenderName,
			Address: app.Settings().Meta.SenderAddress,
		},
		To:      toAddresses,
		Subject: subject,
		HTML:    html,
		Text:    renderTestimonialPlainText(data),
	}

	client := app.NewMailClient()
	if err := client.Send(message); err != nil {
		app.Logger().Error("Email notification: failed to send testimonial notification",
			"error", err,
			"recipients", adminEmails,
		)
		return
	}

	app.Logger().Info("Email notification: testimonial notification sent",
		"author", data.AuthorName,
		"recipients", adminEmails,
		"locale", locale,
	)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func renderTestimonialPlainText(data TestimonialEmailData) string {
	t := data.T
	var b strings.Builder
	b.WriteString(t.NotifPlainHeader + "\n")
	b.WriteString(strings.Repeat("=", len([]rune(t.NotifPlainHeader))) + "\n\n")

	if data.AuthorName != "" {
		b.WriteString(t.NotifPlainFrom + " " + data.AuthorName + "\n")
	}
	if data.AuthorTitle != "" || data.AuthorCompany != "" {
		title := data.AuthorTitle
		if data.AuthorCompany != "" {
			if title != "" {
				title += t.NotifAt
			}
			title += data.AuthorCompany
		}
		b.WriteString(t.NotifPlainTitle + " " + title + "\n")
	}
	if data.Relationship != "" {
		b.WriteString(t.NotifPlainRelation + " " + data.Relationship + "\n")
	}
	if data.RequestLabel != "" {
		b.WriteString(t.NotifPlainRequest + " " + data.RequestLabel + "\n")
	}
	b.WriteString(t.NotifPlainSubmitted + " " + data.SubmittedAt + "\n\n")

	b.WriteString(t.NotifPlainContent + "\n")
	b.WriteString(data.ContentPreview + "\n\n")

	if data.ReviewURL != "" {
		b.WriteString(t.NotifPlainReview + " " + data.ReviewURL + "\n")
	}

	return b.String()
}

var testimonialEmailTemplate = template.Must(template.New("testimonial").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f5;padding:32px 16px;">
<tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1);">

<!-- Header -->
<tr><td style="background-color:#18181b;padding:24px 32px;">
<h1 style="margin:0;color:#ffffff;font-size:18px;font-weight:600;">{{.T.NotifHeader}}</h1>
</td></tr>

<!-- Body -->
<tr><td style="padding:32px;">

<!-- Author info -->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-bottom:24px;">
<tr><td>
<h2 style="margin:0 0 4px;color:#18181b;font-size:16px;font-weight:600;">{{.AuthorName}}</h2>
{{if or .AuthorTitle .AuthorCompany}}
<p style="margin:0 0 4px;color:#52525b;font-size:14px;">{{.AuthorTitle}}{{if and .AuthorTitle .AuthorCompany}}{{.T.NotifAt}}{{end}}{{.AuthorCompany}}</p>
{{end}}
{{if .Relationship}}
<p style="margin:0;color:#71717a;font-size:13px;">{{.Relationship}}</p>
{{end}}
</td></tr>
</table>

<!-- Content preview -->
<div style="padding:16px;background-color:#f9fafb;border-radius:6px;border-left:3px solid #3b82f6;margin-bottom:24px;">
<p style="margin:0;color:#374151;font-size:14px;line-height:1.6;">&#8220;{{.ContentPreview}}&#8221;</p>
</div>

<!-- Metadata -->
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-bottom:24px;font-size:13px;color:#6b7280;">
{{if .RequestLabel}}
<tr><td style="padding:4px 0;"><strong>{{.T.NotifRequestLabel}}</strong> {{.RequestLabel}}</td></tr>
{{end}}
<tr><td style="padding:4px 0;"><strong>{{.T.NotifSubmittedLabel}}</strong> {{.SubmittedAt}}</td></tr>
</table>

<!-- CTA button -->
{{if .ReviewURL}}
<table role="presentation" cellpadding="0" cellspacing="0">
<tr><td style="border-radius:6px;background-color:#3b82f6;">
<a href="{{.ReviewURL}}" target="_blank" style="display:inline-block;padding:12px 24px;color:#ffffff;font-size:14px;font-weight:600;text-decoration:none;">{{.T.NotifReviewButton}}</a>
</td></tr>
</table>
{{end}}

</td></tr>

<!-- Footer -->
<tr><td style="padding:16px 32px;background-color:#fafafa;border-top:1px solid #e5e7eb;">
<p style="margin:0;color:#9ca3af;font-size:12px;">{{printf .T.NotifFooter .SiteName}}</p>
</td></tr>

</table>
</td></tr>
</table>
</body>
</html>`))

func renderTestimonialEmail(data TestimonialEmailData) (string, error) {
	var buf bytes.Buffer
	if err := testimonialEmailTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// VerificationEmailData holds template data for the verification email.
type VerificationEmailData struct {
	SiteName   string
	AuthorName string
	VerifyURL  string
	ExpiresIn  string
	// Translated strings
	T EmailStrings
}

// SendVerificationEmail sends a verification email to a testimonial submitter.
// Unlike SendTestimonialNotification, this returns an error because the caller
// needs to inform the user if the email failed to send.
func SendVerificationEmail(app core.App, email string, authorName string, verifyURL string) error {
	if !app.Settings().SMTP.Enabled {
		return fmt.Errorf("SMTP is not configured")
	}

	locale := GetSiteLocale(app)
	t := GetEmailStrings(locale)

	data := VerificationEmailData{
		SiteName:   app.Settings().Meta.SenderName,
		AuthorName: authorName,
		VerifyURL:  verifyURL,
		ExpiresIn:  t.VerifyExpiresIn,
		T:          t,
	}

	html, err := renderVerificationEmail(data)
	if err != nil {
		return fmt.Errorf("failed to render verification email: %w", err)
	}

	message := &mailer.Message{
		From: mail.Address{
			Name:    app.Settings().Meta.SenderName,
			Address: app.Settings().Meta.SenderAddress,
		},
		To:      []mail.Address{{Address: email}},
		Subject: t.VerifySubject,
		HTML:    html,
		Text:    renderVerificationPlainText(data),
	}

	return app.NewMailClient().Send(message)
}

func renderVerificationPlainText(data VerificationEmailData) string {
	t := data.T
	var b strings.Builder
	b.WriteString(t.VerifyPlainHeader + "\n")
	b.WriteString(strings.Repeat("=", len([]rune(t.VerifyPlainHeader))) + "\n\n")

	if data.AuthorName != "" {
		b.WriteString(fmt.Sprintf(t.VerifyGreeting, data.AuthorName) + "\n\n")
	}

	b.WriteString(t.VerifyPlainBody + "\n\n")
	b.WriteString(t.VerifyPlainLink + "\n")
	b.WriteString(data.VerifyURL + "\n\n")
	b.WriteString(fmt.Sprintf(t.VerifyExpiry, data.ExpiresIn) + "\n\n")

	if data.SiteName != "" {
		b.WriteString("— " + data.SiteName + "\n")
	}

	return b.String()
}

var verificationEmailTemplate = template.Must(template.New("verification").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f5;padding:32px 16px;">
<tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1);">

<!-- Header -->
<tr><td style="background-color:#18181b;padding:24px 32px;">
<h1 style="margin:0;color:#ffffff;font-size:18px;font-weight:600;">{{.T.VerifyHeader}}</h1>
</td></tr>

<!-- Body -->
<tr><td style="padding:32px;">

{{if .AuthorName}}
<p style="margin:0 0 16px;color:#374151;font-size:15px;">{{printf .T.VerifyGreeting .AuthorName}}</p>
{{end}}

<p style="margin:0 0 24px;color:#374151;font-size:15px;line-height:1.6;">
{{.T.VerifyBody}}
</p>

<!-- CTA button -->
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 24px;">
<tr><td style="border-radius:6px;background-color:#3b82f6;">
<a href="{{.VerifyURL}}" target="_blank" style="display:inline-block;padding:12px 24px;color:#ffffff;font-size:14px;font-weight:600;text-decoration:none;">{{.T.VerifyButton}}</a>
</td></tr>
</table>

<p style="margin:0 0 8px;color:#6b7280;font-size:13px;">{{.T.VerifyAltInstructions}}</p>
<p style="margin:0 0 24px;color:#3b82f6;font-size:13px;word-break:break-all;">{{.VerifyURL}}</p>

<p style="margin:0;color:#9ca3af;font-size:12px;">{{printf .T.VerifyExpiry .ExpiresIn}}</p>

</td></tr>

<!-- Footer -->
<tr><td style="padding:16px 32px;background-color:#fafafa;border-top:1px solid #e5e7eb;">
<p style="margin:0;color:#9ca3af;font-size:12px;">{{printf .T.VerifyFooter .SiteName}}</p>
</td></tr>

</table>
</td></tr>
</table>
</body>
</html>`))

func renderVerificationEmail(data VerificationEmailData) (string, error) {
	var buf bytes.Buffer
	if err := verificationEmailTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

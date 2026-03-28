package hooks

import (
	"fmt"
	"html"
)

// EmailLayout wraps body HTML in a branded Facet email shell.
// All styles are inlined for maximum email-client compatibility.
func EmailLayout(bodyHTML string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>Facet</title>
</head>
<body style="margin: 0; padding: 0; background-color: #f6f6f6; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color: #f6f6f6;">
  <tr>
    <td align="center" style="padding: 24px 16px;">
      <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width: 600px; width: 100%; border-radius: 8px; overflow: hidden;">
        <!-- Header -->
        <tr>
          <td align="center" style="background-color: #2563eb; padding: 20px;">
            <span style="color: #ffffff; font-size: 20px; font-weight: 600; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">Facet</span>
          </td>
        </tr>
        <!-- Content -->
        <tr>
          <td style="background-color: #ffffff; padding: 24px;">
` + bodyHTML + `
          </td>
        </tr>
        <!-- Footer -->
        <tr>
          <td style="background-color: #f9fafb; padding: 16px 24px; border-top: 1px solid #e5e7eb;">
            <p style="color: #9CA3AF; font-size: 12px; margin: 0; text-align: center; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">Facet &mdash; Your professional identity, your way.</p>
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>`
}

// NewsletterEmailLayout wraps body HTML in the standard email layout and appends an unsubscribe footer.
// Used for all newsletter emails (welcome, broadcast) to comply with CAN-SPAM/GDPR.
func NewsletterEmailLayout(bodyHTML, unsubscribeURL string) string {
	footer := fmt.Sprintf(
		`<div style="border-top: 1px solid #e5e7eb; margin-top: 32px; padding-top: 16px;">
  <p style="color: #9CA3AF; font-size: 12px; text-align: center; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
    You received this email because you subscribed to our newsletter.<br>
    <a href="%s" style="color: #6B7280; text-decoration: underline;">Unsubscribe</a>
  </p>
</div>`, unsubscribeURL)
	return EmailLayout(bodyHTML + footer)
}

// CTAButton returns a centered, styled call-to-action button for use inside EmailLayout.
func CTAButton(text, url string) string {
	return fmt.Sprintf(`<div style="text-align: center; margin: 32px 0;">
  <a href="%s" style="display: inline-block; padding: 14px 28px; background-color: #2563eb; color: #ffffff; text-decoration: none; border-radius: 8px; font-weight: 600; font-size: 16px; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">%s</a>
</div>`, url, text)
}

// EmailTemplate represents a built-in newsletter template.
type EmailTemplate struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// BuiltInTemplates lists available newsletter templates.
var BuiltInTemplates = []EmailTemplate{
	{Slug: "minimal", Name: "Minimal", Description: "Clean, no header — just your content"},
	{Slug: "branded", Name: "Branded", Description: "Header with your name and accent color"},
	{Slug: "announcement", Name: "Announcement", Description: "Bold header for important updates"},
}

// RenderNewsletterTemplate wraps body HTML in the specified template.
// Falls back to the default layout if template is unknown.
func RenderNewsletterTemplate(templateSlug, bodyHTML, unsubscribeURL, senderName string) string {
	// HTML-escape senderName to prevent XSS via profile name injection
	senderName = html.EscapeString(senderName)

	footer := fmt.Sprintf(
		`<div style="border-top: 1px solid #e5e7eb; margin-top: 32px; padding-top: 16px;">
  <p style="color: #9CA3AF; font-size: 12px; text-align: center; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
    You received this email because you subscribed to our newsletter.<br>
    <a href="%s" style="color: #6B7280; text-decoration: underline;">Unsubscribe</a>
  </p>
</div>`, unsubscribeURL)

	content := bodyHTML + footer

	switch templateSlug {
	case "minimal":
		return minimalTemplate(content)
	case "announcement":
		return announcementTemplate(content, senderName)
	case "branded":
		return brandedTemplate(content, senderName)
	default:
		return EmailLayout(content)
	}
}

func minimalTemplate(content string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: #ffffff; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0">
  <tr>
    <td align="center" style="padding: 32px 16px;">
      <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width: 600px; width: 100%;">
        <tr>
          <td style="padding: 0; color: #374151; font-size: 16px; line-height: 1.6;">
` + content + `
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>`
}

func brandedTemplate(content, senderName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: #f9fafb; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color: #f9fafb;">
  <tr>
    <td align="center" style="padding: 32px 16px;">
      <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width: 600px; width: 100%%;">
        <!-- Branded header -->
        <tr>
          <td style="padding: 16px 0 24px 0;">
            <p style="margin: 0; font-size: 18px; font-weight: 600; color: #111827;">%s</p>
          </td>
        </tr>
        <!-- Content card -->
        <tr>
          <td style="background-color: #ffffff; padding: 32px; border-radius: 8px; border: 1px solid #e5e7eb;">
%s
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>`, senderName, content)
}

func announcementTemplate(content, senderName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: #111827; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color: #111827;">
  <tr>
    <td align="center" style="padding: 48px 16px 24px;">
      <p style="margin: 0; font-size: 14px; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase; color: #9CA3AF;">%s</p>
    </td>
  </tr>
  <tr>
    <td align="center" style="padding: 0 16px 48px;">
      <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width: 600px; width: 100%%;">
        <tr>
          <td style="background-color: #ffffff; padding: 40px 32px; border-radius: 12px;">
%s
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>`, senderName, content)
}

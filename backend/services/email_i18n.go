package services

import (
	"github.com/pocketbase/pocketbase/core"
)

// EmailStrings holds all translatable strings for email templates.
type EmailStrings struct {
	// Notification email (to admins)
	NotifSubjectFrom      string // "New testimonial from {name}"
	NotifSubjectFallback  string // "New testimonial submitted"
	NotifHeader           string // "New Testimonial"
	NotifAt               string // " at " (between title and company)
	NotifRequestLabel     string // "Request:"
	NotifSubmittedLabel   string // "Submitted:"
	NotifReviewButton     string // "Review Testimonial"
	NotifFooter           string // "This notification was sent by {site}. You received this because you are an admin."
	NotifPlainHeader      string // "New Testimonial Submitted"
	NotifPlainFrom        string // "From:"
	NotifPlainTitle       string // "Title:"
	NotifPlainRelation    string // "Relationship:"
	NotifPlainRequest     string // "Request:"
	NotifPlainSubmitted   string // "Submitted:"
	NotifPlainContent     string // "Content:"
	NotifPlainReview      string // "Review:"

	// Verification email (to submitter)
	VerifySubject         string // "Verify your testimonial"
	VerifyHeader          string // "Verify Your Email"
	VerifyGreeting        string // "Hi {name},"
	VerifyBody            string // "Please verify your email address..."
	VerifyButton          string // "Verify Email"
	VerifyAltInstructions string // "Or copy and paste this link into your browser:"
	VerifyExpiry          string // "This link expires in {time}."
	VerifyFooter          string // "This email was sent by {site}. If you did not submit a testimonial..."
	VerifyExpiresIn       string // "15 minutes"
	VerifyPlainHeader     string // "Verify Your Email"
	VerifyPlainBody       string // "Please verify your email to add credibility to your testimonial."
	VerifyPlainLink       string // "Click the link below to verify:"
}

var emailTranslations = map[string]EmailStrings{
	"en": {
		NotifSubjectFrom:      "New testimonial from ",
		NotifSubjectFallback:  "New testimonial submitted",
		NotifHeader:           "New Testimonial",
		NotifAt:               " at ",
		NotifRequestLabel:     "Request:",
		NotifSubmittedLabel:   "Submitted:",
		NotifReviewButton:     "Review Testimonial",
		NotifFooter:           "This notification was sent by %s. You received this because you are an admin.",
		NotifPlainHeader:      "New Testimonial Submitted",
		NotifPlainFrom:        "From:",
		NotifPlainTitle:       "Title:",
		NotifPlainRelation:    "Relationship:",
		NotifPlainRequest:     "Request:",
		NotifPlainSubmitted:   "Submitted:",
		NotifPlainContent:     "Content:",
		NotifPlainReview:      "Review:",
		VerifySubject:         "Verify your testimonial",
		VerifyHeader:          "Verify Your Email",
		VerifyGreeting:        "Hi %s,",
		VerifyBody:            "Please verify your email address to add credibility to your testimonial. Click the button below to confirm.",
		VerifyButton:          "Verify Email",
		VerifyAltInstructions: "Or copy and paste this link into your browser:",
		VerifyExpiry:          "This link expires in %s.",
		VerifyFooter:          "This email was sent by %s. If you did not submit a testimonial, you can safely ignore this email.",
		VerifyExpiresIn:       "15 minutes",
		VerifyPlainHeader:     "Verify Your Email",
		VerifyPlainBody:       "Please verify your email to add credibility to your testimonial.",
		VerifyPlainLink:       "Click the link below to verify:",
	},
	"de": {
		NotifSubjectFrom:      "Neues Testimonial von ",
		NotifSubjectFallback:  "Neues Testimonial eingegangen",
		NotifHeader:           "Neues Testimonial",
		NotifAt:               " bei ",
		NotifRequestLabel:     "Anfrage:",
		NotifSubmittedLabel:   "Eingereicht:",
		NotifReviewButton:     "Testimonial prüfen",
		NotifFooter:           "Diese Benachrichtigung wurde von %s gesendet. Sie erhalten diese, weil Sie Administrator sind.",
		NotifPlainHeader:      "Neues Testimonial eingegangen",
		NotifPlainFrom:        "Von:",
		NotifPlainTitle:       "Titel:",
		NotifPlainRelation:    "Beziehung:",
		NotifPlainRequest:     "Anfrage:",
		NotifPlainSubmitted:   "Eingereicht:",
		NotifPlainContent:     "Inhalt:",
		NotifPlainReview:      "Prüfen:",
		VerifySubject:         "Bestätigen Sie Ihr Testimonial",
		VerifyHeader:          "E-Mail bestätigen",
		VerifyGreeting:        "Hallo %s,",
		VerifyBody:            "Bitte bestätigen Sie Ihre E-Mail-Adresse, um Ihrem Testimonial mehr Glaubwürdigkeit zu verleihen. Klicken Sie auf die Schaltfläche unten.",
		VerifyButton:          "E-Mail bestätigen",
		VerifyAltInstructions: "Oder kopieren Sie diesen Link und fügen Sie ihn in Ihren Browser ein:",
		VerifyExpiry:          "Dieser Link läuft in %s ab.",
		VerifyFooter:          "Diese E-Mail wurde von %s gesendet. Wenn Sie kein Testimonial eingereicht haben, können Sie diese E-Mail ignorieren.",
		VerifyExpiresIn:       "15 Minuten",
		VerifyPlainHeader:     "E-Mail bestätigen",
		VerifyPlainBody:       "Bitte bestätigen Sie Ihre E-Mail, um Ihrem Testimonial mehr Glaubwürdigkeit zu verleihen.",
		VerifyPlainLink:       "Klicken Sie auf den folgenden Link zur Bestätigung:",
	},
	"elvish": {
		NotifSubjectFrom:      "New testimony from ",
		NotifSubjectFallback:  "New testimony received",
		NotifHeader:           "New Testimony",
		NotifAt:               " of ",
		NotifRequestLabel:     "Request:",
		NotifSubmittedLabel:   "Offered:",
		NotifReviewButton:     "Review Testimony",
		NotifFooter:           "This message was sent by %s. You received this as a keeper of the records.",
		NotifPlainHeader:      "New Testimony Received",
		NotifPlainFrom:        "From:",
		NotifPlainTitle:       "Title:",
		NotifPlainRelation:    "Bond:",
		NotifPlainRequest:     "Request:",
		NotifPlainSubmitted:   "Offered:",
		NotifPlainContent:     "Words:",
		NotifPlainReview:      "Review:",
		VerifySubject:         "Verify your testimony",
		VerifyHeader:          "Verify Your Identity",
		VerifyGreeting:        "Mae govannen %s,",
		VerifyBody:            "Please verify your identity to lend weight to your testimony. Press the seal below to confirm.",
		VerifyButton:          "Verify Identity",
		VerifyAltInstructions: "Or follow this path in your browser:",
		VerifyExpiry:          "This seal expires in %s.",
		VerifyFooter:          "This message was sent by %s. If you did not offer testimony, you may disregard this letter.",
		VerifyExpiresIn:       "15 minutes",
		VerifyPlainHeader:     "Verify Your Identity",
		VerifyPlainBody:       "Please verify your identity to lend weight to your testimony.",
		VerifyPlainLink:       "Follow this path to verify:",
	},
	"klingon": {
		NotifSubjectFrom:      "New testimony from ",
		NotifSubjectFallback:  "New testimony received",
		NotifHeader:           "New Testimony",
		NotifAt:               " at ",
		NotifRequestLabel:     "Request:",
		NotifSubmittedLabel:   "Filed:",
		NotifReviewButton:     "Examine Testimony",
		NotifFooter:           "This transmission was sent by %s. You received this because you command this station.",
		NotifPlainHeader:      "New Testimony Filed",
		NotifPlainFrom:        "From:",
		NotifPlainTitle:       "Rank:",
		NotifPlainRelation:    "Alliance:",
		NotifPlainRequest:     "Request:",
		NotifPlainSubmitted:   "Filed:",
		NotifPlainContent:     "Report:",
		NotifPlainReview:      "Examine:",
		VerifySubject:         "Verify your testimony",
		VerifyHeader:          "Verify Your Identity",
		VerifyGreeting:        "Warrior %s,",
		VerifyBody:            "Verify your identity to strengthen your testimony. Strike the button below to confirm.",
		VerifyButton:          "Verify Now",
		VerifyAltInstructions: "Or enter this code into your terminal:",
		VerifyExpiry:          "This code expires in %s.",
		VerifyFooter:          "This transmission was sent by %s. If you did not submit testimony, disregard this message.",
		VerifyExpiresIn:       "15 minutes",
		VerifyPlainHeader:     "Verify Your Identity",
		VerifyPlainBody:       "Verify your identity to strengthen your testimony.",
		VerifyPlainLink:       "Access this link to verify:",
	},
	"lolcat": {
		NotifSubjectFrom:      "New testimoniulz frum ",
		NotifSubjectFallback:  "New testimoniulz submitd",
		NotifHeader:           "New Testimoniulz",
		NotifAt:               " at ",
		NotifRequestLabel:     "Reqwest:",
		NotifSubmittedLabel:   "Submitd:",
		NotifReviewButton:     "Review Testimoniulz",
		NotifFooter:           "Dis notificashun wuz sent by %s. U got dis cuz u iz admin.",
		NotifPlainHeader:      "New Testimoniulz Submitd",
		NotifPlainFrom:        "Frum:",
		NotifPlainTitle:       "Title:",
		NotifPlainRelation:    "Relashunship:",
		NotifPlainRequest:     "Reqwest:",
		NotifPlainSubmitted:   "Submitd:",
		NotifPlainContent:     "Content:",
		NotifPlainReview:      "Review:",
		VerifySubject:         "Verify ur testimoniulz",
		VerifyHeader:          "Verify Ur Email",
		VerifyGreeting:        "Hai %s,",
		VerifyBody:            "Plz verify ur email 2 maek ur testimoniulz moar credibulz. Click da button below kthx.",
		VerifyButton:          "Verify Email",
		VerifyAltInstructions: "Or copee n paste dis link in ur browzr:",
		VerifyExpiry:          "Dis link expirez in %s.",
		VerifyFooter:          "Dis email wuz sent by %s. If u didnt submit testimoniulz, u can safly ignore dis email.",
		VerifyExpiresIn:       "15 minutez",
		VerifyPlainHeader:     "Verify Ur Email",
		VerifyPlainBody:       "Plz verify ur email 2 maek ur testimoniulz moar credibulz.",
		VerifyPlainLink:       "Click da link below 2 verify:",
	},
}

// GetEmailStrings returns the email translations for the given locale.
// Falls back to English if the locale is not supported.
func GetEmailStrings(locale string) EmailStrings {
	if s, ok := emailTranslations[locale]; ok {
		return s
	}
	return emailTranslations["en"]
}

// GetSiteLocale reads the default_locale from site settings.
// Returns "en" if settings can't be loaded or locale is empty.
func GetSiteLocale(app core.App) string {
	records, err := app.FindRecordsByFilter("site_settings", "", "", 1, 0, nil)
	if err != nil || len(records) == 0 {
		return "en"
	}
	locale := records[0].GetString("default_locale")
	if locale == "" {
		return "en"
	}
	return locale
}

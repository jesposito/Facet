package hooks

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// defaultNewsletterSendInterval is the pause between SMTP submissions when
// NEWSLETTER_SEND_INTERVAL_MS is unset. Owners on small SMTP quotas can slow
// it down as far as maxNewsletterSendInterval (one message an hour).
const (
	defaultNewsletterSendInterval = time.Second
	maxNewsletterSendInterval     = time.Hour
)

// newsletterPace waits between submissions. Tests replace it.
var newsletterPace = time.Sleep

// errNewsletterPersistence stops a send when its progress can't be recorded:
// carrying on would mail people the database can't account for.
var errNewsletterPersistence = errors.New("newsletter progress could not be saved")

// errOwnerSMTPUnavailable is returned by the central mailer guard before any
// connection is made.
var errOwnerSMTPUnavailable = errors.New("owner SMTP is not configured; sendmail fallback is disabled")

func newsletterSendInterval() (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv("NEWSLETTER_SEND_INTERVAL_MS"))
	if raw == "" {
		return defaultNewsletterSendInterval, nil
	}
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ms < 0 || ms > maxNewsletterSendInterval.Milliseconds() {
		return 0, fmt.Errorf("NEWSLETTER_SEND_INTERVAL_MS must be an integer from 0 to %d", maxNewsletterSendInterval.Milliseconds())
	}
	return time.Duration(ms) * time.Millisecond, nil
}

// ownerSMTPReady reports whether the owner's own SMTP server is configured
// with a usable sender. Nothing else may carry mail.
func ownerSMTPReady(app core.App) bool {
	settings := app.Settings()
	if !settings.SMTP.Enabled || strings.TrimSpace(settings.SMTP.Host) == "" {
		return false
	}
	_, err := mail.ParseAddress(settings.Meta.SenderAddress)
	return err == nil
}

func newsletterReady(app core.App) bool {
	if !ownerSMTPReady(app) {
		return false
	}
	u, err := url.Parse(publicAppURL(app))
	return err == nil && u.IsAbs() && u.Hostname() != "" && (u.Scheme == "http" || u.Scheme == "https")
}

func createNewsletterRecipients(app core.App, sendID string, subscriberIDs []string) error {
	collection, err := app.FindCollectionByNameOrId("newsletter_recipients")
	if err != nil {
		return err
	}
	for _, subscriberID := range subscriberIDs {
		recipient := core.NewRecord(collection)
		recipient.Set("send_id", sendID)
		recipient.Set("subscriber_id", subscriberID)
		recipient.Set("state", "pending")
		if err := app.Save(recipient); err != nil {
			return err
		}
	}
	return nil
}

func claimNewsletterRecipient(app core.App, recipientID string) (bool, error) {
	result, err := app.DB().NewQuery(`UPDATE newsletter_recipients
		SET state = 'in-flight', claimed_at = {:now}, error = ''
		WHERE id = {:id} AND state IN ('pending', 'failed')`).
		Bind(dbx.Params{"now": types.NowDateTime().String(), "id": recipientID}).Execute()
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// completeNewsletterRecipient records the outcome of a claimed recipient. It
// fails unless exactly that in-flight row changed, so a lost update is never
// mistaken for a recorded one.
func completeNewsletterRecipient(app core.App, recipientID, state, errorMessage string) error {
	if len(errorMessage) > 2000 {
		errorMessage = errorMessage[:2000]
	}
	result, err := app.DB().NewQuery(`UPDATE newsletter_recipients
		SET state = {:state}, error = {:error}, completed_at = {:now}
		WHERE id = {:id} AND state = 'in-flight'`).
		Bind(dbx.Params{"state": state, "error": errorMessage, "now": types.NowDateTime().String(), "id": recipientID}).Execute()
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("recipient %s was not in flight", recipientID)
	}
	return nil
}

// markInflightUncertain turns a send's in-flight recipients into uncertain
// ones: an SMTP attempt may have begun, so they are never retried
// automatically.
func markInflightUncertain(app core.App, sendID, reason string) error {
	_, err := app.DB().NewQuery(`UPDATE newsletter_recipients SET state = 'uncertain',
		error = {:reason}, completed_at = {:now}
		WHERE send_id = {:send} AND state = 'in-flight'`).
		Bind(dbx.Params{"reason": reason, "now": types.NowDateTime().String(), "send": sendID}).Execute()
	return err
}

func syncNewsletterProgress(app core.App, sendID string, finished bool) error {
	var rows []struct {
		State string `db:"state"`
		Count int    `db:"count"`
	}
	if err := app.DB().NewQuery(`SELECT state, COUNT(*) AS count FROM newsletter_recipients
		WHERE send_id = {:send} GROUP BY state`).Bind(dbx.Params{"send": sendID}).All(&rows); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.State] = row.Count
	}
	status := "sending"
	activeLock := newsletterActiveLock
	// PocketBase date columns are NOT NULL; "" means not finished yet.
	completedAt := ""
	if finished {
		activeLock = ""
		completedAt = time.Now().UTC().Format(types.DefaultDateLayout)
		pending := counts["pending"] + counts["in-flight"]
		switch {
		case pending > 0:
			status = "interrupted"
		case counts["uncertain"] > 0:
			status = "interrupted"
		case counts["accepted"] > 0 && (counts["failed"] > 0 || counts["skipped"] > 0):
			status = "partial"
		case counts["accepted"] > 0:
			status = "sent"
		default:
			status = "failed"
		}
	}
	result, err := app.DB().NewQuery(`UPDATE newsletter_sends SET
		sent_count = {:accepted}, failed_count = {:failed}, uncertain_count = {:uncertain},
		skipped_count = {:skipped}, pending_count = {:pending}, status = {:status},
		active_lock = {:lock}, sent_at = {:completed} WHERE id = {:send}`).
		Bind(dbx.Params{
			"accepted": counts["accepted"], "failed": counts["failed"], "uncertain": counts["uncertain"],
			"skipped": counts["skipped"], "pending": counts["pending"] + counts["in-flight"],
			"status": status, "lock": activeLock, "completed": completedAt, "send": sendID,
		}).Execute()
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("send %s no longer exists", sendID)
	}
	return nil
}

func recoverInterruptedNewsletters(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("newsletter_recipients"); err != nil {
		return nil
	}
	sends, err := app.FindRecordsByFilter("newsletter_sends", "status = 'sending' || active_lock != ''", "", 0, 0)
	if err != nil {
		return err
	}
	for _, send := range sends {
		ledgerRows, err := app.CountRecords("newsletter_recipients", dbx.HashExp{"send_id": send.Id})
		if err != nil {
			return err
		}
		if ledgerRows == 0 {
			if err := releaseLegacyNewsletterSend(app, send.Id); err != nil {
				return err
			}
			continue
		}
		if err := markInflightUncertain(app, send.Id, "process stopped after the SMTP attempt began"); err != nil {
			return err
		}
		if err := syncNewsletterProgress(app, send.Id, true); err != nil {
			return err
		}
	}
	return nil
}

// releaseLegacyNewsletterSend stops a send recorded before the recipient
// ledger existed. Its stored counters are the only history of who was mailed,
// so they are kept as they are: rebuilding them from an empty ledger would
// erase them and weaken duplicate protection. Nobody is marked resumable,
// because there is no record of who is left.
func releaseLegacyNewsletterSend(app core.App, sendID string) error {
	_, err := app.DB().NewQuery(`UPDATE newsletter_sends SET
		status = CASE WHEN status = 'sending' THEN 'interrupted' ELSE status END,
		active_lock = ''
		WHERE id = {:send} AND (status = 'sending' OR active_lock != '')`).
		Bind(dbx.Params{"send": sendID}).Execute()
	return err
}

// newsletterResumableCounts returns, for each listed send, how many ledger
// recipients Resume would retry: pending ones and those the server refused.
// Sends without such rows, including ones recorded before the ledger, are
// absent from the map. It runs one grouped query for all the IDs.
func newsletterResumableCounts(app core.App, sendIDs []string) (map[string]int, error) {
	counts := make(map[string]int, len(sendIDs))
	if len(sendIDs) == 0 {
		return counts, nil
	}
	ids := make([]any, len(sendIDs))
	for i, id := range sendIDs {
		ids[i] = id
	}
	var rows []struct {
		SendID string `db:"send_id"`
		Count  int    `db:"count"`
	}
	err := app.DB().Select("send_id", "COUNT(*) AS count").
		From("newsletter_recipients").
		Where(dbx.In("send_id", ids...)).
		AndWhere(dbx.In("state", "pending", "failed")).
		GroupBy("send_id").
		All(&rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.SendID] = row.Count
	}
	return counts, nil
}

// resumeNewsletter restarts an interrupted send for its pending recipients and
// those whose server explicitly refused them. Accepted and uncertain
// recipients are never mailed again.
func resumeNewsletter(app *pocketbase.PocketBase, crypto *services.CryptoService, sendID string) error {
	if !services.IsFeatureEnabled(app, services.FeatureNewsletter, true) {
		return errNewsletterFeatureOff
	}
	if !newsletterReady(app) {
		return errors.New("newsletter SMTP sender and absolute APP_URL must be configured")
	}
	if _, err := newsletterSendInterval(); err != nil {
		return err
	}
	send, err := app.FindRecordById("newsletter_sends", sendID)
	if err != nil {
		return err
	}
	err = app.RunInTransaction(func(txApp core.App) error {
		active, err := txApp.CountRecords("newsletter_sends", dbx.NewExp("status = 'sending' OR active_lock != ''"))
		if err != nil {
			return err
		}
		if active > 0 {
			return errNewsletterSendInProgress
		}
		eligible, err := txApp.CountRecords("newsletter_recipients", dbx.NewExp(
			"send_id = {:send} AND state IN ('pending', 'failed')", dbx.Params{"send": sendID},
		))
		if err != nil {
			return err
		}
		if eligible == 0 {
			return errors.New("no recipients are safe to resume")
		}
		result, err := txApp.DB().NewQuery(`UPDATE newsletter_sends SET status = 'sending', active_lock = {:lock}
			WHERE id = {:id} AND status != 'sending' AND active_lock = ''`).
			Bind(dbx.Params{"lock": newsletterActiveLock, "id": sendID}).Execute()
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err == nil && n != 1 {
			return errNewsletterSendInProgress
		}
		return err
	})
	if err != nil {
		return err
	}
	audience := &newsletterAudience{}
	if listID := send.GetString("list_id"); listID != "" {
		// A deleted list only loses the live record; the stored list_name
		// below still fills {{list.name}}.
		audience.List, _ = app.FindRecordById("newsletter_lists", listID)
	}
	personalization := newNewsletterPersonalization(app, send.GetBool("personalize"), audience)
	personalization.ListName = send.GetString("list_name")
	goSendNewsletter(app, crypto, send.Id, send.GetString("subject"), send.GetString("body_html"), send.GetString("template_slug"), nil, personalization)
	return nil
}

// resumableNewsletterRecipients lists recipients that have not been
// submitted: pending ones, and failed ones whose server refused them
// outright. A query error is returned, never an empty list.
func resumableNewsletterRecipients(app core.App, sendID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("newsletter_recipients",
		"send_id = {:send} && (state = 'pending' || state = 'failed')", "created,id", 0, 0,
		dbx.Params{"send": sendID})
}

// smtpFailureState decides whether a failed submission is safe to retry.
// Only two outcomes prove a message was not accepted: an explicit negative
// SMTP reply, and a failure before any connection existed. Anything else,
// such as a connection dropped after DATA, may have been delivered and is
// uncertain. Error text is never parsed.
func smtpFailureState(err error) string {
	if errors.Is(err, errOwnerSMTPUnavailable) {
		return "failed"
	}
	var reply *textproto.Error
	if errors.As(err, &reply) && reply.Code >= 400 && reply.Code < 600 {
		return "failed"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "failed"
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return "failed"
	}
	return "uncertain"
}

// smtpRelayUnavailable reports failures that say nothing about the current
// recipient and would repeat for every other one: the owner relay is not
// configured, can't be reached, is closing service, or refuses our login.
// Like smtpFailureState it trusts only typed errors, never error text.
func smtpRelayUnavailable(err error) bool {
	if errors.Is(err, errOwnerSMTPUnavailable) {
		return true
	}
	var reply *textproto.Error
	if errors.As(err, &reply) {
		switch reply.Code {
		case 421, 454, 530, 534, 535:
			return true
		}
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}

// deleteNewsletterSend removes a finished send and its recipient ledger in
// one transaction. The send is re-read inside the transaction, which is
// serialized with resume's, so a running or just-resumed send keeps its row,
// its active lock and its receipts.
func deleteNewsletterSend(app core.App, sendID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		record, err := txApp.FindRecordById("newsletter_sends", sendID)
		if err != nil {
			return err
		}
		if record.GetString("status") == "sending" || record.GetString("active_lock") != "" {
			return errNewsletterSendInProgress
		}
		if _, err := txApp.DB().NewQuery("DELETE FROM newsletter_recipients WHERE send_id = {:send}").
			Bind(dbx.Params{"send": sendID}).Execute(); err != nil {
			return err
		}
		return txApp.Delete(record)
	})
}

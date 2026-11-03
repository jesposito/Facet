package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// A paid purchase used to be saved with an access link that was never sent,
// so buyers paid and got nothing. access_email_sent_at records when the link
// went out, which lets a repeated webhook or the checkout return page send it
// again after a mail failure without sending it twice when all went well.
//
// The unique index stops a webhook and the return page racing to record one
// checkout twice. It only covers rows with a session ID and is skipped if an
// older race already left duplicates, so the migration can't fail on old data.
func init() {
	const indexName = "idx_purchases_checkout_session"
	m.Register(func(app core.App) error {
		purchases, err := app.FindCollectionByNameOrId("purchases")
		if err != nil {
			return nil
		}
		if purchases.Fields.GetByName("access_email_sent_at") == nil {
			purchases.Fields.Add(&core.DateField{Name: "access_email_sent_at"})
		}

		var dupes struct {
			N int `db:"n"`
		}
		err = app.DB().NewQuery(`SELECT COUNT(*) AS n FROM (
			SELECT stripe_checkout_session_id FROM purchases
			WHERE stripe_checkout_session_id != ''
			GROUP BY stripe_checkout_session_id HAVING COUNT(*) > 1)`).One(&dupes)
		if err == nil && dupes.N == 0 {
			purchases.AddIndex(indexName, true, "stripe_checkout_session_id", "stripe_checkout_session_id != ''")
		} else {
			app.Logger().Warn("Duplicate checkout sessions in purchases; unique index not added", "duplicates", dupes.N)
		}
		return app.Save(purchases)
	}, func(app core.App) error {
		purchases, err := app.FindCollectionByNameOrId("purchases")
		if err != nil {
			return nil
		}
		purchases.RemoveIndex(indexName)
		if f := purchases.Fields.GetByName("access_email_sent_at"); f != nil {
			purchases.Fields.RemoveById(f.GetId())
		}
		return app.Save(purchases)
	})
}

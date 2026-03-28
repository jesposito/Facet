package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds threaded discussions with two collections: comments and comment_reports.
func init() {
	m.Register(func(app core.App) error {
		// 1. Create comments collection
		if _, err := app.FindCollectionByNameOrId("comments"); err == nil {
			return nil // Already exists
		}

		comments := core.NewBaseCollection("comments")
		// Admin only — custom endpoints handle public access
		comments.ListRule = nil
		comments.ViewRule = nil
		comments.CreateRule = nil
		comments.UpdateRule = nil
		comments.DeleteRule = nil

		comments.Fields.Add(&core.TextField{
			Name:     "content_type",
			Required: true,
			Max:      50,
		})
		comments.Fields.Add(&core.TextField{
			Name:     "content_id",
			Required: true,
			Max:      15,
		})
		comments.Fields.Add(&core.TextField{
			Name: "parent_id",
			Max:  15,
		})
		comments.Fields.Add(&core.TextField{
			Name:     "author_name",
			Required: true,
			Max:      100,
		})
		comments.Fields.Add(&core.EmailField{
			Name: "author_email",
		})
		comments.Fields.Add(&core.TextField{
			Name:     "body",
			Required: true,
			Max:      5000,
		})
		comments.Fields.Add(&core.SelectField{
			Name:      "status",
			MaxSelect: 1,
			Required:  true,
			Values:    []string{"pending", "approved", "rejected", "spam"},
		})
		comments.Fields.Add(&core.BoolField{
			Name: "is_pinned",
		})
		comments.Fields.Add(&core.BoolField{
			Name: "is_admin_reply",
		})
		comments.Fields.Add(&core.TextField{
			Name: "ip_hash",
			Max:  64,
		})
		comments.Fields.Add(&core.TextField{
			Name: "user_agent",
			Max:  500,
		})

		comments.Indexes = []string{
			"CREATE INDEX idx_comments_content ON comments(content_type, content_id, status)",
			"CREATE INDEX idx_comments_parent ON comments(parent_id) WHERE parent_id != ''",
			"CREATE INDEX idx_comments_status ON comments(status)",
		}

		if err := app.Save(comments); err != nil {
			return err
		}

		// 2. Create comment_reports collection
		commentReports := core.NewBaseCollection("comment_reports")
		commentReports.ListRule = nil
		commentReports.ViewRule = nil
		commentReports.CreateRule = nil
		commentReports.UpdateRule = nil
		commentReports.DeleteRule = nil

		commentReports.Fields.Add(&core.RelationField{
			Name:          "comment",
			MaxSelect:     1,
			Required:      true,
			CollectionId:  comments.Id,
			CascadeDelete: true,
		})
		commentReports.Fields.Add(&core.SelectField{
			Name:      "reason",
			MaxSelect: 1,
			Required:  true,
			Values:    []string{"spam", "harassment", "off_topic", "other"},
		})
		commentReports.Fields.Add(&core.TextField{
			Name: "detail",
			Max:  1000,
		})
		commentReports.Fields.Add(&core.TextField{
			Name: "reporter_ip_hash",
			Max:  64,
		})
		commentReports.Fields.Add(&core.SelectField{
			Name:      "status",
			MaxSelect: 1,
			Required:  true,
			Values:    []string{"open", "resolved", "dismissed"},
		})

		commentReports.Indexes = []string{
			"CREATE INDEX idx_comment_reports_status ON comment_reports(status)",
		}

		return app.Save(commentReports)
	}, func(app core.App) error {
		// Rollback: delete collections in reverse order
		collections := []string{"comment_reports", "comments"}
		for _, name := range collections {
			if collection, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

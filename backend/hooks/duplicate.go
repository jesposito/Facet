package hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterDuplicateHooks adds admin duplicate endpoints for cloning records
func RegisterDuplicateHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// POST /api/admin/duplicate/{collection}/{id}
		se.Router.POST("/api/admin/duplicate/{collection}/{id}", func(e *core.RequestEvent) error {
			// Require authentication
			if e.Auth == nil {
				return apis.NewUnauthorizedError("authentication required", nil)
			}

			collection := e.Request.PathValue("collection")
			recordID := e.Request.PathValue("id")

			if collection == "" || recordID == "" {
				return apis.NewBadRequestError("collection and id are required", nil)
			}

			// Get the collection to validate it exists and ensure it's allowed for duplication
			coll, err := app.FindCollectionByNameOrId(collection)
			if err != nil {
				return apis.NewBadRequestError("collection not found", err)
			}

			// Only allow duplication of content collections (not system/auth collections)
			allowedCollections := map[string]bool{
				"experience":      true,
				"projects":        true,
				"education":       true,
				"certifications":  true,
				"awards":          true,
				"skills":          true,
				"posts":           true,
				"talks":           true,
				"contact_methods": true,
			}

			if !allowedCollections[coll.Name] {
				return apis.NewBadRequestError("collection not allowed for duplication", nil)
			}

			// Find the original record
			original, err := app.FindRecordById(coll.Name, recordID)
			if err != nil {
				return apis.NewNotFoundError("record not found", err)
			}

			// Create a new record with the same data
			duplicate := core.NewRecord(coll)

			if title := original.GetString("title"); title != "" {
				duplicate.Set("title", title+" (Copy)")
			}
			if name := original.GetString("name"); name != "" {
				duplicate.Set("name", name+" (Copy)")
			}
			if company := original.GetString("company"); company != "" {
				duplicate.Set("company", company+" (Copy)")
			}

			duplicate.Set("company_group_id", original.GetString("company_group_id"))

			if slug := original.GetString("slug"); slug != "" {
				duplicate.Set("slug", slug+"-copy")
			}

			duplicate.Set("description", original.GetString("description"))
			duplicate.Set("content", original.GetString("content"))
			duplicate.Set("summary", original.GetString("summary"))
			duplicate.Set("excerpt", original.GetString("excerpt"))
			duplicate.Set("location", original.GetString("location"))

			duplicate.Set("start_date", original.GetString("start_date"))
			duplicate.Set("end_date", original.GetString("end_date"))
			duplicate.Set("date", original.GetString("date"))
			duplicate.Set("awarded_at", original.GetString("awarded_at"))
			duplicate.Set("issue_date", original.GetString("issue_date"))
			duplicate.Set("expiry_date", original.GetString("expiry_date"))
			duplicate.Set("published_at", original.GetString("published_at"))

			duplicate.Set("bullets", original.Get("bullets"))
			duplicate.Set("skills", original.Get("skills"))
			duplicate.Set("tech_stack", original.Get("tech_stack"))
			duplicate.Set("categories", original.Get("categories"))
			duplicate.Set("tags", original.Get("tags"))
			duplicate.Set("links", original.Get("links"))
			duplicate.Set("media", original.Get("media"))
			duplicate.Set("media_refs", original.Get("media_refs"))
			duplicate.Set("admin_tags", original.Get("admin_tags"))

			duplicate.Set("institution", original.GetString("institution"))
			duplicate.Set("degree", original.GetString("degree"))
			duplicate.Set("field", original.GetString("field"))
			duplicate.Set("issuer", original.GetString("issuer"))
			duplicate.Set("credential_id", original.GetString("credential_id"))
			duplicate.Set("credential_url", original.GetString("credential_url"))
			duplicate.Set("category", original.GetString("category"))
			duplicate.Set("proficiency", original.GetString("proficiency"))
			duplicate.Set("event", original.GetString("event"))
			duplicate.Set("event_url", original.GetString("event_url"))
			duplicate.Set("slides_url", original.GetString("slides_url"))
			duplicate.Set("video_url", original.GetString("video_url"))
			duplicate.Set("url", original.GetString("url"))
			duplicate.Set("type", original.GetString("type"))
			duplicate.Set("value", original.GetString("value"))
			duplicate.Set("label", original.GetString("label"))
			duplicate.Set("protection", original.GetString("protection"))

			duplicate.Set("visibility", original.GetString("visibility"))
			duplicate.Set("is_draft", original.GetBool("is_draft"))
			duplicate.Set("is_featured", false)

			duplicate.Set("sort_order", 0)

			// Save the duplicate record
			if err := app.Save(duplicate); err != nil {
				return apis.NewBadRequestError("failed to create duplicate", err)
			}

			duplicate, err = app.FindRecordById(coll.Name, duplicate.Id)
			if err != nil {
				return apis.NewBadRequestError("failed to fetch created duplicate", err)
			}

			return e.JSON(http.StatusCreated, map[string]any{
				"success": true,
				"message": "Record duplicated successfully",
				"record":  duplicate,
			})
		})

		return se.Next()
	})
}

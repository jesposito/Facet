package hooks

import (
	"net/http"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ----------------------------------------------------------------------------
// Public API v1 write endpoints.
//
// These mirror the read endpoints registered in api_keys.go. They are gated by
// the same `APIKeyMiddleware` and use new `write:*` scopes that live alongside
// the existing `read:*` scopes in `validScopes`.
//
// Self-hosted is single-tenant: no per-tenant rate limits, no tier gating, no
// credit reservation. We deliberately strip those layers from the cloud version
// of this file.
// ----------------------------------------------------------------------------

// apiV1WriteFields defines which fields are settable via the public write API
// per collection. Anything not in this list is silently ignored on POST/PATCH
// so callers can't sneak in protected fields (e.g. owner ids, audit timestamps,
// file uploads — files require multipart and a separate code path).
//
// Keep this list aligned with the collection schema in
// migrations/1735600000_init.go and with the read mappers in api_keys.go so
// writes round-trip through reads cleanly.
var apiV1WriteFields = map[string][]string{
	"posts": {
		"title", "slug", "excerpt", "content",
		"tags", "visibility", "is_draft", "published_at",
	},
	"projects": {
		"title", "slug", "summary", "description",
		"tech_stack", "links", "categories",
		"visibility", "is_draft", "is_featured", "sort_order",
	},
	"skills": {
		"name", "category", "proficiency",
		"visibility", "sort_order",
	},
	"experience": {
		"company", "title", "location",
		"start_date", "end_date",
		"description", "bullets", "skills",
		"visibility", "is_draft", "sort_order",
	},
	"profile": {
		"name", "headline", "location", "summary",
		"contact_email", "contact_links", "visibility",
	},
}

// RegisterAPIV1WriteHooks wires up the POST/PATCH/DELETE endpoints that
// complement the read-only `/api/v1/*` surface registered in
// RegisterAPIKeyHooks.
func RegisterAPIV1WriteHooks(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		writeAuth := func(scope string, handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return APIKeyMiddleware(app, crypto, scope)(handler)
		}

		// -----------------------------------------------------------------
		// posts
		// -----------------------------------------------------------------
		se.Router.POST("/api/v1/posts", writeAuth("write:posts", func(e *core.RequestEvent) error {
			return apiV1Create(app, e, "posts", mapBasicPost)
		}))
		se.Router.PATCH("/api/v1/posts/{id}", writeAuth("write:posts", func(e *core.RequestEvent) error {
			return apiV1Update(app, e, "posts", mapBasicPost)
		}))
		se.Router.DELETE("/api/v1/posts/{id}", writeAuth("write:posts", func(e *core.RequestEvent) error {
			return apiV1Delete(app, e, "posts")
		}))

		// -----------------------------------------------------------------
		// projects
		// -----------------------------------------------------------------
		se.Router.POST("/api/v1/projects", writeAuth("write:projects", func(e *core.RequestEvent) error {
			return apiV1Create(app, e, "projects", mapBasicProject)
		}))
		se.Router.PATCH("/api/v1/projects/{id}", writeAuth("write:projects", func(e *core.RequestEvent) error {
			return apiV1Update(app, e, "projects", mapBasicProject)
		}))
		se.Router.DELETE("/api/v1/projects/{id}", writeAuth("write:projects", func(e *core.RequestEvent) error {
			return apiV1Delete(app, e, "projects")
		}))

		// -----------------------------------------------------------------
		// skills
		// -----------------------------------------------------------------
		se.Router.POST("/api/v1/skills", writeAuth("write:skills", func(e *core.RequestEvent) error {
			return apiV1Create(app, e, "skills", mapBasicSkill)
		}))
		se.Router.PATCH("/api/v1/skills/{id}", writeAuth("write:skills", func(e *core.RequestEvent) error {
			return apiV1Update(app, e, "skills", mapBasicSkill)
		}))
		se.Router.DELETE("/api/v1/skills/{id}", writeAuth("write:skills", func(e *core.RequestEvent) error {
			return apiV1Delete(app, e, "skills")
		}))

		// -----------------------------------------------------------------
		// experience
		// -----------------------------------------------------------------
		se.Router.POST("/api/v1/experience", writeAuth("write:experience", func(e *core.RequestEvent) error {
			return apiV1Create(app, e, "experience", mapBasicExperience)
		}))
		se.Router.PATCH("/api/v1/experience/{id}", writeAuth("write:experience", func(e *core.RequestEvent) error {
			return apiV1Update(app, e, "experience", mapBasicExperience)
		}))
		se.Router.DELETE("/api/v1/experience/{id}", writeAuth("write:experience", func(e *core.RequestEvent) error {
			return apiV1Delete(app, e, "experience")
		}))

		// -----------------------------------------------------------------
		// profile (singleton — only PATCH, no POST/DELETE)
		// -----------------------------------------------------------------
		se.Router.PATCH("/api/v1/profile", writeAuth("write:profile", func(e *core.RequestEvent) error {
			return apiV1UpdateProfile(app, e)
		}))

		return se.Next()
	})
}

// apiV1Create handles POST /api/v1/{collection}. It binds the JSON body,
// allowlists fields, saves the record, and returns 201 with the mapped read
// representation so callers can use the response as-is.
func apiV1Create(app core.App, e *core.RequestEvent, collection string, mapper func(*core.Record) map[string]any) error {
	body, err := bindAPIBody(e)
	if err != nil {
		return err
	}

	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "collection not found"})
	}

	record := core.NewRecord(coll)
	applyAllowedFields(record, body, apiV1WriteFields[collection])

	if err := app.Save(record); err != nil {
		app.Logger().Warn("api v1 create failed", "collection", collection, "error", err)
		return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "validation failed: " + err.Error()})
	}

	return respondJSON(e, http.StatusCreated, mapper(record))
}

// apiV1Update handles PATCH /api/v1/{collection}/{id}. It loads the existing
// record, applies allowlisted fields from the body, saves, and returns 200
// with the mapped read representation.
func apiV1Update(app core.App, e *core.RequestEvent, collection string, mapper func(*core.Record) map[string]any) error {
	id := strings.TrimSpace(e.Request.PathValue("id"))
	if id == "" {
		return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	record, err := app.FindRecordById(collection, id)
	if err != nil {
		return respondNotFound(e, "record not found")
	}

	body, err := bindAPIBody(e)
	if err != nil {
		return err
	}
	applyAllowedFields(record, body, apiV1WriteFields[collection])

	if err := app.Save(record); err != nil {
		app.Logger().Warn("api v1 update failed", "collection", collection, "id", id, "error", err)
		return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "validation failed: " + err.Error()})
	}

	return respondJSON(e, http.StatusOK, mapper(record))
}

// apiV1Delete handles DELETE /api/v1/{collection}/{id}. It hard-deletes the
// record and returns 204 No Content on success. Returns 404 if the record
// doesn't exist (idempotent-ish — repeat deletes 404).
func apiV1Delete(app core.App, e *core.RequestEvent, collection string) error {
	id := strings.TrimSpace(e.Request.PathValue("id"))
	if id == "" {
		return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	record, err := app.FindRecordById(collection, id)
	if err != nil {
		return respondNotFound(e, "record not found")
	}

	if err := app.Delete(record); err != nil {
		app.Logger().Warn("api v1 delete failed", "collection", collection, "id", id, "error", err)
		return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
	}

	// 204 No Content — explicitly no body.
	e.Response.WriteHeader(http.StatusNoContent)
	return errResponseWritten
}

// apiV1UpdateProfile patches the singleton profile record. Self-hosted has
// exactly one profile, so we find it by "id != ”" rather than by id path
// param, and there is no POST/DELETE counterpart.
func apiV1UpdateProfile(app core.App, e *core.RequestEvent) error {
	records, err := app.FindRecordsByFilter("profile", "id != ''", "", 1, 0)
	if err != nil || len(records) == 0 {
		return respondNotFound(e, "profile not found")
	}
	record := records[0]

	body, err := bindAPIBody(e)
	if err != nil {
		return err
	}
	applyAllowedFields(record, body, apiV1WriteFields["profile"])

	if err := app.Save(record); err != nil {
		app.Logger().Warn("api v1 profile update failed", "error", err)
		return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "validation failed: " + err.Error()})
	}

	// Mirror the shape of GET /api/v1/profile (see api_keys.go).
	return respondJSON(e, http.StatusOK, map[string]any{
		"name":     record.GetString("name"),
		"headline": record.GetString("headline"),
		"summary":  record.GetString("summary"),
		"location": record.GetString("location"),
		"updated":  record.GetString("updated"),
	})
}

// bindAPIBody decodes the JSON request body into a generic map. On parse
// failure it writes a 400 response and returns errResponseWritten so the
// caller can short-circuit with `return err`.
func bindAPIBody(e *core.RequestEvent) (map[string]any, error) {
	var body map[string]any
	if err := e.BindBody(&body); err != nil {
		return nil, respondJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, nil
}

// applyAllowedFields copies values from `body` onto `record` for each name in
// `allowed` that is present in the body. Unknown / disallowed fields are
// silently ignored (preferring forward-compat over chatty 400s).
func applyAllowedFields(record *core.Record, body map[string]any, allowed []string) {
	for _, field := range allowed {
		if val, ok := body[field]; ok {
			record.Set(field, val)
		}
	}
}

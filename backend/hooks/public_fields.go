package hooks

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Public pages receive only presentation fields. New schema fields stay private
// until reviewed here; demo records follow the same projection.
var publicFieldAllowlist = map[string]map[string]bool{
	"experience":      publicFields("company", "title", "location", "start_date", "end_date", "description", "bullets", "skills", "media", "company_logo", "company_logo_library_url", "logo_background", "company_group_id", "visibility", "is_draft", "sort_order", "created", "updated"),
	"projects":        publicFields("title", "slug", "summary", "description", "tech_stack", "links", "media", "cover_image", "categories", "is_featured", "media_refs", "cover_image_library_url", "related_experience", "related_education", "visibility", "is_draft", "sort_order", "created", "updated"),
	"education":       publicFields("institution", "degree", "field", "start_date", "end_date", "description", "institution_logo", "institution_logo_library_url", "logo_background", "visibility", "is_draft", "sort_order", "created", "updated"),
	"certifications":  publicFields("name", "issuer", "issue_date", "expiry_date", "credential_id", "credential_url", "visibility", "is_draft", "sort_order", "created", "updated"),
	"awards":          publicFields("title", "issuer", "awarded_at", "description", "url", "visibility", "is_draft", "sort_order", "created", "updated"),
	"skills":          publicFields("name", "category", "proficiency", "visibility", "is_draft", "sort_order", "created", "updated"),
	"posts":           publicFields("title", "slug", "excerpt", "content", "cover_image", "tags", "published_at", "media_refs", "media", "featured", "cover_image_library_url", "comments_enabled", "visibility", "is_draft", "sort_order", "created", "updated"),
	"talks":           publicFields("title", "slug", "event", "event_url", "date", "location", "description", "slides_url", "video_url", "media_refs", "media", "cover_image_library_url", "visibility", "is_draft", "sort_order", "created", "updated"),
	"testimonials":    publicFields("content", "relationship", "project", "author_name", "author_title", "author_company", "author_photo", "author_website", "verification_method", "featured", "author_photo_library_url", "visibility", "is_draft", "sort_order", "created", "updated"),
	"contact_methods": publicFields("type", "value", "label", "protection_level", "is_primary", "icon", "visibility", "is_draft", "sort_order", "created", "updated"),
	"custom_content":  publicFields("title", "content", "cover_image", "media", "media_refs", "cover_image_library_url", "comments_enabled", "visibility", "is_draft", "sort_order", "created", "updated"),
}

func publicFields(names ...string) map[string]bool {
	fields := make(map[string]bool, len(names))
	for _, name := range names {
		fields[name] = true
	}
	return fields
}

func publicFieldAllowed(collection *core.Collection, name string) bool {
	field := collection.Fields.GetByName(name)
	if field == nil || field.GetHidden() {
		return false
	}
	return publicFieldAllowlist[strings.TrimPrefix(collection.Name, "demo_")][name]
}

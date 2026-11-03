package services

// LibraryURLFields lists every collection field that stores a Media Library URL
// (/api/files/<collection>/<record>/<file>) picked from the library. It is the
// single source of truth for usage reporting, media replacement and orphaned
// upload cleanup, so an omission here means the library can delete or miss a
// file that content still shows.
//
// Each *_library_url field has a matching cached file field named by stripping
// the "_library_url" suffix (e.g. cover_image_library_url -> cover_image).
//
// MAINTENANCE: add new *_library_url fields here when a migration creates them.
// Nested project experience media and external_media mirrors are tracked
// separately (FindProjectExperienceMediaUsage and media_refs).
var LibraryURLFields = map[string][]string{
	"profile":        {"hero_image_library_url", "avatar_library_url"},
	"experience":     {"company_logo_library_url"},
	"education":      {"institution_logo_library_url"},
	"projects":       {"cover_image_library_url"},
	"posts":          {"cover_image_library_url"},
	"talks":          {"cover_image_library_url"},
	"views":          {"hero_image_library_url"},
	"site_settings":  {"favicon_library_url"},
	"custom_content": {"cover_image_library_url"},
	"testimonials":   {"author_photo_library_url"},
	"courses":        {"cover_image_library_url"},
}

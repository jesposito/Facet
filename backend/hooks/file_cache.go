package hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

const (
	// publicFileCacheControl lets browsers and shared caches keep ordinary
	// uploads such as cover images and avatars for a day.
	publicFileCacheControl = "public, max-age=86400"
	// protectedFileCacheControl stops anyone keeping a copy of a protected
	// file, so it can't be served again after the file token expires.
	protectedFileCacheControl = "private, no-store"
)

// RegisterFileCacheHeaders sets Cache-Control on /api/files responses.
// PocketBase only adds its own 30-day default when the header is empty, and
// Caddy only fills it in when missing, so this is the value visitors get.
func RegisterFileCacheHeaders(app core.App) {
	app.OnFileDownloadRequest().BindFunc(func(e *core.FileDownloadRequestEvent) error {
		e.Response.Header().Set("Cache-Control", fileCacheControl(e.FileField))
		return e.Next()
	})
}

func fileCacheControl(field *core.FileField) string {
	if field != nil && field.Protected {
		return protectedFileCacheControl
	}
	return publicFileCacheControl
}

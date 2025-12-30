package hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RequireAuth returns a middleware that requires authentication
func RequireAuth(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Load auth from headers
		info, _ := e.RequestInfo()
		if info != nil && info.Auth != nil {
			e.Auth = info.Auth
		}
		return e.Next()
	}
}

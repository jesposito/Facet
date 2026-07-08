package hooks

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
)

// errResponseWritten is a sentinel error signaling that the HTTP response has
// already been written by a helper. PocketBase's e.JSON() returns nil, so
// returning it directly does NOT stop execution -- always return this sentinel.
var errResponseWritten = errors.New("response already written")

// respondUnauthorized writes a 401 JSON response and returns errResponseWritten.
func respondUnauthorized(e *core.RequestEvent, msg string) error {
	e.JSON(http.StatusUnauthorized, map[string]string{"error": msg})
	return errResponseWritten
}

// respondForbidden writes a 403 JSON response and returns errResponseWritten.
func respondForbidden(e *core.RequestEvent, msg string) error {
	e.JSON(http.StatusForbidden, map[string]string{"error": msg})
	return errResponseWritten
}

// respondJSON writes a JSON response with the given status and returns errResponseWritten.
func respondJSON(e *core.RequestEvent, status int, body any) error {
	e.JSON(status, body)
	return errResponseWritten
}

// respondNotFound writes a 404 JSON response and returns errResponseWritten.
func respondNotFound(e *core.RequestEvent, msg string) error {
	e.JSON(http.StatusNotFound, map[string]string{"error": msg})
	return errResponseWritten
}

// isSuperuser returns true if the request has a valid admin auth.
func isSuperuser(e *core.RequestEvent) bool {
	if e.Auth == nil {
		return false
	}
	name := e.Auth.Collection().Name
	return name == core.CollectionNameSuperusers || name == "users"
}

// requireSuperuser checks if the request is from a superuser.
// Writes 401 JSON and returns errResponseWritten if not authorized.
func requireSuperuser(e *core.RequestEvent) error {
	if e.Auth == nil {
		return respondUnauthorized(e, "unauthorized")
	}
	name := e.Auth.Collection().Name
	if name != core.CollectionNameSuperusers && name != "users" {
		return respondUnauthorized(e, "unauthorized")
	}
	return nil
}

func parsePositiveInt(raw string, def, max int) int {
	if raw == "" {
		return def
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return def
	}
	if max > 0 && parsed > max {
		return max
	}
	return parsed
}

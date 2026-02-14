package services

import (
	"fmt"
	"runtime/debug"

	"github.com/pocketbase/pocketbase"
)

// SafeGo runs fn in a goroutine with panic recovery.
// If fn panics, the error is logged and the goroutine exits cleanly
// instead of crashing the entire process.
func SafeGo(app *pocketbase.PocketBase, name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				app.Logger().Error("goroutine panic recovered",
					"goroutine", name,
					"error", fmt.Sprint(r),
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

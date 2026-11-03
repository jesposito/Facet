package hooks

import (
	"net"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// clientIPMiddlewarePriority runs the middleware ahead of everything
// PocketBase binds, including its activity logger and rate limiter.
const clientIPMiddlewarePriority = -1 << 30

// RegisterClientIPMiddleware settles who sent each request before any other
// code looks:
//
//   - X-Internal is dropped unless the request really is the frontend calling
//     over loopback, so a visitor can't use it to skip limits or analytics.
//   - RemoteAddr is replaced with the client address from services.ClientIP.
//     PocketBase's own sign-in limiter and logs read RemoteAddr, so behind
//     Caddy they now see each visitor instead of one shared 127.0.0.1.
func RegisterClientIPMiddleware(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Bind(&hook.Handler[*core.RequestEvent]{
			Id:       "facetClientIP",
			Priority: clientIPMiddlewarePriority,
			Func: func(e *core.RequestEvent) error {
				normalizeClientAddress(e)
				return e.Next()
			},
		})
		return se.Next()
	})
}

func normalizeClientAddress(e *core.RequestEvent) {
	r := e.Request
	if r.Header.Get("X-Internal") != "" && !services.IsInternalRequest(r) {
		r.Header.Del("X-Internal")
	}
	ip := services.ClientIP(r)
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host == ip {
		return
	}
	r.RemoteAddr = net.JoinHostPort(ip, "0")
}

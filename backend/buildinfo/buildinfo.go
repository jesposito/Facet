package buildinfo

// These values are set by Docker builds via -ldflags. Local development builds
// keep the defaults so the identity endpoint remains available everywhere.
var (
	Version   = "unset"
	SHA       = "unset"
	BuildDate = "unset"
)

func ShortSHA() string {
	if len(SHA) <= 7 {
		return SHA
	}
	return SHA[:7]
}

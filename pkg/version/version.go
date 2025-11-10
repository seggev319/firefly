package version

// These variables are intended to be set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	BuiltAt = "unknown"
)

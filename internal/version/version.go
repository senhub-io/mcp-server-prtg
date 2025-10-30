package version

// AppVersion holds the current version of the application.
// This is set by main package at startup from build-time ldflags.
//
//nolint:gochecknoglobals // Application version is set once at startup and read-only thereafter.
var AppVersion = "dev"

// Set updates the application version.
func Set(v string) {
	AppVersion = v
}

// Get returns the current application version.
func Get() string {
	return AppVersion
}

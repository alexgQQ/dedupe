package utils

// This is to dynamically set the version on builds
// however it doesn't seem to work when packaged with main
// so it lives here instead

var (
	Version = ""
	Branch  = "dev"
	Commit  = ""
)

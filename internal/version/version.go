// Package version holds build information injected via -ldflags.
package version

// Version is set by the linker: -ldflags "-X github.com/philmehew/ali/internal/version.Version=v1.0.0"
var Version = "dev"

// Commit is the git commit hash, set by the linker.
var Commit = "unknown"

// BuildDate is the date the binary was compiled, set by the linker.
var BuildDate = "unknown"

// Author is the project author.
const Author = "Phil Mehew"

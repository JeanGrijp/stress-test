// Package version holds build-time metadata.
package version

import (
	"runtime/debug"
)

// Default values; override via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Attempt to derive version info from Go build info when ldflags are not provided
// (e.g., when installing with `go install module/path@version`).
//
// In that case, debug.ReadBuildInfo() typically yields Main.Version set to the
// semantic version (e.g., v1.2.3). VCS settings may be unavailable depending on
// how the module was built, so Commit/Date may remain as defaults.
//
// This keeps `stress-test version` meaningful for both releases (ldflags) and
// direct installs via `go install` (module version).
func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		// Set semantic version if ldflags didn't override it and build info has it
		if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
		// Best-effort: try to read VCS settings if available
		if Commit == "none" || Date == "unknown" {
			var vcsRev, vcsTime string
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					vcsRev = s.Value
				case "vcs.time":
					vcsTime = s.Value
				}
			}
			if Commit == "none" && vcsRev != "" {
				if len(vcsRev) > 12 {
					Commit = vcsRev[:12]
				} else {
					Commit = vcsRev
				}
			}
			if Date == "unknown" && vcsTime != "" {
				Date = vcsTime
			}
		}
	}
}

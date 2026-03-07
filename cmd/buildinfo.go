package cmd

import "runtime/debug"

func init() {
	if version == "dev" {
		applyBuildInfo()
	}
}

// applyBuildInfo reads Go's embedded build metadata and populates the version,
// commit, and buildDate vars when they were not injected via ldflags (e.g. go install).
func applyBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		version = v
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value != "" {
				commit = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				buildDate = s.Value
			}
		}
	}
}

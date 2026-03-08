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

	hasRealVersion := false
	if v := info.Main.Version; v != "" && v != "(devel)" {
		version = v
		hasRealVersion = true
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

	// go install downloads from the module proxy, which strips git metadata.
	// If we have a real version but no VCS info, mark them explicitly as N/A.
	if hasRealVersion {
		if commit == "none" {
			commit = "N/A"
		}
		if buildDate == "unknown" {
			buildDate = "N/A"
		}
	}
}

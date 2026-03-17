package new

import "github.com/charmbracelet/huh"

// ProjectFile is a reusable template-to-destination mapping for file generation.
type ProjectFile struct {
	TemplatePath string
	Destination  string
}

type projectSetup struct {
	appName              string
	moduleName           string
	useAir               bool
	includeAirConfig     bool
	useEnv               bool
	initGit              bool
	installDeps          bool
	skipInitialCommit    bool
	withoutStarterModule bool
	withFolderStructure  bool
}

var keelTheme = huh.ThemeCharm()

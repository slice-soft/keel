package new

import "github.com/charmbracelet/huh"

type projectFile struct {
	tmpl string
	dest string
}

type projectSetup struct {
	appName              string
	moduleName           string
	useAir               bool
	includeAirConfig     bool
	useEnv               bool
	initGit              bool
	installDeps          bool
	withoutStarterModule bool
	withFolderStructure  bool
}

var keelTheme = huh.ThemeCharm()

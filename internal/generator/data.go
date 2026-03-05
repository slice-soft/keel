package generator

const coreModulePath = "github.com/slice-soft/ss-keel-core"

// Data contains the variables available across all templates.
type Data struct {
	AppName            string // my-app
	ModuleName         string // github.com/user/my-app
	TemplateMode       string // new | init
	UseAir             bool   // use Air in the dev script
	UseAirConfig       bool   // use .air.toml in the dev script
	UseEnv             bool   // include .env support
	UseStarterModule   bool   // generate the default starter module
	UseFolderStructure bool   // create an opinionated folder structure
	PackageName        string // users
	PascalName         string // Users
	CamelName          string // users
	KebabName          string // users
	SnakeName          string // users
	CoreVersion        string // github.com/slice-soft/ss-keel-core v1.2.3
}

// NewData builds Data from a name in any supported format.
func NewData(name string) Data {
	pascal := toPascal(name)
	return Data{
		PackageName: toPackage(name),
		PascalName:  pascal,
		CamelName:   toCamel(pascal),
		KebabName:   toKebab(name),
		SnakeName:   toSnake(name),
	}
}

// NewProjectData builds Data for the `keel new` command.
func NewProjectData(appName, moduleName string, useAir, useAirConfig, useEnv, useStarterModule, useFolderStructure bool) Data {
	d := NewData(appName)
	d.AppName = appName
	d.ModuleName = moduleName
	d.TemplateMode = "new"
	d.UseAir = useAir
	d.UseAirConfig = useAirConfig
	d.UseEnv = useEnv
	d.UseStarterModule = useStarterModule
	d.UseFolderStructure = useFolderStructure
	d.CoreVersion, _ = getLatestModuleVersion(coreModulePath)
	return d
}

// NewInitData builds Data for the `keel init` command.
func NewInitData(appName string, useAir, airConfigExists bool) Data {
	d := NewData(appName)
	d.AppName = appName
	d.TemplateMode = "init"
	d.UseAir = useAir
	d.UseAirConfig = airConfigExists
	d.UseEnv = false
	return d
}

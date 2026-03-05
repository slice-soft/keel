package generator

const coreModulePath = "github.com/slice-soft/ss-keel-core"

// Data contiene las variables disponibles en todos los templates.
type Data struct {
	AppName            string // mi-app
	ModuleName         string // github.com/user/mi-app
	TemplateMode       string // new | init
	UseAir             bool   // usa Air en script dev
	UseAirConfig       bool   // usa .air.toml en script dev
	UseEnv             bool   // incluye soporte para .env
	UseStarterModule   bool   // crea módulo starter "hola"
	UseFolderStructure bool   // crea estructura de carpetas con middleware, guards, scheduler, checkers, events y hooks
	PackageName        string // users
	PascalName         string // Users
	CamelName          string // users
	KebabName          string // users
	SnakeName          string // users
	CoreVersion        string // github.com/slice-soft/ss-keel-core v1.2.3
}

// NewData construye el Data a partir del nombre en cualquier formato.
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

// NewProjectData construye el Data para un proyecto nuevo.
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

// NewInitData construye el Data para keel init.
func NewInitData(appName string, useAir, airConfigExists bool) Data {
	d := NewData(appName)
	d.AppName = appName
	d.TemplateMode = "init"
	d.UseAir = useAir
	d.UseAirConfig = airConfigExists
	d.UseEnv = false
	return d
}

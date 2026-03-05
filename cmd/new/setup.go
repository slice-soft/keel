package new

import (
	"fmt"
	"os"
	"strings"
)

func collectProjectSetup(args []string) (projectSetup, error) {
	initialAppName, err := resolveProjectName(args)
	if err != nil {
		return projectSetup{}, err
	}

	if yesFlag {
		return collectProjectSetupWithDefaults(initialAppName)
	}

	moduleName, err := promptModulePath(initialAppName)
	if err != nil {
		return projectSetup{}, err
	}

	appName, err := projectNameFromModule(moduleName)
	if err != nil {
		return projectSetup{}, err
	}

	if _, err := os.Stat(appName); err == nil {
		return projectSetup{}, fmt.Errorf("directory '%s' already exists", appName)
	}

	useAir, includeAirConfig, err := promptAirSetup()
	if err != nil {
		return projectSetup{}, err
	}

	useEnv, err := promptYesNo("Include .env support?", true)
	if err != nil {
		return projectSetup{}, err
	}

	initGit, err := promptYesNo("Initialize a new git repository?", true)
	if err != nil {
		return projectSetup{}, err
	}

	installDeps, err := promptYesNo("Install dependencies?", true)
	if err != nil {
		return projectSetup{}, err
	}

	return projectSetup{
		appName:              appName,
		moduleName:           moduleName,
		useAir:               useAir,
		includeAirConfig:     includeAirConfig,
		useEnv:               useEnv,
		initGit:              initGit,
		installDeps:          installDeps,
		withoutStarterModule: withoutStarterModule,
		withFolderStructure:  withFolderStructure,
	}, nil
}

func collectProjectSetupWithDefaults(appName string) (projectSetup, error) {
	if _, err := os.Stat(appName); err == nil {
		return projectSetup{}, fmt.Errorf("directory '%s' already exists", appName)
	}

	moduleName := defaultModulePath(appName)
	printAutomaticModulePathWarning()

	if !airInstalled() {
		fmt.Println("  ⚠  Air is not installed on your PATH.")
		fmt.Println("  Installing Air with: go install github.com/air-verse/air@latest")
		if err := installAirBinary(); err != nil {
			fmt.Printf("  ⚠  failed to install Air: %v\n", err)
		}
	}

	return projectSetup{
		appName:              appName,
		moduleName:           moduleName,
		useAir:               true,
		includeAirConfig:     true,
		useEnv:               true,
		initGit:              true,
		installDeps:          true,
		withoutStarterModule: withoutStarterModule,
		withFolderStructure:  withFolderStructure,
	}, nil
}

func printAutomaticModulePathWarning() {
	fmt.Println("  ⚠  Se creó el proyecto en modo automático.")
	fmt.Println("  ⚠  Debes revisar en go.mod que el module path sea el correcto (dominio y usuario, si aplica).")
	fmt.Println("  ⚠  Ajusta ese valor de forma consistente a nivel general del proyecto.")
}

func projectNameFromModule(moduleName string) (string, error) {
	parts := strings.Split(moduleName, "/")
	lastPart := parts[len(parts)-1]
	return resolveProjectName([]string{lastPart})
}

package new

import (
	"fmt"
	"os"
	"strings"
)

var resolveProjectNameFn = resolveProjectName
var promptModulePathFn = promptModulePath
var promptAirSetupFn = promptAirSetup
var promptYesNoFn = promptYesNo

func collectProjectSetup(args []string) (projectSetup, error) {
	initialAppName, err := resolveProjectNameFn(args)
	if err != nil {
		return projectSetup{}, err
	}

	if yesFlag {
		return collectProjectSetupWithDefaults(initialAppName)
	}

	moduleName, err := promptModulePathFn(initialAppName)
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

	useAir, includeAirConfig, err := promptAirSetupFn()
	if err != nil {
		return projectSetup{}, err
	}

	useEnv, err := promptYesNoFn("Include .env support?", true)
	if err != nil {
		return projectSetup{}, err
	}

	initGit, err := promptYesNoFn("Initialize a new git repository?", true)
	if err != nil {
		return projectSetup{}, err
	}

	installDeps, err := promptYesNoFn("Install dependencies?", true)
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

	if !airInstalledFn() {
		fmt.Println("  ⚠  Air is not installed on your PATH.")
		fmt.Println("  Installing Air with: go install github.com/air-verse/air@latest")
		if err := installAirBinaryFn(); err != nil {
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
		skipInitialCommit:    true,
		withoutStarterModule: withoutStarterModule,
		withFolderStructure:  withFolderStructure,
	}, nil
}

func printAutomaticModulePathWarning() {
	fmt.Println("  ⚠  Project was created in automatic mode.")
	fmt.Println("  ⚠  Review go.mod and ensure the module path is correct (domain and username, if applicable).")
	fmt.Println("  ⚠  Apply this change consistently across the entire project.")
	fmt.Println("  ⚠  The initial git commit will be skipped until you replace the placeholder module path.")
}

func projectNameFromModule(moduleName string) (string, error) {
	parts := strings.Split(moduleName, "/")
	lastPart := parts[len(parts)-1]
	return resolveProjectName([]string{lastPart})
}

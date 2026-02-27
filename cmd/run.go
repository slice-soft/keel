package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run [script]",
	Short: "Run a script defined in keel.toml",
	Long: `Runs scripts defined in the [scripts] section of keel.toml.
If no script name is given, an interactive selector is shown.

Example keel.toml:
  [scripts]
  dev   = "air -c .air.toml"
  build = "go build -o bin/app ./cmd/main.go"
  test  = "go test ./..."

Usage:
  keel run dev
  keel run build
  keel run test
  keel run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScript,
}

func runScript(cmd *cobra.Command, args []string) error {
	viper.SetConfigName("keel")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("keel.toml not found in the current directory")
	}

	scriptName := ""
	if len(args) > 0 {
		scriptName = args[0]
	}

	if scriptName == "" {
		scripts := viper.GetStringMapString("scripts")
		if len(scripts) == 0 {
			return fmt.Errorf("no scripts defined in keel.toml — add a [scripts] section")
		}

		// Sort names for stable display
		names := make([]string, 0, len(scripts))
		for n := range scripts {
			names = append(names, n)
		}
		sort.Strings(names)

		options := make([]huh.Option[string], 0, len(scripts))
		for _, n := range names {
			options = append(options, huh.NewOption(n+" — "+scripts[n], n))
		}

		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Which script to run?").
					Options(options...).
					Value(&scriptName),
			),
		).WithTheme(keelTheme).Run(); err != nil {
			return err
		}
	}

	scriptKey := fmt.Sprintf("scripts.%s", scriptName)
	script := viper.GetString(scriptKey)

	if script == "" {
		fmt.Printf("❌ Script '%s' not found in keel.toml\n\n", scriptName)
		fmt.Println("Available scripts:")
		scripts := viper.GetStringMapString("scripts")
		for name, s := range scripts {
			fmt.Printf("  %-12s %s\n", name, s)
		}
		return nil
	}

	fmt.Printf("⚓  keel run %s\n   → %s\n\n", scriptName, script)

	parts := strings.Fields(script)
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	return c.Run()
}

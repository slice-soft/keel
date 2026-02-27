package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run <script>",
	Short: "Run a script defined in keel.toml",
	Long: `Runs scripts defined in the [scripts] section of keel.toml.

Example keel.toml:
  [scripts]
  dev   = "air -c .air.toml"
  build = "go build -o bin/app ./cmd/main.go"
  test  = "go test ./..."

Usage:
  keel run dev
  keel run build
  keel run test`,
	Args: cobra.ExactArgs(1),
	RunE: runScript,
}

func runScript(cmd *cobra.Command, args []string) error {
	scriptName := args[0]

	viper.SetConfigName("keel")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("keel.toml not found in the current directory")
	}

	scriptKey := fmt.Sprintf("scripts.%s", scriptName)
	script := viper.GetString(scriptKey)

	if script == "" {
		fmt.Printf("❌ Script '%s' not found in keel.toml\n\n", scriptName)
		fmt.Println("Available scripts:")
		scripts := viper.GetStringMapString("scripts")
		for name, cmd := range scripts {
			fmt.Printf("  %-12s %s\n", name, cmd)
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

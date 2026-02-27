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
	Short: "Ejecuta un script definido en keel.toml",
	Long: `Ejecuta scripts definidos en la sección [scripts] del keel.toml.

Ejemplo de keel.toml:
  [scripts]
  dev   = "air -c .air.toml"
  build = "go build -o bin/app ./cmd/main.go"
  test  = "go test ./..."

Uso:
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
		return fmt.Errorf("no se encontró keel.toml en el directorio actual")
	}

	scriptKey := fmt.Sprintf("scripts.%s", scriptName)
	script := viper.GetString(scriptKey)

	if script == "" {
		// Mostrar scripts disponibles
		fmt.Printf("❌ Script '%s' no encontrado en keel.toml\n\n", scriptName)
		fmt.Println("Scripts disponibles:")
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

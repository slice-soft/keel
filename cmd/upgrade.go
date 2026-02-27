package cmd

import (
	"fmt"
	"os"

	"github.com/slice-soft/ss-keel-cli/internal/updater"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Actualiza keel a la última versión",
	Long: `Descarga e instala automáticamente la última versión de keel
desde los releases oficiales de GitHub.

El binario actual se reemplaza atómicamente — si algo falla
se restaura la versión anterior automáticamente.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updater.Upgrade(version)
	},
}

// init también agrega el comando version personalizado
// para mostrar más info que el default de cobra.
func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf(`
  ⚓  keel %s
  keel.slice-soft.dev
  github.com/slice-soft/keel-cli

`, "{{.Version}}"))

	// Sobrescribir el handler de --version para mostrar OS/arch también
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Muestra la versión instalada de keel",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\n  ⚓  keel %s\n", version)
			fmt.Printf("  OS/Arch: %s\n", getOSArch())
			fmt.Printf("  keel.slice-soft.dev\n\n")
		},
	})
}

func getOSArch() string {
	info, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	_ = info
	// runtime disponible via updater internamente
	return "ver keel version --help"
}

package cmd

import (
	"fmt"
	"os"

	"github.com/slice-soft/ss-keel-cli/internal/updater"
	"github.com/spf13/cobra"
)

// version es inyectada por GoReleaser en build time.
// En desarrollo local muestra "dev".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "keel",
	Version: version,
	Short:   "⚓ Keel CLI — Framework de Go bajo slice-soft",
	Long: `
  ⚓  K E E L  C L I
  ────────────────────────────────
  Framework de Go opinionado por slice-soft
  keel.slice-soft.dev
  ────────────────────────────────`,

	// PersistentPreRun corre antes de CUALQUIER subcomando.
	// Aquí iniciamos el chequeo de actualización en background.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// No chequeamos en el comando upgrade para evitar loop
		if cmd.Name() == "upgrade" {
			return
		}
		// Guardamos el canal en el contexto para leerlo en PersistentPostRun
		updateCh = updater.CheckAndNotify(version)
	},

	// PersistentPostRun corre después de CUALQUIER subcomando.
	// Aquí leemos el resultado del chequeo y mostramos el aviso si hay update.
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if updateCh == nil {
			return
		}
		if msg := <-updateCh; msg != "" {
			fmt.Print(msg)
		}
	},
}

// updateCh es el canal que conecta PreRun con PostRun.
var updateCh chan string

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(upgradeCmd)
}

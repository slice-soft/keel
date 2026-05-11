package run

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/slice-soft/keel/internal/updater"
	"github.com/spf13/cobra"
)

// updateNoticeFn is injected by the root command. It drains the pending CLI
// update channel and returns the notice string (or "" if none / timed out).
var updateNoticeFn func() string

// SetUpdateGetter wires up the CLI update notice source from the root command.
func SetUpdateGetter(fn func() string) {
	updateNoticeFn = fn
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "run [script]",
		Short:         "Run a script defined in keel.toml",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runScript,
	}

	return cmd
}

func runScript(cmd *cobra.Command, args []string) error {
	const keelConfig = "keel.toml"
	scriptName := args[0]

	if err := ensureKeelConfigExists(keelConfig); err != nil {
		return err
	}

	scripts, err := loadScriptsFromConfig(keelConfig)
	if err != nil {
		return err
	}

	scriptCmd, err := findScriptCommand(scripts, scriptName)
	if err != nil {
		return err
	}

	printStartupNotices()

	return executeScript(scriptCmd)
}

// printStartupNotices shows any pending CLI update (prominent) and outdated
// addon hints (subtle) before the script process starts.
func printStartupNotices() {
	// Start addon check in background while we wait for the CLI update notice.
	var addonCh chan []updater.AddonUpdate
	if goModData, err := os.ReadFile("go.mod"); err == nil {
		addonCh = updater.CheckAddonUpdatesAsync(string(goModData))
	}

	// CLI update — drain the injected channel with a short timeout.
	if updateNoticeFn != nil {
		if notice := updateNoticeFn(); notice != "" {
			fmt.Print(notice)
		}
	}

	// Addon updates — non-intrusive single line.
	if addonCh != nil {
		select {
		case updates := <-addonCh:
			if len(updates) > 0 {
				var names []string
				for _, u := range updates {
					names = append(names, fmt.Sprintf("%s (%s→%s)", u.Name, u.Current, u.Latest))
				}
				fmt.Printf("\n  💡 Outdated addons: %s\n     Run: keel doctor for details\n", strings.Join(names, ", "))
			}
		case <-time.After(time.Second):
			// timeout — skip, don't block startup
		}
	}
}

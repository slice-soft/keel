package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const versionBanner = `
██╗  ██╗███████╗███████╗██╗         ██████╗██╗     ██╗
██║ ██╔╝██╔════╝██╔════╝██║        ██╔════╝██║     ██║
█████╔╝ █████╗  █████╗  ██║        ██║     ██║     ██║
██╔═██╗ ██╔══╝  ██╔══╝  ██║        ██║     ██║     ██║
██║  ██╗███████╗███████╗███████╗   ╚██████╗███████╗██║
╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝    ╚═════╝╚══════╝╚═╝

                 ___/___
           _____/______|
       ____\            \__
       \                <  |
 ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
`

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBlue  = "\033[34m"
	colorWhite = "\033[97m"
)

func renderVersionOutput(cliVersion, cliCommit, cliBuildDate string) string {
	var b strings.Builder
	useColor := colorsEnabled()

	b.WriteString(colorize(versionBanner, colorCyan, useColor))
	b.WriteString("\n")

	b.WriteString(colorize(fmt.Sprintf("keel-cli: %s\n", cliVersion), colorWhite, useColor))
	b.WriteString(colorize(fmt.Sprintf("commit: %s\n", cliCommit), colorWhite, useColor))
	b.WriteString(colorize(fmt.Sprintf("build date: %s\n", cliBuildDate), colorWhite, useColor))
	b.WriteString(colorize(fmt.Sprintf("go: %s\n", runtime.Version()), colorWhite, useColor))
	b.WriteString(colorize(fmt.Sprintf("operating system: %s/%s\n", runtime.GOOS, runtime.GOARCH), colorWhite, useColor))

	b.WriteString("\n")

	b.WriteString(colorize("framework: Keel Framework (https://keel-go.dev)\n", colorBlue, useColor))
	b.WriteString(colorize("repository: Keel CLI Repository (https://github.com/slice-soft/keel)\n", colorBlue, useColor))

	return b.String()
}

func colorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}

func colorize(text, color string, enabled bool) string {
	if !enabled {
		return text
	}
	return color + text + colorReset
}

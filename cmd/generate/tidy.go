package generate

import (
	"os"
	"os/exec"

	"github.com/slice-soft/keel/internal/gomod"
)

var runGoModTidyFn = runGoModTidy

func runGoModTidy() error {
	return gomod.RunTidy(exec.Command, ".", os.Stdout, os.Stderr)
}

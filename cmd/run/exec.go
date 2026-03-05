package run

import (
	"os"
	"os/exec"
	"runtime"
)

func executeScript(script string) error {
	cmd := shellCommand(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func shellCommand(script string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", script)
	}
	return exec.Command("sh", "-c", script)
}

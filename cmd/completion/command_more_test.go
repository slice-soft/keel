package completion

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func stubCompletionPromptInputs(t *testing.T, inputs ...string) {
	t.Helper()
	previous := runCompletionPromptForm
	index := 0
	runCompletionPromptForm = func(form *huh.Form) error {
		if index >= len(inputs) {
			t.Fatalf("missing prompt input for completion form call #%d", index+1)
		}
		in := strings.NewReader(inputs[index])
		index++
		return form.WithAccessible(true).WithInput(in).WithOutput(io.Discard).Run()
	}
	t.Cleanup(func() {
		runCompletionPromptForm = previous
	})
}

func resetInstallDeps(t *testing.T) {
	t.Helper()
	previousUserHomeDirFn := userHomeDirFn
	previousResolveShellFn := resolveShellFn
	previousGenerateCompletionScriptFn := generateCompletionScriptFn
	previousWriteCompletionScriptFn := writeCompletionScriptFn
	previousResolveConfigFileFn := resolveConfigFileFn
	previousEnsureSourceLineFn := ensureSourceLineFn
	previousGenZshCompletionFn := genZshCompletionFn
	previousGenBashCompletionFn := genBashCompletionFn
	previousGenFishCompletionFn := genFishCompletionFn
	previousGenPowerShellCompletionFn := genPowerShellCompletionFn
	previousMkdirAllFn := mkdirAllFn
	previousWriteFileFn := writeFileFn
	previousReadFileFn := readFileFn
	previousOpenFileFn := openFileFn
	t.Cleanup(func() {
		userHomeDirFn = previousUserHomeDirFn
		resolveShellFn = previousResolveShellFn
		generateCompletionScriptFn = previousGenerateCompletionScriptFn
		writeCompletionScriptFn = previousWriteCompletionScriptFn
		resolveConfigFileFn = previousResolveConfigFileFn
		ensureSourceLineFn = previousEnsureSourceLineFn
		genZshCompletionFn = previousGenZshCompletionFn
		genBashCompletionFn = previousGenBashCompletionFn
		genFishCompletionFn = previousGenFishCompletionFn
		genPowerShellCompletionFn = previousGenPowerShellCompletionFn
		mkdirAllFn = previousMkdirAllFn
		writeFileFn = previousWriteFileFn
		readFileFn = previousReadFileFn
		openFileFn = previousOpenFileFn
	})
}

func TestNewGenerateAndInstallCommands(t *testing.T) {
	root := &cobra.Command{Use: "keel"}

	t.Run("generate command runs and supports errors", func(t *testing.T) {
		cmd := newGenerateCommand(root, "zsh")
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("expected zsh generator to succeed, got %v", err)
		}

		unsupported := newGenerateCommand(root, "unsupported")
		if err := unsupported.RunE(unsupported, nil); err == nil {
			t.Fatalf("expected unsupported shell error")
		}
	})

	t.Run("install command delegates to runner", func(t *testing.T) {
		previousRunInstallCommandFn := runInstallCommandFn
		t.Cleanup(func() {
			runInstallCommandFn = previousRunInstallCommandFn
		})

		called := false
		runInstallCommandFn = func(root *cobra.Command) error {
			called = true
			return nil
		}

		cmd := newInstallCommand(root)
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("expected install command to succeed, got %v", err)
		}
		if !called {
			t.Fatalf("expected install runner to be called")
		}
	})
}

func TestRunInstallErrors(t *testing.T) {
	root := &cobra.Command{Use: "keel"}

	t.Run("home resolution error", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "", errors.New("home failed") }

		if err := runInstall(root); err == nil {
			t.Fatalf("expected home resolution error")
		}
	})

	t.Run("resolve shell error", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "/tmp", nil }
		resolveShellFn = func(homeDir string) (string, error) { return "", errors.New("shell failed") }

		if err := runInstall(root); err == nil {
			t.Fatalf("expected resolve shell error")
		}
	})

	t.Run("generate script error", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "/tmp", nil }
		resolveShellFn = func(homeDir string) (string, error) { return "zsh", nil }
		generateCompletionScriptFn = func(root *cobra.Command, shell string) (string, error) {
			return "", errors.New("generate failed")
		}

		if err := runInstall(root); err == nil {
			t.Fatalf("expected generate script error")
		}
	})

	t.Run("write script error", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "/tmp", nil }
		resolveShellFn = func(homeDir string) (string, error) { return "zsh", nil }
		generateCompletionScriptFn = func(root *cobra.Command, shell string) (string, error) { return "script", nil }
		writeCompletionScriptFn = func(homeDir, shell, content string) (string, error) {
			return "", errors.New("write failed")
		}

		if err := runInstall(root); err == nil {
			t.Fatalf("expected write script error")
		}
	})

	t.Run("resolve config error", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "/tmp", nil }
		resolveShellFn = func(homeDir string) (string, error) { return "zsh", nil }
		generateCompletionScriptFn = func(root *cobra.Command, shell string) (string, error) { return "script", nil }
		writeCompletionScriptFn = func(homeDir, shell, content string) (string, error) { return "/tmp/keel.zsh", nil }
		resolveConfigFileFn = func(shell, homeDir string) (string, error) {
			return "", errors.New("config failed")
		}

		if err := runInstall(root); err == nil {
			t.Fatalf("expected resolve config error")
		}
	})

	t.Run("returns nil when config path is empty", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "/tmp", nil }
		resolveShellFn = func(homeDir string) (string, error) { return "zsh", nil }
		generateCompletionScriptFn = func(root *cobra.Command, shell string) (string, error) { return "script", nil }
		writeCompletionScriptFn = func(homeDir, shell, content string) (string, error) { return "/tmp/keel.zsh", nil }
		resolveConfigFileFn = func(shell, homeDir string) (string, error) { return "", nil }

		calledEnsure := false
		ensureSourceLineFn = func(configPath, sourceLine string) error {
			calledEnsure = true
			return nil
		}

		if err := runInstall(root); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if calledEnsure {
			t.Fatalf("ensureSourceLine should not be called when config path is empty")
		}
	})

	t.Run("ensure source line error", func(t *testing.T) {
		resetInstallDeps(t)
		userHomeDirFn = func() (string, error) { return "/tmp", nil }
		resolveShellFn = func(homeDir string) (string, error) { return "zsh", nil }
		generateCompletionScriptFn = func(root *cobra.Command, shell string) (string, error) { return "script", nil }
		writeCompletionScriptFn = func(homeDir, shell, content string) (string, error) { return "/tmp/keel.zsh", nil }
		resolveConfigFileFn = func(shell, homeDir string) (string, error) { return "/tmp/.zshrc", nil }
		ensureSourceLineFn = func(configPath, sourceLine string) error {
			return errors.New("source failed")
		}

		if err := runInstall(root); err == nil {
			t.Fatalf("expected ensure source line error")
		}
	})
}

func TestCompletionFileAndScriptErrorPaths(t *testing.T) {
	root := &cobra.Command{Use: "keel"}

	t.Run("generateCompletionScript returns shell-specific errors", func(t *testing.T) {
		resetInstallDeps(t)

		genZshCompletionFn = func(root *cobra.Command, out io.Writer) error { return errors.New("zsh fail") }
		if _, err := generateCompletionScript(root, "zsh"); err == nil {
			t.Fatalf("expected zsh generation error")
		}

		genBashCompletionFn = func(root *cobra.Command, out io.Writer) error { return errors.New("bash fail") }
		if _, err := generateCompletionScript(root, "bash"); err == nil {
			t.Fatalf("expected bash generation error")
		}

		genFishCompletionFn = func(root *cobra.Command, out io.Writer) error { return errors.New("fish fail") }
		if _, err := generateCompletionScript(root, "fish"); err == nil {
			t.Fatalf("expected fish generation error")
		}

		genPowerShellCompletionFn = func(root *cobra.Command, out io.Writer) error { return errors.New("pwsh fail") }
		if _, err := generateCompletionScript(root, "powershell"); err == nil {
			t.Fatalf("expected powershell generation error")
		}
	})

	t.Run("writeCompletionScript returns mkdir error", func(t *testing.T) {
		resetInstallDeps(t)
		mkdirAllFn = func(path string, perm os.FileMode) error {
			return errors.New("mkdir failed")
		}
		if _, err := writeCompletionScript("/tmp", "zsh", "script"); err == nil {
			t.Fatalf("expected mkdir error")
		}
	})

	t.Run("writeCompletionScript returns write error", func(t *testing.T) {
		resetInstallDeps(t)
		mkdirAllFn = func(path string, perm os.FileMode) error { return nil }
		writeFileFn = func(path string, data []byte, perm os.FileMode) error {
			return errors.New("write failed")
		}
		if _, err := writeCompletionScript("/tmp", "zsh", "script"); err == nil {
			t.Fatalf("expected write error")
		}
	})

	t.Run("ensureSourceLine returns read error", func(t *testing.T) {
		resetInstallDeps(t)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".zshrc")
		if err := os.WriteFile(configPath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed writing config seed: %v", err)
		}

		readFileFn = func(path string) ([]byte, error) {
			return nil, errors.New("read failed")
		}
		if err := ensureSourceLine(configPath, `source "/tmp/keel.zsh"`); err == nil {
			t.Fatalf("expected read error")
		}
	})

	t.Run("ensureSourceLine returns open error", func(t *testing.T) {
		resetInstallDeps(t)
		tmpDir := t.TempDir()
		openFileFn = func(path string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, errors.New("open failed")
		}
		if err := ensureSourceLine(filepath.Join(tmpDir, ".bashrc"), `source "/tmp/keel.bash"`); err == nil {
			t.Fatalf("expected open error")
		}
	})

	t.Run("sourceLineForShell fish branch", func(t *testing.T) {
		line := sourceLineForShell("fish", "/tmp/keel.fish")
		if line != `source "/tmp/keel.fish"` {
			t.Fatalf("unexpected fish source line: %q", line)
		}
	})

	t.Run("resolveConfigFile unknown shell returns empty path", func(t *testing.T) {
		path, err := resolveConfigFile("unknown-shell", t.TempDir())
		if err != nil {
			t.Fatalf("resolveConfigFile returned error: %v", err)
		}
		if path != "" {
			t.Fatalf("expected empty path, got %q", path)
		}
	})
}

func TestCompletionPromptsAndInteractiveBranches(t *testing.T) {
	t.Run("promptSelectShell", func(t *testing.T) {
		stubCompletionPromptInputs(t, "2\n")
		selected, err := promptSelectShell([]string{"zsh", "bash"})
		if err != nil {
			t.Fatalf("promptSelectShell returned error: %v", err)
		}
		if selected != "bash" {
			t.Fatalf("expected bash, got %q", selected)
		}
	})

	t.Run("promptSelectConfigFile", func(t *testing.T) {
		stubCompletionPromptInputs(t, "2\n")
		selected, err := promptSelectConfigFile("zsh", []string{"/tmp/.zshrc", "/tmp/.zprofile"})
		if err != nil {
			t.Fatalf("promptSelectConfigFile returned error: %v", err)
		}
		if selected != "/tmp/.zprofile" {
			t.Fatalf("expected /tmp/.zprofile, got %q", selected)
		}
	})

	t.Run("resolveShell prompts when interactive with multiple options", func(t *testing.T) {
		home := t.TempDir()
		if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# zsh"), 0644); err != nil {
			t.Fatalf("failed writing zshrc: %v", err)
		}
		if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("# bash"), 0644); err != nil {
			t.Fatalf("failed writing bashrc: %v", err)
		}
		t.Setenv("SHELL", "/bin/zsh")

		previousInteractive := isInteractiveTerminalFn
		isInteractiveTerminalFn = func() bool { return true }
		t.Cleanup(func() {
			isInteractiveTerminalFn = previousInteractive
		})

		stubCompletionPromptInputs(t, "2\n")
		shell, err := resolveShell(home)
		if err != nil {
			t.Fatalf("resolveShell returned error: %v", err)
		}
		if shell != "bash" {
			t.Fatalf("expected bash, got %q", shell)
		}
	})

	t.Run("resolveConfigFile prompts when interactive with multiple files", func(t *testing.T) {
		home := t.TempDir()
		if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# zsh"), 0644); err != nil {
			t.Fatalf("failed writing zshrc: %v", err)
		}
		if err := os.WriteFile(filepath.Join(home, ".zprofile"), []byte("# zprofile"), 0644); err != nil {
			t.Fatalf("failed writing zprofile: %v", err)
		}

		previousInteractive := isInteractiveTerminalFn
		isInteractiveTerminalFn = func() bool { return true }
		t.Cleanup(func() {
			isInteractiveTerminalFn = previousInteractive
		})

		stubCompletionPromptInputs(t, "2\n")
		path, err := resolveConfigFile("zsh", home)
		if err != nil {
			t.Fatalf("resolveConfigFile returned error: %v", err)
		}
		if path != filepath.Join(home, ".zprofile") {
			t.Fatalf("expected .zprofile path, got %q", path)
		}
	})
}

func TestIsInteractiveTerminalFalseForRegularFile(t *testing.T) {
	previousStdin := os.Stdin
	tmp, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("failed creating temp file: %v", err)
	}
	defer tmp.Close()

	os.Stdin = tmp
	t.Cleanup(func() {
		os.Stdin = previousStdin
	})

	if isInteractiveTerminal() {
		t.Fatalf("expected regular file stdin to be non-interactive")
	}
}

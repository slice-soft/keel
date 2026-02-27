package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner     = "slice-soft"
	repoName      = "keel-cli"
	checkInterval = 24 * time.Hour
)

// Release representa la respuesta de la GitHub API.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckAndNotify verifica si hay versión nueva en background.
// Retorna un canal que emite el mensaje de aviso (o string vacío si no hay update).
// No bloqueante — el caller lee el canal al final del comando.
func CheckAndNotify(currentVersion string) chan string {
	ch := make(chan string, 1)

	go func() {
		defer close(ch)

		if !shouldCheck() {
			ch <- ""
			return
		}

		latest, err := fetchLatestVersion()
		if err != nil || latest == "" {
			ch <- ""
			return
		}

		saveLastCheck()

		if isNewer(latest, currentVersion) {
			ch <- fmt.Sprintf(
				"\n  💡 Nueva versión disponible: %s (tienes %s)\n     Actualiza con: keel upgrade\n",
				latest, currentVersion,
			)
		} else {
			ch <- ""
		}
	}()

	return ch
}

// Upgrade descarga e instala el binario más reciente desde GitHub Releases.
func Upgrade(currentVersion string) error {
	fmt.Println("\n⚓  Verificando última versión...")

	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("error consultando GitHub: %w", err)
	}

	if !isNewer(release.TagName, currentVersion) {
		fmt.Printf("  ✅ Ya tienes la versión más reciente (%s)\n\n", currentVersion)
		return nil
	}

	fmt.Printf("  Nueva versión: %s (tienes %s)\n", release.TagName, currentVersion)

	assetName := buildAssetName()
	downloadURL := ""
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no se encontró binario para %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("  Descargando %s...\n", assetName)

	tmpFile, err := downloadBinary(downloadURL)
	if err != nil {
		return fmt.Errorf("error descargando binario: %w", err)
	}
	defer os.Remove(tmpFile)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error obteniendo ruta del ejecutable: %w", err)
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	fmt.Println("  Instalando...")

	if err := replaceBinary(tmpFile, execPath); err != nil {
		return fmt.Errorf("error instalando: %w", err)
	}

	fmt.Printf("\n  ✅ keel actualizado a %s\n\n", release.TagName)
	return nil
}

func fetchLatestVersion() (string, error) {
	r, err := fetchLatestRelease()
	if err != nil {
		return "", err
	}
	return r.TagName, nil
}

func fetchLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API respondió %d", resp.StatusCode)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "keel-update-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

// replaceBinary reemplaza el binario actual atómicamente con backup de seguridad.
func replaceBinary(newBinary, targetPath string) error {
	if err := os.Chmod(newBinary, 0755); err != nil {
		return err
	}
	backupPath := targetPath + ".bak"
	if err := os.Rename(targetPath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(newBinary, targetPath); err != nil {
		os.Rename(backupPath, targetPath) // restaurar si falla
		return err
	}
	os.Remove(backupPath)
	return nil
}

// buildAssetName construye el nombre del asset según OS/arch.
// Debe coincidir exactamente con lo que GoReleaser genera.
func buildAssetName() string {
	name := fmt.Sprintf("keel_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// — Control de frecuencia —

func keelDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".keel")
}

func lastCheckFile() string {
	return filepath.Join(keelDir(), "last_check")
}

func shouldCheck() bool {
	data, err := os.ReadFile(lastCheckFile())
	if err != nil {
		return true
	}
	var last time.Time
	if err := last.UnmarshalText(data); err != nil {
		return true
	}
	return time.Since(last) > checkInterval
}

func saveLastCheck() {
	os.MkdirAll(keelDir(), 0755)
	data, _ := time.Now().MarshalText()
	os.WriteFile(lastCheckFile(), data, 0644)
}

// isNewer compara versiones semver de forma simple.
func isNewer(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	return latest != current && latest > current
}

package generate

import (
	"fmt"

	"github.com/slice-soft/keel/internal/addon"
	generator "github.com/slice-soft/keel/internal/generator/generate"
)

var installOfficialAddonFn = addon.InstallOfficialAlias
var ensurePersistenceAddonInstalledFn = ensurePersistenceAddonInstalled

func resolvePersistenceBackend(opts Options) (repositoryBackend, bool, error) {
	switch {
	case opts.UseGormPersistence && opts.UseMongoPersistence:
		return "", false, fmt.Errorf("--mongo and --gorm cannot be used together")
	case opts.UseGormPersistence:
		return repositoryBackendGorm, true, nil
	case opts.UseMongoPersistence:
		return repositoryBackendMongo, true, nil
	default:
		return repositoryBackendStub, false, nil
	}
}

func ensurePersistenceAddonInstalled(backend repositoryBackend) error {
	alias, modulePath, err := addonMetadataForBackend(backend)
	if err != nil {
		return err
	}
	if generator.IsAddonInstalled(modulePath) {
		return nil
	}

	fmt.Printf("  → %s addon not found in go.mod; installing official addon\n", alias)
	if _, err := installOfficialAddonFn(alias, false); err != nil {
		return fmt.Errorf("could not install %s addon: %w", alias, err)
	}
	return nil
}

func addonMetadataForBackend(backend repositoryBackend) (alias string, modulePath string, err error) {
	switch backend {
	case repositoryBackendGorm:
		return "gorm", gormAddonModulePath, nil
	case repositoryBackendMongo:
		return "mongo", mongoAddonModulePath, nil
	default:
		return "", "", fmt.Errorf("unsupported persistence backend: %s", backend)
	}
}

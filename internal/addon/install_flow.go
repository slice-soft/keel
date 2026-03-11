package addon

import "fmt"

// InstallRepo fetches an addon's manifest and executes its installation steps.
func InstallRepo(repo string) (*Manifest, error) {
	manifest, err := FetchManifest(repo)
	if err != nil {
		return nil, err
	}
	if err := Install(manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// InstallOfficialAlias resolves an official addon alias from the registry and installs it.
func InstallOfficialAlias(alias string, forceRefresh bool) (*Manifest, error) {
	registry, err := FetchRegistry(forceRefresh)
	if err != nil {
		return nil, err
	}

	repo, ok := registry.ResolveRepo(alias)
	if !ok {
		return nil, fmt.Errorf("addon alias %q is not in the official Keel addon registry", alias)
	}

	return InstallRepo(repo)
}

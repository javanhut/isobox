package ipkg

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type DependenciesConfig struct {
	Packages PackageGroups `toml:"packages"`
	Options  InstallOptions `toml:"options"`
}

type PackageGroups struct {
	Dev     []string `toml:"dev"`
	Shells  []string `toml:"shells"`
	Network []string `toml:"network"`
	Utils   []string `toml:"utils"`
	Custom  []string `toml:"custom"`
}

type InstallOptions struct {
	InstallDev     bool `toml:"install_dev"`
	InstallShells  bool `toml:"install_shells"`
	InstallNetwork bool `toml:"install_network"`
	InstallUtils   bool `toml:"install_utils"`
	InstallCustom  bool `toml:"install_custom"`
}

func LoadDependencies(path string) (*DependenciesConfig, error) {
	var config DependenciesConfig

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("failed to parse dependencies file: %w", err)
	}

	return &config, nil
}

func (pm *PackageManager) InstallFromConfig(configPath string) error {
	config, err := LoadDependencies(configPath)
	if err != nil {
		return err
	}

	var packagesToInstall []string

	// Collect packages based on options
	if config.Options.InstallDev {
		packagesToInstall = append(packagesToInstall, config.Packages.Dev...)
	}
	if config.Options.InstallShells {
		packagesToInstall = append(packagesToInstall, config.Packages.Shells...)
	}
	if config.Options.InstallNetwork {
		packagesToInstall = append(packagesToInstall, config.Packages.Network...)
	}
	if config.Options.InstallUtils {
		packagesToInstall = append(packagesToInstall, config.Packages.Utils...)
	}
	if config.Options.InstallCustom {
		packagesToInstall = append(packagesToInstall, config.Packages.Custom...)
	}

	if len(packagesToInstall) == 0 {
		fmt.Println("No packages selected for installation")
		return nil
	}

	fmt.Printf("Installing %d packages from configuration...\n", len(packagesToInstall))

	// Install packages
	failed := []string{}
	for _, pkg := range packagesToInstall {
		if pkg == "" {
			continue
		}

		fmt.Printf("\n==> Installing %s\n", pkg)
		if err := pm.Install(pkg); err != nil {
			fmt.Printf("Failed to install %s: %v\n", pkg, err)
			failed = append(failed, pkg)
		}
	}

	if len(failed) > 0 {
		fmt.Printf("\nFailed to install %d packages:\n", len(failed))
		for _, pkg := range failed {
			fmt.Printf("  - %s\n", pkg)
		}
		return fmt.Errorf("%d package(s) failed to install", len(failed))
	}

	fmt.Printf("\nSuccessfully installed all %d packages!\n", len(packagesToInstall))
	return nil
}

func (config *DependenciesConfig) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	return encoder.Encode(config)
}

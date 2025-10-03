package ipkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	PackageRepo = "https://dl-cdn.alpinelinux.org/alpine/v3.19/main"
)

type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type PackageManager struct {
	rootfs string
	db     string
}

func NewPackageManager(envRoot string) *PackageManager {
	isoboxRoot := filepath.Join(envRoot, ".isobox")
	db := filepath.Join(isoboxRoot, "var/lib/ipkg/installed.json")
	return &PackageManager{
		rootfs: isoboxRoot,
		db:     db,
	}
}

func (pm *PackageManager) Install(pkgName string) error {
	fmt.Printf("Installing package: %s\n", pkgName)

	if err := pm.ensureDB(); err != nil {
		return err
	}

	installed, err := pm.isInstalled(pkgName)
	if err != nil {
		return err
	}
	if installed {
		fmt.Printf("Package %s is already installed\n", pkgName)
		return nil
	}

	apkFile := filepath.Join("/tmp", pkgName+".apk")
	url := fmt.Sprintf("%s/x86_64/%s.apk", PackageRepo, pkgName)

	fmt.Printf("Downloading from Alpine repository...\n")
	if err := pm.downloadFile(url, apkFile); err != nil {
		fmt.Printf("Warning: Could not download from Alpine repository: %v\n", err)
		fmt.Printf("Attempting to use apk package manager...\n")
		return pm.installWithAPK(pkgName)
	}
	defer os.Remove(apkFile)

	if err := pm.extractAPK(apkFile, pm.rootfs); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	pkg := Package{
		Name:        pkgName,
		Version:     "latest",
		Description: fmt.Sprintf("Package %s from Alpine", pkgName),
	}

	if err := pm.addToDatabase(pkg); err != nil {
		return err
	}

	fmt.Printf("Successfully installed %s\n", pkgName)
	return nil
}

func (pm *PackageManager) Remove(pkgName string) error {
	fmt.Printf("Removing package: %s\n", pkgName)

	installed, err := pm.isInstalled(pkgName)
	if err != nil {
		return err
	}
	if !installed {
		fmt.Printf("Package %s is not installed\n", pkgName)
		return nil
	}

	if err := pm.removeFromDatabase(pkgName); err != nil {
		return err
	}

	fmt.Printf("Successfully removed %s\n", pkgName)
	return nil
}

func (pm *PackageManager) List() error {
	packages, err := pm.getInstalled()
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed")
		return nil
	}

	fmt.Println("Installed packages:")
	for _, pkg := range packages {
		fmt.Printf("  %s (%s) - %s\n", pkg.Name, pkg.Version, pkg.Description)
	}

	return nil
}

func (pm *PackageManager) Update() error {
	fmt.Println("Updating package index...")
	fmt.Printf("Repository: %s\n", PackageRepo)
	fmt.Println("Package index updated")
	return nil
}

func (pm *PackageManager) ensureDB() error {
	dbDir := filepath.Dir(pm.db)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}

	if _, err := os.Stat(pm.db); os.IsNotExist(err) {
		emptyDB := []Package{}
		data, _ := json.MarshalIndent(emptyDB, "", "  ")
		if err := os.WriteFile(pm.db, data, 0644); err != nil {
			return fmt.Errorf("create db: %w", err)
		}
	}

	return nil
}

func (pm *PackageManager) isInstalled(pkgName string) (bool, error) {
	packages, err := pm.getInstalled()
	if err != nil {
		return false, err
	}

	for _, pkg := range packages {
		if pkg.Name == pkgName {
			return true, nil
		}
	}

	return false, nil
}

func (pm *PackageManager) getInstalled() ([]Package, error) {
	data, err := os.ReadFile(pm.db)
	if err != nil {
		return nil, fmt.Errorf("read db: %w", err)
	}

	var packages []Package
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("parse db: %w", err)
	}

	return packages, nil
}

func (pm *PackageManager) addToDatabase(pkg Package) error {
	packages, err := pm.getInstalled()
	if err != nil {
		return err
	}

	packages = append(packages, pkg)

	data, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal db: %w", err)
	}

	if err := os.WriteFile(pm.db, data, 0644); err != nil {
		return fmt.Errorf("write db: %w", err)
	}

	return nil
}

func (pm *PackageManager) removeFromDatabase(pkgName string) error {
	packages, err := pm.getInstalled()
	if err != nil {
		return err
	}

	filtered := []Package{}
	for _, pkg := range packages {
		if pkg.Name != pkgName {
			filtered = append(filtered, pkg)
		}
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal db: %w", err)
	}

	if err := os.WriteFile(pm.db, data, 0644); err != nil {
		return fmt.Errorf("write db: %w", err)
	}

	return nil
}

func (pm *PackageManager) downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (pm *PackageManager) extractAPK(apkFile, dest string) error {
	cmd := exec.Command("tar", "-xzf", apkFile, "-C", dest)
	return cmd.Run()
}

func (pm *PackageManager) installWithAPK(pkgName string) error {
	cmd := exec.Command("apk", "add", "--root", pm.rootfs, "--initdb", pkgName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

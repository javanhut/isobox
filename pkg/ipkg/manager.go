package ipkg

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	AlpineMainRepo      = "https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/"
	AlpineCommunityRepo = "https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/"
)

var packageAliases = map[string]string{
	"nvim":   "neovim",
	"vim":    "vim",
	"python": "python3",
	"py":     "python3",
	"pip":    "py3-pip",
}

var soLibraryMap = map[string]string{
	"so:libluv.so.1":        "luv",
	"so:libtermkey.so.1":    "libtermkey",
	"so:libvterm.so.0":      "libvterm",
	"so:libmsgpack-c.so.2":  "msgpack-c",
	"so:libtree-sitter.so.0": "tree-sitter",
	"so:libunibilium.so.4":  "unibilium",
	"so:libintl.so.8":       "musl-libintl",
	"so:libluajit-5.1.so.2": "luajit",
	"so:libuv.so.1":         "libuv",
	"so:libssl.so.3":        "libssl3",
	"so:libcrypto.so.3":     "libcrypto3",
	"so:libz.so.1":          "zlib",
	"so:libpcre2-8.so.0":    "pcre2",
	"so:libcurl.so.4":       "libcurl",
	"so:libnghttp2.so.14":   "nghttp2-libs",
	"so:libbrotlidec.so.1":  "brotli-libs",
	"so:libpsl.so.5":        "libpsl",
	"so:libc-ares.so.2":     "c-ares",
	"so:libidn2.so.0":       "libidn2",
	"so:libunistring.so.5":  "libunistring",
	"so:libncursesw.so.6":   "ncurses-libs",
	"so:libreadline.so.8":   "readline",
	"so:libonig.so.5":       "oniguruma",
}

type Package struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Installed   time.Time `json:"installed"`
}

type PackageManager struct {
	rootfs     string
	db         string
	installing map[string]bool
}

func NewPackageManager(envRoot string) *PackageManager {
	isoboxRoot := filepath.Join(envRoot, ".isobox")
	db := filepath.Join(isoboxRoot, "var/lib/ipkg/installed.json")
	return &PackageManager{
		rootfs:     isoboxRoot,
		db:         db,
		installing: make(map[string]bool),
	}
}

// NewInternalPackageManager creates a package manager for use inside the isolated environment
// where the root is already the isolated filesystem
func NewInternalPackageManager() *PackageManager {
	db := "/var/lib/ipkg/installed.json"
	return &PackageManager{
		rootfs:     "/",
		db:         db,
		installing: make(map[string]bool),
	}
}

func (pm *PackageManager) resolvePackageName(pkgName string) string {
	if alias, ok := packageAliases[pkgName]; ok {
		return alias
	}
	return pkgName
}

func (pm *PackageManager) Install(pkgName string) error {
	if err := pm.ensureDB(); err != nil {
		return err
	}

	originalName := pkgName
	pkgName = pm.resolvePackageName(pkgName)

	if originalName != pkgName {
		fmt.Printf("Installing %s (mapped to: %s)...\n", originalName, pkgName)
	} else {
		fmt.Printf("Installing %s...\n", pkgName)
	}

	fmt.Println("Resolving dependencies...")
	return pm.installWithDeps(pkgName)
}

func (pm *PackageManager) installWithDeps(pkgName string) error {
	// Prevent circular dependencies
	if pm.installing[pkgName] {
		return nil
	}

	// Check if already installed
	installed, err := pm.isInstalled(pkgName)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	pm.installing[pkgName] = true
	defer func() { delete(pm.installing, pkgName) }()

	// Find and download package
	apkURL, err := pm.findPackage(pkgName)
	if err != nil {
		return fmt.Errorf("package %s not found in Alpine repositories", pkgName)
	}

	cacheDir := filepath.Join(pm.rootfs, "var/cache/isobox")
	os.MkdirAll(cacheDir, 0755)
	apkFile := filepath.Join(cacheDir, pkgName+".apk")

	fmt.Printf("  Downloading %s...\n", pkgName)
	if err := pm.downloadFile(apkURL, apkFile); err != nil {
		return fmt.Errorf("failed to download %s: %w", pkgName, err)
	}
	defer os.Remove(apkFile)

	// Parse dependencies
	deps, err := pm.parseDependencies(apkFile)
	if err != nil {
		return fmt.Errorf("failed to parse dependencies for %s: %w", pkgName, err)
	}

	// Install dependencies first
	for _, dep := range deps {
		if dep == "" {
			continue
		}

		// Skip invalid dependency types
		if strings.HasPrefix(dep, "cmd:") || strings.HasPrefix(dep, "pc:") || strings.HasPrefix(dep, "/") {
			continue
		}

		// Map so: dependencies
		if strings.HasPrefix(dep, "so:") {
			if mappedPkg, ok := soLibraryMap[dep]; ok {
				dep = mappedPkg
			} else {
				continue // Skip unknown so: dependencies
			}
		}

		// Remove version constraints
		dep = strings.Split(dep, "=")[0]
		dep = strings.Split(dep, "<")[0]
		dep = strings.Split(dep, ">")[0]
		dep = strings.TrimSpace(dep)

		if dep == "" {
			continue
		}

		if err := pm.installWithDeps(dep); err != nil {
			fmt.Printf("  Warning: Failed to install dependency %s: %v\n", dep, err)
		}
	}

	// Extract the package
	fmt.Printf("  Installing %s...\n", pkgName)
	if err := pm.extractAPK(apkFile); err != nil {
		return fmt.Errorf("failed to extract %s: %w", pkgName, err)
	}

	// Add to database
	pkg := Package{
		Name:      pkgName,
		Version:   "latest",
		Installed: time.Now(),
	}

	if err := pm.addToDatabase(pkg); err != nil {
		return err
	}

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
	fmt.Printf("Main repository: %s\n", AlpineMainRepo)
	fmt.Printf("Community repository: %s\n", AlpineCommunityRepo)
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

func (pm *PackageManager) findPackage(pkgName string) (string, error) {
	// Try main repository first
	url, err := pm.searchRepo(AlpineMainRepo, pkgName)
	if err == nil {
		return url, nil
	}

	// Try community repository
	url, err = pm.searchRepo(AlpineCommunityRepo, pkgName)
	if err == nil {
		return url, nil
	}

	return "", fmt.Errorf("package not found")
}

func (pm *PackageManager) searchRepo(repoURL, pkgName string) (string, error) {
	resp, err := http.Get(repoURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch repo index")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Search for package in HTML index
	pattern := fmt.Sprintf(`href="(%s-[^"]*\.apk)"`, regexp.QuoteMeta(pkgName))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return "", fmt.Errorf("package not found in repo")
	}

	return repoURL + matches[1], nil
}

func (pm *PackageManager) parseDependencies(apkFile string) ([]string, error) {
	file, err := os.Open(apkFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Find .PKGINFO file in the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Name == ".PKGINFO" {
			// Read PKGINFO content
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, tr); err != nil {
				return nil, err
			}

			// Parse dependencies
			var deps []string
			scanner := bufio.NewScanner(buf)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "depend = ") {
					dep := strings.TrimPrefix(line, "depend = ")
					deps = append(deps, dep)
				}
			}

			return deps, nil
		}
	}

	return []string{}, nil
}

func (pm *PackageManager) extractAPK(apkFile string) error {
	file, err := os.Open(apkFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip metadata files
		if strings.HasPrefix(header.Name, ".") {
			continue
		}

		target := filepath.Join(pm.rootfs, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			// Create file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

		case tar.TypeSymlink:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			// Remove existing file/symlink
			os.Remove(target)

			// Create symlink
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	return nil
}

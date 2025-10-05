package environment

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Environment struct {
	Root      string    `json:"root"`
	Created   time.Time `json:"created"`
	IsoboxDir string    `json:"isobox_dir"`
	Username  string    `json:"username"`
	Shell     string    `json:"shell"`
}

func getBaseCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/isobox-base-system.tar.gz"
	}
	cacheDir := filepath.Join(home, ".cache", "isobox")
	os.MkdirAll(cacheDir, 0755)
	return filepath.Join(cacheDir, "base-system.tar.gz")
}

func RebuildCache() error {
	cachePath := getBaseCachePath()

	if _, err := os.Stat(cachePath); err == nil {
		fmt.Printf("Deleting old cache: %s\n", cachePath)
		if err := os.Remove(cachePath); err != nil {
			return fmt.Errorf("failed to delete cache: %w", err)
		}
	}

	fmt.Println("Rebuilding base system cache...")
	if err := buildBaseSystem(cachePath); err != nil {
		return fmt.Errorf("rebuild failed: %w", err)
	}

	return nil
}

func buildBaseSystem(cachePath string) error {
	// Install performance optimization tools
	installOptimizationTools()

	tmpDir, err := os.MkdirTemp("", "isobox-base-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("\nSetting up base system...")
	fmt.Println("This will be cached for faster initialization in the future.")

	tmpEnv := &Environment{
		IsoboxDir: tmpDir,
	}

	dirs := []string{
		"bin", "sbin",
		"usr/bin", "usr/sbin", "usr/local/bin",
		"lib", "lib64", "usr/lib", "usr/lib64",
		"etc", "etc/profile.d",
		"dev", "proc", "sys", "tmp",
		"var/lib/ipkg", "var/log", "var/tmp", "var/cache/isobox",
		"mnt", "opt", "srv", "run",
	}

	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	fmt.Println("\nSetting up POSIX binaries...")
	busybox := findBusybox()
	if busybox == "" {
		return fmt.Errorf("BusyBox not found")
	}
	if err := tmpEnv.setupWithBusybox(busybox); err != nil {
		return err
	}

	if err := tmpEnv.setupInternalPackageManager(); err != nil {
		return err
	}

	if err := tmpEnv.setupShells(); err != nil {
		fmt.Printf("  Warning: Shell setup failed: %v\n", err)
	}

	fmt.Println("\nCreating base system tarball...")
	fmt.Print("  Compressing... ")

	// Use pigz for parallel compression if available, otherwise use gzip
	var cmd *exec.Cmd
	if _, err := exec.LookPath("pigz"); err == nil {
		// pigz uses all available CPU cores by default
		cmd = exec.Command("tar", "--use-compress-program=pigz", "-cf", cachePath, "-C", tmpDir, ".")
	} else {
		cmd = exec.Command("tar", "-czf", cachePath, "-C", tmpDir, ".")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create tarball: %w (output: %s)", err, string(output))
	}

	fmt.Println("Done!")
	fmt.Printf("Base system cached at: %s\n", cachePath)
	return nil
}

func installOptimizationTools() {
	// Check if tools are already installed
	hasAria2c := false
	hasPigz := false

	if _, err := exec.LookPath("aria2c"); err == nil {
		hasAria2c = true
	}
	if _, err := exec.LookPath("pigz"); err == nil {
		hasPigz = true
	}

	if hasAria2c && hasPigz {
		return
	}

	// Detect OS
	osRelease, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return
	}

	osInfo := string(osRelease)
	var installCmd []string
	var packages []string

	if !hasAria2c {
		packages = append(packages, "aria2")
	}
	if !hasPigz {
		packages = append(packages, "pigz")
	}

	// Determine package manager based on OS
	if strings.Contains(osInfo, "ID=arch") || strings.Contains(osInfo, "ID_LIKE=arch") {
		installCmd = append([]string{"pacman", "-S", "--noconfirm", "--needed"}, packages...)
	} else if strings.Contains(osInfo, "ID=ubuntu") || strings.Contains(osInfo, "ID=debian") || strings.Contains(osInfo, "ID_LIKE=debian") {
		installCmd = append([]string{"apt-get", "install", "-y"}, packages...)
	} else if strings.Contains(osInfo, "ID=fedora") || strings.Contains(osInfo, "ID=rhel") || strings.Contains(osInfo, "ID_LIKE=\"rhel fedora\"") {
		installCmd = append([]string{"dnf", "install", "-y"}, packages...)
	} else if strings.Contains(osInfo, "ID=centos") {
		installCmd = append([]string{"yum", "install", "-y"}, packages...)
	} else if strings.Contains(osInfo, "ID=opensuse") || strings.Contains(osInfo, "ID_LIKE=\"suse\"") {
		installCmd = append([]string{"zypper", "install", "-y"}, packages...)
	} else if strings.Contains(osInfo, "ID=alpine") {
		installCmd = append([]string{"apk", "add"}, packages...)
	} else {
		// Unknown OS, skip installation
		return
	}

	fmt.Printf("Installing performance tools (%s)...\n", strings.Join(packages, ", "))

	cmd := exec.Command("sudo", installCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("  Warning: Failed to install optimization tools: %v\n", err)
		fmt.Printf("  Continuing with standard tools (wget, gzip)\n")
	} else {
		fmt.Println("  Performance tools installed successfully!")
	}
}

func extractBaseSystem(targetDir, cachePath string) error {
	fmt.Print("Extracting base system... ")

	// Use pigz for parallel decompression if available
	var cmd *exec.Cmd
	if _, err := exec.LookPath("pigz"); err == nil {
		cmd = exec.Command("tar", "--use-compress-program=pigz", "-xf", cachePath, "-C", targetDir)
	} else {
		cmd = exec.Command("tar", "-xzf", cachePath, "-C", targetDir)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract tarball: %w (output: %s)", err, string(output))
	}
	fmt.Println("Done!")
	return nil
}

func Initialize(path string, shell string) (*Environment, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	isoboxDir := filepath.Join(absPath, ".isobox")
	username := filepath.Base(absPath)

	if shell == "" {
		shell = "bash"
	}

	env := &Environment{
		Root:      absPath,
		Created:   time.Now(),
		IsoboxDir: isoboxDir,
		Username:  username,
		Shell:     shell,
	}

	baseCachePath := getBaseCachePath()

	if _, err := os.Stat(baseCachePath); os.IsNotExist(err) {
		fmt.Println("Building base system (first time only, this will be cached)...")
		if err := buildBaseSystem(baseCachePath); err != nil {
			return nil, fmt.Errorf("build base system: %w", err)
		}
	} else {
		fmt.Println("Using cached base system...")
	}

	fmt.Println("Creating isolated Linux filesystem...")

	if err := os.MkdirAll(env.IsoboxDir, 0755); err != nil {
		return nil, fmt.Errorf("create isobox directory: %w", err)
	}

	if err := extractBaseSystem(env.IsoboxDir, baseCachePath); err != nil {
		return nil, fmt.Errorf("extract base system: %w", err)
	}

	if err := env.createIsolatedFilesystem(); err != nil {
		return nil, err
	}

	if err := env.createEssentialFiles(); err != nil {
		return nil, err
	}

	if err := env.save(); err != nil {
		return nil, err
	}

	return env, nil
}

func Load(path string) (*Environment, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	configPath := filepath.Join(absPath, ".isobox/config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &env, nil
}

func (e *Environment) createIsolatedFilesystem() error {
	// Create user-specific directories
	dirs := []string{
		"root",
		"home",
	}

	for _, dir := range dirs {
		path := filepath.Join(e.IsoboxDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		fmt.Printf("  Created: .isobox/%s\n", dir)
	}

	// Create user home directory
	userHome := filepath.Join(e.IsoboxDir, "home", e.Username)
	if err := os.MkdirAll(userHome, 0755); err != nil {
		return fmt.Errorf("create user home: %w", err)
	}

	// Set ownership to user (UID 1000)
	chownCmd := exec.Command("sudo", "chown", "1000:1000", userHome)
	if err := chownCmd.Run(); err != nil {
		fmt.Printf("  Warning: failed to set ownership: %v\n", err)
	}

	fmt.Printf("  Created: .isobox/home/%s\n", e.Username)

	tmpPath := filepath.Join(e.IsoboxDir, "tmp")
	if err := os.Chmod(tmpPath, 01777); err != nil {
		return fmt.Errorf("chmod tmp: %w", err)
	}

	if err := e.createDeviceNodes(); err != nil {
		return fmt.Errorf("create device nodes: %w", err)
	}

	return nil
}

func (e *Environment) createDeviceNodes() error {
	devDir := filepath.Join(e.IsoboxDir, "dev")

	devices := []struct {
		name  string
		major int
		minor int
	}{
		{"null", 1, 3},
		{"zero", 1, 5},
		{"random", 1, 8},
		{"urandom", 1, 9},
		{"tty", 5, 0},
	}

	createdCount := 0
	for _, dev := range devices {
		devPath := filepath.Join(devDir, dev.name)

		// Check if device node already exists
		if _, err := os.Stat(devPath); err == nil {
			continue
		}

		cmd := exec.Command("sudo", "mknod", "-m", "666", devPath, "c", fmt.Sprintf("%d", dev.major), fmt.Sprintf("%d", dev.minor))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("mknod %s: %w", dev.name, err)
		}
		createdCount++
	}

	if createdCount > 0 {
		fmt.Println("  Created device nodes: /dev/null, /dev/zero, /dev/random, /dev/urandom, /dev/tty")
	}
	return nil
}

func (e *Environment) setupBinaries() error {
	fmt.Println("\nSetting up POSIX binaries...")

	busybox := findBusybox()
	if busybox != "" {
		return e.setupWithBusybox(busybox)
	}

	return e.setupWithSystemBinaries()
}

func (e *Environment) setupWithBusybox(busyboxPath string) error {
	fmt.Printf("Found BusyBox at: %s\n", busyboxPath)

	destBusybox := filepath.Join(e.IsoboxDir, "bin/busybox")
	if err := copyBinary(busyboxPath, destBusybox); err != nil {
		return fmt.Errorf("copy busybox: %w", err)
	}

	if err := os.Chmod(destBusybox, 0755); err != nil {
		return err
	}

	binDir := filepath.Join(e.IsoboxDir, "bin")
	cmd := exec.Command("./busybox", "--install", "-s", ".")
	cmd.Dir = binDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install busybox applets: %w", err)
	}

	if err := e.fixBusyboxSymlinks(); err != nil {
		return fmt.Errorf("fix symlinks: %w", err)
	}

	if err := e.addSSLCapableTools(); err != nil {
		fmt.Printf("  Warning: SSL tools setup failed: %v\n", err)
	}

	if err := e.installMuslLibc(); err != nil {
		fmt.Printf("  Warning: musl libc setup failed: %v\n", err)
	}

	if err := e.installAlpineBaseDeps(); err != nil {
		fmt.Printf("  Warning: Alpine base dependencies setup failed: %v\n", err)
	}

	fmt.Println("  Installed BusyBox and 150+ Unix utilities")
	return nil
}

func (e *Environment) installAlpineBaseDeps() error {
	deps := []string{
		// Core C/C++ runtime libraries
		"musl",
		"libgcc",

		// Compression libraries
		"zlib",
		"libbz2",
		"xz-libs",
		"zstd-libs",
		"lz4-libs",

		// SSL/TLS and networking
		"libssl3",
		"libcrypto3",
		"ca-certificates-bundle",
		"libcurl",
		"nghttp2-libs",
		"c-ares",

		// Text processing and regex
		"pcre2",
		"grep",
		"sed",
		"gawk",

		// Internationalization
		"libidn2",
		"libunistring",

		// Common utilities
		"coreutils",
		"findutils",
		"tar",
		"gzip",
		"file",
		"diffutils",
		"patch",

		// Archive handling
		"brotli-libs",
		"libpsl",

		// Common libraries for applications
		"libffi",
		"libuuid",
		"sqlite-libs",
		"expat",
		"libxml2",
		"libxslt",
		"yaml",
		"gmp",
		"mpfr4",
		"libgomp",
		"mpc1",

		// JSON/data processing
		"jansson",
		"jq",

		// Network utilities
		"libevent",
		"libarchive",
		"curl",

		// Development essentials
		"pkgconf",
		"binutils",
		"make",

		// Additional runtime support
		"tzdata",
		"attr",
		"libcap",

		// Process management
		"procps-ng",
		"util-linux",

		// Additional shell utilities
		"less",
		"which",
		"nano",
		// Essential Libraries for utils
		"ncurses-terminfo-base",
		"libncursesw",
		"libformw",
		"libmenuw",
		"libpanelw",
		"readline",
		"libacl",
		"libattr",
		"utmps-libs",
		"s6",
		"skalibs",
		"oniguruma",
		"oniguruma-dev",
		"luv",
		"libtermkey",
		"libvterm",
		"msgpack-c",
		"tree-sitter",
		"unibilium",
		"musl-libintl",
		"luajit",
		"libuv",
	}

	return e.installAlpinePackages(deps)
}

func (e *Environment) addSSLCapableTools() error {
	if err := e.setupSSLCertificates(); err != nil {
		fmt.Printf("  Warning: SSL certificates setup failed: %v\n", err)
		return err
	}

	packages := []string{"wget", "ca-certificates"}
	return e.installAlpinePackages(packages)
}

// installAlpinePackages installs multiple packages efficiently by caching repo indices
func (e *Environment) installAlpinePackages(packages []string) error {
	repos := []string{
		"https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/",
		"https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/",
	}

	// Build package URL cache in parallel
	fmt.Printf("  Building package index...\n")
	pkgURLs := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Download repo indices concurrently
	for _, baseURL := range repos {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			cmd := exec.Command("wget", "-qO-", url)
			output, err := cmd.Output()
			if err != nil {
				return
			}

			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				for _, pkgName := range packages {
					if strings.Contains(line, fmt.Sprintf(`href="%s-`, pkgName)) {
						start := strings.Index(line, `href="`) + 6
						end := strings.Index(line[start:], `"`)
						if end > 0 {
							pkgFile := line[start : start+end]
							if strings.HasPrefix(pkgFile, pkgName+"-") && strings.HasSuffix(pkgFile, ".apk") {
								mu.Lock()
								if _, exists := pkgURLs[pkgName]; !exists {
									pkgURLs[pkgName] = url + pkgFile
								}
								mu.Unlock()
								break
							}
						}
					}
				}
			}
		}(baseURL)
	}
	wg.Wait()

	totalPkgs := len(packages)
	fmt.Printf("  Installing %d packages...\n", totalPkgs)

	// Phase 1: Download all packages in parallel (16 workers)
	type downloadJob struct {
		name string
		url  string
	}

	type downloadResult struct {
		name    string
		tmpFile string
		success bool
	}

	downloadJobs := make(chan downloadJob, totalPkgs)
	downloadResults := make(chan downloadResult, totalPkgs)

	// Start 16 download workers
	numDownloadWorkers := 16
	for w := 0; w < numDownloadWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range downloadJobs {
				tmpFile, success := downloadPackageFile(job.name, job.url)
				downloadResults <- downloadResult{
					name:    job.name,
					tmpFile: tmpFile,
					success: success,
				}
			}
		}()
	}

	// Queue downloads
	for _, pkgName := range packages {
		pkgURL, found := pkgURLs[pkgName]
		if !found {
			downloadResults <- downloadResult{name: pkgName, success: false}
			continue
		}
		downloadJobs <- downloadJob{name: pkgName, url: pkgURL}
	}
	close(downloadJobs)

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(downloadResults)
	}()

	// Collect downloaded files
	downloadedFiles := make(map[string]string)
	downloadedCount := 0

	fmt.Print("  Downloading: [")
	barWidth := 40

	for result := range downloadResults {
		if result.success {
			downloadedFiles[result.name] = result.tmpFile
			downloadedCount++
		}

		// Update progress bar
		progress := float64(len(downloadedFiles)+1) / float64(totalPkgs)
		filledWidth := int(progress * float64(barWidth))

		fmt.Print("\r  Downloading: [")
		for i := 0; i < barWidth; i++ {
			if i < filledWidth {
				fmt.Print("=")
			} else if i == filledWidth {
				fmt.Print(">")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Printf("] %d/%d", len(downloadedFiles), totalPkgs)
	}
	fmt.Println()

	// Phase 2: Extract all packages in parallel (16 workers)
	type extractJob struct {
		name    string
		tmpFile string
	}

	extractJobs := make(chan extractJob, len(downloadedFiles))
	extractResults := make(chan bool, len(downloadedFiles))

	// Start 16 extraction workers
	numExtractWorkers := 16
	for w := 0; w < numExtractWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range extractJobs {
				success := e.extractPackageFile(job.tmpFile)
				os.Remove(job.tmpFile)
				extractResults <- success
			}
		}()
	}

	// Queue extractions
	for name, tmpFile := range downloadedFiles {
		extractJobs <- extractJob{name: name, tmpFile: tmpFile}
	}
	close(extractJobs)

	// Wait for all extractions to complete
	go func() {
		wg.Wait()
		close(extractResults)
	}()

	// Display extraction progress
	installedCount := 0
	extractedCount := 0

	fmt.Print("  Extracting: [")

	for success := range extractResults {
		if success {
			installedCount++
		}
		extractedCount++

		// Update progress bar
		progress := float64(extractedCount) / float64(len(downloadedFiles))
		filledWidth := int(progress * float64(barWidth))

		fmt.Print("\r  Extracting: [")
		for i := 0; i < barWidth; i++ {
			if i < filledWidth {
				fmt.Print("=")
			} else if i == filledWidth {
				fmt.Print(">")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Printf("] %d/%d", extractedCount, len(downloadedFiles))
	}

	fmt.Printf("\n  Added %d Alpine packages\n", installedCount)
	return nil
}

// downloadPackageFile downloads a single package using aria2c or wget
func downloadPackageFile(pkgName, pkgURL string) (string, bool) {
	tmpFile := fmt.Sprintf("/tmp/%s-%d.apk", pkgName, time.Now().UnixNano())

	// Try aria2c first (faster with multi-connection support)
	if _, err := exec.LookPath("aria2c"); err == nil {
		cmd := exec.Command("aria2c",
			"-x", "4", // 4 connections per download
			"-s", "4", // split into 4 segments
			"-k", "1M", // chunk size
			"--quiet=true",
			"--allow-overwrite=true",
			"-d", "/tmp",
			"-o", filepath.Base(tmpFile),
			pkgURL)
		if err := cmd.Run(); err == nil {
			return tmpFile, true
		}
	}

	// Fallback to wget
	cmd := exec.Command("wget", "-q", "-O", tmpFile, pkgURL)
	if err := cmd.Run(); err != nil {
		return "", false
	}

	return tmpFile, true
}

// extractPackageFile extracts a single package to the target directory
func (e *Environment) extractPackageFile(tmpFile string) bool {
	cmd := exec.Command("tar", "-xzf", tmpFile, "-C", e.IsoboxDir)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (e *Environment) installAlpinePackage(pkgName string) error {
	// Use Alpine v3.18 which still uses the old APK format (APKv2)
	// Alpine v3.19+ uses APKv3 which requires apk-tools to extract
	repos := []string{
		"https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/",
		"https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/",
	}

	var pkgURL string
	var pkgFile string

	for _, baseURL := range repos {
		cmd := exec.Command("wget", "-qO-", baseURL)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")

		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(`href="%s-`, pkgName)) {
				start := strings.Index(line, `href="`) + 6
				end := strings.Index(line[start:], `"`)
				if end > 0 {
					pkgFile = line[start : start+end]
					if strings.HasPrefix(pkgFile, pkgName+"-") && strings.HasSuffix(pkgFile, ".apk") {
						pkgURL = baseURL + pkgFile
						break
					}
				}
			}
		}

		if pkgURL != "" {
			break
		}
	}

	if pkgURL == "" {
		return fmt.Errorf("package %s not found in repositories", pkgName)
	}

	tmpFile := fmt.Sprintf("/tmp/%s.apk", pkgName)

	cmd := exec.Command("wget", "-q", "-O", tmpFile, pkgURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	// Alpine v3.18 uses APKv2 format which is a standard tar.gz
	cmd = exec.Command("tar", "-xzf", tmpFile, "-C", e.IsoboxDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract failed: %w (output: %s)", err, string(output))
	}

	return nil
}

func (e *Environment) setupSSLCertificates() error {
	certLocations := []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/ssl/cert.pem",
	}

	var certFile string
	for _, loc := range certLocations {
		if _, err := os.Stat(loc); err == nil {
			certFile = loc
			break
		}
	}

	if certFile == "" {
		return fmt.Errorf("no CA certificates found")
	}

	destDir := filepath.Join(e.IsoboxDir, "etc/ssl/certs")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile := filepath.Join(destDir, "ca-certificates.crt")
	if err := copyBinary(certFile, destFile); err != nil {
		return err
	}

	destFile2 := filepath.Join(e.IsoboxDir, "etc/ssl/cert.pem")
	os.MkdirAll(filepath.Dir(destFile2), 0755)
	copyBinary(certFile, destFile2)

	fmt.Println("  Added SSL certificates")
	return nil
}

func (e *Environment) installMuslLibc() error {
	if err := e.installAlpinePackage("musl"); err != nil {
		return fmt.Errorf("install musl: %w", err)
	}

	fmt.Println("  Added musl libc (for Alpine packages)")
	return nil
}

func (e *Environment) fixBusyboxSymlinks() error {
	binDir := filepath.Join(e.IsoboxDir, "bin")

	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == "busybox" {
			continue
		}

		linkPath := filepath.Join(binDir, entry.Name())
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}

			if filepath.IsAbs(target) {
				os.Remove(linkPath)
				if err := os.Symlink("busybox", linkPath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (e *Environment) setupWithSystemBinaries() error {
	fmt.Println("BusyBox not found, copying essential system binaries...")

	essentialBinaries := []string{
		"sh", "bash", "dash",
		"ls", "cat", "cp", "mv", "rm", "mkdir", "touch", "chmod", "chown",
		"grep", "sed", "awk", "cut", "sort", "uniq", "head", "tail",
		"echo", "printf", "test",
		"find", "which", "whereis", "file",
		"pwd", "cd", "env",
		"tar", "gzip", "gunzip", "bzip2", "xz",
		"wget", "curl",
		"ps", "top", "kill",
		"mount", "umount",
	}

	copiedCount := 0
	for _, bin := range essentialBinaries {
		systemPath, err := exec.LookPath(bin)
		if err != nil {
			continue
		}

		destPath := filepath.Join(e.IsoboxDir, "bin", bin)
		if err := copyBinary(systemPath, destPath); err != nil {
			fmt.Printf("  Warning: failed to copy %s: %v\n", bin, err)
			continue
		}

		copiedCount++
		fmt.Printf("  Copied: %s\n", bin)
	}

	fmt.Printf("  Copied %d essential binaries\n", copiedCount)
	return nil
}

func (e *Environment) setupLibraries() error {
	fmt.Println("\nSetting up shared libraries...")

	binDir := filepath.Join(e.IsoboxDir, "bin")
	binaries, err := os.ReadDir(binDir)
	if err != nil {
		return fmt.Errorf("read bin dir: %w", err)
	}

	copiedLibs := make(map[string]bool)
	processedBinaries := make(map[string]bool)

	for _, entry := range binaries {
		if entry.IsDir() {
			continue
		}

		binPath := filepath.Join(binDir, entry.Name())

		realBinPath := binPath
		info, err := os.Lstat(binPath)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(binPath)
			if err != nil {
				continue
			}
			realBinPath = target
		}

		if processedBinaries[realBinPath] {
			continue
		}
		processedBinaries[realBinPath] = true

		libs, err := getRequiredLibraries(realBinPath)
		if err != nil {
			continue
		}

		for _, lib := range libs {
			if copiedLibs[lib] {
				continue
			}

			if err := e.copyLibrary(lib); err != nil {
				fmt.Printf("  Warning: failed to copy %s: %v\n", lib, err)
				continue
			}

			copiedLibs[lib] = true
		}
	}

	fmt.Printf("  Copied %d shared libraries\n", len(copiedLibs))

	if err := e.setupLdConfig(); err != nil {
		fmt.Printf("  Warning: ld config setup failed: %v\n", err)
	}

	return nil
}

func (e *Environment) copyLibrary(libPath string) error {
	if !filepath.IsAbs(libPath) {
		return fmt.Errorf("not absolute path: %s", libPath)
	}

	relPath := strings.TrimPrefix(libPath, "/")
	destPath := filepath.Join(e.IsoboxDir, relPath)

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	info, err := os.Lstat(libPath)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(libPath)
		if err != nil {
			return err
		}

		if err := os.Symlink(target, destPath); err != nil {
			return err
		}

		realPath, err := filepath.EvalSymlinks(libPath)
		if err == nil && realPath != libPath {
			return e.copyLibrary(realPath)
		}
		return nil
	}

	return copyBinary(libPath, destPath)
}

func (e *Environment) setupLdConfig() error {
	ldSoConf := filepath.Join(e.IsoboxDir, "etc/ld.so.conf")
	content := `/lib
/lib64
/usr/lib
/usr/lib64
`
	return os.WriteFile(ldSoConf, []byte(content), 0644)
}

func (e *Environment) setupInternalPackageManager() error {
	fmt.Println("\nSetting up internal package manager...")

	scriptPath := findInternalScript()
	if scriptPath == "" {
		return fmt.Errorf("internal package manager script not found (isobox-internal.sh)")
	}

	destPath := filepath.Join(e.IsoboxDir, "bin/isobox")

	if err := copyBinary(scriptPath, destPath); err != nil {
		return fmt.Errorf("copy isobox script: %w", err)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("chmod isobox script: %w", err)
	}

	fmt.Println("  Installed: /bin/isobox (internal package manager)")
	return nil
}

func (e *Environment) setupShells() error {
	fmt.Println("\nSetting up shells (bash, zsh, sh)...")

	// Install shell dependencies and shells together
	packages := []string{
		"python3",
		"gcc",
		"go",
		"vim",
		"bash",
		"zsh",
	}

	if err := e.installAlpinePackages(packages); err != nil {
		return fmt.Errorf("shell installation failed: %w", err)
	}

	fmt.Println("  sh is provided by BusyBox")
	return nil
}

func findInternalScript() string {
	searchPaths := []string{}

	// System installation paths (checked first)
	searchPaths = append(searchPaths, "/usr/local/share/isobox/scripts/isobox-internal.sh")
	searchPaths = append(searchPaths, "/usr/share/isobox/scripts/isobox-internal.sh")

	// Executable directory (for local builds)
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		searchPaths = append(searchPaths, filepath.Join(exeDir, "scripts/isobox-internal.sh"))
	}

	// Working directory (for development)
	if workingDir, err := os.Getwd(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(workingDir, "scripts/isobox-internal.sh"))

		gitRoot := findGitRoot(workingDir)
		if gitRoot != "" {
			searchPaths = append(searchPaths, filepath.Join(gitRoot, "scripts/isobox-internal.sh"))
		}
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func findGitRoot(startDir string) string {
	dir := startDir
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func (e *Environment) createEssentialFiles() error {
	fmt.Println("\nCreating essential configuration files...")

	shellPath := "/bin/" + e.Shell

	etcPasswd := filepath.Join(e.IsoboxDir, "etc/passwd")
	passwdContent := fmt.Sprintf(`root:x:0:0:root:/root:/bin/sh
%s:x:1000:1000:%s:/home/%s:%s
nobody:x:65534:65534:nobody:/:/bin/false
`, e.Username, e.Username, e.Username, shellPath)
	if err := os.WriteFile(etcPasswd, []byte(passwdContent), 0644); err != nil {
		return fmt.Errorf("create passwd: %w", err)
	}

	etcShadow := filepath.Join(e.IsoboxDir, "etc/shadow")
	shadowContent := fmt.Sprintf(`root:!:19000:0:99999:7:::
%s:!:19000:0:99999:7:::
nobody:!:19000:0:99999:7:::
`, e.Username)
	if err := os.WriteFile(etcShadow, []byte(shadowContent), 0600); err != nil {
		return fmt.Errorf("create shadow: %w", err)
	}

	etcGroup := filepath.Join(e.IsoboxDir, "etc/group")
	groupContent := fmt.Sprintf(`root:x:0:
%s:x:1000:
nogroup:x:65534:
`, e.Username)
	if err := os.WriteFile(etcGroup, []byte(groupContent), 0644); err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	etcGshadow := filepath.Join(e.IsoboxDir, "etc/gshadow")
	gshadowContent := fmt.Sprintf(`root:!::
%s:!::
nogroup:!::
`, e.Username)
	if err := os.WriteFile(etcGshadow, []byte(gshadowContent), 0600); err != nil {
		return fmt.Errorf("create gshadow: %w", err)
	}

	etcHosts := filepath.Join(e.IsoboxDir, "etc/hosts")
	hostsContent := `127.0.0.1	localhost isobox
::1		localhost ip6-localhost ip6-loopback
`
	if err := os.WriteFile(etcHosts, []byte(hostsContent), 0644); err != nil {
		return fmt.Errorf("create hosts: %w", err)
	}

	etcResolv := filepath.Join(e.IsoboxDir, "etc/resolv.conf")
	resolvContent := `nameserver 8.8.8.8
nameserver 8.8.4.4
`
	if err := os.WriteFile(etcResolv, []byte(resolvContent), 0644); err != nil {
		return fmt.Errorf("create resolv.conf: %w", err)
	}

	bashrc := filepath.Join(e.IsoboxDir, "etc/bash.bashrc")
	bashrcContent := `export PS1="(isobox) \u@\h:\w\$ "
export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin

echo "========================================="
echo "   ISOBOX Isolated Environment"
echo "========================================="
echo "You are in a completely isolated Linux"
echo "environment. You CANNOT access the host"
echo "system from here."
echo ""
echo "Package Manager:"
echo "  isobox install <package>"
echo "  isobox remove <package>"
echo "  isobox list"
echo ""
echo "Type 'exit' to leave this environment"
echo "========================================="
echo ""

alias ll='ls -lah'
alias la='ls -A'
alias l='ls -CF'
`
	if err := os.WriteFile(bashrc, []byte(bashrcContent), 0644); err != nil {
		return fmt.Errorf("create bashrc: %w", err)
	}

	profile := filepath.Join(e.IsoboxDir, "etc/profile")
	profileContent := `export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
export PS1="(isobox) \u@\h:\w\$ "

if [ -f /etc/bash.bashrc ]; then
    . /etc/bash.bashrc
fi
`
	if err := os.WriteFile(profile, []byte(profileContent), 0644); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}

	hostSystem := getHostSystemName()

	osRelease := filepath.Join(e.IsoboxDir, "etc/os-release")
	osReleaseContent := fmt.Sprintf(`NAME="ISOBOX-(%s))"
PRETTY_NAME="ISOBOX Isolated Environment (%s)"
ID=isobox
ID_LIKE=arch
VERSION_ID=1.0
HOME_URL="https://github.com/javanhut/isobox"
HOST_SYSTEM="%s"
`, hostSystem, hostSystem, hostSystem)
	if err := os.WriteFile(osRelease, []byte(osReleaseContent), 0644); err != nil {
		return fmt.Errorf("create os-release: %w", err)
	}

	nsswitch := filepath.Join(e.IsoboxDir, "etc/nsswitch.conf")
	nsswitchContent := `passwd:     files
group:      files
shadow:     files
hosts:      files dns
networks:   files
protocols:  files
services:   files
`
	if err := os.WriteFile(nsswitch, []byte(nsswitchContent), 0644); err != nil {
		return fmt.Errorf("create nsswitch.conf: %w", err)
	}

	fmt.Println("  Created: /etc/passwd, /etc/shadow, /etc/group, /etc/gshadow")
	fmt.Println("  Created: /etc/hosts, /etc/resolv.conf, /etc/nsswitch.conf")
	fmt.Println("  Created: /etc/bash.bashrc, /etc/profile, /etc/os-release")

	return nil
}

func (e *Environment) save() error {
	configPath := filepath.Join(e.IsoboxDir, "config.json")
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

func (e *Environment) copyProjectFiles() error {
	fmt.Println("\nCopying project files to isolated environment...")

	userHome := filepath.Join(e.IsoboxDir, "home", e.Username)
	if err := os.MkdirAll(userHome, 0755); err != nil {
		return fmt.Errorf("create user home: %w", err)
	}

	tarFile := filepath.Join(e.IsoboxDir, "tmp", "project.tar.gz")
	if err := os.MkdirAll(filepath.Dir(tarFile), 0755); err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}

	fmt.Printf("  Creating archive of project files...\n")

	tarCmd := exec.Command("tar",
		"-czf", tarFile,
		"--exclude=.isobox",
		"--exclude=.git",
		"-C", e.Root,
		".")

	if output, err := tarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("  Extracting to /home/%s...\n", e.Username)

	untarCmd := exec.Command("tar",
		"-xzf", tarFile,
		"-C", userHome)

	if output, err := untarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("untar failed: %w\nOutput: %s", err, string(output))
	}

	if err := os.Remove(tarFile); err != nil {
		fmt.Printf("  Warning: failed to remove temp tar file: %v\n", err)
	}

	chownCmd := exec.Command("sudo", "chown", "-R", "1000:1000", userHome)
	if err := chownCmd.Run(); err != nil {
		fmt.Printf("  Warning: failed to set ownership: %v\n", err)
	}

	fmt.Printf("  Project files copied to /home/%s\n", e.Username)

	return nil
}

func (e *Environment) EnterShell() error {
	shell := "/bin/" + e.Shell
	isoboxShell := filepath.Join(e.IsoboxDir, "bin", e.Shell)

	if _, err := os.Stat(isoboxShell); os.IsNotExist(err) {
		fmt.Printf("Warning: Configured shell '%s' not found, falling back to sh\n", e.Shell)
		isoboxShell = filepath.Join(e.IsoboxDir, "bin/sh")
		shell = "/bin/sh"
	}

	fmt.Printf("Entering isolated environment as user '%s'...\n", e.Username)
	fmt.Printf("Root filesystem: %s\n", e.IsoboxDir)
	fmt.Printf("Shell: %s\n", shell)
	fmt.Printf("Working directory: /home/%s\n\n", e.Username)

	homeDir := fmt.Sprintf("/home/%s", e.Username)

	initCmd := fmt.Sprintf("cd %s && exec %s -l", homeDir, shell)

	cmd := exec.Command("sudo", "chroot", "--userspec=1000:1000", e.IsoboxDir, shell, "-c", initCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=" + homeDir,
		"USER=" + e.Username,
		"LOGNAME=" + e.Username,
		"TERM=" + os.Getenv("TERM"),
	}

	return cmd.Run()
}

func (e *Environment) Execute(command []string) error {
	fmt.Printf("Executing in isolated environment as user '%s': %v\n", e.Username, command)

	homeDir := fmt.Sprintf("/home/%s", e.Username)
	cmdStr := strings.Join(command, " ")
	execCmd := fmt.Sprintf("cd %s && %s", homeDir, cmdStr)

	args := []string{"chroot", "--userspec=1000:1000", e.IsoboxDir, "/bin/sh", "-c", execCmd}

	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=" + homeDir,
		"USER=" + e.Username,
		"LOGNAME=" + e.Username,
		"TERM=" + os.Getenv("TERM"),
	}

	return cmd.Run()
}

func (e *Environment) PrintStatus() {
	fmt.Printf("ISOBOX Environment Status\n")
	fmt.Printf("=========================\n\n")
	fmt.Printf("Project Root: %s\n", e.Root)
	fmt.Printf("Isolated Root: %s\n", e.IsoboxDir)
	fmt.Printf("Created: %s\n", e.Created.Format("2006-01-02 15:04:05"))

	binDir := filepath.Join(e.IsoboxDir, "bin")
	binCount := 0
	if entries, err := os.ReadDir(binDir); err == nil {
		binCount = len(entries)
	}
	fmt.Printf("Available Commands: %d\n", binCount)

	libDir := filepath.Join(e.IsoboxDir, "lib")
	libCount := 0
	if entries, err := os.ReadDir(libDir); err == nil {
		libCount = len(entries)
	}
	fmt.Printf("Shared Libraries: %d\n", libCount)

	pkgDB := filepath.Join(e.IsoboxDir, "var/lib/ipkg/installed.json")
	if data, err := os.ReadFile(pkgDB); err == nil {
		var packages []any
		if json.Unmarshal(data, &packages) == nil {
			fmt.Printf("Installed Packages: %d\n", len(packages))
		}
	}

	fmt.Printf("\nThis is a COMPLETELY ISOLATED environment.\n")
	fmt.Printf("The host system is NOT accessible from within.\n")
	fmt.Printf("\nTo enter:\n")
	fmt.Printf("  cd %s && isobox enter\n", e.Root)
}

func (e *Environment) Migrate(sourceDir, destPath string) error {
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}

	if _, err := os.Stat(absSource); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", absSource)
	}

	destInIsobox := filepath.Join(e.IsoboxDir, destPath)

	fmt.Printf("Copying %s to %s in isobox...\n", absSource, destPath)

	destDir := filepath.Dir(destInIsobox)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	cmd := exec.Command("cp", "-r", absSource, destInIsobox)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy failed: %w (output: %s)", err, string(output))
	}

	chownCmd := exec.Command("sudo", "chown", "-R", "1000:1000", destInIsobox)
	if err := chownCmd.Run(); err != nil {
		fmt.Printf("  Warning: failed to set ownership to user: %v\n", err)
	}

	fmt.Printf("  Successfully copied to %s\n", destPath)
	return nil
}

func (e *Environment) Destroy() error {
	cmd := exec.Command("sudo", "rm", "-rf", e.IsoboxDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove %s: %w", e.IsoboxDir, err)
	}

	return nil
}

func findBusybox() string {
	locations := []string{
		"/usr/bin/busybox",
		"/bin/busybox",
		"/usr/local/bin/busybox",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	if path, err := exec.LookPath("busybox"); err == nil {
		return path
	}

	return ""
}

func copyBinary(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func getRequiredLibraries(binaryPath string) ([]string, error) {
	cmd := exec.Command("ldd", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var libs []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=>") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "=>" && i+1 < len(parts) {
					libPath := parts[i+1]
					if libPath != "" && libPath != "(0x" && filepath.IsAbs(libPath) {
						libs = append(libs, libPath)

						// For the dynamic linker, also include the interpreter path
						// e.g., "/lib64/ld-linux-x86-64.so.2 => /usr/lib64/ld-linux-x86-64.so.2"
						// We need both paths since the binary hardcodes the first path
						if i > 0 && filepath.IsAbs(parts[i-1]) && strings.Contains(parts[i-1], "ld-linux") {
							libs = append(libs, parts[i-1])
						}
					}
					break
				}
			}
		} else if strings.HasPrefix(line, "/") {
			parts := strings.Fields(line)
			if len(parts) > 0 && filepath.IsAbs(parts[0]) {
				libs = append(libs, parts[0])
			}
		}
	}

	return libs, nil
}

func getHostSystemName() string {
	osReleaseData, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(osReleaseData), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name := strings.TrimPrefix(line, "PRETTY_NAME=")
			name = strings.Trim(name, "\"")
			return name
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			name := strings.TrimPrefix(line, "NAME=")
			name = strings.Trim(name, "\"")
			return name
		}
	}

	return "Unknown"
}

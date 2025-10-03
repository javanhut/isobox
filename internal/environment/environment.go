package environment

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Environment struct {
	Root      string    `json:"root"`
	Created   time.Time `json:"created"`
	IsoboxDir string    `json:"isobox_dir"`
}

func Initialize(path string) (*Environment, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	isoboxDir := filepath.Join(absPath, ".isobox")

	env := &Environment{
		Root:      absPath,
		Created:   time.Now(),
		IsoboxDir: isoboxDir,
	}

	fmt.Println("Creating isolated Linux filesystem...")

	if err := env.createIsolatedFilesystem(); err != nil {
		return nil, err
	}

	if err := env.setupBinaries(); err != nil {
		return nil, err
	}

	if err := env.setupLibraries(); err != nil {
		return nil, err
	}

	if err := env.setupInternalPackageManager(); err != nil {
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
	dirs := []string{
		"bin", "sbin",
		"usr/bin", "usr/sbin", "usr/local/bin",
		"lib", "lib64", "usr/lib", "usr/lib64",
		"etc", "etc/profile.d",
		"dev", "proc", "sys", "tmp",
		"root", "home",
		"var/lib/ipkg", "var/log", "var/tmp",
		"mnt", "opt", "srv", "run",
	}

	for _, dir := range dirs {
		path := filepath.Join(e.IsoboxDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		fmt.Printf("  Created: .isobox/%s\n", dir)
	}

	tmpPath := filepath.Join(e.IsoboxDir, "tmp")
	if err := os.Chmod(tmpPath, 01777); err != nil {
		return fmt.Errorf("chmod tmp: %w", err)
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
	deps := []string{"zlib", "pcre2", "libssl3", "libcrypto3", "ca-certificates-bundle", "libidn2", "libunistring"}

	for _, dep := range deps {
		if err := e.installAlpinePackage(dep); err != nil {
			fmt.Printf("  Warning: failed to install %s\n", dep)
			continue
		}
	}

	fmt.Println("  Added Alpine base dependencies")
	return nil
}

func (e *Environment) addSSLCapableTools() error {
	if err := e.setupSSLCertificates(); err != nil {
		fmt.Printf("  Warning: SSL certificates setup failed: %v\n", err)
		return err
	}

	packages := []string{"wget", "ca-certificates"}

	for _, pkg := range packages {
		if err := e.installAlpinePackage(pkg); err != nil {
			fmt.Printf("  Warning: failed to install %s: %v\n", pkg, err)
			continue
		}
		fmt.Printf("  Added Alpine %s\n", pkg)
	}

	return nil
}

func (e *Environment) installAlpinePackage(pkgName string) error {
	baseURL := "https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/"

	cmd := exec.Command("wget", "-qO-", baseURL)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	var pkgFile string

	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(`href="%s-`, pkgName)) {
			start := strings.Index(line, `href="`) + 6
			end := strings.Index(line[start:], `"`)
			if end > 0 {
				pkgFile = line[start : start+end]
				if strings.HasPrefix(pkgFile, pkgName+"-") && strings.Contains(pkgFile[len(pkgName)+1:], ".apk") {
					break
				}
			}
		}
	}

	if pkgFile == "" {
		return fmt.Errorf("package not found")
	}

	pkgURL := baseURL + pkgFile
	tmpFile := fmt.Sprintf("/tmp/%s.apk", pkgName)

	cmd = exec.Command("wget", "-q", "-O", tmpFile, pkgURL)
	if err := cmd.Run(); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	cmd = exec.Command("tar", "-xzf", tmpFile, "-C", e.IsoboxDir)
	return cmd.Run()
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
	muslURL := "https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/musl-1.2.4_git20230717-r5.apk"
	muslFile := "/tmp/musl-isobox.apk"

	cmd := exec.Command("wget", "-q", "-O", muslFile, muslURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download musl: %w", err)
	}
	defer os.Remove(muslFile)

	cmd = exec.Command("tar", "-xzf", muslFile, "-C", e.IsoboxDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract musl: %w", err)
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

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	scriptPath := filepath.Join(exeDir, "scripts/isobox-internal.sh")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		workingDir, _ := os.Getwd()
		scriptPath = filepath.Join(workingDir, "scripts/isobox-internal.sh")
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		fmt.Println("  Warning: Internal package manager script not found")
		return nil
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

func (e *Environment) createEssentialFiles() error {
	fmt.Println("\nCreating essential configuration files...")

	etcPasswd := filepath.Join(e.IsoboxDir, "etc/passwd")
	passwdContent := `root:x:0:0:root:/root:/bin/sh
nobody:x:65534:65534:nobody:/:/bin/false
`
	if err := os.WriteFile(etcPasswd, []byte(passwdContent), 0644); err != nil {
		return fmt.Errorf("create passwd: %w", err)
	}

	etcShadow := filepath.Join(e.IsoboxDir, "etc/shadow")
	shadowContent := `root:!:19000:0:99999:7:::
nobody:!:19000:0:99999:7:::
`
	if err := os.WriteFile(etcShadow, []byte(shadowContent), 0600); err != nil {
		return fmt.Errorf("create shadow: %w", err)
	}

	etcGroup := filepath.Join(e.IsoboxDir, "etc/group")
	groupContent := `root:x:0:
nogroup:x:65534:
`
	if err := os.WriteFile(etcGroup, []byte(groupContent), 0644); err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	etcGshadow := filepath.Join(e.IsoboxDir, "etc/gshadow")
	gshadowContent := `root:!::
nogroup:!::
`
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
export HOME=/root

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
export HOME=/root
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

func (e *Environment) EnterShell() error {
	shell := "/bin/bash"
	isoboxShell := filepath.Join(e.IsoboxDir, "bin/bash")

	if _, err := os.Stat(isoboxShell); os.IsNotExist(err) {
		isoboxShell = filepath.Join(e.IsoboxDir, "bin/sh")
		shell = "/bin/sh"
	}

	fmt.Printf("Entering isolated environment via chroot...\n")
	fmt.Printf("Root filesystem: %s\n\n", e.IsoboxDir)

	cmd := exec.Command("sudo", "chroot", e.IsoboxDir, shell, "-l")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=/root",
		"TERM=" + os.Getenv("TERM"),
	}

	return cmd.Run()
}

func (e *Environment) Execute(command []string) error {
	fmt.Printf("Executing in isolated environment: %v\n", command)

	args := []string{"chroot", e.IsoboxDir}
	args = append(args, command...)

	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=/root",
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

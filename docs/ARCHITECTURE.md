# IsoBox Architecture

IsoBox creates a **completely isolated Linux environment** using chroot. Unlike typical containerization, IsoBox provides true filesystem isolation where users cannot escape to the host system.

## Core Concept

IsoBox builds a self-contained Linux filesystem in `.isobox/` with:
- Complete directory structure (`/bin`, `/lib`, `/etc`, `/usr`, etc.)
- All necessary binaries copied from host
- All required shared libraries
- Essential configuration files
- Package management database

When you enter IsoBox via `isobox enter`, you are **chrooted** into `.isobox/` - you cannot access anything outside this directory.

## Architecture Diagram

```
Host System
├── /home/user/myproject/
│   ├── .isobox/                    ← ISOLATED ROOT
│   │   ├── bin/                    ← Copied binaries
│   │   │   ├── sh
│   │   │   ├── bash
│   │   │   ├── ls, cat, grep...
│   │   │   └── (40+ commands)
│   │   ├── lib/, lib64/            ← Copied libraries
│   │   ├── usr/bin/, usr/lib/      ← Additional binaries/libs
│   │   ├── etc/                    ← Config files
│   │   │   ├── passwd, group
│   │   │   ├── hosts, resolv.conf
│   │   │   ├── bash.bashrc
│   │   │   └── profile
│   │   ├── var/lib/ipkg/           ← Package database
│   │   ├── root/                   ← Home in isolation
│   │   ├── tmp/                    ← Temp files
│   │   └── dev/, proc/, sys/       ← System directories
│   │
│   ├── your-code.py                ← NOT accessible from inside
│   └── README.md                   ← NOT accessible from inside
│
└── (chroot barrier - cannot cross from inside .isobox/)
```

## Components

### 1. Isolated Filesystem Creator

**Location:** `internal/environment/environment.go`

**Responsibilities:**
- Creates complete Linux directory structure
- Copies binaries from host system
- Copies required shared libraries
- Creates essential configuration files
- Manages environment metadata

**Directory Structure:**
```
.isobox/
├── bin/          # Essential commands (40+ binaries)
├── sbin/         # System binaries
├── lib/          # Shared libraries (.so files)
├── lib64/        # 64-bit libraries
├── usr/
│   ├── bin/      # User binaries
│   ├── sbin/     # User system binaries
│   ├── lib/      # User libraries
│   └── lib64/    # User 64-bit libraries
├── etc/          # Configuration
│   ├── passwd    # User database
│   ├── group     # Group database
│   ├── hosts     # Hostname resolution
│   ├── resolv.conf # DNS configuration
│   ├── bash.bashrc # Bash configuration
│   ├── profile   # Shell profile
│   └── ld.so.conf # Library paths
├── var/
│   ├── lib/ipkg/ # Package database
│   └── log/      # Logs
├── root/         # Root home directory
├── home/         # User home directories
├── tmp/          # Temporary files (1777)
├── dev/          # Device files
├── proc/         # Process info (mounted at runtime)
├── sys/          # System info
├── mnt/          # Mount points
├── opt/          # Optional software
├── srv/          # Service data
└── run/          # Runtime data
```

### 2. Binary Setup

**Two modes:**

#### Mode 1: BusyBox (Preferred)
- Looks for BusyBox at `/usr/bin/busybox`, `/bin/busybox`
- Copies single BusyBox binary to `.isobox/bin/busybox`
- Runs `busybox --install -s` to create 150+ symlinks
- Result: Full POSIX environment in ~2MB

#### Mode 2: System Binaries (Fallback)
- Copies essential binaries from host:
  - Shells: `sh`, `bash`, `dash`
  - File ops: `ls`, `cat`, `cp`, `mv`, `rm`, `mkdir`, `chmod`
  - Text: `grep`, `sed`, `awk`, `cut`, `sort`, `head`, `tail`
  - Archive: `tar`, `gzip`, `bzip2`, `xz`
  - Network: `wget`, `curl`
  - Process: `ps`, `top`, `kill`
  - Utils: `find`, `which`, `echo`, `pwd`, `env`
- Result: ~40 binaries, larger footprint

### 3. Library Resolution

**Process:**
1. Read all binaries in `.isobox/bin/`
2. For each binary, run `ldd` to find shared library dependencies
3. Parse `ldd` output to extract library paths
4. Copy each library to `.isobox/lib/` or `.isobox/lib64/`
5. Copy symlinks as symlinks
6. Create `/etc/ld.so.conf` with library paths

**Example:**
```bash
$ ldd /bin/bash
    linux-vdso.so.1 =>  (0x00007ffd...)
    libtinfo.so.6 => /lib/x86_64-linux-gnu/libtinfo.so.6
    libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6
    /lib64/ld-linux-x86-64.so.2

# These libraries are copied to .isobox/lib/x86_64-linux-gnu/
```

### 4. Chroot Isolation

**Entry mechanism:**
```bash
sudo chroot /path/to/.isobox /bin/bash -l
```

**What chroot does:**
1. Changes root directory `/` to `/path/to/.isobox`
2. Changes current directory to `/`
3. Executes `/bin/bash -l`
4. From inside: cannot access parent directories

**Environment variables set:**
- `PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin`
- `HOME=/root`
- `TERM=` (inherited from host)

**Login shell (`-l` flag):**
- Sources `/etc/profile`
- Sources `/etc/bash.bashrc`
- Displays welcome message
- Sets up aliases and PS1

### 5. Package Manager (IPKG)

**Location:** `pkg/ipkg/manager.go`

**Functionality:**
- Downloads Alpine Linux packages (.apk files)
- Extracts to `.isobox/` (respects directory structure)
- Tracks installations in `.isobox/var/lib/ipkg/installed.json`
- Supports install, remove, list, update

**Package flow:**
1. User runs: `isobox pkg install curl`
2. Downloads: `https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/curl.apk`
3. Extracts APK (tar.gz) to `.isobox/`
4. Updates database:
   ```json
   {
     "name": "curl",
     "version": "8.5.0-r0",
     "description": "Package curl from Alpine",
     "files": ["/usr/bin/curl"]
   }
   ```
5. Binary now available in isolated environment

## Isolation Guarantees

### ✅ What IS Isolated:

1. **Filesystem**: Cannot access host files outside `.isobox/`
2. **Binaries**: All commands come from `.isobox/bin`
3. **Libraries**: All `.so` files from `.isobox/lib`
4. **Configuration**: Uses `.isobox/etc/` config files
5. **Packages**: Installed to `.isobox/`, not host
6. **Home directory**: `/root` maps to `.isobox/root/`

### ❌ What is NOT Isolated:

1. **Process namespace**: Can see host processes if `/proc` mounted
2. **Network**: Shares host network stack
3. **User**: Runs as root inside, but still same user on host
4. **Kernel**: Shares host kernel
5. **Devices**: `/dev` might expose host devices

**Chroot is a filesystem jail, not a complete sandbox.**

## Comparison with Other Technologies

### vs Docker

| Aspect | IsoBox | Docker |
|--------|--------|--------|
| Isolation | chroot (filesystem) | namespaces (full) |
| Process isolation | No | Yes |
| Network isolation | No | Yes |
| User isolation | No | Yes |
| Overhead | Minimal | Moderate |
| Setup time | Seconds | Minutes (pull image) |
| Size | 10-50MB | 100MB-1GB+ |
| Requires root | Yes (chroot) | Yes (daemon) |
| Escape possibility | Moderate | Difficult |

### vs Containers (LXC/Podman)

IsoBox is **not a container**. It's a chroot-based isolated environment:
- Containers use Linux namespaces for full isolation
- IsoBox uses chroot for filesystem isolation only
- Containers have separate PID/network/IPC namespaces
- IsoBox shares these with host

### vs Virtual Machines

| Aspect | IsoBox | VM |
|--------|--------|-----|
| Kernel | Shared | Separate |
| Boot time | Instant | Minutes |
| Memory | Minimal | GBs |
| Isolation | Filesystem | Complete |
| Performance | Native | Virtualized |

## Security Considerations

**IsoBox is NOT a security boundary!**

### Chroot Escape Possibilities:

1. **As root inside chroot**: Possible to escape via:
   - Creating device files
   - Using `ptrace` on host processes
   - Exploiting kernel vulnerabilities

2. **Access to host /proc**: Can potentially:
   - Read host filesystem via `/proc/[pid]/root/`
   - See host processes and memory

3. **Shared kernel**: Kernel exploits affect both host and chroot

### Recommended Use Cases:

✅ **Safe for:**
- Development environment isolation
- Testing different tool versions
- Clean build environments
- Learning/educational purposes
- Avoiding pollution of host system

❌ **NOT safe for:**
- Running untrusted code
- Security sandboxing
- Production isolation
- Multi-tenant environments
- Hostile users

**For production security, use Docker, Podman, or VMs.**

## Performance Characteristics

### Initialization:
- Directory creation: < 100ms
- Binary copying: 1-5 seconds (40 binaries)
- Library copying: 2-10 seconds (depends on dependencies)
- Total: **3-15 seconds**

### Runtime:
- Entry (chroot): < 50ms
- Command execution: **Native speed** (no virtualization)
- Filesystem I/O: **Native speed** (no overlay)
- Network: **Native speed** (no network namespace)

### Storage:
- **Without BusyBox**: 20-50MB (individual binaries + libs)
- **With BusyBox**: 5-15MB (single binary + minimal libs)
- **Per package**: 1-10MB average (Alpine packages are small)

## Implementation Details

### Binary Copying
```go
// Copy file preserving permissions
func copyBinary(src, dst string) error {
    sourceFile, _ := os.Open(src)
    destFile, _ := os.Create(dst)
    io.Copy(destFile, sourceFile)
    sourceInfo, _ := os.Stat(src)
    os.Chmod(dst, sourceInfo.Mode())
}
```

### Library Detection
```go
// Find libraries with ldd
func getRequiredLibraries(binaryPath string) ([]string, error) {
    cmd := exec.Command("ldd", binaryPath)
    output, _ := cmd.Output()
    // Parse output for /path/to/lib.so
    // Return list of absolute library paths
}
```

### Chroot Execution
```go
// Enter chroot environment
func (e *Environment) EnterShell() error {
    cmd := exec.Command("sudo", "chroot", e.IsoboxDir, "/bin/bash", "-l")
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
```

## Limitations

1. **Requires sudo**: chroot is a privileged operation
2. **Linux only**: chroot is Linux/Unix-specific
3. **No process isolation**: Shares process namespace with host
4. **No network isolation**: Shares network with host
5. **No device isolation**: May access host devices via /dev
6. **Escape possible**: Root user inside can potentially escape
7. **No checkpoint/restore**: Cannot save/restore state like containers
8. **No clustering**: No orchestration like Kubernetes

## Future Enhancements

Potential improvements:

1. **Namespace integration**: Add PID/network namespaces for better isolation
2. **Overlay filesystem**: Use overlayfs for copy-on-write
3. **Resource limits**: Use cgroups for CPU/memory limits
4. **Device whitelist**: Restrict /dev access
5. **Rootless mode**: Explore user namespaces to avoid sudo
6. **Auto library updates**: Refresh libraries when host updates
7. **Template environments**: Pre-built environment templates
8. **Environment export/import**: Share environments between systems

## Design Philosophy

IsoBox follows these principles:

1. **Complete Isolation**: User cannot escape `.isobox/` filesystem
2. **Self-Contained**: All dependencies copied, nothing shared
3. **POSIX Compliant**: Standard Unix/Linux environment
4. **Lightweight**: Minimal overhead, fast initialization
5. **Transparent**: Clear about what is and isn't isolated
6. **Simple**: Easy to understand chroot-based approach
7. **Per-Directory**: Independent environment per project

IsoBox is designed for **development isolation**, not production security.

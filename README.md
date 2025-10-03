# IsoBox

A truly isolated Linux development environment using chroot. IsoBox creates a complete, self-contained Linux filesystem that you **cannot escape from** - all commands and packages are isolated from the host system.

## What is IsoBox?

IsoBox creates a **completely isolated Linux environment** in a `.isobox` directory. When you enter this environment:

- ❌ You CANNOT access the host filesystem
- ❌ You CANNOT see host processes
- ❌ You CANNOT escape to the parent system
- ✅ You get a complete, isolated Linux environment
- ✅ All binaries are copied into `.isobox/`
- ✅ All libraries are copied into `.isobox/lib`
- ✅ All packages install to `.isobox/`

**This is TRUE isolation** - like a lightweight container but using chroot.

## Features

- **Complete Isolation**: Cannot access host system from within
- **Full Linux Filesystem**: Standard `/bin`, `/lib`, `/etc`, `/usr` structure
- **Self-Contained**: All binaries and libraries copied in
- **POSIX Compliant**: Full set of Unix commands (via BusyBox or system binaries)
- **Package Management**: Install packages with IPKG (isolated to environment)
- **Per-Directory**: Each project gets its own isolated environment

## How It Works

```
Host System
└── myproject/
    ├── .isobox/              ← ISOLATED LINUX FILESYSTEM
    │   ├── bin/              ← All commands here
    │   ├── sbin/
    │   ├── lib/              ← All libraries here
    │   ├── lib64/
    │   ├── usr/
    │   ├── etc/              ← Config files
    │   ├── var/lib/ipkg/     ← Package database
    │   ├── dev/
    │   ├── proc/
    │   ├── sys/
    │   ├── tmp/
    │   └── root/
    └── [your project files - NOT accessible from inside]
```

When you run `isobox enter`, it executes:
```bash
sudo chroot .isobox /bin/bash
```

You are now **jailed** in `.isobox/` - you cannot escape!

## Quick Start

### Installation

```bash
git clone https://github.com/javanhut/isobox
cd isobox
go build -o isobox
sudo mv isobox /usr/local/bin/
```

### Basic Usage

```bash
# Create a project
mkdir ~/myproject
cd ~/myproject

# Initialize isolated environment (requires sudo for chroot later)
isobox init

# Enter the isolated environment (uses sudo chroot)
isobox enter

# You are now in a completely isolated Linux environment!
(isobox) root@host:/# ls
bin  boot  dev  etc  home  lib  lib64  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var

(isobox) root@host:/# pwd
/

(isobox) root@host:/# ls /
bin  boot  dev  etc  home  lib  lib64  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var

# You CANNOT see the host system!
(isobox) root@host:/# exit

# Back to host system
```

## Commands

```bash
isobox init [path]        # Initialize isolated environment
isobox enter              # Enter isolated environment (uses sudo chroot)
isobox exec <cmd>         # Execute command in isolation (uses sudo chroot)
isobox status             # Show environment status
isobox destroy            # Remove isolated environment

# Package management
isobox pkg install <pkg>  # Install package to isolated environment
isobox pkg remove <pkg>   # Remove package
isobox pkg list           # List installed packages
isobox pkg update         # Update package index
```

## Requirements

- **Linux system** (chroot is Linux-specific)
- **sudo access** (chroot requires root privileges)
- Go 1.20+ (for building)
- **BusyBox** (optional, provides 150+ Unix commands in one binary)

### Installing BusyBox (Highly Recommended)

Ubuntu/Debian:
```bash
sudo apt-get install busybox-static
```

Alpine:
```bash
apk add busybox-static
```

Arch:
```bash
sudo pacman -S busybox
```

Without BusyBox, IsoBox will copy system binaries instead (larger, more dependencies).

## What Gets Created

### During `isobox init`:

1. **Complete Linux directory structure** in `.isobox/`:
   ```
   .isobox/
   ├── bin/          ← All executables
   ├── sbin/         ← System binaries
   ├── lib/          ← Shared libraries
   ├── lib64/        ← 64-bit libraries
   ├── usr/bin/
   ├── usr/lib/
   ├── etc/          ← passwd, group, hosts, resolv.conf, bash.bashrc
   ├── dev/
   ├── proc/
   ├── sys/
   ├── tmp/
   ├── var/lib/ipkg/ ← Package database
   ├── root/         ← Home directory in isolation
   └── home/
   ```

2. **Binaries copied**:
   - If BusyBox found: Single busybox binary + 150+ symlinks
   - Otherwise: Copies essential commands from system

3. **Libraries copied**:
   - Automatically detects and copies shared libraries needed by binaries
   - Uses `ldd` to find dependencies

4. **Configuration files**:
   - `/etc/passwd`, `/etc/group` (minimal)
   - `/etc/hosts`, `/etc/resolv.conf` (networking)
   - `/etc/bash.bashrc`, `/etc/profile` (shell config)

## True Isolation

### What You CAN Do Inside:

✅ Run any command in `/bin`, `/usr/bin`  
✅ Install packages with `isobox pkg install`  
✅ Create files in `/tmp`, `/root`  
✅ Use networking (DNS configured)  
✅ Everything stays in `.isobox/`

### What You CANNOT Do Inside:

❌ Access parent directories  
❌ See host processes  
❌ Access host files  
❌ Break out of the chroot jail  
❌ Affect the host system

## Examples

### Example 1: Development Environment

```bash
mkdir ~/dev-env
cd ~/dev-env

isobox init
isobox pkg install git curl vim

isobox enter
(isobox) # git --version
(isobox) # curl --version
(isobox) # ls /bin
(isobox) # exit
```

### Example 2: Testing Environment

```bash
mkdir ~/test-env
cd ~/test-env

isobox init
isobox enter

(isobox) # echo "I'm completely isolated!"
(isobox) # ls ../  # This shows /, not parent directory!
(isobox) # exit
```

### Example 3: Execute Commands

```bash
cd ~/dev-env

# Execute without entering shell
isobox exec ls -la /
isobox exec pwd
isobox exec cat /etc/os-release
```

## Package Management

Install packages into the isolated environment:

```bash
cd ~/myproject

# Packages install to .isobox/
isobox pkg install curl
isobox pkg install python3
isobox pkg list

isobox enter
(isobox) # curl --version
(isobox) # python3 --version
```

Packages are fetched from Alpine Linux and extracted to `.isobox/`.

## Understanding Chroot

IsoBox uses **chroot** to create isolation:

```bash
# What happens when you run: isobox enter
sudo chroot /path/to/project/.isobox /bin/bash
```

**Chroot changes the root directory** `/` to `.isobox/`. From inside:
- `/` is actually `/path/to/project/.isobox/`
- You cannot access anything outside `.isobox/`
- This is a security jail

## Security Note

**Chroot is NOT a complete security boundary**. Determined users with root inside the chroot might escape. For production security, use:
- Docker/Podman (containers with namespaces)
- VMs (complete virtualization)

IsoBox is designed for **development isolation**, not production security.

## Comparison

| Feature | IsoBox | Docker | chroot | Virtual Machine |
|---------|--------|--------|--------|-----------------|
| Isolation | Filesystem | Full | Filesystem | Complete |
| Setup | 1 command | Pull images | Manual | Install OS |
| Size | ~10-50MB | ~100MB+ | Varies | GBs |
| Speed | Native | Near-native | Native | Slower |
| Escape | Possible | Hard | Possible | Nearly impossible |
| Use Case | Dev env | Deployment | Testing | Production |

## Multiple Environments

Each directory is completely independent:

```bash
# Environment A
cd ~/project-a
isobox init
isobox pkg install git
isobox enter
(isobox) # which git
/usr/bin/git

# Environment B
cd ~/project-b  
isobox init
isobox pkg install curl
isobox enter
(isobox) # which git
(not found)
(isobox) # which curl
/usr/bin/curl
```

## Troubleshooting

### "Permission denied" when entering

Chroot requires root. IsoBox uses `sudo`:
```bash
isobox enter  # This runs: sudo chroot .isobox /bin/bash
```

Make sure you have sudo access.

### "Command not found" inside

Install packages or check binaries:
```bash
isobox exec ls /bin
isobox pkg install <package>
```

### Libraries missing

If you see library errors, the shared library copy may have failed. Try:
```bash
isobox destroy
isobox init  # Rebuild environment
```

### BusyBox not found

Install busybox-static for best results:
```bash
sudo apt-get install busybox-static
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - How IsoBox works internally
- [Quick Start](docs/QUICKSTART.md) - Detailed getting started
- [Package Manager](docs/PACKAGE_MANAGER.md) - IPKG documentation

## Why IsoBox?

**Problem**: You want isolated development environments but:
- Docker is too heavy
- Python venv / Node nvm only isolate one language
- You want FULL Linux environment isolation
- You don't want to affect your host system

**Solution**: IsoBox
- Lightweight (10-50MB)
- Complete Linux environment
- True filesystem isolation via chroot
- Per-directory environments
- Simple to use

## License

MIT

## Author

javanhut

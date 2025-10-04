# IsoBox

A truly isolated Linux development environment using chroot. IsoBox creates a complete, self-contained Linux filesystem that you **cannot escape from** - all commands and packages are isolated from the host system.

## What is IsoBox?

IsoBox creates a **completely isolated Linux environment** in a `.isobox` directory. When you enter this environment:

**Restrictions:**
- You CANNOT access the host filesystem
- You CANNOT see host processes
- You CANNOT escape to the parent system
- You CANNOT affect the host system

**Capabilities:**
- You get a complete, isolated Linux environment
- All binaries are available in `.isobox/bin`
- All libraries are in `.isobox/lib`
- All packages install to `.isobox/`
- Full networking with DNS configured

**This is TRUE isolation** - like a lightweight container but using chroot.

## Features

- **Complete Isolation**: Cannot access host system from within
- **Full Linux Filesystem**: Standard `/bin`, `/lib`, `/etc`, `/usr` structure
- **Self-Contained**: All binaries and libraries installed automatically
- **POSIX Compliant**: 150+ Unix commands via BusyBox
- **Package Management**: Install Alpine Linux packages inside the environment
- **Per-Directory**: Each project gets its own isolated environment
- **Automatic Dependencies**: musl libc and SSL libraries installed automatically
- **Multiple Shells**: Choose between bash, zsh, or sh as your default shell

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
make install
```

This installs:
- Binary to `/usr/local/bin/isobox`
- Scripts to `/usr/local/share/isobox/scripts/`

To uninstall:
```bash
cd isobox
make uninstall
```

### Basic Usage

```bash
# Create a project
mkdir ~/myproject
cd ~/myproject

# Initialize isolated environment (requires sudo for chroot later)
isobox init

# Or initialize with a specific shell (bash, zsh, or sh)
isobox init --shell zsh

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

### Host Commands

```bash
isobox init [path] [--shell <shell>]  # Initialize isolated environment (shells: bash, zsh, sh)
isobox enter                           # Enter isolated environment (uses sudo chroot)
isobox exec <cmd>                      # Execute command in isolation (uses sudo chroot)
isobox migrate <src> <dest>            # Copy directory from host to isobox
isobox recache                         # Delete and rebuild the base system cache
isobox status                          # Show environment status
isobox destroy                         # Remove isolated environment (uses sudo)
```

### Inside Environment Commands

Once inside the environment with `isobox enter`:

```bash
isobox install <package>  # Install Alpine package
isobox remove <package>   # Remove package
isobox list              # List installed packages
isobox update            # Update package index
isobox help              # Show help
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

2. **Binaries installed**:
   - BusyBox: 150+ Unix commands via symlinks
   - Alpine wget: SSL-capable for package downloads
   - Internal package manager: `/bin/isobox` script

3. **Libraries installed**:
   - musl libc: Required for Alpine packages
   - libssl3 & libcrypto3: SSL/TLS support
   - pcre2, zlib: Common dependencies
   - libidn2, libunistring: Internationalization support

4. **Configuration files**:
   - `/etc/passwd`, `/etc/group`, `/etc/shadow`, `/etc/gshadow`
   - `/etc/hosts`, `/etc/resolv.conf` (networking)
   - `/etc/bash.bashrc`, `/etc/profile` (shell config)
   - `/etc/os-release` (system identification)
   - `/etc/ssl/certs/ca-certificates.crt` (SSL certificates)

## True Isolation

### What You CAN Do Inside:

- Run any command in `/bin`, `/usr/bin`
- Install packages with `isobox install <package>`
- Create files in `/tmp`, `/root`
- Use networking (DNS configured)
- Everything stays isolated in `.isobox/`

### What You CANNOT Do Inside:

- Access parent directories
- See host processes
- Access host files
- Break out of the chroot jail
- Affect the host system

## Examples

### Example 1: Development Environment

```bash
mkdir ~/dev-env
cd ~/dev-env

isobox init
isobox enter

# Inside the environment:
(isobox) # isobox install git
(isobox) # isobox install curl
(isobox) # isobox install vim
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

### Example 4: Migrating Projects

```bash
cd ~/dev-env
isobox init

# Copy a project from host into the isobox
isobox migrate ./myproject /home/username/myproject

# Enter and use the migrated files
isobox enter
(isobox) # cd /home/username/myproject
(isobox) # ls -la
(isobox) # exit
```

### Example 5: Rebuilding Cache

```bash
# If the package manager stops working after cache changes
isobox recache

# Then reinitialize your environments
cd ~/myproject
isobox destroy
isobox init
```

## Package Management

IsoBox uses an internal package manager that runs inside the isolated environment. It fetches packages from Alpine Linux repositories.

### Installing Packages

```bash
cd ~/myproject
isobox init
isobox enter

# Inside the environment:
(isobox) # isobox install git
(isobox) # isobox install python3
(isobox) # isobox install nodejs
(isobox) # isobox list

# Use the installed packages:
(isobox) # git --version
(isobox) # python3 --version
```

### How It Works

1. Packages are downloaded from Alpine Linux v3.19 repositories
2. Automatically installs required dependencies (pcre2, zlib, etc.)
3. Extracts packages directly into the isolated filesystem
4. All files stay within `.isobox/` - completely isolated from host

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

# Or enter the environment and install:
isobox enter
(isobox) # isobox install <package>
```

### Library errors

If you see library errors like "symbol not found" or "cannot open shared object":
```bash
# Destroy and rebuild to get latest dependencies
isobox destroy
isobox init
```

The init process automatically installs musl libc and common libraries.

### Package manager not working

If the internal package manager (`isobox install`) doesn't work inside the environment:
```bash
# Rebuild the base system cache
isobox recache

# Then destroy and reinitialize your environment
cd ~/myproject
isobox destroy
isobox init
```

This ensures the `isobox` script is properly installed in `/bin/isobox` inside the environment.

### BusyBox not found

Install busybox-static for best results:
```bash
sudo apt-get install busybox-static
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - How IsoBox works internally
- [Quick Start](docs/QUICKSTART.md) - Detailed getting started
- [Package Manager](docs/PACKAGE_MANAGER.md) - Internal package manager documentation

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

# IsoBox Package Manager

The IsoBox package manager installs packages from Alpine Linux repositories into isolated environments.

## Overview

The package manager is implemented in pure Go and downloads Alpine Linux packages, extracts them directly into the isolated filesystem, and tracks installed packages.

**Key Features:**
- Pure Go implementation (no shell scripts required)
- Installs packages from Alpine Linux v3.18 repositories
- **Automatic dependency resolution** - Recursively installs all package dependencies
- **Batch installation via TOML configuration** - Define dependencies in a file
- Per-environment package isolation
- Simple JSON-based package tracking
- Supports both main and community repositories
- Package name aliases (nvim → neovim, python → python3, etc.)

## Basic Usage

All package commands run **inside** the isolated environment.

### Enter the Environment

```bash
cd ~/myproject
isobox enter
```

You're now inside the isolated environment with the `(isobox)` prompt.

### Install a Package

```bash
(isobox) # isobox install <package-name>
```

Examples:
```bash
(isobox) # isobox install git
(isobox) # isobox install curl
(isobox) # isobox install vim
(isobox) # isobox install python3
```

### Remove a Package

```bash
(isobox) # isobox remove <package-name>
```

Example:
```bash
(isobox) # isobox remove git
```

Note: This removes the package from the database but does not delete files.

### List Installed Packages

```bash
(isobox) # isobox list
```

Output:
```
Installed packages:
  git (latest) - installed 2025-10-03T14:00:00Z
  curl (latest) - installed 2025-10-03T14:15:00Z
```

### Update Package Index

```bash
(isobox) # isobox update
```

This displays information about updating the package index.

### Show Help

```bash
(isobox) # isobox help
```

## Batch Package Installation with dependencies.toml

You can define all your environment's dependencies in a TOML configuration file and install them in batch.

### Create a dependencies.toml File

```toml
# dependencies.toml
[packages]
# Core development tools
dev = ["git", "vim", "neovim", "python3", "go", "gcc", "make"]

# Shells
shells = ["bash", "zsh"]

# Network tools
network = ["curl", "wget", "openssh-client"]

# System utilities
utils = ["htop", "tmux", "jq"]

# Custom packages
custom = []

[options]
# Control which package groups to install
install_dev = true
install_shells = true
install_network = false
install_utils = false
install_custom = false
```

### Install During Initialization

Install packages automatically when creating a new environment:

```bash
isobox init --install-dep dependencies.toml
```

This will:
1. Create the isolated environment
2. Install all packages defined in the TOML file
3. Resolve and install all dependencies automatically

### Install in Existing Environment

From the host system:

```bash
cd ~/myproject
isobox pkg install-deps dependencies.toml
```

### Package Name Aliases

The package manager supports common aliases that map to Alpine package names:

- `nvim` → `neovim`
- `vim` → `vim`
- `python` → `python3`
- `py` → `python3`
- `pip` → `py3-pip`

You can use either the alias or the actual package name in your dependencies.toml.

### Example Workflow

```bash
# Create dependencies.toml for a Python project
cat > dependencies.toml << 'EOF'
[packages]
dev = ["git", "python3", "py3-pip", "gcc", "musl-dev"]
utils = ["curl", "jq"]

[options]
install_dev = true
install_utils = true
EOF

# Initialize environment with dependencies
isobox init myproject --install-dep dependencies.toml

# Enter the environment
cd myproject && isobox enter

# Packages are already installed and ready to use
(isobox) python3 --version
(isobox) git --version
```

## How It Works

### Package Sources

The package manager uses Alpine Linux v3.18 repositories:

**Main Repository:** https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/
**Community Repository:** https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/

Alpine packages are used because:
- **Small size**: Built with musl libc (smaller than glibc)
- **Security**: Security-focused distribution
- **Compatibility**: Works in isolated environments with musl
- **Complete**: Large package selection across main and community repos
- **Actively maintained**: Regular updates and security patches

### Installation Process

The package manager is implemented in pure Go with no external dependencies:

1. **Check if installed**: Query the package database (`/var/lib/ipkg/installed.json`)
2. **Resolve package aliases**: Map common names (nvim → neovim, python → python3)
3. **Search repositories**:
   - Fetch HTML index from Alpine main repository
   - If not found, search community repository
   - Use regex pattern matching to find package file
4. **Download package**: HTTP GET request to download .apk file
5. **Parse dependencies**:
   - Open .apk as gzip-compressed tar archive
   - Extract `.PKGINFO` file
   - Parse `depend =` lines
   - Filter out invalid dependencies (cmd:, pc:, absolute paths)
   - Map shared library dependencies (so:*) to package names
6. **Install dependencies recursively**:
   - Check circular dependency prevention map
   - Install each dependency before the main package
7. **Extract package**:
   - Read .apk as tar.gz archive
   - Extract files directly to root filesystem
   - Create directories, regular files, and symlinks
   - Skip metadata files (.*)
8. **Update database**: Add package entry with name, version, and timestamp
9. **Cleanup**: Remove downloaded .apk file from cache

All extraction and parsing is done in pure Go using the standard library's `archive/tar`, `compress/gzip`, and `net/http` packages.

### Automatic Dependency Installation

During initialization (`isobox init`), the following are automatically installed:

**Core Libraries:**
- musl libc (required for all Alpine packages)
- libssl3 and libcrypto3 (SSL/TLS support)
- ca-certificates-bundle (SSL certificates)
- zlib (compression library)
- pcre2 (regex library)
- libidn2 and libunistring (internationalization)
- ncurses libraries (libncursesw, libformw, libmenuw, libpanelw)
- readline (command-line editing)
- libacl and libattr (ACL support)
- oniguruma (regex engine)
- libcap (capability support)
- s6 and skalibs (process supervision)
- utmps-libs (utmp/wtmp handling)

**Tools:**
- BusyBox (150+ Unix commands)
- Alpine wget (SSL-capable for package downloads)
- bash and zsh shells
- python3 (Python 3 interpreter)
- gcc (C compiler)
- go (Go programming language)
- vim (text editor)

When you install your first package, the package manager also installs common dependencies (pcre2, zlib) if not already marked as installed.

### Package Database

**Location:** `/var/lib/ipkg/installed.json`

This is inside the isolated environment, so from the host it's at `.isobox/var/lib/ipkg/installed.json`.

**Format:**
```json
[
  {
    "name": "git",
    "version": "latest",
    "installed": "2025-10-03T14:00:00Z"
  },
  {
    "name": "curl",
    "version": "latest",
    "installed": "2025-10-03T14:15:00Z"
  }
]
```

## Environment-Specific Packages

Each IsoBox environment has its own isolated package manager and database:

```
project-a/
└── .isobox/
    ├── bin/isobox              # Statically-linked isobox binary
    └── var/lib/ipkg/
        └── installed.json      # Contains: git, vim

project-b/
└── .isobox/
    ├── bin/isobox              # Same binary, different database
    └── var/lib/ipkg/
        └── installed.json      # Contains: python3, curl
```

The `isobox` binary automatically detects when it's running inside an isolated environment (by checking `/etc/os-release` for `ID=isobox`) and operates on the local package database.

Installing packages in `project-a` doesn't affect `project-b`.

## Common Packages

### Development Tools

```bash
(isobox) # isobox install git
(isobox) # isobox install make
(isobox) # isobox install gcc
(isobox) # isobox install cmake
```

### Networking Tools

```bash
(isobox) # isobox install curl
(isobox) # isobox install wget
(isobox) # isobox install openssh-client
(isobox) # isobox install netcat-openbsd
```

### Text Editors

Note: vim is pre-installed in all environments.

```bash
(isobox) # vim --version        # Already available
(isobox) # isobox install nano
(isobox) # isobox install emacs
```

### Programming Languages

Note: python3 and go are pre-installed in all environments.

```bash
(isobox) # python3 --version   # Already available
(isobox) # go version           # Already available
(isobox) # isobox install nodejs
(isobox) # isobox install ruby
```

### Text Processing

```bash
(isobox) # isobox install grep
(isobox) # isobox install sed
(isobox) # isobox install gawk
(isobox) # isobox install jq
```

### Build Tools

Note: gcc is pre-installed in all environments.

```bash
(isobox) # gcc --version        # Already available
(isobox) # isobox install g++
(isobox) # isobox install make
(isobox) # isobox install cmake
(isobox) # isobox install autoconf
```

## Complete Workflows

### Development Environment

```bash
# From host
mkdir ~/myproject
cd ~/myproject
isobox init
isobox enter

# Inside environment - vim, gcc, python3, go are pre-installed
(isobox) # vim --version
(isobox) # gcc --version
(isobox) # python3 --version
(isobox) # go version

# Install additional tools
(isobox) # isobox install git
(isobox) # isobox install make

# Use the tools
(isobox) # git --version
(isobox) # exit
```

### Build Environment

```bash
# From host
mkdir ~/build-env
cd ~/build-env
isobox init
isobox enter

# Inside environment - gcc is pre-installed
(isobox) # gcc --version
(isobox) # isobox install make
(isobox) # isobox install cmake
(isobox) # isobox install git
(isobox) # exit
```

### Python Development

```bash
# From host
mkdir ~/pyproject
cd ~/pyproject
isobox init
isobox enter

# Inside environment - python3 and gcc are pre-installed
(isobox) # python3 --version
(isobox) # isobox install py3-pip
(isobox) # isobox install python3-dev
(isobox) # pip --version
(isobox) # exit
```

### Web Development

```bash
# From host
mkdir ~/webapp
cd ~/webapp
isobox init
isobox enter

# Inside environment
(isobox) # isobox install nodejs
(isobox) # isobox install npm
(isobox) # isobox install git
(isobox) # isobox install curl
(isobox) # node --version
(isobox) # npm --version
(isobox) # exit
```

## Finding Packages

Search Alpine packages online:
https://pkgs.alpinelinux.org/packages

Search for a specific package:
```
https://pkgs.alpinelinux.org/packages?name=<package>&branch=v3.18
```

Examples:
- https://pkgs.alpinelinux.org/packages?name=git&branch=v3.18
- https://pkgs.alpinelinux.org/packages?name=python3&branch=v3.18
- https://pkgs.alpinelinux.org/packages?name=nodejs&branch=v3.18

## Repository Search Order

When you install a package, the package manager:

1. Searches the **main** repository first
2. If not found, searches the **community** repository
3. If not found in either, reports an error

Example:
```bash
(isobox) # isobox install git
Installing git (host: Arch Linux)...
Downloading from Alpine Linux repository...
Fetching package list...
Downloading git-2.43.7-r0.apk...
Extracting package...
Successfully installed git
```

If package is in community repo:
```bash
(isobox) # isobox install nodejs
Installing nodejs (host: Arch Linux)...
Downloading from Alpine Linux repository...
Fetching package list...
Warning: Package not found in main repo, trying community repo...
Downloading nodejs-20.10.0-r1.apk...
Extracting package...
Successfully installed nodejs
```

## Dependency Management

### Automatic Dependency Resolution

The package manager now **automatically resolves and installs all dependencies**. When you install a package, it:

1. Downloads the package
2. Extracts the `.PKGINFO` file to read dependencies
3. Recursively installs each dependency before installing the main package
4. Skips already-installed packages to avoid duplicates
5. Prevents circular dependency loops

**Example:**
```bash
(isobox) # isobox install git
Installing git (host: Arch Linux)...
Resolving dependencies...
  Resolving dependencies for git...
  Installing dependency: pcre2
  Installing dependency: zlib
  Installing dependency: libcurl
  Resolving dependencies for libcurl...
  Installing dependency: nghttp2-libs
  Installing dependency: libssl3
  ...
  Installing git...
Successfully installed git and its dependencies
```

### Base System Dependencies

**During `isobox init`:**
- musl libc (required for all Alpine packages)
- SSL libraries (libssl3, libcrypto3)
- CA certificates (ca-certificates-bundle)
- Base libraries (zlib, pcre2, libidn2, libunistring)
- ncurses libraries (libncursesw, libformw, libmenuw, libpanelw)
- readline (command-line editing)
- Neovim runtime libraries (luv, libtermkey, libvterm, msgpack-c, tree-sitter, unibilium, musl-libintl, luajit, libuv)
- And many more common libraries

### Shared Library Dependencies

Alpine packages list runtime dependencies using `so:` notation (like `so:libluv.so.1`). The dependency resolver automatically:

1. Detects `so:*` dependencies in package metadata
2. Maps them to actual package names (e.g., `so:libluv.so.1` → `luv`)
3. Installs the corresponding package if not already installed

**Common mappings:**
- `so:libluv.so.1` → `luv`
- `so:libtermkey.so.1` → `libtermkey`
- `so:libvterm.so.0` → `libvterm`
- `so:libmsgpack-c.so.2` → `msgpack-c`
- `so:libtree-sitter.so.0` → `tree-sitter`
- `so:libunibilium.so.4` → `unibilium`
- And many more...

### No Manual Dependency Management Needed

You can now install packages directly without worrying about dependencies:

**Python development:**
```bash
(isobox) # isobox install py3-pip
# Dependencies (python3, etc.) are installed automatically
```

**Node.js development:**
```bash
(isobox) # isobox install nodejs
# Dependencies (libstdc++, c-ares, etc.) are installed automatically
```

**Git with SSH:**
```bash
(isobox) # isobox install git
# Dependencies (openssh-client, libcurl, etc.) are installed automatically
```

## Package Database Operations

### View Database from Host

```bash
cat .isobox/var/lib/ipkg/installed.json
```

### View Database from Inside Environment

```bash
(isobox) # cat /var/lib/ipkg/installed.json
```

### Format with jq

If jq is installed:
```bash
(isobox) # isobox install jq
(isobox) # cat /var/lib/ipkg/installed.json | jq
```

Output:
```json
[
  {
    "name": "git",
    "version": "latest",
    "installed": "2025-10-03T14:00:00Z"
  }
]
```

### Manual Database Editing

Not recommended, but possible:
```bash
(isobox) # vi /var/lib/ipkg/installed.json
```

The database is plain JSON, so you can manually add or remove entries if needed.

## Troubleshooting

### Package Not Found

**Error:** `Error: Package <name> not found in Alpine repositories`

**Causes:**
- Package name is misspelled
- Package doesn't exist in Alpine v3.18
- Package is in a different repository (testing, edge)

**Solutions:**
1. Check spelling of package name
2. Search Alpine package database: https://pkgs.alpinelinux.org/packages
3. Try alternative names (e.g., `python3` not `python`)
4. Check if package exists in v3.18 specifically

### Download Failed

**Error:** `Error: Failed to download package`

**Causes:**
- No internet connection
- Repository is down
- DNS not working inside environment

**Solutions:**
1. Check internet from host: `ping dl-cdn.alpinelinux.org`
2. Check DNS inside environment: `(isobox) # cat /etc/resolv.conf`
3. Verify wget/curl is working: `(isobox) # wget --version`
4. Try again later if repository is temporarily down

### Extract Failed

**Error:** `Error: Failed to extract package`

**Causes:**
- Corrupted download
- Insufficient permissions
- tar not available

**Solutions:**
1. Try reinstalling: `(isobox) # isobox remove <pkg> && isobox install <pkg>`
2. Check available space: `(isobox) # df -h`
3. Verify tar is available: `(isobox) # which tar`

### wget or curl Not Found

**Error:** `Error: wget or curl required for package installation`

**Cause:** Both wget and curl are missing from the environment

**Solution:**
This should not happen after `isobox init` because Alpine wget is automatically installed. If it does:

```bash
# From host
cd ~/myproject
isobox destroy
isobox init
```

This rebuilds the environment with all required tools.

### Package Already Installed

**Message:** `Package <name> is already installed`

**Cause:** Package exists in database

**Solution:**
If you want to reinstall:
```bash
(isobox) # isobox remove <package>
(isobox) # isobox install <package>
```

### Library Errors After Install

**Error:** `error while loading shared libraries: libfoo.so.1`

**Cause:** Package dependency was not properly resolved (rare with automatic dependency resolution)

**Solution:**
This should rarely happen with automatic dependency resolution. If it does:

1. **Check if base system is up to date:**
   ```bash
   # Exit environment and rebuild cache
   exit
   isobox recache

   # Reinitialize environment
   isobox destroy
   isobox init
   isobox enter
   ```

2. **If problem persists, manually install the missing library:**
   - Identify missing library: `(isobox) # ldd /usr/bin/<binary>`
   - Search for package providing library: https://pkgs.alpinelinux.org/contents
   - Install the library package: `(isobox) # isobox install <library-package>`

3. **Report the issue:**
   - This may indicate a missing mapping in the `so:` dependency resolver
   - The mapping can be added to `scripts/isobox-internal.sh` in the `map_so_to_package()` function

### Package Manager Not Found

**Error:** `/bin/sh: isobox: not found` when running package commands inside the environment

**Cause:** The package manager script (`/bin/isobox`) is missing from the isolated environment. This can happen if:
- The cache was created before the script was properly included
- The environment was created with an older version of IsoBox
- The base system cache is corrupted

**Solution:**

```bash
# Exit the environment if you're inside it
exit

# Rebuild the base system cache from the host
isobox recache

# Destroy and reinitialize your environment
cd ~/myproject
isobox destroy
isobox init

# Enter and verify the package manager works
isobox enter
(isobox) # isobox help
(isobox) # isobox install git
```

The `recache` command:
- Deletes the old base system cache at `~/.cache/isobox/base-system.tar.gz`
- Rebuilds it from scratch with the latest package manager script
- Ensures all future `isobox init` commands use the updated cache

## Technical Details

### Package Manager Script Location

Inside environment: `/bin/isobox`
From host: `.isobox/bin/isobox`

This is a POSIX shell script that runs inside the chroot environment.

### APK File Format

Alpine packages (.apk) are gzip-compressed tar archives containing:

```
package-name-version.apk
├── .PKGINFO          # Package metadata
├── .trigger          # Installation triggers (optional)
├── usr/
│   ├── bin/          # Executables
│   ├── lib/          # Libraries
│   └── share/        # Data files
└── etc/              # Configuration files
```

The package manager extracts the entire archive to `/` (the isolated root).

### Download URL Structure

```
https://dl-cdn.alpinelinux.org/alpine/v3.18/{repo}/x86_64/{package}.apk
```

Components:
- `v3.18` - Alpine version
- `{repo}` - Repository: `main` or `community`
- `x86_64` - Architecture
- `{package}` - Package filename with version

Example:
```
https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/git-2.43.0-r0.apk
```

### Package Discovery

The script scrapes the repository index HTML:
1. Fetch repository index page with wget/curl
2. Parse HTML for package links matching pattern: `href="<package>-[0-9]`
3. Extract the first matching package filename
4. Construct full download URL
5. Download and extract

### Database Operations

All database operations use either jq (if available) or sed:

**Add package (with jq):**
```bash
jq ". += [{\"name\": \"$pkg\", \"version\": \"latest\", \"installed\": \"$date\"}]" installed.json
```

**Add package (with sed):**
```bash
sed -i 's/\]$/,{"name":"'$pkg'","version":"latest","installed":"'$date'"}]/' installed.json
```

**Remove package (with jq):**
```bash
jq "map(select(.name != \"$pkg\"))" installed.json
```

### Cache Location

Downloaded packages are temporarily stored at:
```
/var/cache/isobox/<package>.apk
```

This cache is inside the isolated environment, so from the host:
```
.isobox/var/cache/isobox/<package>.apk
```

Packages are deleted immediately after extraction to save space.

## Limitations

### 1. No Version Pinning

Always installs the latest version from Alpine v3.18 repositories. You cannot specify a version.

### 2. No Package Verification

No GPG signature checking. Packages are downloaded over HTTPS but signatures are not verified.

### 3. Basic Removal

`isobox remove <package>` only removes the database entry. Files remain on disk.

To fully remove:
```bash
# From host
sudo rm -rf .isobox/usr/bin/<binary>
sudo rm -rf .isobox/usr/lib/<libraries>
```

### 4. No Search Command

No built-in package search. Use the Alpine package website:
https://pkgs.alpinelinux.org/packages

### 5. No Upgrade Command

No way to upgrade packages. To update a package:
```bash
(isobox) # isobox remove <package>
(isobox) # isobox install <package>
```

### 6. No Conflict Detection

If two packages provide the same file, the last one installed wins (files are overwritten).

### 7. Architecture Locked to x86_64

Only x86_64 packages are supported. No ARM, ARM64, or other architectures.

## Best Practices

### 1. Initialize Before Installing

Always run `isobox init` before entering the environment:
```bash
mkdir ~/myproject
cd ~/myproject
isobox init
isobox enter
```

### 2. Install Packages Inside Environment

Package commands only work inside the environment:
```bash
# Wrong - from host
isobox install git

# Right - from inside
isobox enter
(isobox) # isobox install git
```

### 3. Document Package Dependencies

Keep a list of packages in a script or README:
```bash
#!/bin/sh
# setup-env.sh
isobox install git
isobox install vim
isobox install make
isobox install gcc
```

### 4. One Environment Per Project

Don't share environments across projects:
```
~/project-a/.isobox    # Separate environment
~/project-b/.isobox    # Separate environment
```

### 5. Don't Commit .isobox/

Add to `.gitignore`:
```
.isobox/
```

Environments are easily recreated with `isobox init`.

### 6. Install Common Dependencies First

For development, install base tools first:
```bash
(isobox) # isobox install git
(isobox) # isobox install vim
(isobox) # isobox install make
```

Then install project-specific packages.

### 7. Check Package Availability

Before installing, verify the package exists:
```
https://pkgs.alpinelinux.org/packages?name=<package>&branch=v3.18
```

This prevents failed installations.

## Examples

### Full Python Environment

```bash
# From host
mkdir ~/pyproject
cd ~/pyproject
isobox init
isobox enter

# Inside environment - python3 and gcc are pre-installed
(isobox) # python3 --version
(isobox) # gcc --version
(isobox) # isobox install py3-pip
(isobox) # isobox install python3-dev
(isobox) # isobox install musl-dev

# Use Python
(isobox) # pip install requests
(isobox) # python3 -c "import requests; print(requests.__version__)"
(isobox) # exit
```

### Full Node.js Environment

```bash
# From host
mkdir ~/nodeapp
cd ~/nodeapp
isobox init
isobox enter

# Inside environment
(isobox) # isobox install nodejs
(isobox) # isobox install npm
(isobox) # isobox install git

# Use Node
(isobox) # node --version
(isobox) # npm --version
(isobox) # npm init -y
(isobox) # npm install express
(isobox) # exit
```

### System Administration Tools

```bash
# From host
mkdir ~/sysadmin
cd ~/sysadmin
isobox init
isobox enter

# Inside environment
(isobox) # isobox install bash
(isobox) # isobox install coreutils
(isobox) # isobox install findutils
(isobox) # isobox install grep
(isobox) # isobox install sed
(isobox) # isobox install gawk
(isobox) # isobox install curl
(isobox) # isobox install jq

# Use tools
(isobox) # bash --version
(isobox) # ls --version
(isobox) # grep --version
(isobox) # exit
```

### Build Environment for C/C++

```bash
# From host
mkdir ~/build
cd ~/build
isobox init
isobox enter

# Inside environment - gcc is pre-installed
(isobox) # gcc --version
(isobox) # isobox install g++
(isobox) # isobox install make
(isobox) # isobox install cmake
(isobox) # isobox install git
(isobox) # isobox install musl-dev

# Build a project
(isobox) # git clone https://github.com/user/project.git
(isobox) # cd project
(isobox) # make
(isobox) # exit
```

## Comparison with Other Package Managers

| Feature | IsoBox | apk | apt | npm | pip |
|---------|--------|-----|-----|-----|-----|
| Scope | Per-directory | System | System | Project | Project |
| Isolation | Complete | None | None | Partial | Partial |
| Dependencies | **Auto** | Auto | Auto | Auto | Auto |
| Database | JSON | SQLite | dpkg | package.json | Metadata |
| Size | Tiny | Small | Large | Medium | Medium |
| Language | **Go** | C | C | JavaScript | Python |
| Verification | None | GPG | GPG | SHA | SHA |
| Static Binary | **Yes** | No | No | No | No |

IsoBox package manager is designed for simplicity and environment isolation, not as a replacement for full-featured system package managers.

## Future Enhancements

Planned improvements:

1. ~~**Full dependency resolution** - Automatically install all required packages~~ **Completed**
2. ~~**Package name aliases** - Map common names to Alpine packages~~ **Completed**
3. ~~**Batch installation** - Install packages from configuration files~~ **Completed**
4. **GPG verification** - Verify package signatures
5. **Version pinning** - Install specific package versions
6. **Package search** - Built-in search: `isobox search <term>`
7. **Clean removal** - Delete files on `isobox remove`
8. **Upgrade command** - `isobox upgrade <package>`
9. **List available** - Show all available packages
10. **Package info** - Display package details
11. **Multiple architectures** - Support ARM, ARM64
12. **Custom repositories** - Add non-Alpine repositories

## Technical Implementation

The package manager is implemented in pure Go with the following architecture:

### Core Components

- **`pkg/ipkg/manager.go`**: Main package manager implementation
  - Repository searching and package discovery
  - HTTP-based package downloads
  - Tar/gzip archive extraction
  - Dependency parsing and resolution
  - JSON database management

- **`pkg/ipkg/dependencies.go`**: TOML configuration support
  - Parse dependencies.toml files
  - Batch package installation
  - Group-based package management

### Key Features

**Statically Linked Binary:**
- Built with `CGO_ENABLED=0` for static compilation
- No external dependencies required
- Works in minimal environments (musl libc compatible)
- Single ~8-10MB binary includes all functionality

**Auto-Detection:**
- Detects when running inside IsoBox by checking `/etc/os-release`
- Automatically switches between host mode and internal mode
- Uses different root paths depending on context

**Pure Go Implementation:**
- No shell script dependencies
- Standard library only (`archive/tar`, `compress/gzip`, `net/http`)
- TOML parsing via `github.com/BurntSushi/toml`

## Contributing

To improve the package manager:

1. Edit source files in `pkg/ipkg/` directory
2. Build with `make build` (creates static binary)
3. Test in a clean environment
4. Update this documentation
5. Submit a pull request

The binary is automatically copied to `.isobox/bin/isobox` during `isobox init`.
